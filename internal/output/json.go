package output

import (
	"encoding/json"
	"io"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

// JSONGame represents a game in JSON format.
type JSONGame struct {
	Tags       map[string]string `json:"tags"`
	Moves      []JSONMove        `json:"moves,omitempty"`
	Result     string            `json:"result,omitempty"`
	PlyCount   int               `json:"plyCount,omitempty"`
	FinalFEN   string            `json:"finalFEN,omitempty"`
	InitialFEN string            `json:"initialFEN,omitempty"`
}

// JSONMove represents a move in JSON format.
type JSONMove struct {
	MoveNumber int          `json:"moveNumber,omitempty"`
	Color      string       `json:"color"` // "white" or "black"
	SAN        string       `json:"san"`
	UCI        string       `json:"uci,omitempty"`
	From       string       `json:"from,omitempty"`
	To         string       `json:"to,omitempty"`
	Piece      string       `json:"piece,omitempty"`
	Captured   string       `json:"captured,omitempty"`
	Promotion  string       `json:"promotion,omitempty"`
	NAGs       []string     `json:"nags,omitempty"`
	Comments   []string     `json:"comments,omitempty"`
	Variations [][]JSONMove `json:"variations,omitempty"`
	FEN        string       `json:"fen,omitempty"`
}

// JSONOutput holds multiple games for array output.
type JSONOutput struct {
	Games []*JSONGame `json:"games"`
}

// OutputGameJSON outputs a single game in JSON format.
func OutputGameJSON(game *chess.Game, cfg *config.Config) {
	jsonGame := GameToJSON(game, cfg)
	enc := json.NewEncoder(cfg.OutputFile)
	enc.SetIndent("", "  ")
	enc.Encode(jsonGame) //nolint:gosec // G104: error handled via writer
}

// OutputGamesJSON outputs multiple games as a JSON array.
func OutputGamesJSON(games []*chess.Game, cfg *config.Config, w io.Writer) {
	jsonGames := make([]*JSONGame, len(games))
	for i, game := range games {
		jsonGames[i] = GameToJSON(game, cfg)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(&JSONOutput{Games: jsonGames}) //nolint:gosec // G104: error handled via writer
}

// GameToJSON converts a chess game to JSON format.
func GameToJSON(game *chess.Game, cfg *config.Config) *JSONGame {
	jg := &JSONGame{
		Tags: copyTags(game.Tags),
	}

	// Get starting position
	board, initialFEN := getInitialBoard(game)
	jg.InitialFEN = initialFEN

	// Convert moves and count plies
	jg.Moves = convertMoveList(game.Moves, board, cfg, true)
	jg.PlyCount = countPlies(game.Moves)

	// Get result
	if result := game.GetTag("Result"); result != "" {
		jg.Result = result
	} else {
		jg.Result = "*"
	}

	// Final FEN if requested
	if cfg.Annotation.OutputFEN {
		jg.FinalFEN = engine.BoardToFEN(board)
	}

	return jg
}

// copyTags copies game tags and ensures seven tag roster has values.
func copyTags(tags map[string]string) map[string]string {
	result := make(map[string]string, len(tags)+len(chess.SevenTagRoster))
	for k, v := range tags {
		result[k] = v
	}
	for _, tag := range chess.SevenTagRoster {
		if _, ok := result[tag]; !ok {
			result[tag] = "?"
		}
	}
	return result
}

// getInitialBoard returns the starting board and initial FEN (if any).
func getInitialBoard(game *chess.Game) (*chess.Board, string) {
	if fen := game.GetTag("FEN"); fen != "" {
		board, err := engine.NewBoardFromFEN(fen)
		if err == nil && board != nil {
			return board, fen
		}
	}
	return engine.NewInitialBoard(), ""
}

// countPlies counts the number of moves in a move list.
func countPlies(moves *chess.Move) int {
	count := 0
	for move := moves; move != nil; move = move.Next {
		count++
	}
	return count
}

// convertMoveList converts a move list to JSON format.
// The includeFEN parameter controls whether FEN is added after each move.
func convertMoveList(moves *chess.Move, board *chess.Board, cfg *config.Config, includeFEN bool) []JSONMove {
	result := make([]JSONMove, 0, 80) // Preallocate for typical game length

	moveNum := board.MoveNumber
	isWhite := board.ToMove == chess.White

	for move := moves; move != nil; move = move.Next {
		jm := convertSingleMove(move, board, cfg, moveNum, isWhite)

		// Add FEN after move if requested (only for main line)
		if includeFEN && cfg.Annotation.AddFENComments {
			jm.FEN = engine.BoardToFEN(board)
		}

		result = append(result, jm)

		if !isWhite {
			moveNum++
		}
		isWhite = !isWhite
	}

	return result
}

// convertSingleMove converts a single move to JSON format and applies it to the board.
func convertSingleMove(move *chess.Move, board *chess.Board, cfg *config.Config, moveNum uint, isWhite bool) JSONMove {
	jm := JSONMove{
		SAN:   move.Text,
		Color: colorName(isWhite),
		Piece: pieceTypeName(move.PieceToMove),
		UCI:   formatUCI(move, board),
	}

	if isWhite {
		jm.MoveNumber = int(moveNum)
	}

	// Source and destination squares
	fromCol, fromRank := move.FromCol, move.FromRank
	if fromCol == 0 || fromRank == 0 {
		fromCol, fromRank = findSourceFromMove(move, board)
	}
	if fromCol != 0 && fromRank != 0 {
		jm.From = string([]byte{byte(fromCol), byte(fromRank)})
	}
	if move.ToCol != 0 && move.ToRank != 0 {
		jm.To = string([]byte{byte(move.ToCol), byte(move.ToRank)})
	}

	// Captured piece
	jm.Captured = getCapturedPiece(move, board)

	// Promotion
	if move.Class == chess.PawnMoveWithPromotion && move.PromotedPiece != chess.Empty {
		jm.Promotion = pieceTypeName(move.PromotedPiece)
	}

	// NAGs
	if cfg.Output.KeepNAGs {
		jm.NAGs = collectNAGs(move)
	}

	// Comments
	if cfg.Output.KeepComments {
		jm.Comments = collectComments(move)
	}

	// Variations
	if cfg.Output.KeepVariations {
		jm.Variations = convertVariationsJSON(move.Variations, board, cfg)
	}

	// Apply move to track position
	engine.ApplyMove(board, move)

	return jm
}

// colorName returns "white" or "black" based on the boolean.
func colorName(isWhite bool) string {
	if isWhite {
		return "white"
	}
	return "black"
}

// getCapturedPiece returns the name of the captured piece, if any.
func getCapturedPiece(move *chess.Move, board *chess.Board) string {
	captured := board.Get(move.ToCol, move.ToRank)
	if captured != chess.Empty && captured != chess.Off {
		return pieceTypeName(chess.ExtractPiece(captured))
	}
	if move.Class == chess.EnPassantPawnMove {
		return "pawn"
	}
	return ""
}

// collectNAGs collects all NAG strings from a move.
func collectNAGs(move *chess.Move) []string {
	if len(move.NAGs) == 0 {
		return nil
	}
	var result []string
	for _, nag := range move.NAGs {
		result = append(result, nag.Text...)
	}
	return result
}

// collectComments collects all comment strings from a move.
func collectComments(move *chess.Move) []string {
	if len(move.Comments) == 0 {
		return nil
	}
	result := make([]string, len(move.Comments))
	for i, comment := range move.Comments {
		result[i] = comment.Text
	}
	return result
}

// convertVariationsJSON converts all variations of a move to JSON format.
func convertVariationsJSON(variations []*chess.Variation, board *chess.Board, cfg *config.Config) [][]JSONMove {
	if len(variations) == 0 {
		return nil
	}
	var result [][]JSONMove
	for _, v := range variations {
		varMoves := convertMoveList(v.Moves, board.Copy(), cfg, false)
		if len(varMoves) > 0 {
			result = append(result, varMoves)
		}
	}
	return result
}

// pieceTypeName returns the piece type as a string.
func pieceTypeName(p chess.Piece) string {
	switch p {
	case chess.Pawn:
		return "pawn"
	case chess.Knight:
		return "knight"
	case chess.Bishop:
		return "bishop"
	case chess.Rook:
		return "rook"
	case chess.Queen:
		return "queen"
	case chess.King:
		return "king"
	default:
		return ""
	}
}
