---
phase: test-coverage-processing
plan: 01
wave: 1
dependencies: []
must_haves:
  - Test all pure helper functions in filters.go (parseIntSet, parseRange, checkPlyBounds, checkMoveBounds, checkGamePosition, truncateMoveList, findCommentPly, countPieces, checkRatingWinner)
  - Test filter sub-pipelines (applyEndingFilters, applyGameInfoFilters, applyFeatureFilters, addAnnotations)
  - Test truncateMoves end-to-end with game objects
  - All tests pass with go test -race
files_touched:
  - cmd/pgn-extract/filters_test.go
tdd: false
---

# Plan 01: Filter Pure Functions and Sub-Pipeline Tests

## Goal

Cover the 25+ uncovered functions in `filters.go`. This file has the highest density of
testable pure functions and filter sub-pipelines. Current coverage is 7.4% with only
`parseElo`, `cleanString`, and `MatchedCountOperations` tested. Most functions are pure
(take inputs, return outputs) or require only simple flag state setup.

## Key Challenge

Many filter functions read from global `flag` pointers (e.g., `*checkmateFilter`,
`*exactPly`). Tests must save and restore these pointers between test cases to avoid
cross-test pollution. A helper pattern like `defer func(old int) { *exactPly = old }(*exactPly)`
should be used consistently.

## Tasks

<task id="1" files="cmd/pgn-extract/filters_test.go" tdd="false">
  <action>
    Add table-driven tests for all pure helper functions in filters.go:
    - parseIntSet: empty string, single value, multiple values, invalid entries, whitespace
    - parseRange: valid "20-40", missing dash, empty, extra parts
    - checkPlyBounds: with exactPly, minPly, maxPly, parsedPlyRange, combinations, already-false matched
    - checkMoveBounds: same pattern as checkPlyBounds but for moves
    - checkGamePosition: with selectOnlySet, skipMatchingSet, both empty, position in/out of set
    - countPieces: initial board (32 pieces), empty board concept
    - checkRatingWinner: higher wins, lower wins, equal ratings, missing ratings, draw results
    - findCommentPly: game with comments, no comments, pattern not found
    - truncateMoveList: skip 0, skip N, limit N, skip+limit, nil moves, skip past end

    Each test must save/restore any global flag state it modifies.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestParseIntSet|TestParseRange|TestCheckPlyBounds|TestCheckMoveBounds|TestCheckGamePosition|TestCountPieces|TestCheckRatingWinner|TestFindCommentPly|TestTruncateMoveList" ./cmd/pgn-extract/ -v</verify>
  <done>All pure helper function tests pass. Each function has at least 3 test cases covering normal, boundary, and edge conditions.</done>
</task>

<task id="2" files="cmd/pgn-extract/filters_test.go" tdd="false">
  <action>
    Add tests for the filter sub-pipeline functions that require Board or GameAnalysis objects:
    - applyEndingFilters: nil board (should pass unless filters enabled), with checkmateFilter true and checkmate board, with stalemateFilter true
    - applyGameInfoFilters: nil info with no filters (pass), nil info with filter enabled (fail), info with each flag set (fiftyMoveFilter, repetitionFilter, underpromotionFilter, etc.)
    - applyFeatureFilters: with commentedFilter and game with/without comments, with noSetupTags/onlySetupTags and game with/without SetUp tag, with pieceCount filter
    - addAnnotations: with AddPlyCount enabled (check PlyCount tag added), with AddHashTag and non-nil board (check HashCode tag added)
    - applyTagFilters: with nil gameFilter (pass-through), with gameFilter that has criteria matching/not-matching
    - applyPatternFilters: verify it returns matched unchanged (no-op)

    Use testutil.MustParseGame to create test games. Save/restore global flag pointers.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestApplyEndingFilters|TestApplyGameInfoFilters|TestApplyFeatureFilters|TestAddAnnotations|TestApplyTagFilters|TestApplyPatternFilters" ./cmd/pgn-extract/ -v</verify>
  <done>All filter sub-pipeline tests pass. applyEndingFilters, applyGameInfoFilters, applyFeatureFilters, addAnnotations, applyTagFilters, and applyPatternFilters each have tests covering enabled/disabled filter paths.</done>
</task>

<task id="3" files="cmd/pgn-extract/filters_test.go" tdd="false">
  <action>
    Add integration-level tests for the top-level filter orchestration:
    - truncateMoves: test with dropPly, startPly, plyLimit, dropBefore (with comment), combined flags, no flags (no-op)
    - initSelectionSets: test that it populates selectOnlySet and skipMatchingSet from flag values
    - IncrementGamePosition + checkGamePosition: integration test showing position tracking with selectOnly/skipMatching
    - applyValidation: with strictMode on valid/invalid game, with validateMode on valid/invalid game, with both off (nil return)
    - needsGameAnalysis: with various flag combinations

    Restore all global state (selectOnlySet, skipMatchingSet, parsedPlyRange, parsedMoveRange, flag pointers, gamePositionCounter) after each test.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestTruncateMoves|TestInitSelectionSets|TestGamePositionTracking|TestApplyValidation|TestNeedsGameAnalysis" ./cmd/pgn-extract/ -v</verify>
  <done>All orchestration-level filter tests pass. truncateMoves, initSelectionSets, applyValidation, and needsGameAnalysis are covered with multiple scenarios each.</done>
</task>
