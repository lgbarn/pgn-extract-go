// filters.go - Game filtering logic
package main

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/hashing"
	"github.com/lgbarn/pgn-extract-go/internal/processing"
)

// FilterResult holds the result of applying all filters to a game.
type FilterResult struct {
	Matched      bool
	Board        *chess.Board
	GameInfo     *GameAnalysis
	PlyCount     int
	SkipOutput   bool   // True if validation failed (don't output anywhere)
	ErrorMessage string // For logging validation errors
}

// applyFilters applies all game filters and returns the result.
// This is the shared filter logic used by both sequential and parallel processing.
func applyFilters(game *chess.Game, ctx *ProcessingContext) FilterResult {
	result := FilterResult{Matched: true}

	// Apply fixes if requested (do this before validation)
	if *fixableMode {
		fixGame(game)
	}

	// Validation checks
	if *strictMode || *validateMode {
		validResult := validateGame(game)

		// In strict mode, skip games with any parse errors
		if *strictMode && len(validResult.ParseErrors) > 0 {
			result.Matched = false
			result.SkipOutput = true
			if len(validResult.ParseErrors) > 0 {
				result.ErrorMessage = validResult.ParseErrors[0]
			}
			return result
		}

		// In validate mode, skip games with illegal moves
		if *validateMode && !validResult.Valid {
			result.Matched = false
			result.SkipOutput = true
			result.ErrorMessage = validResult.ErrorMsg
			return result
		}
	}

	// Add ECO tags if classifier is available
	if ctx.ecoClassifier != nil {
		ctx.ecoClassifier.AddECOTags(game)
	}

	// Check filter criteria
	if ctx.gameFilter != nil && ctx.gameFilter.HasCriteria() {
		result.Matched = ctx.gameFilter.MatchGame(game)
	}

	// Check CQL filter
	if result.Matched && ctx.cqlNode != nil {
		result.Matched = matchesCQL(game, ctx.cqlNode)
	}

	// Check variation matcher
	if result.Matched && ctx.variationMatcher != nil {
		result.Matched = ctx.variationMatcher.MatchGame(game)
	}

	// Check material matcher
	if result.Matched && ctx.materialMatcher != nil {
		result.Matched = ctx.materialMatcher.MatchGame(game)
	}

	// Calculate ply count for bounds checking
	result.PlyCount = processing.CountPlies(game)

	// Check ply bounds
	if result.Matched && *minPly > 0 && result.PlyCount < *minPly {
		result.Matched = false
	}
	if result.Matched && *maxPly > 0 && result.PlyCount > *maxPly {
		result.Matched = false
	}

	// Check move bounds (moves = plies / 2, rounded up)
	moveCount := (result.PlyCount + 1) / 2
	if result.Matched && *minMoves > 0 && moveCount < *minMoves {
		result.Matched = false
	}
	if result.Matched && *maxMoves > 0 && moveCount > *maxMoves {
		result.Matched = false
	}

	// Game analysis (creates its own board - thread-safe)
	cfg := ctx.cfg
	needsReplay := *checkmateFilter || *stalemateFilter || ctx.detector != nil ||
		*fiftyMoveFilter || *repetitionFilter || *underpromotionFilter ||
		*higherRatedWinner || *lowerRatedWinner || cfg.Annotation.AddFENComments || cfg.Annotation.AddHashComments ||
		cfg.Annotation.AddHashTag

	if needsReplay {
		result.Board, result.GameInfo = analyzeGame(game)
	}

	// Check checkmate filter
	if result.Matched && *checkmateFilter {
		if !engine.IsCheckmate(result.Board) {
			result.Matched = false
		}
	}

	// Check stalemate filter
	if result.Matched && *stalemateFilter {
		if !engine.IsStalemate(result.Board) {
			result.Matched = false
		}
	}

	// Check fifty-move rule filter
	if result.Matched && *fiftyMoveFilter {
		if result.GameInfo == nil || !result.GameInfo.HasFiftyMoveRule {
			result.Matched = false
		}
	}

	// Check repetition filter
	if result.Matched && *repetitionFilter {
		if result.GameInfo == nil || !result.GameInfo.HasRepetition {
			result.Matched = false
		}
	}

	// Check underpromotion filter
	if result.Matched && *underpromotionFilter {
		if result.GameInfo == nil || !result.GameInfo.HasUnderpromotion {
			result.Matched = false
		}
	}

	// Check commented filter
	if result.Matched && *commentedFilter {
		if !processing.HasComments(game) {
			result.Matched = false
		}
	}

	// Check rating-based winner filters
	if result.Matched && (*higherRatedWinner || *lowerRatedWinner) {
		whiteElo := parseElo(game.Tags["WhiteElo"])
		blackElo := parseElo(game.Tags["BlackElo"])
		gameResult := game.Tags["Result"]

		if whiteElo > 0 && blackElo > 0 {
			if *higherRatedWinner {
				higherWon := (whiteElo > blackElo && gameResult == "1-0") ||
					(blackElo > whiteElo && gameResult == "0-1")
				if !higherWon {
					result.Matched = false
				}
			}
			if *lowerRatedWinner {
				lowerWon := (whiteElo < blackElo && gameResult == "1-0") ||
					(blackElo < whiteElo && gameResult == "0-1")
				if !lowerWon {
					result.Matched = false
				}
			}
		} else {
			result.Matched = false // No rating info available
		}
	}

	// Handle negated matching
	if *negateMatch {
		result.Matched = !result.Matched
	}

	// Add plycount tag if requested (and matched)
	if result.Matched && cfg.Annotation.AddPlyCount {
		game.Tags["PlyCount"] = strconv.Itoa(result.PlyCount)
	}

	// Add hashcode tag if requested (and matched)
	if result.Matched && cfg.Annotation.AddHashTag && result.Board != nil {
		hash := hashing.GenerateZobristHash(result.Board)
		game.Tags["HashCode"] = fmt.Sprintf("%016x", hash)
	}

	return result
}

// parseElo parses an Elo rating string to int
func parseElo(s string) int {
	if s == "" || s == "-" || s == "?" {
		return 0
	}
	elo, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return elo
}

// Global state for stopAfter (atomic for thread safety)
var matchedCount int64

// IncrementMatchedCount atomically increments the matched game counter
func IncrementMatchedCount() int64 {
	return atomic.AddInt64(&matchedCount, 1)
}

// GetMatchedCount returns the current matched game count
func GetMatchedCount() int64 {
	return atomic.LoadInt64(&matchedCount)
}
