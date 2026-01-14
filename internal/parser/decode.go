package parser

import (
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// isCol returns true if c is a valid column (file) character.
func isCol(c byte) bool {
	return c >= chess.FirstCol && c <= chess.LastCol
}

// isRank returns true if c is a valid rank character.
func isRank(c byte) bool {
	return c >= chess.FirstRank && c <= chess.LastRank
}

// isPiece returns the piece type represented by the character(s) at the start of move.
func isPiece(move string) chess.Piece {
	if len(move) == 0 {
		return chess.Empty
	}

	switch move[0] {
	case 'K', 'k':
		return chess.King
	case 'Q', 'q', 'D': // D = Dutch/German Queen
		return chess.Queen
	case 'R', 'r', 'T': // T = Dutch/German Rook
		return chess.Rook
	case 'N', 'n', 'P', 'S': // P = Dutch Knight, S = German Knight
		return chess.Knight
	case 'B', 'L': // L = Dutch/German Bishop
		// Note: lowercase 'b' is most likely a pawn reference
		return chess.Bishop
	case RussianQueen:
		return chess.Queen
	case RussianRook:
		return chess.Rook
	case RussianBishop:
		return chess.Bishop
	case RussianKnightOrKing:
		// Check for two-character Russian King
		if len(move) > 1 && move[1] == RussianKingSecondLetter {
			return chess.King
		}
		return chess.Knight
	}
	return chess.Empty
}

// isCapture returns true if c is a capture or separator character.
func isCapture(c byte) bool {
	return c == 'x' || c == 'X' || c == ':' || c == '-'
}

// isCastlingChar returns true if c is a castling character.
func isCastlingChar(c byte) bool {
	return c == 'O' || c == '0' || c == 'o'
}

// isCheck returns true if c is a check indicator.
func isCheck(c byte) bool {
	return c == '+' || c == '#'
}

// moveDecoder holds state for parsing a move string.
type moveDecoder struct {
	input         string
	pos           int
	fromRank      chess.Rank
	toRank        chess.Rank
	fromCol       chess.Col
	toCol         chess.Col
	class         chess.MoveClass
	pieceToMove   chess.Piece
	promotedPiece chess.Piece
	ok            bool
}

func newMoveDecoder(input string) *moveDecoder {
	return &moveDecoder{input: input, ok: true}
}

func (d *moveDecoder) currentChar() byte {
	if d.pos >= len(d.input) {
		return 0
	}
	return d.input[d.pos]
}

func (d *moveDecoder) advance() {
	if d.pos < len(d.input) {
		d.pos++
	}
}

func (d *moveDecoder) remaining() string {
	if d.pos >= len(d.input) {
		return ""
	}
	return d.input[d.pos:]
}

func (d *moveDecoder) skipCapture() {
	if isCapture(d.currentChar()) {
		d.advance()
	}
}

// DecodeMove parses a move string and returns a Move structure with decoded information.
func DecodeMove(moveString string) *chess.Move {
	d := newMoveDecoder(moveString)
	d.decode()

	move := chess.NewMove()
	move.Text = moveString
	move.Class = d.class
	move.PieceToMove = d.pieceToMove
	move.PromotedPiece = d.promotedPiece
	move.FromCol = d.fromCol
	move.FromRank = d.fromRank
	move.ToCol = d.toCol
	move.ToRank = d.toRank

	return move
}

func (d *moveDecoder) decode() {
	switch {
	case isCol(d.currentChar()):
		d.decodePawnMove()
	case isPiece(d.remaining()) != chess.Empty:
		d.decodePieceMove()
	case isCastlingChar(d.currentChar()):
		d.decodeCastling()
	case d.input == chess.NullMoveString:
		d.class = chess.NullMove
	default:
		d.ok = false
	}

	d.validateTrailing()

	if !d.ok {
		d.class = chess.UnknownMove
	}
}

func (d *moveDecoder) decodePawnMove() {
	d.class = chess.PawnMove
	d.pieceToMove = chess.Pawn
	col := chess.Col(d.currentChar())
	d.advance()

	if isRank(d.currentChar()) {
		d.decodePawnMoveWithRank(col)
	} else {
		d.decodePawnCapture(col)
	}

	if d.ok {
		d.checkPromotion()
	}
}

func (d *moveDecoder) decodePawnMoveWithRank(col chess.Col) {
	rank := chess.Rank(d.currentChar())
	d.advance()
	d.skipCapture()

	if isCol(d.currentChar()) {
		d.fromCol = col
		d.fromRank = rank
		d.toCol = chess.Col(d.currentChar())
		d.advance()

		if isRank(d.currentChar()) {
			d.toRank = chess.Rank(d.currentChar())
			d.advance()
		}
	} else {
		d.toCol = col
		d.toRank = rank
	}
}

func (d *moveDecoder) decodePawnCapture(col chess.Col) {
	d.skipCapture()

	if !isCol(d.currentChar()) {
		d.ok = false
		return
	}

	d.fromCol = col
	d.toCol = chess.Col(d.currentChar())
	d.advance()

	if isRank(d.currentChar()) {
		d.toRank = chess.Rank(d.currentChar())
		d.advance()
	}

	// Sanity check: from column must be adjacent to target column (or 'b' for ambiguity)
	if d.fromCol != 'b' && !d.isAdjacentCol(d.fromCol, d.toCol) {
		d.ok = false
	}
}

func (d *moveDecoder) isAdjacentCol(from, to chess.Col) bool {
	return from == chess.Col(byte(to)+1) || from == chess.Col(byte(to)-1)
}

func (d *moveDecoder) checkPromotion() {
	if d.currentChar() == '=' {
		d.advance()
	}

	if piece := isPiece(d.remaining()); piece != chess.Empty {
		d.class = chess.PawnMoveWithPromotion
		d.promotedPiece = piece
		d.advance()
	} else if d.currentChar() == 'b' {
		d.class = chess.PawnMoveWithPromotion
		d.promotedPiece = chess.Bishop
		d.advance()
	}
}

func (d *moveDecoder) decodePieceMove() {
	d.pieceToMove = isPiece(d.remaining())
	d.class = chess.PieceMove

	// Handle two-character Russian King
	if d.currentChar() == RussianKnightOrKing && d.pieceToMove == chess.King {
		d.advance()
	}
	d.advance()

	if isRank(d.currentChar()) {
		d.decodePieceMoveWithDisambiguatingRank()
	} else {
		d.decodePieceMoveStandard()
	}
}

func (d *moveDecoder) decodePieceMoveWithDisambiguatingRank() {
	d.fromRank = chess.Rank(d.currentChar())
	d.advance()
	d.skipCapture()

	if !isCol(d.currentChar()) {
		d.ok = false
		return
	}

	d.toCol = chess.Col(d.currentChar())
	d.advance()

	if isRank(d.currentChar()) {
		d.toRank = chess.Rank(d.currentChar())
		d.advance()
	}
}

func (d *moveDecoder) decodePieceMoveStandard() {
	if isCapture(d.currentChar()) {
		d.advance()
		d.decodeTargetSquare()
		return
	}

	if !isCol(d.currentChar()) {
		d.ok = false
		return
	}

	col := chess.Col(d.currentChar())
	d.advance()
	d.skipCapture()

	if isRank(d.currentChar()) {
		d.decodePieceMoveWithColAndRank(col)
	} else if isCol(d.currentChar()) {
		d.decodePieceMoveWithDisambiguatingCol(col)
	} else {
		d.ok = false
	}
}

func (d *moveDecoder) decodeTargetSquare() {
	if !isCol(d.currentChar()) {
		d.ok = false
		return
	}

	d.toCol = chess.Col(d.currentChar())
	d.advance()

	if !isRank(d.currentChar()) {
		d.ok = false
		return
	}

	d.toRank = chess.Rank(d.currentChar())
	d.advance()
}

func (d *moveDecoder) decodePieceMoveWithColAndRank(col chess.Col) {
	rank := chess.Rank(d.currentChar())
	d.advance()
	d.skipCapture()

	if isCol(d.currentChar()) {
		// Full coordinates: Re1d1
		d.fromCol = col
		d.fromRank = rank
		d.toCol = chess.Col(d.currentChar())
		d.advance()

		if !isRank(d.currentChar()) {
			d.ok = false
			return
		}
		d.toRank = chess.Rank(d.currentChar())
		d.advance()
	} else {
		// Simple move: Re1
		d.toCol = col
		d.toRank = rank
	}
}

func (d *moveDecoder) decodePieceMoveWithDisambiguatingCol(col chess.Col) {
	d.fromCol = col
	d.toCol = chess.Col(d.currentChar())
	d.advance()

	if !isRank(d.currentChar()) {
		d.ok = false
		return
	}
	d.toRank = chess.Rank(d.currentChar())
	d.advance()
}

func (d *moveDecoder) decodeCastling() {
	d.advance()

	if d.currentChar() == '-' {
		d.advance()
	}

	if !isCastlingChar(d.currentChar()) {
		d.ok = false
		return
	}

	d.advance()
	if d.currentChar() == '-' {
		d.advance()
	}

	if isCastlingChar(d.currentChar()) {
		d.class = chess.QueensideCastle
		d.advance()
	} else {
		d.class = chess.KingsideCastle
	}
	d.pieceToMove = chess.King
}

func (d *moveDecoder) validateTrailing() {
	if !d.ok || d.class == chess.NullMove {
		return
	}

	// Skip trailing check symbols
	for isCheck(d.currentChar()) {
		d.advance()
	}

	if d.currentChar() == 0 {
		return
	}

	// Check for en passant notation
	remaining := d.remaining()
	if d.class == chess.PawnMove && (strings.HasSuffix(remaining, "ep") || strings.HasSuffix(remaining, "e.p.")) {
		d.class = chess.EnPassantPawnMove
		return
	}

	d.ok = false
}

// DecodeAlgebraic refines move details using board context.
func DecodeAlgebraic(move *chess.Move, board *chess.Board) *chess.Move {
	fromR := chess.RankConvert(move.FromRank)
	fromC := chess.ColConvert(move.FromCol)

	if fromR == 0 || fromC == 0 {
		return move
	}

	colouredPiece := board.GetByIndex(fromC, fromR)
	pieceToMove := chess.ExtractPiece(colouredPiece)

	if pieceToMove == chess.Empty {
		return move
	}

	// Check for castling (king moving from e-file)
	if pieceToMove == chess.King && move.FromCol == 'e' {
		switch move.ToCol {
		case 'g':
			move.Class = chess.KingsideCastle
		case 'c':
			move.Class = chess.QueensideCastle
		default:
			move.Class = chess.PieceMove
			move.PieceToMove = pieceToMove
		}
		return move
	}

	// Standard move classification
	if pieceToMove == chess.Pawn {
		move.Class = chess.PawnMove
	} else {
		move.Class = chess.PieceMove
	}
	move.PieceToMove = pieceToMove

	return move
}
