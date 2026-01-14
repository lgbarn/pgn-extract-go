package engine

import "github.com/lgbarn/pgn-extract-go/internal/chess"

// Direction offsets for piece movement.
var (
	knightOffsets = [][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}
	kingOffsets   = [][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}
	diagonalDirs  = [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
	straightDirs  = [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
)

// IsInCheck returns true if the given colour's king is in check.
func IsInCheck(board *chess.Board, colour chess.Colour) bool {
	kingCol, kingRank := getKingPosition(board, colour)
	if kingCol == 0 {
		return false
	}
	return isSquareAttacked(board, kingCol, kingRank, colour.Opposite())
}

// getKingPosition returns the king position for the given colour.
// Returns (0, 0) if no king is found.
func getKingPosition(board *chess.Board, colour chess.Colour) (chess.Col, chess.Rank) {
	if colour == chess.White {
		if board.WKingCol != 0 && board.WKingRank != 0 {
			return board.WKingCol, board.WKingRank
		}
	} else {
		if board.BKingCol != 0 && board.BKingRank != 0 {
			return board.BKingCol, board.BKingRank
		}
	}
	return findKing(board, colour)
}

// findKing finds the king of the given colour on the board.
func findKing(board *chess.Board, colour chess.Colour) (chess.Col, chess.Rank) {
	king := chess.MakeColouredPiece(colour, chess.King)
	for col := chess.Col('a'); col <= 'h'; col++ {
		for rank := chess.Rank('1'); rank <= '8'; rank++ {
			if board.Get(col, rank) == king {
				return col, rank
			}
		}
	}
	return 0, 0
}

// isSquareAttacked returns true if the square is attacked by the given colour.
func isSquareAttacked(board *chess.Board, col chess.Col, rank chess.Rank, byColour chess.Colour) bool {
	return isPawnAttacking(board, col, rank, byColour) ||
		isKnightAttacking(board, col, rank, byColour) ||
		isKingAttacking(board, col, rank, byColour) ||
		isDiagonalAttacking(board, col, rank, byColour) ||
		isStraightAttacking(board, col, rank, byColour)
}

// isPawnAttacking checks if a pawn of the given colour attacks the square.
func isPawnAttacking(board *chess.Board, col chess.Col, rank chess.Rank, byColour chess.Colour) bool {
	pawn := chess.MakeColouredPiece(byColour, chess.Pawn)
	pawnDir := -1
	if byColour == chess.Black {
		pawnDir = 1
	}

	pawnRank := chess.Rank(int(rank) + pawnDir)
	if pawnRank < '1' || pawnRank > '8' {
		return false
	}

	if col > 'a' && board.Get(col-1, pawnRank) == pawn {
		return true
	}
	if col < 'h' && board.Get(col+1, pawnRank) == pawn {
		return true
	}
	return false
}

// isKnightAttacking checks if a knight of the given colour attacks the square.
func isKnightAttacking(board *chess.Board, col chess.Col, rank chess.Rank, byColour chess.Colour) bool {
	knight := chess.MakeColouredPiece(byColour, chess.Knight)
	for _, offset := range knightOffsets {
		c := chess.Col(int(col) + offset[0])
		r := chess.Rank(int(rank) + offset[1])
		if isOnBoard(c, r) && board.Get(c, r) == knight {
			return true
		}
	}
	return false
}

// isKingAttacking checks if a king of the given colour attacks the square.
func isKingAttacking(board *chess.Board, col chess.Col, rank chess.Rank, byColour chess.Colour) bool {
	king := chess.MakeColouredPiece(byColour, chess.King)
	for _, offset := range kingOffsets {
		c := chess.Col(int(col) + offset[0])
		r := chess.Rank(int(rank) + offset[1])
		if isOnBoard(c, r) && board.Get(c, r) == king {
			return true
		}
	}
	return false
}

// isDiagonalAttacking checks if a bishop or queen attacks along diagonals.
func isDiagonalAttacking(board *chess.Board, col chess.Col, rank chess.Rank, byColour chess.Colour) bool {
	bishop := chess.MakeColouredPiece(byColour, chess.Bishop)
	queen := chess.MakeColouredPiece(byColour, chess.Queen)
	return isSlidingAttacking(board, col, rank, diagonalDirs, bishop, queen)
}

// isStraightAttacking checks if a rook or queen attacks along straight lines.
func isStraightAttacking(board *chess.Board, col chess.Col, rank chess.Rank, byColour chess.Colour) bool {
	rook := chess.MakeColouredPiece(byColour, chess.Rook)
	queen := chess.MakeColouredPiece(byColour, chess.Queen)
	return isSlidingAttacking(board, col, rank, straightDirs, rook, queen)
}

// isSlidingAttacking checks if either attacker piece attacks along the given directions.
func isSlidingAttacking(board *chess.Board, col chess.Col, rank chess.Rank, dirs [][2]int, attacker1, attacker2 chess.Piece) bool {
	for _, dir := range dirs {
		c := chess.Col(int(col) + dir[0])
		r := chess.Rank(int(rank) + dir[1])
		for isOnBoard(c, r) {
			piece := board.Get(c, r)
			if piece != chess.Empty {
				if piece == attacker1 || piece == attacker2 {
					return true
				}
				break
			}
			c = chess.Col(int(c) + dir[0])
			r = chess.Rank(int(r) + dir[1])
		}
	}
	return false
}

// isOnBoard returns true if the coordinates are within the board bounds.
func isOnBoard(col chess.Col, rank chess.Rank) bool {
	return col >= 'a' && col <= 'h' && rank >= '1' && rank <= '8'
}
