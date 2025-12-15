// Package chess provides core chess types and operations.
package chess

// Colour represents the colour of a piece or player.
type Colour int

const (
	Black Colour = iota
	White
)

// String returns the string representation of a colour.
func (c Colour) String() string {
	if c == White {
		return "White"
	}
	return "Black"
}

// Opposite returns the opposite colour.
func (c Colour) Opposite() Colour {
	if c == White {
		return Black
	}
	return White
}

// Piece represents a chess piece type.
type Piece int

const (
	Off   Piece = iota // Off the board (hedge square)
	Empty              // Empty square
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
	NumPieceValues
)

// String returns the string representation of a piece.
func (p Piece) String() string {
	names := []string{"Off", "Empty", "Pawn", "Knight", "Bishop", "Rook", "Queen", "King"}
	if int(p) < len(names) {
		return names[p]
	}
	return "Unknown"
}

// Letter returns the single letter representation of a piece (uppercase).
func (p Piece) Letter() byte {
	letters := []byte{' ', ' ', 'P', 'N', 'B', 'R', 'Q', 'K'}
	if int(p) < len(letters) {
		return letters[p]
	}
	return '?'
}

// MoveClass categorizes different types of chess moves.
type MoveClass int

const (
	PawnMove MoveClass = iota
	PawnMoveWithPromotion
	EnPassantPawnMove
	PieceMove
	KingsideCastle
	QueensideCastle
	NullMove
	UnknownMove
)

// WhoseMove indicates whose turn it is for positional matching.
type WhoseMove int

const (
	WhiteToMove WhoseMove = iota
	BlackToMove
	EitherToMove
)

// Rank represents a chess rank (row) - '1' to '8'.
type Rank byte

// Col represents a chess file (column) - 'a' to 'h'.
type Col byte

// Constants for board dimensions and coordinates.
const (
	BoardSize = 8
	Hedge     = 2 // Hedge size for knight move calculations

	RankBase  = '1'
	ColBase   = 'a'
	FirstRank = RankBase
	LastRank  = RankBase + BoardSize - 1
	FirstCol  = ColBase
	LastCol   = ColBase + BoardSize - 1
)

// RankConvert converts a rank character to a board array index.
func RankConvert(rank Rank) int {
	if rank >= FirstRank && rank <= LastRank {
		return int(rank-RankBase) + Hedge
	}
	return 0
}

// ColConvert converts a column character to a board array index.
func ColConvert(col Col) int {
	if col >= FirstCol && col <= LastCol {
		return int(col-ColBase) + Hedge
	}
	return 0
}

// ToRank converts a board array index back to a rank character.
func ToRank(r int) Rank {
	return Rank(r + int(RankBase) - Hedge)
}

// ToCol converts a board array index back to a column character.
func ToCol(c int) Col {
	return Col(c + int(ColBase) - Hedge)
}

// ColourOffset returns +1 for White, -1 for Black (for pawn direction).
func ColourOffset(colour Colour) int {
	if colour == White {
		return 1
	}
	return -1
}

// HashCode is the type for position hashing.
type HashCode uint64

// PieceShift is used for encoding coloured pieces.
const PieceShift = 3

// MakeColouredPiece creates a coloured piece value.
func MakeColouredPiece(colour Colour, piece Piece) Piece {
	return Piece((int(piece) << PieceShift) | int(colour))
}

// W creates a white piece.
func W(piece Piece) Piece {
	return MakeColouredPiece(White, piece)
}

// B creates a black piece.
func B(piece Piece) Piece {
	return MakeColouredPiece(Black, piece)
}

// ExtractColour extracts the colour from a coloured piece.
func ExtractColour(colouredPiece Piece) Colour {
	return Colour(colouredPiece & 0x01)
}

// ExtractPiece extracts the piece type from a coloured piece.
func ExtractPiece(colouredPiece Piece) Piece {
	return Piece(colouredPiece >> PieceShift)
}

// NullMoveString is the PGN representation of a null move.
const NullMoveString = "--"

// CheckStatus indicates whether a move gives check or checkmate.
type CheckStatus int

const (
	NoCheck CheckStatus = iota
	Check
	Checkmate
)

// MaxMoveLen is the maximum length of a move text string.
const MaxMoveLen = 15
