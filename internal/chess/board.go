package chess

// Board represents a chess board with all state needed for the game.
type Board struct {
	// The board squares with a hedge of 2 around for knight move calculation.
	// board[col][rank] where col and rank are 0-11 (with hedge).
	Squares [Hedge + BoardSize + Hedge][Hedge + BoardSize + Hedge]Piece

	// Who has the next move.
	ToMove Colour

	// The current move number.
	MoveNumber uint

	// Rook starting columns for the 4 castling options.
	// This accommodates Chess960.
	WKingCastle  Col
	WQueenCastle Col
	BKingCastle  Col
	BQueenCastle Col

	// Keep track of where the two kings are for check detection.
	WKingCol  Col
	WKingRank Rank
	BKingCol  Col
	BKingRank Rank

	// Is EnPassant capture possible? If so then EPRank and EPCol have
	// the square on which this can be made.
	EnPassant bool
	EPRank    Rank
	EPCol     Col

	// Weak hash value based on a simple hashing approach.
	WeakHashValue HashCode

	// Zobrist hash value (Polyglot-compatible).
	Zobrist uint64

	// The half-move clock since the last pawn move or capture.
	HalfmoveClock uint
}

// NewBoard creates a new empty board.
func NewBoard() *Board {
	b := &Board{
		ToMove:     White,
		MoveNumber: 1,
	}
	// Initialize all squares to Off (hedge) or Empty
	for col := 0; col < Hedge+BoardSize+Hedge; col++ {
		for rank := 0; rank < Hedge+BoardSize+Hedge; rank++ {
			if col >= Hedge && col < Hedge+BoardSize &&
				rank >= Hedge && rank < Hedge+BoardSize {
				b.Squares[col][rank] = Empty
			} else {
				b.Squares[col][rank] = Off
			}
		}
	}
	return b
}

// SetupInitialPosition sets up the standard chess starting position.
func (b *Board) SetupInitialPosition() {
	// Clear the board first
	for col := Hedge; col < Hedge+BoardSize; col++ {
		for rank := Hedge; rank < Hedge+BoardSize; rank++ {
			b.Squares[col][rank] = Empty
		}
	}

	// Place white pieces (rank 1)
	backRank := []Piece{Rook, Knight, Bishop, Queen, King, Bishop, Knight, Rook}
	for col := 0; col < BoardSize; col++ {
		b.Squares[col+Hedge][Hedge] = W(backRank[col])
		b.Squares[col+Hedge][Hedge+1] = W(Pawn)
		b.Squares[col+Hedge][Hedge+6] = B(Pawn)
		b.Squares[col+Hedge][Hedge+7] = B(backRank[col])
	}

	// Set king positions
	b.WKingCol = 'e'
	b.WKingRank = '1'
	b.BKingCol = 'e'
	b.BKingRank = '8'

	// Set castling rights (standard chess)
	b.WKingCastle = 'h'  // h1 rook
	b.WQueenCastle = 'a' // a1 rook
	b.BKingCastle = 'h'  // h8 rook
	b.BQueenCastle = 'a' // a8 rook

	b.ToMove = White
	b.MoveNumber = 1
	b.EnPassant = false
	b.HalfmoveClock = 0
}

// Get returns the piece at the given coordinates (using char coords 'a'-'h', '1'-'8').
func (b *Board) Get(col Col, rank Rank) Piece {
	c := ColConvert(col)
	r := RankConvert(rank)
	if c == 0 || r == 0 {
		return Off
	}
	return b.Squares[c][r]
}

// Set places a piece at the given coordinates.
func (b *Board) Set(col Col, rank Rank, piece Piece) {
	c := ColConvert(col)
	r := RankConvert(rank)
	if c != 0 && r != 0 {
		b.Squares[c][r] = piece
	}
}

// GetByIndex returns the piece at the given board array indices.
func (b *Board) GetByIndex(col, rank int) Piece {
	return b.Squares[col][rank]
}

// SetByIndex places a piece at the given board array indices.
func (b *Board) SetByIndex(col, rank int, piece Piece) {
	b.Squares[col][rank] = piece
}

// Copy creates a deep copy of the board.
func (b *Board) Copy() *Board {
	newBoard := &Board{}
	*newBoard = *b
	return newBoard
}

// BoardState captures all mutable board state for save/restore operations.
// This is more efficient than Copy() when you need to temporarily modify
// the board and then restore it (e.g., exploring variations).
type BoardState struct {
	Squares       [Hedge + BoardSize + Hedge][Hedge + BoardSize + Hedge]Piece
	ToMove        Colour
	MoveNumber    uint
	WKingCastle   Col
	WQueenCastle  Col
	BKingCastle   Col
	BQueenCastle  Col
	WKingCol      Col
	WKingRank     Rank
	BKingCol      Col
	BKingRank     Rank
	EnPassant     bool
	EPRank        Rank
	EPCol         Col
	WeakHashValue HashCode
	Zobrist       uint64
	HalfmoveClock uint
}

// SaveState captures the current board state for later restoration.
func (b *Board) SaveState() BoardState {
	return BoardState{
		Squares:       b.Squares,
		ToMove:        b.ToMove,
		MoveNumber:    b.MoveNumber,
		WKingCastle:   b.WKingCastle,
		WQueenCastle:  b.WQueenCastle,
		BKingCastle:   b.BKingCastle,
		BQueenCastle:  b.BQueenCastle,
		WKingCol:      b.WKingCol,
		WKingRank:     b.WKingRank,
		BKingCol:      b.BKingCol,
		BKingRank:     b.BKingRank,
		EnPassant:     b.EnPassant,
		EPRank:        b.EPRank,
		EPCol:         b.EPCol,
		WeakHashValue: b.WeakHashValue,
		Zobrist:       b.Zobrist,
		HalfmoveClock: b.HalfmoveClock,
	}
}

// RestoreState restores the board to a previously saved state.
func (b *Board) RestoreState(s BoardState) {
	b.Squares = s.Squares
	b.ToMove = s.ToMove
	b.MoveNumber = s.MoveNumber
	b.WKingCastle = s.WKingCastle
	b.WQueenCastle = s.WQueenCastle
	b.BKingCastle = s.BKingCastle
	b.BQueenCastle = s.BQueenCastle
	b.WKingCol = s.WKingCol
	b.WKingRank = s.WKingRank
	b.BKingCol = s.BKingCol
	b.BKingRank = s.BKingRank
	b.EnPassant = s.EnPassant
	b.EPRank = s.EPRank
	b.EPCol = s.EPCol
	b.WeakHashValue = s.WeakHashValue
	b.Zobrist = s.Zobrist
	b.HalfmoveClock = s.HalfmoveClock
}

// MovePair represents a source-destination square pair for move generation.
type MovePair struct {
	FromCol  Col
	FromRank Rank
	ToCol    Col
	ToRank   Rank
}
