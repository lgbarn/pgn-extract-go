// filters.go - Game filtering logic
package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/hashing"
	"github.com/lgbarn/pgn-extract-go/internal/processing"
)

// Parsed selection sets (initialized once at startup)
var (
	selectOnlySet   map[int]bool
	skipMatchingSet map[int]bool
	parsedPlyRange  [2]int // [min, max]
	parsedMoveRange [2]int // [min, max]
)

// initSelectionSets parses the selection flags into sets for O(1) lookup.
func initSelectionSets() {
	if *selectOnly != "" {
		selectOnlySet = parseIntSet(*selectOnly)
	}
	if *skipMatching != "" {
		skipMatchingSet = parseIntSet(*skipMatching)
	}
	if *plyRange != "" {
		parsedPlyRange = parseRange(*plyRange)
	}
	if *moveRange != "" {
		parsedMoveRange = parseRange(*moveRange)
	}
}

// parseIntSet parses a comma-separated list of integers into a set.
func parseIntSet(s string) map[int]bool {
	result := make(map[int]bool)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if n, err := strconv.Atoi(part); err == nil {
			result[n] = true
		}
	}
	return result
}

// parseRange parses a range string like "20-40" into [min, max].
func parseRange(s string) [2]int {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return [2]int{0, 0}
	}
	min, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	max, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return [2]int{min, max}
}

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

	// Check for same-setup duplicates (deleteSameSetup flag)
	if ctx.setupDetector != nil && ctx.setupDetector.CheckAndAdd(game) {
		return FilterResult{Matched: false}
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

	// Exact ply match takes precedence
	if *exactPly > 0 && plyCount != *exactPly {
		return false
	}

	// Determine effective min/max from range or individual bounds
	minBound := *minPly
	if parsedPlyRange[0] > minBound {
		minBound = parsedPlyRange[0]
	}

	maxBound := *maxPly
	if parsedPlyRange[1] > 0 && (maxBound == 0 || parsedPlyRange[1] < maxBound) {
		maxBound = parsedPlyRange[1]
	}

	if minBound > 0 && plyCount < minBound {
		return false
	}
	if maxBound > 0 && plyCount > maxBound {
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

	// Exact move match takes precedence
	if *exactMove > 0 && moveCount != *exactMove {
		return false
	}

	// Determine effective min/max from range or individual bounds
	minBound := *minMoves
	if parsedMoveRange[0] > minBound {
		minBound = parsedMoveRange[0]
	}

	maxBound := *maxMoves
	if parsedMoveRange[1] > 0 && (maxBound == 0 || parsedMoveRange[1] < maxBound) {
		maxBound = parsedMoveRange[1]
	}

	if minBound > 0 && moveCount < minBound {
		return false
	}
	if maxBound > 0 && moveCount > maxBound {
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
		*seventyFiveMoveFilter || *fiveFoldRepFilter ||
		*insufficientFilter || *materialOddsFilter ||
		cfg.Annotation.AddFENComments || cfg.Annotation.AddHashComments || cfg.Annotation.AddHashTag
}

// applyFeatureFilters applies game feature filters (checkmate, stalemate, etc).
func applyFeatureFilters(result *FilterResult, game *chess.Game, matched bool) bool {
	if !matched {
		return false
	}

	// Board-based ending filters
	if !applyEndingFilters(result.Board) {
		return false
	}

	// GameInfo-based filters
	if !applyGameInfoFilters(result.GameInfo) {
		return false
	}

	// Game-based filters
	if *commentedFilter && !processing.HasComments(game) {
		return false
	}

	if (*higherRatedWinner || *lowerRatedWinner) && !checkRatingWinner(game) {
		return false
	}

	if *pieceCount > 0 && !checkPieceCount(game, *pieceCount) {
		return false
	}

	// Setup tag filtering
	if *noSetupTags && game.HasTag("SetUp") {
		return false
	}

	if *onlySetupTags && !game.HasTag("SetUp") {
		return false
	}

	return true
}

// applyEndingFilters checks board-based ending conditions.
func applyEndingFilters(board *chess.Board) bool {
	if *checkmateFilter && !engine.IsCheckmate(board) {
		return false
	}
	if *stalemateFilter && !engine.IsStalemate(board) {
		return false
	}
	return true
}

// applyGameInfoFilters checks GameInfo-based conditions.
func applyGameInfoFilters(info *GameAnalysis) bool {
	if info == nil {
		// If any GameInfo filter is enabled but info is nil, fail
		if *fiftyMoveFilter || *repetitionFilter || *underpromotionFilter ||
			*seventyFiveMoveFilter || *fiveFoldRepFilter ||
			*insufficientFilter || *materialOddsFilter {
			return false
		}
		return true
	}

	if *fiftyMoveFilter && !info.HasFiftyMoveRule {
		return false
	}
	if *repetitionFilter && !info.HasRepetition {
		return false
	}
	if *underpromotionFilter && !info.HasUnderpromotion {
		return false
	}
	if *seventyFiveMoveFilter && !info.Has75MoveRule {
		return false
	}
	if *fiveFoldRepFilter && !info.Has5FoldRepetition {
		return false
	}
	if *insufficientFilter && !info.HasInsufficientMaterial {
		return false
	}
	if *materialOddsFilter && !info.HasMaterialOdds {
		return false
	}
	return true
}

// checkPieceCount checks if the game ever reaches a position with exactly N pieces.
func checkPieceCount(game *chess.Game, targetCount int) bool {
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN) //nolint:errcheck // InitialFEN is known valid

	// Check initial position
	if countPieces(board) == targetCount {
		return true
	}

	// Check after each move
	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}
		if countPieces(board) == targetCount {
			return true
		}
	}

	return false
}

// countPieces counts the total number of pieces on the board (including kings).
func countPieces(board *chess.Board) int {
	count := 0
	for rank := chess.Rank(chess.FirstRank); rank <= chess.Rank(chess.LastRank); rank++ {
		for col := chess.Col(chess.FirstCol); col <= chess.Col(chess.LastCol); col++ {
			piece := board.Get(col, rank)
			if piece != chess.Empty && piece != chess.Off {
				count++
			}
		}
	}
	return count
}

// checkRatingWinner checks if the game result matches the rating-based winner filter.
func checkRatingWinner(game *chess.Game) bool {
	whiteElo := parseElo(game.Tags["WhiteElo"])
	blackElo := parseElo(game.Tags["BlackElo"])

	if whiteElo <= 0 || blackElo <= 0 {
		return false
	}

	result := game.Tags["Result"]
	whiteWon := result == "1-0"
	blackWon := result == "0-1"

	if *higherRatedWinner {
		higherWon := (whiteElo > blackElo && whiteWon) || (blackElo > whiteElo && blackWon)
		if !higherWon {
			return false
		}
	}

	if *lowerRatedWinner {
		lowerWon := (whiteElo < blackElo && whiteWon) || (blackElo < whiteElo && blackWon)
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

// gamePositionCounter tracks the position of games being processed (1-indexed)
var gamePositionCounter int64

// IncrementMatchedCount atomically increments the matched game counter
func IncrementMatchedCount() int64 {
	return atomic.AddInt64(&matchedCount, 1)
}

// GetMatchedCount returns the current matched game count
func GetMatchedCount() int64 {
	return atomic.LoadInt64(&matchedCount)
}

// IncrementGamePosition atomically increments the game position counter and returns the new position
func IncrementGamePosition() int64 {
	return atomic.AddInt64(&gamePositionCounter, 1)
}

// checkGamePosition checks if the game at the given position should be processed.
// Returns true if the game should be processed, false if it should be skipped.
func checkGamePosition(position int) bool {
	// If selectOnly is specified, only include games at those positions
	if len(selectOnlySet) > 0 {
		return selectOnlySet[position]
	}
	// If skipMatching is specified, exclude games at those positions
	if len(skipMatchingSet) > 0 {
		return !skipMatchingSet[position]
	}
	return true
}

// truncateMoves applies move truncation options to the game.
// This modifies the game's move list based on dropPly, startPly, plyLimit.
func truncateMoves(game *chess.Game) {
	if *dropPly <= 0 && *startPly <= 0 && *plyLimit <= 0 && *dropBefore == "" {
		return
	}

	// Handle dropBefore - find comment matching the string
	dropBeforePly := 0
	if *dropBefore != "" {
		dropBeforePly = findCommentPly(game, *dropBefore)
	}

	// Calculate effective start ply
	effectiveStart := 0
	if *dropPly > 0 {
		effectiveStart = *dropPly
	}
	if *startPly > effectiveStart {
		effectiveStart = *startPly
	}
	if dropBeforePly > effectiveStart {
		effectiveStart = dropBeforePly
	}

	// Calculate effective limit
	effectiveLimit := 0
	if *plyLimit > 0 {
		effectiveLimit = *plyLimit
	}

	// Apply truncation
	if effectiveStart > 0 || effectiveLimit > 0 {
		game.Moves = truncateMoveList(game.Moves, effectiveStart, effectiveLimit)
	}
}

// findCommentPly finds the ply number where a comment contains the given string.
// Returns 0 if not found.
func findCommentPly(game *chess.Game, pattern string) int {
	ply := 0
	for move := game.Moves; move != nil; move = move.Next {
		ply++
		for _, comment := range move.Comments {
			if comment != nil && strings.Contains(comment.Text, pattern) {
				return ply
			}
		}
	}
	return 0
}

// truncateMoveList truncates the move list, skipping the first 'skip' plies
// and limiting to 'limit' plies (0 = no limit).
func truncateMoveList(moves *chess.Move, skip, limit int) *chess.Move {
	if moves == nil {
		return nil
	}

	// Skip first N plies
	current := moves
	skipped := 0
	for current != nil && skipped < skip {
		current = current.Next
		skipped++
	}

	if current == nil {
		return nil
	}

	// Create new head
	newHead := current
	newHead.Prev = nil

	// Apply limit if specified
	if limit > 0 {
		count := 1
		for current.Next != nil && count < limit {
			current = current.Next
			count++
		}
		current.Next = nil
	}

	return newHead
}
