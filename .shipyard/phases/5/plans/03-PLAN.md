---
phase: test-coverage-processing
plan: 03
wave: 1
dependencies: []
must_haves:
  - Test pure argument parsing functions in main.go (splitArgsLine, loadArgsFile, loadFileList)
  - Test reportStatistics output formatting
  - Test setup helper functions that do not call os.Exit
  - All tests pass with go test -race
files_touched:
  - cmd/pgn-extract/main_test.go
tdd: false
---

# Plan 03: Main Package Argument Parsing and Helper Tests

## Goal

Cover the testable functions in `main.go`. Current coverage is 0%. While `main()` itself
and functions that call `os.Exit` cannot be unit-tested, there are several pure functions
and setup helpers that can be tested directly.

## Testable Functions (no os.Exit paths)

- `splitArgsLine` -- pure string splitting with quote handling
- `loadArgsFile` -- reads file, parses args (testable with temp files)
- `loadFileList` -- reads file list (testable with temp files)
- `reportStatistics` -- writes formatted output (capture with bytes.Buffer via stderr redirect)
- `loadMaterialMatcher` -- returns MaterialMatcher based on flag state
- `loadVariationMatcher` -- returns nil or VariationMatcher based on flag state
- `setupGameFilter` -- returns configured GameFilter based on flag state
- `parseCQLQuery` -- returns nil or parsed CQL node based on flag state (skip os.Exit paths)
- `usage` -- prints usage info

## Functions to Skip

- `main()` -- orchestration with os.Exit
- `setupLogFile`, `setupOutputFile`, `setupDuplicateFile` -- os.Exit on error
- `setupDuplicateDetector` -- os.Exit on error, complex file I/O
- `loadECOClassifier` -- os.Exit on error
- `loadArgsFromFileIfSpecified` -- modifies os.Args
- `processAllInputs` -- requires full context setup

## Tasks

<task id="1" files="cmd/pgn-extract/main_test.go" tdd="false">
  <action>
    Create main_test.go with table-driven tests for the pure parsing functions:
    - splitArgsLine: simple args "a b c", quoted strings '"hello world" foo', single-quoted "'hello world' foo", mixed quotes, empty string, tabs as separators, adjacent quotes, no spaces (single arg)
    - loadArgsFile: create temp file with args (one per line), comments (#), empty lines, quoted args. Verify parsed args list. Test error case with non-existent file.
    - loadFileList: create temp file with file paths, comments, empty lines. Verify returned list. Test error case with non-existent file.
    - reportStatistics: use a mock DuplicateChecker (or nil) and capture stderr output. Verify "game(s) matched" vs "game(s) output, N duplicate(s)" format based on whether detector is nil.

    For reportStatistics, redirect os.Stderr temporarily or use the function's fmt.Fprintf pattern.
    Note: reportStatistics writes to os.Stderr directly, so capture it by redirecting os.Stderr to a pipe/buffer.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestSplitArgsLine|TestLoadArgsFile|TestLoadFileList|TestReportStatistics" ./cmd/pgn-extract/ -v</verify>
  <done>All argument parsing tests pass. splitArgsLine handles quoted strings correctly. loadArgsFile and loadFileList handle comments, empty lines, and file errors.</done>
</task>

<task id="2" files="cmd/pgn-extract/main_test.go" tdd="false">
  <action>
    Add tests for the setup helper functions that can be tested without os.Exit risks:
    - setupGameFilter: test with no flags set (empty filter, HasCriteria=false). Test with playerFilter set (HasCriteria=true). Test with whiteFilter, blackFilter, ecoFilter, resultFilter individually. Save/restore flag pointers.
    - loadMaterialMatcher: test with materialMatch="" and materialMatchExact="" (returns nil). Test with materialMatch="Q:q" (returns non-nil). Test with materialMatchExact="KQR:kqr" (returns non-nil, exact=true).
    - loadVariationMatcher: test with variationFile="" and positionFile="" (returns nil). Cannot easily test with files without creating test fixture files, so just test the nil case.
    - parseCQLQuery: test with cqlQuery="" and cqlFile="" (returns nil). Test with cqlQuery="mate" (returns non-nil node). Skip file-based test (os.Exit on error).
    - usage: call usage() and verify it does not panic (smoke test only).

    Save and restore all flag pointers modified during tests.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestSetupGameFilter|TestLoadMaterialMatcher|TestLoadVariationMatcher|TestParseCQLQuery|TestUsage" ./cmd/pgn-extract/ -v</verify>
  <done>All setup helper tests pass. setupGameFilter configures filter criteria from flags. loadMaterialMatcher and parseCQLQuery return correct values based on flag state.</done>
</task>

<task id="3" files="cmd/pgn-extract/main_test.go" tdd="false">
  <action>
    Add tests for processInput and withOutputFile (from processor.go, but tested via main_test.go to avoid processor_test.go conflicts):

    Actually -- processInput is in processor.go and should go in Plan 04's processor_test.go.

    Instead, add these tests to main_test.go:
    - Test setupGameFilter with tagFile flag pointing to a real tag criteria file (create temp file with "White \"Fischer\"" content). Verify filter matches a game with White=Fischer.
    - Test setupGameFilter with fenFilter flag (use a simple FEN string). Verify non-nil return and HasCriteria=true.
    - Test setupGameFilter with useSoundex and tagSubstring flags enabled. Verify filter has those options set.
    - Test loadArgsFromFileIfSpecified: save/restore os.Args. Set os.Args to include -A pointing to a temp args file. Verify returned args match file contents. Test with -A=filename syntax. Test with no -A flag (returns nil).

    Note: loadArgsFromFileIfSpecified has an os.Exit path for file errors; only test the success and no-flag cases.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestSetupGameFilterWithTagFile|TestSetupGameFilterWithFEN|TestSetupGameFilterOptions|TestLoadArgsFromFileIfSpecified" ./cmd/pgn-extract/ -v</verify>
  <done>Extended setup helper tests pass. setupGameFilter works with tag files and FEN filters. loadArgsFromFileIfSpecified correctly parses -A and -A= syntax.</done>
</task>
