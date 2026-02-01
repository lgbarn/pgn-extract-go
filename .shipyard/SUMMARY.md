# Phase 6 / Plan 03 Summary: Replace Known-Valid FEN Calls and Fix Close/Type-Assertion Cleanup

## What Was Done

### Task 1: Replace NewBoardFromFEN(InitialFEN) with MustBoardFromFEN(InitialFEN)

Replaced all call sites where `NewBoardFromFEN(InitialFEN)` was used with the error-discarding `//nolint:errcheck` pattern:

| File | Change |
|------|--------|
| `internal/eco/eco.go` (line 84) | `board, _ := ...` -> `board := engine.MustBoardFromFEN(engine.InitialFEN)` |
| `internal/eco/eco.go` (line 183) | Same replacement in `boardForGame` |
| `internal/matching/position.go` (line 126) | Same replacement in `getStartingBoard` |
| `internal/matching/material.go` (line 75) | Same replacement in `MatchGame` |
| `internal/matching/variation.go` (line 150) | Same replacement in `matchPositionSequence` |
| `internal/processing/analyzer.go` (line 152) | `board, _ = ...` -> `board = engine.MustBoardFromFEN(engine.InitialFEN)` (assignment, not declaration) |
| `cmd/pgn-extract/filters.go` (line 347) | Same replacement in `checkPieceCount` |
| `internal/output/output.go` (line 134-138) | User-supplied FEN: replaced `board, _ = ... //nolint` with proper `if b, err := ...; err == nil { board = b }` |

### Task 2: Fix Close() and Type Assertion Nolints

| File | Line | Change |
|------|------|--------|
| `cmd/pgn-extract/processor.go` | 76 | `sw.currentFile.Close() //nolint:...` -> `_ = sw.currentFile.Close()` |
| `cmd/pgn-extract/processor.go` | 241 | `entry.file.Close() //nolint:...` -> `_ = entry.file.Close()` |
| `cmd/pgn-extract/processor.go` | 485 | Removed `//nolint:errcheck` from type assertion (blank identifier is fine) |
| `cmd/pgn-extract/main.go` | 395 | `file.Close() //nolint:...` -> `_ = file.Close()` |
| `cmd/pgn-extract/main.go` | 400 | `splitWriter.Close() //nolint:...` -> `_ = splitWriter.Close()` |
| `cmd/pgn-extract/main.go` | 405 | `ctx.ecoSplitWriter.Close() //nolint:...` -> `_ = ctx.ecoSplitWriter.Close()` |

### Task 3: Final Verification

- All tests pass with `-race` flag
- 14 remaining nolint directives in non-test files, all justified:
  - 8x G304 (os.Open with user-specified filenames - CLI tool requirement)
  - 3x G302 (0644 file permissions - appropriate for user output files)
  - 2x G602 (bounds checking with verified indices)
  - 1x G304 on os.Create (derived from user-specified base name)

## Deviations

None. All changes followed the plan exactly.

## Final State

- Commit: `29f29b7` on `main`
- All 14 packages build and pass tests with race detector
- Nolint count reduced from ~28 to 14 in production code (all remaining are justified security suppressions)
