package chess

// Comment represents a PGN comment.
type Comment struct {
	Text string
}

// NAG represents a Numeric Annotation Glyph with optional comments.
type NAG struct {
	Text     []string
	Comments []*Comment
}

// Variation represents a variation (alternative line) in a game.
type Variation struct {
	PrefixComment []*Comment
	Moves         *Move
	SuffixComment []*Comment
}

// Move represents a single chess move with all associated data.
type Move struct {
	// The move text (e.g., "Nf3", "e4", "O-O").
	Text string

	// Class of move (pawn move, piece move, castle, etc.).
	Class MoveClass

	// Source square.
	FromCol  Col
	FromRank Rank

	// Destination square.
	ToCol  Col
	ToRank Rank

	// The piece being moved.
	PieceToMove Piece

	// The piece captured (Empty if no capture).
	CapturedPiece Piece

	// The piece promoted to (Empty if not a promotion).
	PromotedPiece Piece

	// Whether this move gives check or checkmate.
	CheckStatus CheckStatus

	// EPD representation of the board immediately before this move.
	EPD string

	// The move count suffix for FEN (e.g., "1 1" for halfmove clock and fullmove).
	FENSuffix string

	// Zobrist hash code of the position after this move.
	Zobrist uint64

	// Evaluation of the position after this move.
	Evaluation float64

	// Numeric Annotation Glyphs (!, ?, !!, ??, etc.).
	NAGs []*NAG

	// Comments associated with this move.
	Comments []*Comment

	// Terminating result if this is the last move (e.g., "1-0", "0-1", "1/2-1/2").
	TerminatingResult string

	// Alternative variations from this position.
	Variations []*Variation

	// Links to previous and next moves in the game.
	Prev *Move
	Next *Move
}

// NewMove creates a new empty move.
func NewMove() *Move {
	return &Move{
		CapturedPiece: Empty,
		PromotedPiece: Empty,
		CheckStatus:   NoCheck,
	}
}

// IsCapture returns true if this move is a capture.
func (m *Move) IsCapture() bool {
	return m.CapturedPiece != Empty || m.Class == EnPassantPawnMove
}

// IsPromotion returns true if this move is a pawn promotion.
func (m *Move) IsPromotion() bool {
	return m.Class == PawnMoveWithPromotion
}

// IsCastle returns true if this move is a castling move.
func (m *Move) IsCastle() bool {
	switch m.Class {
	case KingsideCastle, QueensideCastle:
		return true
	default:
		return false
	}
}

// IsNull returns true if this is a null move.
func (m *Move) IsNull() bool {
	return m.Class == NullMove
}

// HasNAGs returns true if this move has any NAGs.
func (m *Move) HasNAGs() bool {
	return len(m.NAGs) > 0
}

// HasComments returns true if this move has any comments.
func (m *Move) HasComments() bool {
	return len(m.Comments) > 0
}

// HasVariations returns true if this move has any variations.
func (m *Move) HasVariations() bool {
	return len(m.Variations) > 0
}

// AppendComment adds a comment to this move.
func (m *Move) AppendComment(text string) {
	m.Comments = append(m.Comments, &Comment{Text: text})
}

// AppendNAG adds a NAG to this move.
func (m *Move) AppendNAG(text string) {
	m.NAGs = append(m.NAGs, &NAG{Text: []string{text}})
}

// AppendVariation adds a variation to this move.
func (m *Move) AppendVariation(v *Variation) {
	m.Variations = append(m.Variations, v)
}
