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
			if hasLegalMovesForPiece(board, col, rank, chess.ExtractPiece(piece), colour) {
				return true
			}
		}
	}
	return false
}

// hasLegalMovesForPiece checks if a specific piece has any legal moves.
func hasLegalMovesForPiece(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, pieceType chess.Piece, colour chess.Colour) bool {
	switch pieceType {
	case chess.Pawn:
		return hasPawnMoves(board, fromCol, fromRank, colour)
	case chess.Knight:
		return hasJumpMoves(board, fromCol, fromRank, colour, knightOffsets)
	case chess.King:
		return hasJumpMoves(board, fromCol, fromRank, colour, kingOffsets)
	case chess.Bishop:
		return hasSlidingMoves(board, fromCol, fromRank, colour, diagonalDirs)
	case chess.Rook:
		return hasSlidingMoves(board, fromCol, fromRank, colour, straightDirs)
	case chess.Queen:
		return hasSlidingMoves(board, fromCol, fromRank, colour, diagonalDirs) ||
			hasSlidingMoves(board, fromCol, fromRank, colour, straightDirs)
	}
	return false
}

// hasPawnMoves checks if a pawn has any legal moves.
func hasPawnMoves(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, colour chess.Colour) bool {
	dir := chess.ColourOffset(colour)
	toRank := chess.Rank(int(fromRank) + dir)

	if !isOnBoard(fromCol, toRank) {
		return false
	}

	// Forward move
	if board.Get(fromCol, toRank) == chess.Empty {
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

	// Captures (including en passant)
	for _, dc := range []int{-1, 1} {
		toCol := chess.Col(int(fromCol) + dc)
		if !isOnBoard(toCol, toRank) {
			continue
		}
		target := board.Get(toCol, toRank)
		isCapture := target != chess.Empty && chess.ExtractColour(target) != colour
		isEnPassant := board.EnPassant && toCol == board.EPCol && toRank == board.EPRank
		if (isCapture || isEnPassant) && tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
			return true
		}
	}

	return false
}

// hasJumpMoves checks if a knight or king has any legal moves.
func hasJumpMoves(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, colour chess.Colour, offsets [][2]int) bool {
	for _, offset := range offsets {
		toCol := chess.Col(int(fromCol) + offset[0])
		toRank := chess.Rank(int(fromRank) + offset[1])
		if !isOnBoard(toCol, toRank) {
			continue
		}
		target := board.Get(toCol, toRank)
		if target == chess.Empty || chess.ExtractColour(target) != colour {
			if tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
				return true
			}
		}
	}
	return false
}

// hasSlidingMoves checks if a sliding piece (bishop, rook, queen) has legal moves.
func hasSlidingMoves(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, colour chess.Colour, dirs [][2]int) bool {
	for _, dir := range dirs {
		toCol := chess.Col(int(fromCol) + dir[0])
		toRank := chess.Rank(int(fromRank) + dir[1])
		for isOnBoard(toCol, toRank) {
			target := board.Get(toCol, toRank)
			if target != chess.Empty {
				if chess.ExtractColour(target) != colour {
					if tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
						return true
					}
				}
				break
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
	testBoard := board.Copy()
	piece := testBoard.Get(fromCol, fromRank)
	testBoard.Set(fromCol, fromRank, chess.Empty)
	testBoard.Set(toCol, toRank, piece)

	if chess.ExtractPiece(piece) == chess.King {
		if colour == chess.White {
			testBoard.WKingCol = toCol
			testBoard.WKingRank = toRank
		} else {
			testBoard.BKingCol = toCol
			testBoard.BKingRank = toRank
		}
	}

	return !IsInCheck(testBoard, colour)
}
