package engine

import "github.com/lgbarn/pgn-extract-go/internal/chess"

// IsCheckmate returns true if the position is checkmate for the side to move.
func IsCheckmate(board *chess.Board) bool {
	colour := board.ToMove
	return IsInCheck(board, colour) && !HasLegalMoves(board, colour)
}

// IsStalemate returns true if the position is stalemate for the side to move.
func IsStalemate(board *chess.Board) bool {
	colour := board.ToMove
	return !IsInCheck(board, colour) && !HasLegalMoves(board, colour)
}
