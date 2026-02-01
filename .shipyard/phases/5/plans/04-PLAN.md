---
phase: test-coverage-processing
plan: 04
wave: 2
dependencies: [01, 02, 03]
must_haves:
  - Test the full applyFilters pipeline with ProcessingContext
  - Test outputGamesSequential and outputGamesParallel produce same results
  - Test SplitWriter file rotation, processInput, and game output helpers
  - Combined coverage exceeds 70% for the package
  - All tests pass with go test -race
files_touched:
  - cmd/pgn-extract/processor_test.go
tdd: false
---

# Plan 04: Processing Pipeline Integration and SplitWriter Tests

## Goal

Cover the remaining high-value functions in `processor.go` that tie the processing
pipeline together. This plan depends on Wave 1 plans because the pipeline delegates to
filter and analysis functions already tested there. Current processor.go coverage is 19.3%
(only ECO split writer and duplicate detection tests exist).

## Key Functions to Cover

- `processInput` -- parses games from a reader (thin wrapper around parser)
- `applyFilters` -- the main filter orchestration function
- `outputGamesSequential` -- sequential processing pipeline
- `outputGamesWithProcessing` -- routing between sequential/parallel
- `handleGameOutput` -- duplicate detection + output routing
- `outputNonMatchingGame`, `outputDuplicateGame` -- conditional output helpers
- `outputGameWithECOSplit` -- ECO split routing + JSON collection
- `shouldOutputUnique` -- simple boolean logic
- `withOutputFile` -- output redirection helper
- `processGameWorker` -- worker function for parallel processing
- `SplitWriter.Write`, `SplitWriter.IncrementGameCount`, `SplitWriter.Close`

## Key Challenge

These functions use global flag state and require ProcessingContext setup. Tests must:
1. Create a minimal ProcessingContext with config.NewConfig()
2. Save/restore global flag pointers and counters (matchedCount, gamePositionCounter)
3. Use bytes.Buffer as cfg.OutputFile for output capture
4. Reset atomic counters between tests

## Tasks

<task id="1" files="cmd/pgn-extract/processor_test.go" tdd="false">
  <action>
    Add tests for SplitWriter, processInput, withOutputFile, and simple output helpers:
    - SplitWriter: create with temp dir, write 3 games to a 2-games-per-file writer. Verify file rotation (2 files created). Test custom pattern. Test Close on nil file. Test IncrementGameCount advances counter.
    - processInput: create a strings.Reader with valid PGN, call processInput, verify returned games slice length and first game's tags. Test with empty input (0 games). Test with malformed input (returns what it can parse).
    - withOutputFile: set cfg.OutputFile to buffer A, call withOutputFile with buffer B and a function that writes. Verify B has the write and A is restored as cfg.OutputFile.
    - outputNonMatchingGame: with cfg.NonMatchingFile set to a buffer, verify game is written. With cfg.NonMatchingFile nil, verify no panic.
    - outputDuplicateGame: with cfg.Duplicate.DuplicateFile set to a buffer, verify game is written. With nil, verify no panic. Test with JSONFormat enabled.
    - shouldOutputUnique: test all combinations of Suppress and SuppressOriginals flags.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestSplitWriterRotation|TestSplitWriterCustomPattern|TestProcessInput|TestWithOutputFile|TestOutputNonMatchingGame|TestOutputDuplicateGame|TestShouldOutputUnique" ./cmd/pgn-extract/ -v</verify>
  <done>SplitWriter creates correct number of files with rotation. processInput returns parsed games. withOutputFile correctly swaps and restores output. All output helper tests pass.</done>
</task>

<task id="2" files="cmd/pgn-extract/processor_test.go" tdd="false">
  <action>
    Add integration tests for the applyFilters pipeline and handleGameOutput:
    - applyFilters with minimal context (no filters, no ECO, no detector): game passes through matched=true
    - applyFilters with fixableMode enabled: verify game gets fixed tags
    - applyFilters with negateMatch enabled: verify matched is inverted
    - applyFilters with minPly filter: short game fails, long game passes
    - applyFilters with checkmateFilter: checkmate game passes, non-checkmate fails
    - handleGameOutput with no detector: game is output, returns (1, 0)
    - handleGameOutput with detector, unique game: returns (1, 0)
    - handleGameOutput with detector, duplicate game: returns (0, 1)
    - handleGameOutput with detector, duplicate + SuppressOriginals: returns (1, 1)
    - outputGameWithECOSplit with JSONFormat: verify game added to jsonGames slice
    - outputGameWithECOSplit without JSON or ECO: verify game written to cfg.OutputFile

    Each test must:
    1. Reset matchedCount and gamePositionCounter to 0 using atomic.StoreInt64
    2. Save/restore all global flag pointers modified
    3. Create ProcessingContext with config.NewConfig() and bytes.Buffer as OutputFile
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestApplyFiltersMinimal|TestApplyFiltersFixable|TestApplyFiltersNegate|TestApplyFiltersPlyBounds|TestApplyFiltersCheckmate|TestHandleGameOutput|TestOutputGameWithECOSplit" ./cmd/pgn-extract/ -v</verify>
  <done>applyFilters integration tests cover the main pipeline paths. handleGameOutput correctly routes based on detector presence and duplicate state. All tests pass with -race.</done>
</task>

<task id="3" files="cmd/pgn-extract/processor_test.go" tdd="false">
  <action>
    Add tests for the sequential and parallel processing pipelines:
    - outputGamesSequential: parse 3 test games, create minimal ProcessingContext with bytes.Buffer output. Verify output count matches input count. Verify output contains all game tags.
    - outputGamesSequential with stopAfter: set stopAfter=1, verify only 1 game output.
    - outputGamesSequential with selectOnly: set selectOnly="2", verify only second game output.
    - outputGamesSequential with reportOnly: verify games are counted but not written to output.
    - processGameWorker: create WorkItem with a test game, call processGameWorker, verify ProcessResult has correct Matched, Board, and ShouldOutput fields.
    - outputGamesWithProcessing routing: test with workers=1 (routes to sequential), test with workers>1 and >2 games (routes to parallel). Compare output counts.
    - outputGamesParallel: test with 5+ games and 2 workers. Verify output count matches expected. Verify no data race (run with -race flag).

    Reset all global state (matchedCount, gamePositionCounter, selectOnlySet, skipMatchingSet, flag pointers) before and after each test.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestOutputGamesSequential|TestOutputGamesSequentialStopAfter|TestOutputGamesSequentialSelectOnly|TestOutputGamesSequentialReportOnly|TestProcessGameWorker|TestOutputGamesWithProcessingRouting|TestOutputGamesParallel" ./cmd/pgn-extract/ -v</verify>
  <done>Sequential and parallel pipeline tests pass. Both paths produce correct output counts. processGameWorker maps FilterResult to ProcessResult correctly. All tests pass with -race. Combined package coverage exceeds 70%.</done>
</task>
