// Package processing provides game analysis, validation, and processing logic.
package processing

import (
	"fmt"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/hashing"
)

// GameAnalysis holds analysis results from replaying a game.
type GameAnalysis struct {
	FinalBoard        *chess.Board
	HasFiftyMoveRule  bool
	HasRepetition     bool
	HasUnderpromotion bool
	Positions         []uint64 // Zobrist hashes for repetition detection
}

// FiftyMoveTriggered implements worker.GameInfo.
func (ga *GameAnalysis) FiftyMoveTriggered() bool {
	return ga.HasFiftyMoveRule
}

// RepetitionDetected implements worker.GameInfo.
func (ga *GameAnalysis) RepetitionDetected() bool {
	return ga.HasRepetition
}

// UnderpromotionFound implements worker.GameInfo.
func (ga *GameAnalysis) UnderpromotionFound() bool {
	return ga.HasUnderpromotion
}

// ValidationResult holds the result of game validation.
type ValidationResult struct {
	Valid       bool
	ErrorPly    int
	ErrorMsg    string
	ParseErrors []string
}

// AnalyzeGame replays a game and analyzes it for various features.
func AnalyzeGame(game *chess.Game) (*chess.Board, *GameAnalysis) {
	board := engine.NewBoardForGame(game)

	analysis := &GameAnalysis{
		Positions: make([]uint64, 0),
	}

	// Track initial position
	posHash := hashing.GenerateZobristHash(board)
	analysis.Positions = append(analysis.Positions, posHash)
	positionCount := make(map[uint64]int)
	positionCount[posHash] = 1

	// Apply all moves
	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}

		// Check for fifty-move rule
		if board.HalfmoveClock >= 100 {
			analysis.HasFiftyMoveRule = true
		}

		// Check for underpromotion
		if move.PromotedPiece != chess.Empty && move.PromotedPiece != chess.Queen {
			analysis.HasUnderpromotion = true
		}

		// Track position for repetition
		posHash = hashing.GenerateZobristHash(board)
		analysis.Positions = append(analysis.Positions, posHash)
		positionCount[posHash]++
		if positionCount[posHash] >= 3 {
			analysis.HasRepetition = true
		}
	}

	analysis.FinalBoard = board
	return board, analysis
}

// ReplayGame replays a game from the initial position to get the final board state.
func ReplayGame(game *chess.Game) *chess.Board {
	board := engine.NewBoardForGame(game)

	// Apply all moves
	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			// Move application failed - return current board state
			break
		}
	}

	return board
}

// ValidateGame validates all moves in a game are legal.
func ValidateGame(game *chess.Game) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Check for missing required tags
	requiredTags := []string{"Event", "Site", "Date", "Round", "White", "Black", "Result"}
	for _, tag := range requiredTags {
		if game.GetTag(tag) == "" {
			result.ParseErrors = append(result.ParseErrors, fmt.Sprintf("missing required tag: %s", tag))
		}
	}

	// Check for valid result
	resultTag := game.GetTag("Result")
	if resultTag != "" && resultTag != "1-0" && resultTag != "0-1" && resultTag != "1/2-1/2" && resultTag != "*" {
		result.ParseErrors = append(result.ParseErrors, fmt.Sprintf("invalid result: %s", resultTag))
	}

	// If we have no moves, game is valid (just tags)
	if game.Moves == nil {
		return result
	}

	// Replay game to validate moves
	var board *chess.Board
	var err error

	if fen, ok := game.Tags["FEN"]; ok {
		board, err = engine.NewBoardFromFEN(fen)
		if err != nil {
			result.Valid = false
			result.ErrorMsg = fmt.Sprintf("invalid FEN: %s", fen)
			return result
		}
	} else {
		board, _ = engine.NewBoardFromFEN(engine.InitialFEN)
	}

	plyCount := 0
	for move := game.Moves; move != nil; move = move.Next {
		plyCount++
		if !engine.ApplyMove(board, move) {
			result.Valid = false
			result.ErrorPly = plyCount
			result.ErrorMsg = fmt.Sprintf("illegal move at ply %d: %s", plyCount, move.Text)
			return result
		}
	}

	// Mark game as validated
	game.MovesChecked = true
	game.MovesOK = true

	return result
}

// CountPlies counts the number of plies (half-moves) in a game.
func CountPlies(game *chess.Game) int {
	count := 0
	for move := game.Moves; move != nil; move = move.Next {
		count++
	}
	return count
}

// HasComments checks if a game has any comments.
func HasComments(game *chess.Game) bool {
	for move := game.Moves; move != nil; move = move.Next {
		if move.HasComments() {
			return true
		}
	}
	return false
}
