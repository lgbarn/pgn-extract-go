---
phase: nolint-cleanup
plan: 01
wave: 1
dependencies: []
must_haves:
  - MustBoardFromFEN helper that panics on invalid FEN (for known-valid constants)
  - Sscanf nolint removal in fen.go and lexer.go
  - Test helper nolint removal in testutil/game.go
  - All existing tests pass
files_touched:
  - internal/engine/fen.go
  - internal/parser/lexer.go
  - internal/testutil/game.go
tdd: false
---

# Plan 01: MustBoardFromFEN Helper, Sscanf Fixes, and Test Helper

## Goal

Create the `MustBoardFromFEN` helper function and fix 4 nolint suppressions in
`internal/engine/fen.go`, `internal/parser/lexer.go`, and `internal/testutil/game.go`.
This plan establishes the helper that Plans 02 and 03 depend on for replacing
known-valid FEN nolint suppressions.

## Nolint Reductions

| File | Line | Category | Action |
|------|------|----------|--------|
| `internal/engine/fen.go:300` | `NewInitialBoard` | A (known-valid FEN) | Use `MustBoardFromFEN` |
| `internal/engine/fen.go:206` | `Sscanf` halfmove | B (redundant) | Remove nolint; Sscanf errcheck excluded in .golangci.yml |
| `internal/engine/fen.go:209` | `Sscanf` movenum | B (redundant) | Remove nolint; Sscanf errcheck excluded in .golangci.yml |
| `internal/parser/lexer.go:545` | `Sscanf` movenum | B (redundant) | Remove nolint; Sscanf errcheck excluded in .golangci.yml |
| `internal/testutil/game.go:31` | `ParseAllGames` | H (test helper) | Add explicit error check with `t.Fatal` |

**Net reduction: -5 nolints** (adds 1 new function, no new nolints)

Note: The `MustBoardFromFEN` function itself will NOT have a nolint. It calls
`NewBoardFromFEN` and panics on error, so the error return is properly handled.

## Tasks

<task id="1" files="internal/engine/fen.go" tdd="false">
  <action>
    Add a `MustBoardFromFEN(fen string) *chess.Board` function to `internal/engine/fen.go`
    that calls `NewBoardFromFEN` and panics if it returns an error. Then update
    `NewInitialBoard()` on line 300 to call `MustBoardFromFEN(InitialFEN)` instead of
    `NewBoardFromFEN(InitialFEN)` with the nolint suppression.

    Also remove the `//nolint:errcheck,gosec` comments from the two `fmt.Sscanf` calls
    on lines 206 and 209. These are already excluded in `.golangci.yml` via the
    `exclude-rules` for `fmt.Sscanf` errcheck, so the nolint directives are redundant.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go build ./internal/engine/ && go test ./internal/engine/...</verify>
  <done>
    - `MustBoardFromFEN` exists and is exported
    - `NewInitialBoard` uses `MustBoardFromFEN` without nolint
    - Lines 206 and 209 have no nolint comments
    - `go test ./internal/engine/...` passes
  </done>
</task>

<task id="2" files="internal/parser/lexer.go,internal/testutil/game.go" tdd="false">
  <action>
    In `internal/parser/lexer.go` line 545, remove the `//nolint:errcheck,gosec`
    comment from the `fmt.Sscanf` call. This is redundant with `.golangci.yml` exclusion.

    In `internal/testutil/game.go` line 31, change the test helper function signature
    to accept `testing.TB` (or keep current signature if it already does) and replace
    the `//nolint:errcheck` suppressed call with an explicit error check:
    ```go
    games, err := p.ParseAllGames()
    if err != nil {
        // Use panic since test helpers may not always have *testing.T
        panic(fmt.Sprintf("ParseAllGames failed: %v", err))
    }
    ```
    If the function already has a `t testing.TB` parameter, use `t.Fatalf` instead.
    Check the actual function signature before making changes.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go build ./internal/parser/ && go test ./internal/parser/... && go test ./internal/testutil/... && go test ./...</verify>
  <done>
    - `internal/parser/lexer.go:545` has no nolint comment
    - `internal/testutil/game.go:31` has no nolint comment and has explicit error handling
    - All tests pass (`go test ./...`)
  </done>
</task>

<task id="3" files="" tdd="false">
  <action>
    Run `golangci-lint run ./internal/engine/ ./internal/parser/ ./internal/testutil/`
    to verify no new lint warnings are introduced.
    Confirm the 4 nolint directives have been removed by grepping the touched files.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && golangci-lint run ./internal/engine/ ./internal/parser/ ./internal/testutil/ && ! grep -n 'nolint' internal/engine/fen.go internal/parser/lexer.go internal/testutil/game.go</verify>
  <done>
    - `golangci-lint run` passes with no new warnings on touched packages
    - Zero nolint directives remain in the three touched files
  </done>
</task>
