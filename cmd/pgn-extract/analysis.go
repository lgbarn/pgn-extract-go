// analysis.go - Game analysis, validation, and fixing functions
package main

import (
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/cql"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/processing"
)

// Type aliases for backward compatibility
type GameAnalysis = processing.GameAnalysis
type ValidationResult = processing.ValidationResult

// analyzeGame replays a game and analyzes it for various features.
// This is a thin wrapper around processing.AnalyzeGame.
func analyzeGame(game *chess.Game) (*chess.Board, *GameAnalysis) {
	return processing.AnalyzeGame(game)
}

// replayGame replays a game from the initial position to get the final board state.
// This is a thin wrapper around processing.ReplayGame.
func replayGame(game *chess.Game) *chess.Board {
	return processing.ReplayGame(game)
}

// validateGame validates all moves in a game are legal.
// This is a thin wrapper around processing.ValidateGame.
func validateGame(game *chess.Game) *ValidationResult {
	return processing.ValidateGame(game)
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
