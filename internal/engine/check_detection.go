package engine

import "github.com/lgbarn/pgn-extract-go/internal/chess"

// IsInCheck returns true if the given colour's king is in check.
func IsInCheck(board *chess.Board, colour chess.Colour) bool {
	// Find the king
	var kingCol chess.Col
	var kingRank chess.Rank
	if colour == chess.White {
		kingCol = board.WKingCol
		kingRank = board.WKingRank
	} else {
		kingCol = board.BKingCol
		kingRank = board.BKingRank
	}

	// If king position not tracked, search for it
	if kingCol == 0 || kingRank == 0 {
		kingCol, kingRank = findKing(board, colour)
		if kingCol == 0 {
			return false // No king found
		}
	}

	return isSquareAttacked(board, kingCol, kingRank, colour.Opposite())
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
	// Check pawn attacks
	pawn := chess.MakeColouredPiece(byColour, chess.Pawn)
	var pawnDir int
	if byColour == chess.White {
		pawnDir = -1 // White pawns attack from below
	} else {
		pawnDir = 1 // Black pawns attack from above
	}
	pawnRank := chess.Rank(int(rank) + pawnDir)
	if pawnRank >= '1' && pawnRank <= '8' {
		if col > 'a' && board.Get(col-1, pawnRank) == pawn {
			return true
		}
		if col < 'h' && board.Get(col+1, pawnRank) == pawn {
			return true
		}
	}

	// Check knight attacks
	knight := chess.MakeColouredPiece(byColour, chess.Knight)
	knightMoves := [][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}
	for _, move := range knightMoves {
		c := chess.Col(int(col) + move[0])
		r := chess.Rank(int(rank) + move[1])
		if c >= 'a' && c <= 'h' && r >= '1' && r <= '8' {
			if board.Get(c, r) == knight {
				return true
			}
		}
	}

	// Check king attacks
	king := chess.MakeColouredPiece(byColour, chess.King)
	for dc := -1; dc <= 1; dc++ {
		for dr := -1; dr <= 1; dr++ {
			if dc == 0 && dr == 0 {
				continue
			}
			c := chess.Col(int(col) + dc)
			r := chess.Rank(int(rank) + dr)
			if c >= 'a' && c <= 'h' && r >= '1' && r <= '8' {
				if board.Get(c, r) == king {
					return true
				}
			}
		}
	}

	// Check sliding pieces (bishop, rook, queen) along diagonals
	bishop := chess.MakeColouredPiece(byColour, chess.Bishop)
	queen := chess.MakeColouredPiece(byColour, chess.Queen)
	diagonalDirs := [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
	for _, dir := range diagonalDirs {
		c := chess.Col(int(col) + dir[0])
		r := chess.Rank(int(rank) + dir[1])
		for c >= 'a' && c <= 'h' && r >= '1' && r <= '8' {
			piece := board.Get(c, r)
			if piece != chess.Empty {
				if piece == bishop || piece == queen {
					return true
				}
				break // Blocked
			}
			c = chess.Col(int(c) + dir[0])
			r = chess.Rank(int(r) + dir[1])
		}
	}

	// Check sliding pieces along straight lines
	rook := chess.MakeColouredPiece(byColour, chess.Rook)
	straightDirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for _, dir := range straightDirs {
		c := chess.Col(int(col) + dir[0])
		r := chess.Rank(int(rank) + dir[1])
		for c >= 'a' && c <= 'h' && r >= '1' && r <= '8' {
			piece := board.Get(c, r)
			if piece != chess.Empty {
				if piece == rook || piece == queen {
					return true
				}
				break // Blocked
			}
			c = chess.Col(int(c) + dir[0])
			r = chess.Rank(int(r) + dir[1])
		}
	}

	return false
}
