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
	MoveNumber int           `json:"moveNumber,omitempty"`
	Color      string        `json:"color"` // "white" or "black"
	SAN        string        `json:"san"`
	UCI        string        `json:"uci,omitempty"`
	From       string        `json:"from,omitempty"`
	To         string        `json:"to,omitempty"`
	Piece      string        `json:"piece,omitempty"`
	Captured   string        `json:"captured,omitempty"`
	Promotion  string        `json:"promotion,omitempty"`
	NAGs       []string      `json:"nags,omitempty"`
	Comments   []string      `json:"comments,omitempty"`
	Variations [][]JSONMove  `json:"variations,omitempty"`
	FEN        string        `json:"fen,omitempty"`
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
	enc.Encode(jsonGame)
}

// OutputGamesJSON outputs multiple games as a JSON array.
func OutputGamesJSON(games []*chess.Game, cfg *config.Config, w io.Writer) {
	output := &JSONOutput{
		Games: make([]*JSONGame, 0, len(games)),
	}

	for _, game := range games {
		jsonGame := GameToJSON(game, cfg)
		output.Games = append(output.Games, jsonGame)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(output)
}

// GameToJSON converts a chess game to JSON format.
func GameToJSON(game *chess.Game, cfg *config.Config) *JSONGame {
	jg := &JSONGame{
		Tags: make(map[string]string),
	}

	// Copy tags
	for k, v := range game.Tags {
		jg.Tags[k] = v
	}

	// Ensure seven tag roster has values
	for _, tag := range chess.SevenTagRoster {
		if _, ok := jg.Tags[tag]; !ok {
			jg.Tags[tag] = "?"
		}
	}

	// Get starting position
	var board *chess.Board
	if fen := game.GetTag("FEN"); fen != "" {
		board, _ = engine.NewBoardFromFEN(fen)
		jg.InitialFEN = fen
	}
	if board == nil {
		board = engine.NewInitialBoard()
	}

	// Convert moves
	jg.Moves = convertMoves(game.Moves, board, cfg)

	// Count plies
	plyCount := 0
	for move := game.Moves; move != nil; move = move.Next {
		plyCount++
	}
	jg.PlyCount = plyCount

	// Get result
	result := game.GetTag("Result")
	if result == "" {
		result = "*"
	}
	jg.Result = result

	// Final FEN if requested
	if cfg.OutputFENString {
		jg.FinalFEN = engine.BoardToFEN(board)
	}

	return jg
}

// convertMoves converts a move list to JSON format.
func convertMoves(moves *chess.Move, board *chess.Board, cfg *config.Config) []JSONMove {
	var result []JSONMove

	moveNum := board.MoveNumber
	isWhite := board.ToMove == chess.White

	for move := moves; move != nil; move = move.Next {
		jm := JSONMove{}

		// Move number
		if isWhite {
			jm.MoveNumber = int(moveNum)
		}

		// Color
		if isWhite {
			jm.Color = "white"
		} else {
			jm.Color = "black"
		}

		// SAN
		jm.SAN = move.Text

		// Find source square
		fromCol, fromRank := move.FromCol, move.FromRank
		if fromCol == 0 || fromRank == 0 {
			fromCol, fromRank = findSourceFromMove(move, board)
		}

		// From/To squares
		if fromCol != 0 && fromRank != 0 {
			jm.From = string([]byte{byte(fromCol), byte(fromRank)})
		}
		if move.ToCol != 0 && move.ToRank != 0 {
			jm.To = string([]byte{byte(move.ToCol), byte(move.ToRank)})
		}

		// UCI format
		jm.UCI = formatUCI(move, board)

		// Piece type
		jm.Piece = pieceTypeName(move.PieceToMove)

		// Check for capture
		captured := board.Get(move.ToCol, move.ToRank)
		if captured != chess.Empty && captured != chess.Off {
			jm.Captured = pieceTypeName(chess.ExtractPiece(captured))
		} else if move.Class == chess.EnPassantPawnMove {
			jm.Captured = "pawn"
		}

		// Promotion
		if move.Class == chess.PawnMoveWithPromotion && move.PromotedPiece != chess.Empty {
			jm.Promotion = pieceTypeName(move.PromotedPiece)
		}

		// NAGs
		if cfg.KeepNAGs && len(move.NAGs) > 0 {
			for _, nag := range move.NAGs {
				for _, text := range nag.Text {
					jm.NAGs = append(jm.NAGs, text)
				}
			}
		}

		// Comments
		if cfg.KeepComments && len(move.Comments) > 0 {
			for _, comment := range move.Comments {
				jm.Comments = append(jm.Comments, comment.Text)
			}
		}

		// Variations
		if cfg.KeepVariations && len(move.Variations) > 0 {
			for _, v := range move.Variations {
				varMoves := convertVariation(v, board.Copy(), cfg)
				if len(varMoves) > 0 {
					jm.Variations = append(jm.Variations, varMoves)
				}
			}
		}

		// Apply move to track position
		engine.ApplyMove(board, move)

		// Add FEN after move if requested
		if cfg.AddFENComments {
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

// convertVariation converts a variation to JSON format.
func convertVariation(v *chess.Variation, board *chess.Board, cfg *config.Config) []JSONMove {
	var result []JSONMove

	moveNum := board.MoveNumber
	isWhite := board.ToMove == chess.White

	for move := v.Moves; move != nil; move = move.Next {
		jm := JSONMove{}

		if isWhite {
			jm.MoveNumber = int(moveNum)
		}

		if isWhite {
			jm.Color = "white"
		} else {
			jm.Color = "black"
		}

		jm.SAN = move.Text

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

		jm.UCI = formatUCI(move, board)
		jm.Piece = pieceTypeName(move.PieceToMove)

		captured := board.Get(move.ToCol, move.ToRank)
		if captured != chess.Empty && captured != chess.Off {
			jm.Captured = pieceTypeName(chess.ExtractPiece(captured))
		}

		if move.Class == chess.PawnMoveWithPromotion && move.PromotedPiece != chess.Empty {
			jm.Promotion = pieceTypeName(move.PromotedPiece)
		}

		if cfg.KeepNAGs && len(move.NAGs) > 0 {
			for _, nag := range move.NAGs {
				for _, text := range nag.Text {
					jm.NAGs = append(jm.NAGs, text)
				}
			}
		}

		if cfg.KeepComments && len(move.Comments) > 0 {
			for _, comment := range move.Comments {
				jm.Comments = append(jm.Comments, comment.Text)
			}
		}

		// Nested variations
		if cfg.KeepVariations && len(move.Variations) > 0 {
			for _, nested := range move.Variations {
				nestedMoves := convertVariation(nested, board.Copy(), cfg)
				if len(nestedMoves) > 0 {
					jm.Variations = append(jm.Variations, nestedMoves)
				}
			}
		}

		engine.ApplyMove(board, move)

		result = append(result, jm)

		if !isWhite {
			moveNum++
		}
		isWhite = !isWhite
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
