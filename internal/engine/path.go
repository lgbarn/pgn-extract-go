package engine

import "github.com/lgbarn/pgn-extract-go/internal/chess"

// canPieceMove checks if a piece can move from one square to another.
func canPieceMove(board *chess.Board, pieceType chess.Piece, fromCol chess.Col, fromRank chess.Rank, toCol chess.Col, toRank chess.Rank) bool {
	colDiff := abs(int(toCol) - int(fromCol))
	rankDiff := abs(int(toRank) - int(fromRank))

	switch pieceType {
	case chess.Knight:
		return (colDiff == 1 && rankDiff == 2) || (colDiff == 2 && rankDiff == 1)

	case chess.Bishop:
		return colDiff == rankDiff && isPathClear(board, fromCol, fromRank, toCol, toRank)

	case chess.Rook:
		return (colDiff == 0 || rankDiff == 0) && isPathClear(board, fromCol, fromRank, toCol, toRank)

	case chess.Queen:
		isDiagonal := colDiff == rankDiff
		isStraight := colDiff == 0 || rankDiff == 0
		return (isDiagonal || isStraight) && isPathClear(board, fromCol, fromRank, toCol, toRank)

	case chess.King:
		return colDiff <= 1 && rankDiff <= 1
	}

	return false
}

// isPathClear checks if the path between two squares is clear.
// Works for both diagonal and straight paths.
func isPathClear(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, toCol chess.Col, toRank chess.Rank) bool {
	colDir := sign(int(toCol) - int(fromCol))
	rankDir := sign(int(toRank) - int(fromRank))

	col := chess.Col(int(fromCol) + colDir)
	rank := chess.Rank(int(fromRank) + rankDir)

	for col != toCol || rank != toRank {
		if board.Get(col, rank) != chess.Empty {
			return false
		}
		col = chess.Col(int(col) + colDir)
		rank = chess.Rank(int(rank) + rankDir)
	}

	return true
}
