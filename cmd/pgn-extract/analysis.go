// analysis.go - Game analysis, validation, and fixing functions
package main

import (
	"fmt"
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/cql"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/hashing"
)

// GameAnalysis holds analysis results from replaying a game
type GameAnalysis struct {
	FinalBoard        *chess.Board
	HasFiftyMoveRule  bool
	HasRepetition     bool
	HasUnderpromotion bool
	Positions         []uint64 // Zobrist hashes for repetition detection
}

// analyzeGame replays a game and analyzes it for various features
func analyzeGame(game *chess.Game) (*chess.Board, *GameAnalysis) {
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

// matchesCQL checks if any position in the game matches the CQL query.
func matchesCQL(game *chess.Game, cqlNode cql.Node) bool {
	board := engine.NewBoardForGame(game)

	// Create evaluator once and reuse for all positions
	eval := cql.NewEvaluator(board)

	// Check starting position
	if eval.Evaluate(cqlNode) {
		return true
	}

	// Check each position after a move
	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}
		// Board is modified in place, evaluator already has pointer to it
		if eval.Evaluate(cqlNode) {
			return true
		}
	}

	return false
}

// replayGame replays a game from the initial position to get the final board state.
func replayGame(game *chess.Game) *chess.Board {
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

// ValidationResult holds the result of game validation
type ValidationResult struct {
	Valid       bool
	ErrorPly    int
	ErrorMsg    string
	ParseErrors []string
}

// validateGame validates all moves in a game are legal
func validateGame(game *chess.Game) *ValidationResult {
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

// fixGame attempts to fix common issues in a game
func fixGame(game *chess.Game) bool {
	fixed := false

	// Fix missing required tags with placeholder values
	if game.GetTag("Event") == "" {
		game.SetTag("Event", "?")
		fixed = true
	}
	if game.GetTag("Site") == "" {
		game.SetTag("Site", "?")
		fixed = true
	}
	if game.GetTag("Date") == "" {
		game.SetTag("Date", "????.??.??")
		fixed = true
	}
	if game.GetTag("Round") == "" {
		game.SetTag("Round", "?")
		fixed = true
	}
	if game.GetTag("White") == "" {
		game.SetTag("White", "?")
		fixed = true
	}
	if game.GetTag("Black") == "" {
		game.SetTag("Black", "?")
		fixed = true
	}
	if game.GetTag("Result") == "" {
		game.SetTag("Result", "*")
		fixed = true
	}

	// Fix invalid result tag
	resultTag := game.GetTag("Result")
	validResults := map[string]bool{"1-0": true, "0-1": true, "1/2-1/2": true, "*": true}
	if !validResults[resultTag] {
		// Try to normalize common variations
		switch strings.ToLower(strings.TrimSpace(resultTag)) {
		case "1-0", "white", "white wins":
			game.SetTag("Result", "1-0")
			fixed = true
		case "0-1", "black", "black wins":
			game.SetTag("Result", "0-1")
			fixed = true
		case "1/2", "draw", "1/2-1/2", "0.5-0.5":
			game.SetTag("Result", "1/2-1/2")
			fixed = true
		default:
			game.SetTag("Result", "*")
			fixed = true
		}
	}

	// Fix common date format issues
	date := game.GetTag("Date")
	if date != "" && date != "????.??.??" {
		// Replace common separators with dots
		normalizedDate := strings.ReplaceAll(date, "/", ".")
		normalizedDate = strings.ReplaceAll(normalizedDate, "-", ".")
		if normalizedDate != date {
			game.SetTag("Date", normalizedDate)
			fixed = true
		}
	}

	// Trim whitespace from all tags
	for tag, value := range game.Tags {
		trimmed := strings.TrimSpace(value)
		if trimmed != value {
			game.Tags[tag] = trimmed
			fixed = true
		}
	}

	// Fix encoding issues - remove control characters
	for tag, value := range game.Tags {
		cleaned := cleanString(value)
		if cleaned != value {
			game.Tags[tag] = cleaned
			fixed = true
		}
	}

	return fixed
}

// cleanString removes control characters and fixes common encoding issues
func cleanString(s string) string {
	var result strings.Builder
	for _, r := range s {
		// Keep printable ASCII, space, and common Unicode
		if r >= 32 && r != 127 {
			result.WriteRune(r)
		}
	}
	return result.String()
}
