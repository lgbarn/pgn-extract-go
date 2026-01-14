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

// fixGame attempts to fix common issues in a game.
func fixGame(game *chess.Game) bool {
	fixed := fixMissingTags(game)
	fixed = fixResultTag(game) || fixed
	fixed = fixDateFormat(game) || fixed
	fixed = cleanAllTags(game) || fixed
	return fixed
}

// fixMissingTags adds placeholder values for missing required tags.
func fixMissingTags(game *chess.Game) bool {
	requiredTags := map[string]string{
		"Event":  "?",
		"Site":   "?",
		"Date":   "????.??.??",
		"Round":  "?",
		"White":  "?",
		"Black":  "?",
		"Result": "*",
	}

	fixed := false
	for tag, defaultValue := range requiredTags {
		if game.GetTag(tag) == "" {
			game.SetTag(tag, defaultValue)
			fixed = true
		}
	}
	return fixed
}

// fixResultTag normalizes invalid result tags.
func fixResultTag(game *chess.Game) bool {
	resultTag := game.GetTag("Result")
	validResults := map[string]bool{"1-0": true, "0-1": true, "1/2-1/2": true, "*": true}

	if validResults[resultTag] {
		return false
	}

	normalized := strings.ToLower(strings.TrimSpace(resultTag))
	var newResult string

	switch normalized {
	case "1-0", "white", "white wins":
		newResult = "1-0"
	case "0-1", "black", "black wins":
		newResult = "0-1"
	case "1/2", "draw", "1/2-1/2", "0.5-0.5":
		newResult = "1/2-1/2"
	default:
		newResult = "*"
	}

	game.SetTag("Result", newResult)
	return true
}

// fixDateFormat normalizes date separators to dots.
func fixDateFormat(game *chess.Game) bool {
	date := game.GetTag("Date")
	if date == "" || date == "????.??.??" {
		return false
	}

	normalizedDate := strings.ReplaceAll(date, "/", ".")
	normalizedDate = strings.ReplaceAll(normalizedDate, "-", ".")

	if normalizedDate == date {
		return false
	}

	game.SetTag("Date", normalizedDate)
	return true
}

// cleanAllTags trims whitespace and removes control characters from all tags.
func cleanAllTags(game *chess.Game) bool {
	fixed := false
	for tag, value := range game.Tags {
		cleaned := cleanString(strings.TrimSpace(value))
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
