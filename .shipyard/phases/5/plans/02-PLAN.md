---
phase: test-coverage-processing
plan: 02
wave: 1
dependencies: []
must_haves:
  - Test all fix functions in analysis.go (fixGame, fixMissingTags, fixResultTag, fixDateFormat, cleanAllTags)
  - Test matchesCQL with simple CQL nodes
  - Test all applyFlags sub-functions in flags.go
  - All tests pass with go test -race
files_touched:
  - cmd/pgn-extract/analysis_test.go
  - cmd/pgn-extract/flags_test.go
tdd: false
---

# Plan 02: Analysis Fix Functions and Flag Application Tests

## Goal

Cover the 8 uncovered functions in `analysis.go` and the 9 uncovered functions in
`flags.go`. The analysis fix functions are pure transformations on Game objects, making
them straightforward to test. The flags functions are simple config mapping -- lower value
but easy wins for coverage percentage.

## Key Observations

- `fixMissingTags`, `fixResultTag`, `fixDateFormat`, `cleanAllTags` are all pure functions
  that modify a Game's tags and return a bool indicating whether changes were made.
- `matchesCQL` requires a parsed CQL node and a game -- use `cql.Parse()` to create test nodes.
- `analyzeGame` and `validateGame` are thin wrappers around `processing` package functions;
  test them with simple games to confirm delegation works.
- Flag functions just map flag values to config fields; test with table-driven approach.

## Tasks

<task id="1" files="cmd/pgn-extract/analysis_test.go" tdd="false">
  <action>
    Create analysis_test.go with table-driven tests for all fix and analysis functions:
    - fixMissingTags: game with all tags present (no change), game missing Event (adds "?"), game missing multiple tags
    - fixResultTag: valid results ("1-0", "0-1", "1/2-1/2", "*") unchanged, "white" -> "1-0", "draw" -> "1/2-1/2", "0.5-0.5" -> "1/2-1/2", garbage -> "*"
    - fixDateFormat: normal date unchanged, "2024/01/01" -> "2024.01.01", "2024-01-01" -> "2024.01.01", empty date (no change), "????.??.??" (no change)
    - cleanAllTags: tags with control chars get cleaned, normal tags unchanged, mixed tags
    - fixGame: verify it calls all sub-fixers (game missing tags + bad date + bad result gets all fixed)
    - analyzeGame: simple game returns non-nil board and GameAnalysis
    - validateGame: valid game returns Valid=true, game with moves returns ValidationResult
    - matchesCQL: parse "mate" CQL node, test against checkmate game (true) and non-checkmate game (false); parse "check" node, test against game with check

    Use chess.NewGame() + SetTag() for creating test games, and testutil.MustParseGame for games with moves.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestFixMissingTags|TestFixResultTag|TestFixDateFormat|TestCleanAllTags|TestFixGame|TestAnalyzeGame|TestValidateGame|TestMatchesCQL" ./cmd/pgn-extract/ -v</verify>
  <done>All analysis function tests pass. Fix functions tested with before/after assertions on tags. matchesCQL tested with at least 2 CQL patterns against matching and non-matching games.</done>
</task>

<task id="2" files="cmd/pgn-extract/flags_test.go" tdd="false">
  <action>
    Create flags_test.go with tests for the applyFlags function and its sub-functions:
    - applyFlags: verify it calls all sub-functions by checking resulting config state after setting various flags
    - applyTagOutputFlags: test noTags -> config.NoTags, sevenTagOnly -> config.SevenTagRoster, neither -> default
    - applyContentFlags: test noComments, noNAGs, noVariations, noResults, noClocks, jsonOutput, lineLength mappings
    - applyOutputFormatFlags: test each format string ("lalg", "halg", "elalg", "uci", "epd", "fen") maps to correct config.OutputFormat, unknown string defaults to SAN
    - applyMoveBoundsFlags: test with minPly/maxPly/minMoves/maxMoves set, and with none set (early return)
    - applyAnnotationFlags: test each annotation flag maps correctly
    - applyFilterFlags: test each filter flag maps correctly
    - applyDuplicateFlags: test duplicateCapacity mapping
    - applyPhase4Flags: test nestedComments, splitVariants, chess960Mode, fuzzyDepth

    Each test must save and restore the flag pointer values it modifies. Use a helper:
    ```go
    func withFlag[T any](ptr *T, val T, fn func()) {
        old := *ptr; *ptr = val; defer func() { *ptr = old }(); fn()
    }
    ```
    Or use the defer-restore pattern directly.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestApplyFlags|TestApplyTagOutputFlags|TestApplyContentFlags|TestApplyOutputFormatFlags|TestApplyMoveBoundsFlags|TestApplyAnnotationFlags|TestApplyFilterFlags|TestApplyDuplicateFlags|TestApplyPhase4Flags" ./cmd/pgn-extract/ -v</verify>
  <done>All flag application tests pass. Each applyXxxFlags function has at least 2 test cases verifying correct config field mapping.</done>
</task>

<task id="3" files="cmd/pgn-extract/analysis_test.go" tdd="false">
  <action>
    Add tests for the remaining analysis.go wrapper functions and edge cases:
    - replayGame: test with a simple game (1. e4 e5), verify returned board has pawn on e4 and e5
    - analyzeGame with a game containing underpromotion, verify GameAnalysis.HasUnderpromotion is true
    - analyzeGame with a game containing repetition, verify GameAnalysis.HasRepetition is true
    - matchesCQL with "piece P e4" CQL node against a game starting with 1.e4 (should match after the move)
    - matchesCQL with a game that has no matching positions (returns false)
    - fixResultTag edge cases: "Black" -> "0-1", "1/2" -> "1/2-1/2", empty result tag
    - cleanAllTags with tags containing leading/trailing whitespace
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestReplayGame|TestAnalyzeGameFeatures|TestMatchesCQLPiece|TestFixResultTagEdge|TestCleanAllTagsWhitespace" ./cmd/pgn-extract/ -v</verify>
  <done>Wrapper functions and edge cases all pass. replayGame returns correct board state. analyzeGame detects game features. matchesCQL works with piece placement queries.</done>
</task>
