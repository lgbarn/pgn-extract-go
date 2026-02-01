---
phase: nolint-cleanup
plan: 03
wave: 2
dependencies: [01]
must_haves:
  - All NewBoardFromFEN(InitialFEN) calls replaced with MustBoardFromFEN
  - Close() calls properly handle errors or use defer with logging
  - Type assertion in processor.go uses comma-ok pattern
  - All existing tests pass
  - No regressions in golangci-lint
files_touched:
  - internal/eco/eco.go
  - internal/matching/position.go
  - internal/matching/material.go
  - internal/matching/variation.go
  - internal/output/output.go
  - internal/processing/analyzer.go
  - cmd/pgn-extract/filters.go
  - cmd/pgn-extract/processor.go
  - cmd/pgn-extract/main.go
tdd: false
---

# Plan 03: Replace Known-Valid FEN Calls and Fix Close/Type-Assertion Cleanup

## Goal

Replace all remaining `NewBoardFromFEN(InitialFEN)` calls with `MustBoardFromFEN`
(created in Plan 01), fix `Close()` error handling, and fix the type assertion in
`processor.go`. This plan removes 13 nolint suppressions across consumer files.

## Dependencies

- **Plan 01** must complete first (provides `MustBoardFromFEN` in `internal/engine/fen.go`).

## Nolint Reductions

| File | Line | Category | Action |
|------|------|----------|--------|
| `internal/eco/eco.go:84` | `NewBoardFromFEN(InitialFEN)` | A | Replace with `MustBoardFromFEN` |
| `internal/eco/eco.go:183` | `NewBoardFromFEN(InitialFEN)` | A | Replace with `MustBoardFromFEN` |
| `internal/matching/position.go:126` | `NewBoardFromFEN(InitialFEN)` | A | Replace with `MustBoardFromFEN` |
| `internal/matching/material.go:75` | `NewBoardFromFEN(InitialFEN)` | A | Replace with `MustBoardFromFEN` |
| `internal/matching/variation.go:150` | `NewBoardFromFEN(InitialFEN)` | A | Replace with `MustBoardFromFEN` |
| `internal/output/output.go:135` | `NewBoardFromFEN(fen)` | A | Use err check pattern (not InitialFEN) |
| `internal/processing/analyzer.go:152` | `NewBoardFromFEN(InitialFEN)` | A | Replace with `MustBoardFromFEN` |
| `cmd/pgn-extract/filters.go:347` | `NewBoardFromFEN(InitialFEN)` | A | Replace with `MustBoardFromFEN` |
| `cmd/pgn-extract/processor.go:76` | `Close()` | E | Use `_ =` explicit discard (cleanup before new file) |
| `cmd/pgn-extract/processor.go:241` | `Close()` | E | Use `_ =` explicit discard (LRU eviction cleanup) |
| `cmd/pgn-extract/processor.go:485` | Type assertion | F | Use comma-ok pattern |
| `cmd/pgn-extract/main.go:395` | `file.Close()` | E | Use `_ =` explicit discard (cleanup on exit) |
| `cmd/pgn-extract/main.go:400` | `splitWriter.Close()` | E | Use `_ =` explicit discard (cleanup on exit) |
| `cmd/pgn-extract/main.go:405` | `ecoSplitWriter.Close()` | E | Use `_ =` explicit discard (cleanup on exit) |

**Net reduction: -14 nolints**

Note: `internal/output/output.go:135` uses a user-supplied FEN (not `InitialFEN`),
so it should use the error-check pattern like json.go:110 rather than `MustBoardFromFEN`.

## Remaining Nolints After All Plans (14 non-test, all justified)

These are **Category C** (user file paths, G304) and **Category D** (file permissions, G302)
suppressions plus test-only suppressions, all with clear justification:

| File | Category | Justification |
|------|----------|---------------|
| `internal/eco/eco.go:50` | C (G304) | CLI opens user-specified file |
| `internal/matching/variation.go:31` | C (G304) | CLI opens user-specified file |
| `internal/matching/variation.go:57` | C (G304) | CLI opens user-specified file |
| `internal/matching/filter.go:34` | C (G304) | CLI opens user-specified file |
| `cmd/pgn-extract/processor.go:80` | C (G304) | Filename from user-specified base |
| `cmd/pgn-extract/processor.go:196` | C+D (G304+G302) | User output file, 0644 perms |
| `cmd/pgn-extract/processor.go:208` | C (G304) | User output file |
| `cmd/pgn-extract/main.go:146` | D (G302) | User log file, 0644 perms |
| `cmd/pgn-extract/main.go:165` | D (G302) | User output file, 0644 perms |
| `cmd/pgn-extract/main.go:383` | C (G304) | CLI opens user-specified file |
| `cmd/pgn-extract/main.go:439` | C (G304) | CLI opens user-specified file |
| `cmd/pgn-extract/main.go:500` | C (G304) | CLI opens user-specified file |
| `internal/matching/position.go:162` | gosec G602 | Bounds checked above |
| `internal/matching/position.go:184` | gosec G602 | Bounded by loop condition |
| `cmd/pgn-extract/golden_test.go:49,68,337` | gosec G204/G304 | Test-only exec/read |

## Tasks

<task id="1" files="internal/eco/eco.go,internal/matching/position.go,internal/matching/material.go,internal/matching/variation.go,internal/output/output.go,internal/processing/analyzer.go,cmd/pgn-extract/filters.go" tdd="false">
  <action>
    Replace all `board, _ := engine.NewBoardFromFEN(engine.InitialFEN) //nolint:errcheck`
    patterns with `board := engine.MustBoardFromFEN(engine.InitialFEN)` in:

    - `internal/eco/eco.go` lines 84 and 183
    - `internal/matching/position.go` line 126
    - `internal/matching/material.go` line 75
    - `internal/matching/variation.go` line 150
    - `internal/processing/analyzer.go` line 152
    - `cmd/pgn-extract/filters.go` line 347

    For `internal/output/output.go` line 135, this uses a user-supplied FEN (not
    InitialFEN), so replace with explicit error handling:
    ```go
    board, err := engine.NewBoardFromFEN(fen)
    if err == nil {
        // use board
    }
    ```
    Remove the nolint comment. Ensure the nil check logic is preserved.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go build ./... && go test ./internal/eco/... ./internal/matching/... ./internal/output/... ./internal/processing/... ./cmd/pgn-extract/...</verify>
  <done>
    - All 8 known-valid FEN nolints removed
    - `MustBoardFromFEN` used for `InitialFEN` calls
    - Proper error handling for user-supplied FEN in output.go
    - All tests pass
  </done>
</task>

<task id="2" files="cmd/pgn-extract/processor.go,cmd/pgn-extract/main.go" tdd="false">
  <action>
    **Close() cleanup (Category E):**
    Replace bare `obj.Close() //nolint:errcheck,gosec` with explicit discard
    `_ = obj.Close()` pattern. The `_ =` satisfies errcheck without nolint. Add
    a brief comment explaining this is cleanup-on-exit / cleanup-before-reopen:

    - `cmd/pgn-extract/processor.go:76`:
      `_ = sw.currentFile.Close() // cleanup before creating new file`
    - `cmd/pgn-extract/processor.go:241`:
      `_ = entry.file.Close() // cleanup on LRU eviction`
    - `cmd/pgn-extract/main.go:395`:
      `_ = file.Close() // cleanup on exit`
    - `cmd/pgn-extract/main.go:400`:
      `_ = splitWriter.Close() // cleanup on exit`
    - `cmd/pgn-extract/main.go:405`:
      `_ = ctx.ecoSplitWriter.Close() // cleanup on exit`

    **Type assertion (Category F):**
    `cmd/pgn-extract/processor.go:485`: Replace with comma-ok pattern:
    ```go
    gameInfo, ok := result.GameInfo.(*GameAnalysis)
    if !ok {
        gameInfo = nil // or handle appropriately
    }
    ```
    Remove the nolint comment. Check surrounding code to ensure nil gameInfo
    is handled correctly downstream.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go build ./cmd/pgn-extract/ && go test ./cmd/pgn-extract/...</verify>
  <done>
    - 5 Close() nolints removed (lines 76, 241, 395, 400, 405)
    - 1 type assertion nolint removed (line 485)
    - All cmd tests pass
  </done>
</task>

<task id="3" files="" tdd="false">
  <action>
    Run full verification:
    1. `go test ./...` -- all tests pass
    2. `golangci-lint run ./...` -- no new warnings
    3. Count remaining nolint suppressions in .go source files (excluding test files
       if desired, but count all .go files for accuracy)
    4. Verify non-test nolint count is 15 or fewer (target: 14)
    5. Verify each remaining nolint has a clear justification comment
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test ./... && golangci-lint run ./... && echo "--- Remaining nolints ---" && grep -rn 'nolint' --include='*.go' . | grep -v vendor | grep -v '.git/' | wc -l</verify>
  <done>
    - All tests pass
    - golangci-lint reports no new warnings
    - Non-test nolint count is 15 or fewer (target: 14 remaining from 47, a 70% reduction)
    - Every remaining nolint has a justification comment explaining why it is necessary
  </done>
</task>
