# Phase 5 Research: Test Coverage for cmd/pgn-extract Package

**Date:** 2026-02-01
**Current Coverage:** 7.8% (down from 36.1% noted in ROADMAP.md)
**Target Coverage:** 70%+

## Executive Summary

The `cmd/pgn-extract` package has significantly lower coverage than expected (7.8% vs 36.1% reported in ROADMAP). The existing tests are mostly integration tests that verify end-to-end behavior but don't exercise the internal processing logic. The key challenge is that most functions at 0% coverage are tightly coupled to CLI initialization (`main`, `os.Exit` calls) or require extensive setup (file I/O, duplicate detection, ECO classification).

## Coverage Analysis

### Current Coverage by File

```
analysis.go:      Mixed (0-100%)
filters.go:       7.4% (3 functions at 100%, rest at 0%)
flags.go:         0% (all flag application functions)
main.go:          0% (all setup and CLI functions)
processor.go:     19.3% (ECO split writer partially tested)
```

### Per-Function Coverage Breakdown

#### processor.go (19.3% coverage)

**Fully Covered (100%):**
- `replayGame` - wrapper function tested through processor tests
- `cleanString` - has dedicated unit test
- `ECOSplitWriter.Close` - tested in ECO split tests
- `ECOSplitWriter.FileCount` - tested in ECO split tests
- `ECOSplitWriter.OpenHandleCount` - tested in ECO split tests

**Partially Covered (66-90%):**
- `NewECOSplitWriter` - 66.7% (basic instantiation tested)
- `ECOSplitWriter.WriteGame` - 88.9% (tested but not all error paths)
- `ECOSplitWriter.getOrCreateFile` - 81.0% (LRU logic tested)
- `ECOSplitWriter.evictIfNeeded` - 90.9% (edge cases covered)
- `ECOSplitWriter.getECOPrefix` - 45.5% (only tested for valid ECO codes)

**Not Covered (0%):**
- `withOutputFile` - helper for temporary output redirection
- `NewSplitWriter` - game-count based file splitter
- `NewSplitWriterWithPattern` - custom pattern splitter
- `SplitWriter.Write` - file rotation logic
- `SplitWriter.IncrementGameCount` - counter management
- `SplitWriter.Close` - cleanup
- `processInput` - PGN parsing from reader
- `outputGamesWithProcessing` - main processing orchestrator
- `outputGamesSequential` - single-threaded game output
- `outputNonMatchingGame` - negated match output
- `handleGameOutput` - duplicate detection + output
- `shouldOutputUnique` - duplicate suppression logic
- `outputDuplicateGame` - duplicate file output
- `outputGamesParallel` - worker pool orchestration
- `processGameWorker` - parallel game processing
- `outputGameWithECOSplit` - game output with ECO splitting

#### filters.go (7.4% coverage)

**Fully Covered (100%):**
- `parseElo` - has dedicated unit test
- `cleanString` - has dedicated unit test
- `IncrementMatchedCount` - tested through integration
- `GetMatchedCount` - tested through integration

**Not Covered (0%):**
- `initSelectionSets` - parses selectOnly/skipMatching flags
- `parseIntSet` - comma-separated integer parsing
- `parseRange` - range string parsing (e.g., "20-40")
- `applyFilters` - **CRITICAL** main filter pipeline
- `applyValidation` - strict/validate mode checks
- `applyTagFilters` - game filter, CQL, variation, material matching
- `applyPatternFilters` - currently a no-op
- `checkPlyBounds` - ply count filtering
- `checkMoveBounds` - move count filtering
- `needsGameAnalysis` - determines if board analysis needed
- `applyFeatureFilters` - checkmate/stalemate/feature filters
- `applyEndingFilters` - board state ending checks
- `applyGameInfoFilters` - 50-move, repetition, underpromotion
- `checkPieceCount` - piece count filter
- `countPieces` - board piece counting
- `checkRatingWinner` - rating-based winner filter
- `addAnnotations` - add PlyCount, HashCode tags
- `IncrementGamePosition` - game position tracking
- `checkGamePosition` - selectOnly/skipMatching logic
- `truncateMoves` - move truncation (dropPly, startPly, etc.)
- `findCommentPly` - find comment matching pattern
- `truncateMoveList` - move list manipulation

#### main.go (0% coverage)

**All functions at 0%:**
- `main` - CLI entry point (calls os.Exit, hard to test)
- `setupLogFile` - log file creation
- `setupOutputFile` - output file creation
- `setupDuplicateFile` - duplicate output file creation
- `setupDuplicateDetector` - detector initialization + checkfile loading
- `loadECOClassifier` - ECO file loading
- `setupGameFilter` - filter configuration from flags
- `loadVariationMatcher` - variation/position file loading
- `loadMaterialMatcher` - material match criteria parsing
- `parseCQLQuery` - CQL query/file parsing
- `processAllInputs` - main processing loop
- `reportStatistics` - final stats output
- `usage` - help text
- `loadArgsFile` - argument file parsing
- `splitArgsLine` - quote-aware argument splitting
- `loadFileList` - file list loading
- `loadArgsFromFileIfSpecified` - -A flag handling

#### analysis.go (Mixed coverage)

**Fully Covered (100%):**
- `replayGame` - wrapper tested through integration tests
- `cleanString` - has dedicated unit test

**Not Covered (0%):**
- `analyzeGame` - wrapper around processing.AnalyzeGame
- `validateGame` - wrapper around processing.ValidateGame
- `matchesCQL` - CQL query evaluation
- `fixGame` - game fixing orchestrator
- `fixMissingTags` - add missing required tags
- `fixResultTag` - normalize result tags
- `fixDateFormat` - normalize date format
- `cleanAllTags` - strip control characters from all tags

#### flags.go (0% coverage)

**All flag application functions at 0%:**
- `applyFlags` - main flag application orchestrator
- `applyPhase4Flags` - Phase 4 feature flags
- `applyTagOutputFlags` - tag output configuration
- `applyContentFlags` - content filtering flags
- `applyOutputFormatFlags` - output format configuration
- `applyMoveBoundsFlags` - move/ply bounds configuration
- `applyAnnotationFlags` - annotation flags
- `applyFilterFlags` - filter flags
- `applyDuplicateFlags` - duplicate detection flags

## Existing Test Infrastructure

### Test Helper Functions

**From golden_test.go:**
- `testdataDir()` - returns testdata path
- `inputFile(name)` - constructs testdata/infiles path
- `testEcoFile()` - returns ECO file path
- `buildTestBinary(t)` - builds pgn-extract binary for integration tests
- `runPgnExtract(t, args...)` - runs binary with args, returns stdout/stderr
- `countGames(pgn)` - counts "[Event " occurrences
- `containsMove(output, move)` - checks for move in output

**From processor_test.go:**
- `makeMinimalGame(eco)` - creates game with ECO tag for testing

**From clock_test.go:**
- `createTempPGN(t, filename, content)` - creates temp PGN file
- `createTempPGNWithClocks(t)` - PGN with clock annotations
- `createTempPGNWithMixedComments(t)` - PGN with mixed comment types

### Test Patterns Observed

1. **Integration-style tests** - Build binary, run with flags, verify output
2. **Golden file tests** - Compare output against expected results
3. **Parallel vs Sequential comparison** - Verify parallel matches sequential
4. **ECO split writer tests** - LRU cache behavior, file creation
5. **Duplicate detection tests** - Sequential vs thread-safe comparison

### Synthetic PGN Generation

Tests use embedded PGN strings with:
- Minimal games (just tags + one move)
- Specific features (checkmate, stalemate, comments)
- Duplicate patterns (same moves, different tags)
- Multiple games for parallel testing

### Test Data Dependencies

Tests rely on files in `testdata/infiles/`:
- `fischer.pgn` - 34 games for integration tests
- `fools-mate.pgn` - Simple checkmate game
- `petrosian.pgn` - Additional test games
- `najdorf.pgn` - Opening-specific games
- `test-*.pgn` - Feature-specific test files (7, C, N, V, etc.)

## What's Testable vs. Requires Refactoring

### Directly Testable (No Refactoring Needed)

1. **filters.go helper functions:**
   - `parseIntSet` - pure function
   - `parseRange` - pure function
   - `parseElo` - already tested
   - `countPieces` - pure function
   - `checkPlyBounds` - can test with mock values
   - `checkMoveBounds` - can test with mock values
   - `findCommentPly` - needs synthetic game
   - `truncateMoveList` - needs synthetic move list

2. **SplitWriter (processor.go):**
   - `NewSplitWriter` - can test with temp dir
   - `Write` - can test file rotation
   - `IncrementGameCount` - simple counter
   - `Close` - verify cleanup

3. **analysis.go helpers:**
   - `matchesCQL` - needs CQL node + game
   - `fixMissingTags` - pure transformation
   - `fixResultTag` - pure transformation
   - `fixDateFormat` - pure transformation
   - `cleanAllTags` - pure transformation

### Requires Context Setup (But Testable)

1. **Filter pipeline (filters.go):**
   - `applyFilters` - needs ProcessingContext
   - `applyValidation` - needs game + flags
   - `applyTagFilters` - needs context with matchers
   - `applyFeatureFilters` - needs FilterResult + game
   - `checkPieceCount` - needs game
   - `checkRatingWinner` - needs game with ELO tags
   - `addAnnotations` - needs FilterResult

2. **Game processing (processor.go):**
   - `processInput` - needs reader + config
   - `outputGamesSequential` - needs games + context
   - `outputGamesParallel` - needs games + context
   - `handleGameOutput` - needs game + context
   - `outputGameWithECOSplit` - needs game + config

3. **Main setup functions (main.go):**
   - Most setup functions can be tested with temp files
   - Argument parsing functions are pure
   - File loading functions need temp files

### Difficult to Test (Requires Refactoring)

1. **main() function:**
   - Calls `os.Exit` directly
   - Tightly coupled to CLI parsing
   - Would need extract-method refactoring

2. **Global flag variables:**
   - Tests would interfere with each other
   - Need to reset between tests or use dependency injection

3. **Functions calling os.Exit:**
   - `setupLogFile` (exits on error)
   - `setupOutputFile` (exits on error)
   - `loadECOClassifier` (exits on error)
   - All setup functions in main.go

### Recommended Approach for os.Exit Functions

**Option 1:** Test the happy path only, skip error paths
**Option 2:** Refactor to return errors instead of calling os.Exit
**Option 3:** Use integration tests that expect exit codes

For this phase, **Option 1** is recommended to avoid major refactoring.

## Test Strategy by File

### processor.go (Target: 70%+)

**Priority 1: Core processing pipeline (0% → 60%)**
- `outputGamesSequential` - test with synthetic games, verify output
- `handleGameOutput` - test duplicate detection logic
- `processGameWorker` - test parallel processing worker
- `outputGameWithECOSplit` - test ECO splitting logic

**Priority 2: SplitWriter (0% → 100%)**
- `NewSplitWriter` - instantiation test
- `Write` - test file rotation at boundary
- `IncrementGameCount` - verify counter
- `Close` - verify cleanup

**Priority 3: Helpers (50% → 100%)**
- `withOutputFile` - test output redirection
- `processInput` - test PGN parsing
- `getECOPrefix` - test edge cases (unknown, short ECO)

**Strategy:**
- Create synthetic games with specific features
- Use temp directories for output
- Verify file creation, content, rotation
- Test both sequential and parallel paths

### filters.go (Target: 70%+)

**Priority 1: Main filter pipeline (0% → 60%)**
- `applyFilters` - test with various filter combinations
- `applyTagFilters` - test tag matching
- `applyFeatureFilters` - test game feature detection
- `checkPlyBounds` - test boundary conditions
- `checkMoveBounds` - test move count filtering

**Priority 2: Helper functions (0% → 100%)**
- `parseIntSet` - test comma-separated parsing
- `parseRange` - test range parsing
- `initSelectionSets` - test global initialization
- `truncateMoveList` - test move truncation
- `findCommentPly` - test comment search

**Priority 3: Complex filters (0% → 60%)**
- `checkPieceCount` - test piece counting
- `checkRatingWinner` - test rating-based filtering
- `addAnnotations` - test PlyCount/HashCode addition

**Strategy:**
- Unit test pure functions
- Integration test filter pipeline with ProcessingContext
- Use testutil.MustParseGame for synthetic games
- Mock flags using test-specific values

### main.go (Target: 50%+)

**Priority 1: Argument parsing (0% → 80%)**
- `loadArgsFile` - test comment handling, quotes
- `splitArgsLine` - test quote parsing
- `loadFileList` - test file list loading
- `loadArgsFromFileIfSpecified` - test -A flag

**Priority 2: Setup functions - happy path only (0% → 40%)**
- `setupGameFilter` - test filter creation from flags
- `loadVariationMatcher` - test variation file loading
- `loadMaterialMatcher` - test material criteria parsing
- `parseCQLQuery` - test CQL parsing

**Priority 3: Processing loop (0% → 30%)**
- `processAllInputs` - integration test with temp files
- `reportStatistics` - test output formatting

**Skip:**
- `main()` - too tightly coupled
- Error paths in setup functions (call os.Exit)

**Strategy:**
- Test pure functions first (parsing, splitting)
- Use temp files for I/O functions
- Create minimal test fixtures
- Focus on happy path for setup functions

### analysis.go (Target: 80%+)

**Priority 1: Fixing functions (0% → 100%)**
- `fixGame` - test orchestration
- `fixMissingTags` - test tag addition
- `fixResultTag` - test result normalization
- `fixDateFormat` - test date format fixes
- `cleanAllTags` - test control character removal

**Priority 2: Matching (0% → 80%)**
- `matchesCQL` - test CQL evaluation on games
- `analyzeGame` - wrapper test
- `validateGame` - wrapper test

**Strategy:**
- Create games with specific defects
- Verify fixes are applied correctly
- Test CQL matching with simple queries
- Wrappers are thin, just verify they call underlying functions

### flags.go (Target: 30%+)

**Priority: Flag application functions (0% → 30%)**
- `applyFlags` - test orchestration
- `applyTagOutputFlags` - test tag configuration
- `applyContentFlags` - test content filtering setup
- `applyOutputFormatFlags` - test format configuration

**Strategy:**
- Test that flags correctly populate config
- Focus on non-trivial transformations
- Skip simple boolean assignments
- Integration style: set flags, verify config state

## Error Conditions to Test

### File I/O Errors
- Invalid file paths (write tests with read-only dirs)
- Permission errors (skip on Windows, hard to mock)
- Disk full (difficult to mock, skip)

### Split Writer Edge Cases
- Rotation at exact boundary (gameCount == gamesPerFile)
- Rotation with 1 game per file
- Empty games (zero bytes written)
- Close before any writes

### ECO Split Writer Edge Cases
- Games with no ECO tag (→ "unknown" file)
- Games with short ECO codes (< 3 chars)
- Games with invalid ECO codes
- LRU eviction and reopen (already tested)

### Filter Edge Cases
- Empty game (no moves)
- Game with only tags
- Ply/move bounds at 0
- Negative values in filters (should be rejected or ignored)
- Empty selectOnly/skipMatching sets

### Duplicate Detection
- First game (never a duplicate)
- Exact duplicate
- Same position, different tags
- Parallel vs sequential differences

### Parallel Processing
- Single worker (should match sequential)
- More workers than games
- stopAfter in parallel mode
- Worker pool shutdown

## Dependencies on Other Packages

### Internal packages used:
- `internal/chess` - Game, Board, Move structures
- `internal/config` - Config struct
- `internal/parser` - PGN parsing
- `internal/output` - Game output formatting
- `internal/eco` - ECO classification
- `internal/hashing` - Duplicate detection
- `internal/matching` - Tag/variation/material matching
- `internal/cql` - CQL query evaluation
- `internal/processing` - Game analysis functions
- `internal/worker` - Parallel processing worker pool
- `internal/engine` - Chess engine (move application, validation)
- `internal/testutil` - Test utilities (MustParseGame)

### Test dependencies:
- Most internal packages have their own tests
- Can rely on them being correct
- Focus on integration and pipeline logic

## Constraints and Challenges

### 1. Global State
- Flag variables are global
- `matchedCount` and `gamePositionCounter` are global atomics
- Tests may need to reset state between runs

### 2. os.Exit Calls
- Many setup functions call os.Exit on error
- Can't test error paths without refactoring
- Recommendation: skip error paths, test happy path only

### 3. File I/O
- Many functions require actual files
- Use `t.TempDir()` for isolation
- Clean up is automatic with testing.T

### 4. CLI Integration
- Many tests require building the binary
- `buildTestBinary()` is slow but necessary
- Consider separating unit tests from integration tests

### 5. Parallel Processing
- Non-deterministic output order
- Need to sort/compare sets rather than sequences
- Tests must handle race conditions gracefully

### 6. Coverage Measurement
- Current 7.8% is much lower than expected
- May be due to:
  - Integration tests not triggering internal functions
  - Functions only called by main() (not reachable in tests)
  - Dead code paths

## Test Grouping for Implementation Plans

### Plan 1: Filter Pipeline Tests (filters.go)
**Goal:** Raise filters.go from 7.4% to 70%+

**Tasks:**
1. Unit test helper functions (parseIntSet, parseRange, countPieces)
2. Test ply/move bounds checking (checkPlyBounds, checkMoveBounds)
3. Test move truncation (truncateMoveList, findCommentPly)
4. Integration test filter pipeline (applyFilters, applyTagFilters, applyFeatureFilters)
5. Test game position selection (checkGamePosition, initSelectionSets)
6. Test rating filters (checkRatingWinner, parseElo)
7. Test piece count filter (checkPieceCount)
8. Test annotation addition (addAnnotations)

**Estimated effort:** Medium (pure functions + integration)

### Plan 2: Processor Core Tests (processor.go)
**Goal:** Raise processor.go from 19.3% to 70%+

**Tasks:**
1. Test SplitWriter (Write, IncrementGameCount, Close)
2. Test output helpers (withOutputFile, outputNonMatchingGame, outputDuplicateGame)
3. Test game processing pipeline (outputGamesSequential, handleGameOutput)
4. Test parallel processing (outputGamesParallel, processGameWorker)
5. Test processInput (PGN parsing integration)
6. Test ECO split writer edge cases (getECOPrefix for short/invalid codes)
7. Test shouldOutputUnique logic

**Estimated effort:** High (complex pipeline, requires extensive setup)

### Plan 3: Main Setup Tests (main.go)
**Goal:** Raise main.go from 0% to 50%+

**Tasks:**
1. Test argument parsing (loadArgsFile, splitArgsLine, loadArgsFromFileIfSpecified)
2. Test file list loading (loadFileList)
3. Test filter setup (setupGameFilter, loadVariationMatcher, loadMaterialMatcher)
4. Test CQL parsing (parseCQLQuery)
5. Test statistics reporting (reportStatistics)
6. Test processAllInputs (integration with temp files)
7. Skip: main(), error paths in setup functions

**Estimated effort:** Medium (mostly I/O and setup)

### Plan 4: Analysis and Fixing Tests (analysis.go)
**Goal:** Raise analysis.go to 80%+

**Tasks:**
1. Test game fixing functions (fixGame, fixMissingTags, fixResultTag, fixDateFormat)
2. Test tag cleaning (cleanAllTags)
3. Test CQL matching (matchesCQL)
4. Test wrapper functions (analyzeGame, validateGame)

**Estimated effort:** Low (pure functions, mostly)

### Plan 5: Flag Application Tests (flags.go)
**Goal:** Raise flags.go from 0% to 30%+

**Tasks:**
1. Test flag application orchestration (applyFlags)
2. Test tag output configuration (applyTagOutputFlags)
3. Test content flags (applyContentFlags)
4. Test output format configuration (applyOutputFormatFlags)
5. Skip: Simple boolean assignments

**Estimated effort:** Low (configuration testing)

## Recommended Implementation Order

1. **Plan 4 (Analysis/Fixing)** - Low-hanging fruit, pure functions
2. **Plan 1 (Filter Pipeline)** - Core logic, high value
3. **Plan 2 (Processor Core)** - Complex but critical
4. **Plan 3 (Main Setup)** - Integration tests
5. **Plan 5 (Flag Application)** - Nice-to-have, lower priority

## Overall Strategy

1. **Start with pure functions** - parseIntSet, parseRange, fix functions
2. **Move to integration tests** - filter pipeline, processing pipeline
3. **Use synthetic PGN** - embed test games in test code
4. **Leverage existing helpers** - testutil.MustParseGame, createTempPGN
5. **Focus on happy paths** - skip os.Exit error paths
6. **Parallel testing** - verify parallel matches sequential
7. **Edge case coverage** - empty games, boundary conditions

## Success Metrics

- **Overall package coverage:** 7.8% → 70%+
- **filters.go:** 7.4% → 70%+
- **processor.go:** 19.3% → 70%+
- **main.go:** 0% → 50%+
- **analysis.go:** Mixed → 80%+
- **flags.go:** 0% → 30%+

## Known Limitations

1. Can't test main() without refactoring
2. Can't test os.Exit error paths
3. Global flag state may cause test interference
4. Integration tests are slow (require binary build)
5. Parallel tests may be flaky (timing-dependent)

## References

- Existing tests: `*_test.go` files in cmd/pgn-extract/
- Test utilities: `internal/testutil/testutil.go`
- Test data: `testdata/infiles/*.pgn`
- Coverage tool: `go test -coverprofile=cov.out && go tool cover -func=cov.out`
