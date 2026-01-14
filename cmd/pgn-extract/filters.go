// filters.go - Game filtering logic
package main

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
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

	if *fixableMode {
		fixGame(game)
	}

	if failed := applyValidation(game); failed != nil {
		return *failed
	}

	if ctx.ecoClassifier != nil {
		ctx.ecoClassifier.AddECOTags(game)
	}

	// Apply tag and pattern filters
	result.Matched = applyTagFilters(game, ctx, result.Matched)
	result.Matched = applyPatternFilters(game, ctx, result.Matched)

	// Calculate and check ply/move bounds
	result.PlyCount = processing.CountPlies(game)
	result.Matched = checkPlyBounds(result.PlyCount, result.Matched)
	result.Matched = checkMoveBounds(result.PlyCount, result.Matched)

	// Analyze game if needed for feature filters
	if needsGameAnalysis(ctx) {
		result.Board, result.GameInfo = analyzeGame(game)
	}

	// Apply game feature filters
	result.Matched = applyFeatureFilters(&result, game, result.Matched)

	if *negateMatch {
		result.Matched = !result.Matched
	}

	if result.Matched {
		addAnnotations(game, &result, ctx.cfg)
	}

	return result
}

// applyValidation checks validation modes and returns a failure result if validation fails.
func applyValidation(game *chess.Game) *FilterResult {
	if !*strictMode && !*validateMode {
		return nil
	}

	validResult := validateGame(game)

	if *strictMode && len(validResult.ParseErrors) > 0 {
		return &FilterResult{
			Matched:      false,
			SkipOutput:   true,
			ErrorMessage: validResult.ParseErrors[0],
		}
	}

	if *validateMode && !validResult.Valid {
		return &FilterResult{
			Matched:      false,
			SkipOutput:   true,
			ErrorMessage: validResult.ErrorMsg,
		}
	}

	return nil
}

// applyTagFilters applies tag-based filters (game filter, CQL, variation, material).
func applyTagFilters(game *chess.Game, ctx *ProcessingContext, matched bool) bool {
	if !matched {
		return false
	}

	if ctx.gameFilter != nil && ctx.gameFilter.HasCriteria() && !ctx.gameFilter.MatchGame(game) {
		return false
	}

	if ctx.cqlNode != nil && !matchesCQL(game, ctx.cqlNode) {
		return false
	}

	if ctx.variationMatcher != nil && !ctx.variationMatcher.MatchGame(game) {
		return false
	}

	if ctx.materialMatcher != nil && !ctx.materialMatcher.MatchGame(game) {
		return false
	}

	return true
}

// applyPatternFilters is kept for extensibility but currently a no-op.
func applyPatternFilters(_ *chess.Game, _ *ProcessingContext, matched bool) bool {
	return matched
}

// checkPlyBounds checks if the game meets ply count requirements.
func checkPlyBounds(plyCount int, matched bool) bool {
	if !matched {
		return false
	}
	if *minPly > 0 && plyCount < *minPly {
		return false
	}
	if *maxPly > 0 && plyCount > *maxPly {
		return false
	}
	return true
}

// checkMoveBounds checks if the game meets move count requirements.
func checkMoveBounds(plyCount int, matched bool) bool {
	if !matched {
		return false
	}
	moveCount := (plyCount + 1) / 2
	if *minMoves > 0 && moveCount < *minMoves {
		return false
	}
	if *maxMoves > 0 && moveCount > *maxMoves {
		return false
	}
	return true
}

// needsGameAnalysis returns true if game analysis is required for any enabled filter.
func needsGameAnalysis(ctx *ProcessingContext) bool {
	cfg := ctx.cfg
	return *checkmateFilter || *stalemateFilter || ctx.detector != nil ||
		*fiftyMoveFilter || *repetitionFilter || *underpromotionFilter ||
		*higherRatedWinner || *lowerRatedWinner ||
		cfg.Annotation.AddFENComments || cfg.Annotation.AddHashComments || cfg.Annotation.AddHashTag
}

// applyFeatureFilters applies game feature filters (checkmate, stalemate, etc).
func applyFeatureFilters(result *FilterResult, game *chess.Game, matched bool) bool {
	if !matched {
		return false
	}

	if *checkmateFilter && !engine.IsCheckmate(result.Board) {
		return false
	}

	if *stalemateFilter && !engine.IsStalemate(result.Board) {
		return false
	}

	if *fiftyMoveFilter && (result.GameInfo == nil || !result.GameInfo.HasFiftyMoveRule) {
		return false
	}

	if *repetitionFilter && (result.GameInfo == nil || !result.GameInfo.HasRepetition) {
		return false
	}

	if *underpromotionFilter && (result.GameInfo == nil || !result.GameInfo.HasUnderpromotion) {
		return false
	}

	if *commentedFilter && !processing.HasComments(game) {
		return false
	}

	if (*higherRatedWinner || *lowerRatedWinner) && !checkRatingWinner(game) {
		return false
	}

	return true
}

// checkRatingWinner checks if the game result matches the rating-based winner filter.
func checkRatingWinner(game *chess.Game) bool {
	whiteElo := parseElo(game.Tags["WhiteElo"])
	blackElo := parseElo(game.Tags["BlackElo"])

	if whiteElo <= 0 || blackElo <= 0 {
		return false
	}

	gameResult := game.Tags["Result"]

	if *higherRatedWinner {
		higherWon := (whiteElo > blackElo && gameResult == "1-0") ||
			(blackElo > whiteElo && gameResult == "0-1")
		if !higherWon {
			return false
		}
	}

	if *lowerRatedWinner {
		lowerWon := (whiteElo < blackElo && gameResult == "1-0") ||
			(blackElo < whiteElo && gameResult == "0-1")
		if !lowerWon {
			return false
		}
	}

	return true
}

// addAnnotations adds requested annotations to a matched game.
func addAnnotations(game *chess.Game, result *FilterResult, cfg *config.Config) {
	if cfg.Annotation.AddPlyCount {
		game.Tags["PlyCount"] = strconv.Itoa(result.PlyCount)
	}

	if cfg.Annotation.AddHashTag && result.Board != nil {
		hash := hashing.GenerateZobristHash(result.Board)
		game.Tags["HashCode"] = fmt.Sprintf("%016x", hash)
	}
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
