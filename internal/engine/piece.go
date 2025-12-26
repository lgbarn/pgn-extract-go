package engine

import "github.com/lgbarn/pgn-extract-go/internal/chess"

// applyPieceMove applies a piece (non-pawn) move.
func applyPieceMove(board *chess.Board, move *chess.Move) bool {
	colour := board.ToMove
	fromCol := move.FromCol
	fromRank := move.FromRank
	toCol := move.ToCol
	toRank := move.ToRank
	pieceType := move.PieceToMove

	// If source square not specified, find the piece
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

	// Update king position if king moved
	if pieceType == chess.King {
		if colour == chess.White {
			board.WKingCol = toCol
			board.WKingRank = toRank
			board.WKingCastle = 0
			board.WQueenCastle = 0
		} else {
			board.BKingCol = toCol
			board.BKingRank = toRank
			board.BKingCastle = 0
			board.BQueenCastle = 0
		}
	}

	// Update castling rights if rook moved or captured
	if pieceType == chess.Rook {
		updateCastlingRightsForRook(board, colour, fromCol, fromRank)
	}
	if capturedPiece != chess.Empty && chess.ExtractPiece(capturedPiece) == chess.Rook {
		capturedColour := chess.ExtractColour(capturedPiece)
		updateCastlingRightsForRook(board, capturedColour, toCol, toRank)
	}

	board.EnPassant = false

	// Update halfmove clock
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
	toCol := move.ToCol
	toRank := move.ToRank
	pieceType := move.PieceToMove
	fromCol := move.FromCol
	fromRank := move.FromRank

	piece := chess.MakeColouredPiece(colour, pieceType)

	// Search for the piece that can move to the target square
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

			// Check if this piece can reach the target
			if canPieceMove(board, pieceType, col, rank, toCol, toRank) {
				return col, rank
			}
		}
	}

	return 0, 0
}
