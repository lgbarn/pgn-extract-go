package cql

import (
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// evalPiece checks if a specific piece type is on specific squares.
func (e *Evaluator) evalPiece(args []Node) bool {
	if len(args) < 2 {
		return false
	}

	pieceArg, ok := args[0].(*PieceNode)
	if !ok {
		return false
	}

	squareArg, ok := args[1].(*SquareNode)
	if !ok {
		return false
	}

	squares := e.parseSquareSet(squareArg.Designator)
	if len(squares) == 0 {
		return false
	}

	pieces := e.parsePieceDesignator(pieceArg.Designator)

	for _, sq := range squares {
		piece := e.getPieceAt(sq.col, sq.rank)
		if containsPiece(pieces, piece) {
			return true
		}
	}

	return false
}

// evalCount counts pieces matching the designator on the board.
func (e *Evaluator) evalCount(args []Node) int {
	if len(args) < 1 {
		return 0
	}

	pieceArg, ok := args[0].(*PieceNode)
	if !ok {
		return 0
	}

	pieces := e.parsePieceDesignator(pieceArg.Designator)
	count := 0

	for rank := chess.Rank(0); rank < 8; rank++ {
		for col := chess.Col(0); col < 8; col++ {
			if containsPiece(pieces, e.getPieceAt(col, rank)) {
				count++
			}
		}
	}

	return count
}

// evalMaterial calculates the material value for one side.
// Standard values: P=1, N=3, B=3, R=5, Q=9
func (e *Evaluator) evalMaterial(args []Node) int {
	if len(args) < 1 {
		return 0
	}

	// Get the color argument (can be string "white"/"black" or filter node)
	var color string
	switch arg := args[0].(type) {
	case *StringNode:
		color = arg.Value
	case *FilterNode:
		color = arg.Name
	default:
		return 0
	}

	var targetColour chess.Colour
	switch color {
	case "white":
		targetColour = chess.White
	case "black":
		targetColour = chess.Black
	default:
		return 0
	}

	material := 0
	for rank := chess.Rank(0); rank < 8; rank++ {
		for col := chess.Col(0); col < 8; col++ {
			piece := e.getPieceAt(col, rank)
			if piece == chess.Empty {
				continue
			}

			pieceColour := chess.ExtractColour(piece)
			if pieceColour != targetColour {
				continue
			}

			pieceType := chess.ExtractPiece(piece)
			switch pieceType {
			case chess.Pawn:
				material++
			case chess.Knight, chess.Bishop:
				material += 3
			case chess.Rook:
				material += 5
			case chess.Queen:
				material += 9
			}
		}
	}

	return material
}

// parsePieceDesignator parses a piece designator string into a list of pieces.
func (e *Evaluator) parsePieceDesignator(desig string) []chess.Piece {
	var pieces []chess.Piece

	// Handle piece sets like [RQ]
	if strings.HasPrefix(desig, "[") && strings.HasSuffix(desig, "]") {
		inner := desig[1 : len(desig)-1]
		for _, c := range inner {
			pieces = append(pieces, e.charToPieces(byte(c))...)
		}
		return pieces
	}

	// Single character designator
	if len(desig) == 1 {
		return e.charToPieces(desig[0])
	}

	return pieces
}

// charToPieces converts a character to a list of chess pieces.
func (e *Evaluator) charToPieces(c byte) []chess.Piece {
	switch c {
	case 'K':
		return []chess.Piece{chess.W(chess.King)}
	case 'Q':
		return []chess.Piece{chess.W(chess.Queen)}
	case 'R':
		return []chess.Piece{chess.W(chess.Rook)}
	case 'B':
		return []chess.Piece{chess.W(chess.Bishop)}
	case 'N':
		return []chess.Piece{chess.W(chess.Knight)}
	case 'P':
		return []chess.Piece{chess.W(chess.Pawn)}
	case 'k':
		return []chess.Piece{chess.B(chess.King)}
	case 'q':
		return []chess.Piece{chess.B(chess.Queen)}
	case 'r':
		return []chess.Piece{chess.B(chess.Rook)}
	case 'b':
		return []chess.Piece{chess.B(chess.Bishop)}
	case 'n':
		return []chess.Piece{chess.B(chess.Knight)}
	case 'p':
		return []chess.Piece{chess.B(chess.Pawn)}
	case 'A':
		// Any white piece
		return []chess.Piece{chess.W(chess.King), chess.W(chess.Queen), chess.W(chess.Rook), chess.W(chess.Bishop), chess.W(chess.Knight), chess.W(chess.Pawn)}
	case 'a':
		// Any black piece
		return []chess.Piece{chess.B(chess.King), chess.B(chess.Queen), chess.B(chess.Rook), chess.B(chess.Bishop), chess.B(chess.Knight), chess.B(chess.Pawn)}
	case '_':
		// Empty square
		return []chess.Piece{chess.Empty}
	case '?':
		// Any piece or empty
		return []chess.Piece{
			chess.Empty,
			chess.W(chess.King), chess.W(chess.Queen), chess.W(chess.Rook), chess.W(chess.Bishop), chess.W(chess.Knight), chess.W(chess.Pawn),
			chess.B(chess.King), chess.B(chess.Queen), chess.B(chess.Rook), chess.B(chess.Bishop), chess.B(chess.Knight), chess.B(chess.Pawn),
		}
	}
	return nil
}

// getPieceAt returns the piece at the given board coordinates.
func (e *Evaluator) getPieceAt(col chess.Col, rank chess.Rank) chess.Piece {
	// Board uses hedged 12x12 array with offset 2
	return e.board.Squares[col+chess.Hedge][rank+chess.Hedge]
}

// containsPiece checks if a piece is in the given list.
func containsPiece(pieces []chess.Piece, piece chess.Piece) bool {
	for _, p := range pieces {
		if p == piece {
			return true
		}
	}
	return false
}
