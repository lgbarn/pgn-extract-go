// Package engine provides chess move validation and board manipulation.
package engine

import (
	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// DrawRuleResult contains the results of draw rule detection.
type DrawRuleResult struct {
	// Has75MoveRule is true if a position was reached where 75 moves
	// (150 half-moves) have been made without a pawn move or capture.
	Has75MoveRule bool

	// Has5FoldRepetition is true if any position occurred 5 or more times.
	Has5FoldRepetition bool

	// HasInsufficientMaterial is true if the final position has insufficient
	// mating material for either side.
	HasInsufficientMaterial bool

	// HasMaterialOdds is true if the game started with unequal material.
	HasMaterialOdds bool
}

// AnalyzeDrawRules analyzes a game for various draw conditions.
func AnalyzeDrawRules(game *chess.Game) DrawRuleResult {
	result := DrawRuleResult{}

	board, err := NewBoardFromFEN(InitialFEN)
	if err != nil {
		return result
	}

	// Check for material odds at the start
	if game.FEN() != "" {
		// Game has a custom starting position - check for material odds
		startBoard, err := NewBoardFromFEN(game.FEN())
		if err == nil {
			result.HasMaterialOdds = !isStandardMaterial(startBoard)
		}
	}

	// Track position counts for 5-fold repetition
	positionCounts := make(map[uint64]int)

	// Track the initial position
	positionCounts[board.Zobrist]++

	// Replay the game
	for move := game.Moves; move != nil; move = move.Next {
		if !ApplyMove(board, move) {
			break
		}

		// Check 75-move rule (150 half-moves without pawn move or capture)
		if board.HalfmoveClock >= 150 {
			result.Has75MoveRule = true
		}

		// Track position for 5-fold repetition
		positionCounts[board.Zobrist]++
		if positionCounts[board.Zobrist] >= 5 {
			result.Has5FoldRepetition = true
		}
	}

	// Check insufficient material at final position
	result.HasInsufficientMaterial = HasInsufficientMaterial(board)

	return result
}

// HasInsufficientMaterial returns true if the position has insufficient
// mating material for either side.
// Insufficient material includes:
// - K vs K
// - K+B vs K
// - K+N vs K
// - K+B vs K+B (same color bishops)
func HasInsufficientMaterial(board *chess.Board) bool {
	var whitePieces, blackPieces []chess.Piece
	var whiteBishopOnLight, blackBishopOnLight bool

	// Count pieces for each side
	for rank := chess.Rank(chess.FirstRank); rank <= chess.Rank(chess.LastRank); rank++ {
		for col := chess.Col(chess.FirstCol); col <= chess.Col(chess.LastCol); col++ {
			piece := board.Get(col, rank)
			if piece == chess.Empty || piece == chess.Off {
				continue
			}

			colour := chess.ExtractColour(piece)
			pieceType := chess.ExtractPiece(piece)

			// Kings don't count for material
			if pieceType == chess.King {
				continue
			}

			// Any pawn, rook, or queen means sufficient material
			if pieceType == chess.Pawn || pieceType == chess.Rook || pieceType == chess.Queen {
				return false
			}

			if colour == chess.White {
				whitePieces = append(whitePieces, pieceType)
				if pieceType == chess.Bishop {
					whiteBishopOnLight = isLightSquare(col, rank)
				}
			} else {
				blackPieces = append(blackPieces, pieceType)
				if pieceType == chess.Bishop {
					blackBishopOnLight = isLightSquare(col, rank)
				}
			}
		}
	}

	// K vs K
	if len(whitePieces) == 0 && len(blackPieces) == 0 {
		return true
	}

	// K+B vs K or K+N vs K
	if len(whitePieces) == 0 && len(blackPieces) == 1 {
		return blackPieces[0] == chess.Bishop || blackPieces[0] == chess.Knight
	}
	if len(blackPieces) == 0 && len(whitePieces) == 1 {
		return whitePieces[0] == chess.Bishop || whitePieces[0] == chess.Knight
	}

	// K+B vs K+B (same color bishops)
	if len(whitePieces) == 1 && len(blackPieces) == 1 {
		if whitePieces[0] == chess.Bishop && blackPieces[0] == chess.Bishop {
			// Check if both bishops are on the same color squares
			if whiteBishopOnLight == blackBishopOnLight {
				return true
			}
		}
	}

	return false
}

// isLightSquare returns true if the given square is a light square.
func isLightSquare(col chess.Col, rank chess.Rank) bool {
	colNum := int(col - chess.FirstCol)
	rankNum := int(rank - chess.FirstRank)
	return (colNum+rankNum)%2 == 1
}

// isStandardMaterial checks if the board has standard starting material.
func isStandardMaterial(board *chess.Board) bool {
	// Standard material: 8 pawns, 2 rooks, 2 knights, 2 bishops, 1 queen, 1 king per side
	expectedPieces := map[chess.Piece]int{
		chess.W(chess.Pawn):   8,
		chess.W(chess.Rook):   2,
		chess.W(chess.Knight): 2,
		chess.W(chess.Bishop): 2,
		chess.W(chess.Queen):  1,
		chess.W(chess.King):   1,
		chess.B(chess.Pawn):   8,
		chess.B(chess.Rook):   2,
		chess.B(chess.Knight): 2,
		chess.B(chess.Bishop): 2,
		chess.B(chess.Queen):  1,
		chess.B(chess.King):   1,
	}

	actualPieces := make(map[chess.Piece]int)

	for rank := chess.Rank(chess.FirstRank); rank <= chess.Rank(chess.LastRank); rank++ {
		for col := chess.Col(chess.FirstCol); col <= chess.Col(chess.LastCol); col++ {
			piece := board.Get(col, rank)
			if piece != chess.Empty && piece != chess.Off {
				actualPieces[piece]++
			}
		}
	}

	// Compare actual vs expected
	for piece, expected := range expectedPieces {
		if actualPieces[piece] != expected {
			return false
		}
	}

	return true
}

// CheckMaterialOdds checks if a game started with material odds.
func CheckMaterialOdds(game *chess.Game) bool {
	fenStr := game.FEN()
	if fenStr == "" {
		return false // Standard starting position
	}

	board, err := NewBoardFromFEN(fenStr)
	if err != nil {
		return false
	}

	return !isStandardMaterial(board)
}
