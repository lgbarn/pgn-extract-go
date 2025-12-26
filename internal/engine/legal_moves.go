package engine

import "github.com/lgbarn/pgn-extract-go/internal/chess"

// HasLegalMoves returns true if the given colour has at least one legal move.
func HasLegalMoves(board *chess.Board, colour chess.Colour) bool {
	for col := chess.Col('a'); col <= 'h'; col++ {
		for rank := chess.Rank('1'); rank <= '8'; rank++ {
			piece := board.Get(col, rank)
			if piece == chess.Empty || piece == chess.Off {
				continue
			}
			if chess.ExtractColour(piece) != colour {
				continue
			}

			pieceType := chess.ExtractPiece(piece)
			if hasLegalMovesForPiece(board, col, rank, pieceType, colour) {
				return true
			}
		}
	}
	return false
}

// hasLegalMovesForPiece checks if a specific piece has any legal moves.
func hasLegalMovesForPiece(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, pieceType chess.Piece, colour chess.Colour) bool {
	// Generate potential target squares based on piece type
	var targets [][2]int

	switch pieceType {
	case chess.Pawn:
		dir := chess.ColourOffset(colour)
		// Forward move
		toRank := chess.Rank(int(fromRank) + dir)
		if toRank >= '1' && toRank <= '8' && board.Get(fromCol, toRank) == chess.Empty {
			if tryMove(board, fromCol, fromRank, fromCol, toRank, colour) {
				return true
			}
			// Double push from starting rank
			startRank := chess.Rank('2')
			if colour == chess.Black {
				startRank = '7'
			}
			if fromRank == startRank {
				toRank2 := chess.Rank(int(fromRank) + 2*dir)
				if board.Get(fromCol, toRank2) == chess.Empty {
					if tryMove(board, fromCol, fromRank, fromCol, toRank2, colour) {
						return true
					}
				}
			}
		}
		// Captures
		for dc := -1; dc <= 1; dc += 2 {
			toCol := chess.Col(int(fromCol) + dc)
			toRank := chess.Rank(int(fromRank) + dir)
			if toCol >= 'a' && toCol <= 'h' && toRank >= '1' && toRank <= '8' {
				target := board.Get(toCol, toRank)
				if target != chess.Empty && chess.ExtractColour(target) != colour {
					if tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
						return true
					}
				}
				// En passant
				if board.EnPassant && toCol == board.EPCol && toRank == board.EPRank {
					if tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
						return true
					}
				}
			}
		}
		return false

	case chess.Knight:
		targets = [][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}

	case chess.King:
		targets = [][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}

	case chess.Bishop:
		return hasSlidingMoves(board, fromCol, fromRank, colour, true, false)

	case chess.Rook:
		return hasSlidingMoves(board, fromCol, fromRank, colour, false, true)

	case chess.Queen:
		return hasSlidingMoves(board, fromCol, fromRank, colour, true, true)
	}

	// Check each target for knight and king
	for _, offset := range targets {
		toCol := chess.Col(int(fromCol) + offset[0])
		toRank := chess.Rank(int(fromRank) + offset[1])
		if toCol >= 'a' && toCol <= 'h' && toRank >= '1' && toRank <= '8' {
			target := board.Get(toCol, toRank)
			if target == chess.Empty || chess.ExtractColour(target) != colour {
				if tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
					return true
				}
			}
		}
	}

	return false
}

// hasSlidingMoves checks if a sliding piece (bishop, rook, queen) has legal moves.
func hasSlidingMoves(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, colour chess.Colour, diagonal, straight bool) bool {
	var dirs [][2]int
	if diagonal {
		dirs = append(dirs, [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}...)
	}
	if straight {
		dirs = append(dirs, [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}...)
	}

	for _, dir := range dirs {
		toCol := chess.Col(int(fromCol) + dir[0])
		toRank := chess.Rank(int(fromRank) + dir[1])
		for toCol >= 'a' && toCol <= 'h' && toRank >= '1' && toRank <= '8' {
			target := board.Get(toCol, toRank)
			if target != chess.Empty {
				if chess.ExtractColour(target) != colour {
					if tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
						return true
					}
				}
				break // Blocked
			}
			if tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
				return true
			}
			toCol = chess.Col(int(toCol) + dir[0])
			toRank = chess.Rank(int(toRank) + dir[1])
		}
	}
	return false
}

// tryMove makes a move on a copied board and checks if it leaves the king in check.
func tryMove(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, toCol chess.Col, toRank chess.Rank, colour chess.Colour) bool {
	// Make a copy of the board
	testBoard := board.Copy()

	// Make the move
	piece := testBoard.Get(fromCol, fromRank)
	testBoard.Set(fromCol, fromRank, chess.Empty)
	testBoard.Set(toCol, toRank, piece)

	// Update king position if needed
	if chess.ExtractPiece(piece) == chess.King {
		if colour == chess.White {
			testBoard.WKingCol = toCol
			testBoard.WKingRank = toRank
		} else {
			testBoard.BKingCol = toCol
			testBoard.BKingRank = toRank
		}
	}

	// Check if our king is in check after the move
	return !IsInCheck(testBoard, colour)
}
