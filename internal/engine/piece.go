package engine

import "github.com/lgbarn/pgn-extract-go/internal/chess"

// applyPieceMove applies a piece (non-pawn) move.
func applyPieceMove(board *chess.Board, move *chess.Move) bool {
	colour := board.ToMove
	fromCol, fromRank := move.FromCol, move.FromRank
	toCol, toRank := move.ToCol, move.ToRank
	pieceType := move.PieceToMove

	if fromCol == 0 || fromRank == 0 {
		fromCol, fromRank = findPieceSource(board, move, colour)
		if fromCol == 0 {
			return false
		}
	}

	piece := board.Get(fromCol, fromRank)
	capturedPiece := board.Get(toCol, toRank)

	// Move the piece
	board.Set(fromCol, fromRank, chess.Empty)
	board.Set(toCol, toRank, piece)

	// Update king position and castling rights if king moved
	if pieceType == chess.King {
		if colour == chess.White {
			board.WKingCol, board.WKingRank = toCol, toRank
			board.WKingCastle, board.WQueenCastle = 0, 0
		} else {
			board.BKingCol, board.BKingRank = toCol, toRank
			board.BKingCastle, board.BQueenCastle = 0, 0
		}
	}

	// Update castling rights if rook moved
	if pieceType == chess.Rook {
		updateCastlingRightsForRook(board, colour, fromCol, fromRank)
	}

	// Update castling rights if rook captured
	if capturedPiece != chess.Empty && chess.ExtractPiece(capturedPiece) == chess.Rook {
		updateCastlingRightsForRook(board, chess.ExtractColour(capturedPiece), toCol, toRank)
	}

	board.EnPassant = false

	if capturedPiece != chess.Empty {
		board.HalfmoveClock = 0
	} else {
		board.HalfmoveClock++
	}

	if colour == chess.Black {
		board.MoveNumber++
	}
	board.ToMove = colour.Opposite()

	return true
}

// findPieceSource finds the source square of a piece move.
func findPieceSource(board *chess.Board, move *chess.Move, colour chess.Colour) (chess.Col, chess.Rank) {
	toCol, toRank := move.ToCol, move.ToRank
	pieceType := move.PieceToMove
	fromCol, fromRank := move.FromCol, move.FromRank
	piece := chess.MakeColouredPiece(colour, pieceType)

	for col := chess.Col('a'); col <= 'h'; col++ {
		for rank := chess.Rank('1'); rank <= '8'; rank++ {
			if board.Get(col, rank) != piece {
				continue
			}
			// Check disambiguation
			if fromCol != 0 && col != fromCol {
				continue
			}
			if fromRank != 0 && rank != fromRank {
				continue
			}
			if canPieceMove(board, pieceType, col, rank, toCol, toRank) {
				return col, rank
			}
		}
	}

	return 0, 0
}
