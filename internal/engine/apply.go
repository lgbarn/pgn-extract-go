package engine

import (
	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// ApplyMove applies a move to the board and updates the board state.
// Returns true if the move was applied successfully.
func ApplyMove(board *chess.Board, move *chess.Move) bool {
	if move == nil {
		return false
	}

	switch move.Class {
	case chess.NullMove:
		// Just switch sides
		board.ToMove = board.ToMove.Opposite()
		board.EnPassant = false
		return true

	case chess.KingsideCastle:
		return applyCastle(board, true)

	case chess.QueensideCastle:
		return applyCastle(board, false)

	case chess.PawnMove, chess.PawnMoveWithPromotion, chess.EnPassantPawnMove:
		return applyPawnMove(board, move)

	case chess.PieceMove:
		return applyPieceMove(board, move)

	default:
		return false
	}
}
