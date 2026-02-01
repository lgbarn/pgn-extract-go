// Package output provides game output formatting in various notations.
package output

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

// clockAnnotationRegex matches clock annotations like [%clk H:MM:SS] or [%clk H:MM:SS.d]
var clockAnnotationRegex = regexp.MustCompile(`\[%clk\s+\d+:\d{2}:\d{2}(?:\.\d+)?\]`)

// stripClockAnnotations removes clock annotations from comment text.
func stripClockAnnotations(text string) string {
	return strings.TrimSpace(clockAnnotationRegex.ReplaceAllString(text, ""))
}

// OutputWriter handles formatted output with line length control.
type OutputWriter struct {
	w             io.Writer
	lineLength    int
	maxLineLength int
	needsSpace    bool
}

// NewOutputWriter creates a new output writer.
func NewOutputWriter(w io.Writer, maxLineLength int) *OutputWriter {
	if maxLineLength <= 0 {
		maxLineLength = 80
	}
	return &OutputWriter{
		w:             w,
		maxLineLength: maxLineLength,
	}
}

// Write writes a string, adding a space separator if needed.
func (o *OutputWriter) Write(s string) {
	if o.needsSpace && len(s) > 0 {
		// Check if we need a new line
		if o.lineLength+1+len(s) > o.maxLineLength {
			fmt.Fprintln(o.w)
			o.lineLength = 0
			o.needsSpace = false
		} else {
			fmt.Fprint(o.w, " ")
			o.lineLength++
		}
	}

	fmt.Fprint(o.w, s)
	o.lineLength += len(s)
	o.needsSpace = true
}

// WriteNoSpace writes without adding a leading space.
func (o *OutputWriter) WriteNoSpace(s string) {
	fmt.Fprint(o.w, s)
	o.lineLength += len(s)
	o.needsSpace = true
}

// NewLine starts a new line.
func (o *OutputWriter) NewLine() {
	fmt.Fprintln(o.w)
	o.lineLength = 0
	o.needsSpace = false
}

// OutputGame outputs a game in the configured format.
func OutputGame(game *chess.Game, cfg *config.Config) {
	w := cfg.OutputFile

	// Output tags
	outputTags(game, cfg, w)

	// Blank line between tags and moves
	fmt.Fprintln(w)

	// Output moves
	outputMoves(game, cfg, w)

	// Blank line between games
	fmt.Fprintln(w)
}

// outputTags outputs the game tags.
func outputTags(game *chess.Game, cfg *config.Config, w io.Writer) {
	if cfg.Output.TagFormat == config.NoTags {
		return
	}

	// Output seven tag roster first (common to both SevenTagRoster and AllTags)
	for _, tag := range chess.SevenTagRoster {
		value := game.GetTag(tag)
		if value == "" {
			value = "?"
		}
		fmt.Fprintf(w, "[%s \"%s\"]\n", tag, escapeTagValue(value))
	}

	// Output additional tags if not restricted to seven tag roster
	if cfg.Output.TagFormat != config.SevenTagRoster {
		for tag, value := range game.Tags {
			if !chess.IsSevenTagRosterTag(tag) {
				fmt.Fprintf(w, "[%s \"%s\"]\n", tag, escapeTagValue(value))
			}
		}
	}
}

// escapeTagValue escapes special characters in tag values.
func escapeTagValue(s string) string {
	// Fast path: if no escaping needed, return original string
	if !strings.ContainsAny(s, "\\\"") {
		return s
	}
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// outputMoves outputs the game moves.
func outputMoves(game *chess.Game, cfg *config.Config, w io.Writer) {
	ow := NewOutputWriter(w, int(cfg.Output.MaxLineLength))

	// Start with initial position or FEN
	var board *chess.Board
	if fen := game.GetTag("FEN"); fen != "" {
		if b, err := engine.NewBoardFromFEN(fen); err == nil {
			board = b
		}
	}
	if board == nil {
		board = engine.NewInitialBoard()
	}

	moveNum := board.MoveNumber
	isWhite := board.ToMove == chess.White

	for move := game.Moves; move != nil; move = move.Next {
		// Output move number
		if cfg.Output.KeepMoveNumbers {
			if isWhite {
				ow.Write(fmt.Sprintf("%d.", moveNum))
			} else if move.Prev == nil {
				// Black to move at start
				ow.Write(fmt.Sprintf("%d...", moveNum))
			}
		}

		// Output the move in the configured format
		moveText := formatMove(move, board, cfg.Output.Format)
		ow.Write(moveText)

		// Output NAGs
		if cfg.Output.KeepNAGs && len(move.NAGs) > 0 {
			outputNAGs(move, ow)
		}

		// Output comments
		if cfg.Output.KeepComments {
			for _, comment := range move.Comments {
				outputComment(comment, cfg, ow, false)
			}
		}

		// Output variations
		if cfg.Output.KeepVariations {
			outputVariations(move.Variations, board, cfg, ow)
		}

		// Apply the move to track position
		engine.ApplyMove(board, move)

		if !isWhite {
			moveNum++
		}
		isWhite = !isWhite
	}

	// Output result
	if cfg.Output.KeepResults {
		result := getGameResult(game)
		ow.Write(result)
	}

	ow.NewLine()
}

// getGameResult returns the result of a game, checking terminating result first.
func getGameResult(game *chess.Game) string {
	if game.Moves != nil {
		if lastMove := game.LastMove(); lastMove != nil && lastMove.TerminatingResult != "" {
			return lastMove.TerminatingResult
		}
	}
	if result := game.GetTag("Result"); result != "" {
		return result
	}
	return "*"
}

// outputComment writes a comment, optionally stripping clock annotations.
func outputComment(comment *chess.Comment, cfg *config.Config, ow *OutputWriter, useNoSpace bool) {
	text := comment.Text
	if cfg.Output.StripClockAnnotations {
		text = stripClockAnnotations(text)
	}
	if text == "" {
		return
	}
	formatted := "{" + text + "}"
	if useNoSpace {
		ow.WriteNoSpace(formatted)
	} else {
		ow.Write(formatted)
	}
}

// outputNAGs writes NAGs for a move.
func outputNAGs(move *chess.Move, ow *OutputWriter) {
	for _, nag := range move.NAGs {
		for _, text := range nag.Text {
			ow.Write(text)
		}
	}
}

// outputVariations outputs all variations for a move.
func outputVariations(variations []*chess.Variation, board *chess.Board, cfg *config.Config, ow *OutputWriter) {
	for _, variation := range variations {
		savedState := board.SaveState()
		outputVariation(variation, board, cfg, ow)
		board.RestoreState(savedState)
	}
}

// outputVariation outputs a variation.
func outputVariation(variation *chess.Variation, board *chess.Board, cfg *config.Config, ow *OutputWriter) {
	ow.Write("(")

	// Prefix comments
	if cfg.Output.KeepComments {
		for _, comment := range variation.PrefixComment {
			outputComment(comment, cfg, ow, true)
		}
	}

	// Moves
	moveNum := board.MoveNumber
	isWhite := board.ToMove == chess.White
	first := true

	for move := variation.Moves; move != nil; move = move.Next {
		// Output move number
		if cfg.Output.KeepMoveNumbers && (isWhite || first) {
			if isWhite {
				ow.Write(fmt.Sprintf("%d.", moveNum))
			} else {
				ow.Write(fmt.Sprintf("%d...", moveNum))
			}
		}
		first = false

		// Output the move
		ow.Write(formatMove(move, board, cfg.Output.Format))

		// Output NAGs
		if cfg.Output.KeepNAGs && len(move.NAGs) > 0 {
			outputNAGs(move, ow)
		}

		// Output comments
		if cfg.Output.KeepComments {
			for _, comment := range move.Comments {
				outputComment(comment, cfg, ow, false)
			}
		}

		// Nested variations
		if cfg.Output.KeepVariations {
			outputVariations(move.Variations, board, cfg, ow)
		}

		// Apply the move
		engine.ApplyMove(board, move)

		if !isWhite {
			moveNum++
		}
		isWhite = !isWhite
	}

	// Result in variation
	if cfg.Output.KeepResults {
		if result := getVariationResult(variation); result != "" {
			ow.Write(result)
		}
	}

	ow.WriteNoSpace(")")

	// Suffix comments
	if cfg.Output.KeepComments {
		for _, comment := range variation.SuffixComment {
			outputComment(comment, cfg, ow, false)
		}
	}
}

// getVariationResult returns the terminating result of a variation, if any.
func getVariationResult(variation *chess.Variation) string {
	if variation.Moves == nil {
		return ""
	}
	lastMove := variation.Moves
	for lastMove.Next != nil {
		lastMove = lastMove.Next
	}
	return lastMove.TerminatingResult
}

// formatMove formats a move in the specified notation.
func formatMove(move *chess.Move, board *chess.Board, format config.OutputFormat) string {
	switch format {
	case config.LALG:
		return formatLongAlgebraic(move, board, false, false)
	case config.HALG:
		return formatLongAlgebraic(move, board, true, false)
	case config.ELALG:
		return formatLongAlgebraic(move, board, false, true)
	case config.UCI:
		return formatUCI(move, board)
	default:
		// SAN or Source - use original move text
		return move.Text
	}
}

// formatLongAlgebraic formats a move in long algebraic notation.
func formatLongAlgebraic(move *chess.Move, board *chess.Board, hyphenated bool, enhanced bool) string {
	switch move.Class {
	case chess.KingsideCastle:
		return "O-O"
	case chess.QueensideCastle:
		return "O-O-O"
	case chess.NullMove:
		return "--"
	}

	var sb strings.Builder

	// Piece letter for enhanced notation
	if enhanced && move.PieceToMove != chess.Pawn && move.PieceToMove != chess.Empty {
		sb.WriteByte(engine.SANPieceLetter(move.PieceToMove))
	}

	// Source square
	fromCol, fromRank := move.FromCol, move.FromRank
	if fromCol == 0 || fromRank == 0 {
		fromCol, fromRank = findSourceFromMove(move, board)
	}

	if fromCol != 0 && fromRank != 0 {
		sb.WriteByte(byte(fromCol))
		sb.WriteByte(byte(fromRank))
	}

	// Separator for hyphenated notation
	if hyphenated {
		isCapture := board.Get(move.ToCol, move.ToRank) != chess.Empty || move.Class == chess.EnPassantPawnMove
		if isCapture {
			sb.WriteByte('x')
		} else {
			sb.WriteByte('-')
		}
	}

	// Destination square
	sb.WriteByte(byte(move.ToCol))
	sb.WriteByte(byte(move.ToRank))

	// Promotion
	if move.Class == chess.PawnMoveWithPromotion && move.PromotedPiece != chess.Empty {
		sb.WriteByte('=')
		sb.WriteByte(engine.SANPieceLetter(move.PromotedPiece))
	}

	return sb.String()
}

// formatUCI formats a move in UCI notation.
func formatUCI(move *chess.Move, board *chess.Board) string {
	switch move.Class {
	case chess.KingsideCastle:
		if board.ToMove == chess.White {
			return "e1g1"
		}
		return "e8g8"
	case chess.QueensideCastle:
		if board.ToMove == chess.White {
			return "e1c1"
		}
		return "e8c8"
	case chess.NullMove:
		return "0000"
	}

	var sb strings.Builder

	fromCol := move.FromCol
	fromRank := move.FromRank
	if fromCol == 0 || fromRank == 0 {
		fromCol, fromRank = findSourceFromMove(move, board)
	}

	sb.WriteByte(byte(fromCol))
	sb.WriteByte(byte(fromRank))
	sb.WriteByte(byte(move.ToCol))
	sb.WriteByte(byte(move.ToRank))

	// Promotion (lowercase in UCI)
	if move.Class == chess.PawnMoveWithPromotion && move.PromotedPiece != chess.Empty {
		sb.WriteByte(engine.SANPieceLetter(move.PromotedPiece) + 'a' - 'A')
	}

	return sb.String()
}

// findSourceFromMove attempts to find the source square from a move.
func findSourceFromMove(move *chess.Move, board *chess.Board) (chess.Col, chess.Rank) {
	// This is a simplified version - the engine has more complete logic
	if move.FromCol != 0 && move.FromRank != 0 {
		return move.FromCol, move.FromRank
	}

	colour := board.ToMove
	pieceType := move.PieceToMove
	toCol := move.ToCol
	toRank := move.ToRank

	if pieceType == chess.Empty || pieceType == chess.Pawn {
		// Pawn move
		return findPawnSource(board, move, colour)
	}

	// Piece move - search for the piece
	piece := chess.MakeColouredPiece(colour, pieceType)
	for col := chess.Col('a'); col <= 'h'; col++ {
		for rank := chess.Rank('1'); rank <= '8'; rank++ {
			if board.Get(col, rank) == piece {
				if move.FromCol != 0 && col != move.FromCol {
					continue
				}
				if move.FromRank != 0 && rank != move.FromRank {
					continue
				}
				if canPieceReach(pieceType, col, rank, toCol, toRank, board) {
					return col, rank
				}
			}
		}
	}

	return 0, 0
}

// findPawnSource finds the source of a pawn move.
func findPawnSource(board *chess.Board, move *chess.Move, colour chess.Colour) (chess.Col, chess.Rank) {
	toCol := move.ToCol
	toRank := move.ToRank
	fromCol := move.FromCol

	pawn := chess.MakeColouredPiece(colour, chess.Pawn)
	direction := chess.ColourOffset(colour)

	if fromCol != 0 {
		// Capture
		fromRank := chess.Rank(byte(toRank) - byte(direction))
		if board.Get(fromCol, fromRank) == pawn {
			return fromCol, fromRank
		}
		return 0, 0
	}

	// Non-capture
	fromRank := chess.Rank(byte(toRank) - byte(direction))
	if board.Get(toCol, fromRank) == pawn {
		return toCol, fromRank
	}

	// Double push
	if (colour == chess.White && toRank == '4') || (colour == chess.Black && toRank == '5') {
		fromRank = chess.Rank(byte(toRank) - byte(2*direction))
		if board.Get(toCol, fromRank) == pawn {
			return toCol, fromRank
		}
	}

	return 0, 0
}

// canPieceReach checks if a piece can reach a square.
func canPieceReach(pieceType chess.Piece, fromCol chess.Col, fromRank chess.Rank, toCol chess.Col, toRank chess.Rank, board *chess.Board) bool {
	colDiff := abs(int(toCol) - int(fromCol))
	rankDiff := abs(int(toRank) - int(fromRank))

	switch pieceType {
	case chess.Knight:
		return (colDiff == 1 && rankDiff == 2) || (colDiff == 2 && rankDiff == 1)
	case chess.Bishop:
		return colDiff == rankDiff && isDiagonalClear(board, fromCol, fromRank, toCol, toRank)
	case chess.Rook:
		return (colDiff == 0 || rankDiff == 0) && isStraightClear(board, fromCol, fromRank, toCol, toRank)
	case chess.Queen:
		if colDiff == rankDiff {
			return isDiagonalClear(board, fromCol, fromRank, toCol, toRank)
		}
		if colDiff == 0 || rankDiff == 0 {
			return isStraightClear(board, fromCol, fromRank, toCol, toRank)
		}
		return false
	case chess.King:
		return colDiff <= 1 && rankDiff <= 1
	}
	return false
}

func isDiagonalClear(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, toCol chess.Col, toRank chess.Rank) bool {
	colDir := sign(int(toCol) - int(fromCol))
	rankDir := sign(int(toRank) - int(fromRank))

	col := chess.Col(int(fromCol) + colDir)
	rank := chess.Rank(int(fromRank) + rankDir)

	for col != toCol {
		if board.Get(col, rank) != chess.Empty {
			return false
		}
		col = chess.Col(int(col) + colDir)
		rank = chess.Rank(int(rank) + rankDir)
	}
	return true
}

func isStraightClear(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, toCol chess.Col, toRank chess.Rank) bool {
	colDir := sign(int(toCol) - int(fromCol))
	rankDir := sign(int(toRank) - int(fromRank))

	col := chess.Col(int(fromCol) + colDir)
	rank := chess.Rank(int(fromRank) + rankDir)

	for col != toCol || rank != toRank {
		if board.Get(col, rank) != chess.Empty {
			return false
		}
		col = chess.Col(int(col) + colDir)
		rank = chess.Rank(int(rank) + rankDir)
	}
	return true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sign(x int) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}
