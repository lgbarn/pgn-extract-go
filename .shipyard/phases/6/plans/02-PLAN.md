---
phase: nolint-cleanup
plan: 02
wave: 1
dependencies: []
must_haves:
  - AddCriterion callers propagate or explicitly handle errors
  - Type assertions use comma-ok pattern without nolint
  - JSON Encode errors are propagated
  - All existing tests pass
  - No API changes to exported function signatures
files_touched:
  - internal/matching/filter.go
  - internal/matching/tags.go
  - internal/output/json.go
  - internal/worker/pool.go
tdd: false
---

# Plan 02: Fix Error Handling in Matching, Output, and Worker Packages

## Goal

Remove 13 nolint suppressions related to unchecked errors from `AddCriterion` calls,
JSON `Encode` calls, and type assertions. This plan touches only `internal/matching/`,
`internal/output/`, and `internal/worker/` -- no overlap with Plan 01 or Plan 03 files.

## Nolint Reductions

| File | Line(s) | Category | Action |
|------|---------|----------|--------|
| `internal/matching/tags.go:101` | `AddSimpleCriterion` | G | Remove nolint; `OpEqual` never triggers regex compile error |
| `internal/matching/tags.go:111` | `AddPlayerCriterion` | G | Remove nolint; `OpContains`/`OpSoundex` never trigger error |
| `internal/matching/filter.go:58` | `AddFEN` in LoadTagFile | G | Log/skip error, remove nolint |
| `internal/matching/filter.go:61` | `ParseCriterion` | G | Log/skip error, remove nolint |
| `internal/matching/filter.go:70` | `AddTagCriterion` | G | Return error from `AddCriterion`, remove nolint |
| `internal/matching/filter.go:80` | `AddWhiteFilter` | G | Return error, remove nolint |
| `internal/matching/filter.go:85` | `AddBlackFilter` | G | Return error, remove nolint |
| `internal/matching/filter.go:90` | `AddECOFilter` | G | Return error, remove nolint |
| `internal/matching/filter.go:95` | `AddResultFilter` | G | Return error, remove nolint |
| `internal/matching/filter.go:100` | `AddDateFilter` | G | Return error, remove nolint |
| `internal/output/json.go:49` | `enc.Encode` | G | Return error, remove nolint |
| `internal/output/json.go:61` | `enc.Encode` | G | Return error, remove nolint |
| `internal/output/json.go:110` | `NewBoardFromFEN(fen)` | A variant | Use explicit error check, remove nolint |
| `internal/worker/pool.go:45` | Type assertion | F | Use comma-ok properly, remove nolint |

**Net reduction: -14 nolints**

## Strategy

### AddCriterion callers (tags.go)
`AddCriterion` only returns an error when `op == OpRegex` and `regexp.Compile` fails.
For `AddSimpleCriterion` (uses `OpEqual`) and `AddPlayerCriterion` (uses `OpContains`
or `OpSoundex`), the error is structurally impossible. The cleanest fix is to remove
the nolint and instead explicitly discard the error with a brief comment explaining
why it is safe -- OR better, just check the error and return it, making these functions
return `error`. Since `AddSimpleCriterion` and `AddPlayerCriterion` are internal helpers,
changing their signatures is acceptable. However, to minimize churn, just use `_ =`
assignment to explicitly acknowledge the discarded error, which satisfies errcheck.

### filter.go AddCriterion callers
For `AddTagCriterion`, `AddWhiteFilter`, `AddBlackFilter`, `AddECOFilter`,
`AddResultFilter`, `AddDateFilter`: these are called from `cmd/pgn-extract/main.go`.
The safe approach is to make them return `error` so callers can handle it. However,
since none of these pass `OpRegex`, the error is structurally impossible. Use `_ =`
to explicitly acknowledge, or propagate the error. Prefer propagating error for
`AddTagCriterion` (it accepts arbitrary `op` which could be `OpRegex`) and use
explicit discard for the rest (hardcoded non-regex operators).

### JSON Encode
Change `OutputGameJSON` and `OutputGamesJSON` to return `error` from `enc.Encode`.

### Type assertion (worker/pool.go)
The comma-ok pattern `gi, _ := r.GameInfo.(GameInfo)` is already correct Go -- the
`_` captures the bool. The nolint is for `errcheck` which does not apply to type
assertions. Remove the nolint comment; it is likely a false positive or stale.

## Tasks

<task id="1" files="internal/matching/tags.go,internal/matching/filter.go" tdd="false">
  <action>
    **tags.go changes:**
    - Line 101 (`AddSimpleCriterion`): Replace nolint with explicit discard:
      `_ = tm.AddCriterion(tagName, value, OpEqual) // OpEqual never fails`
      Remove the `//nolint:errcheck,gosec` comment.
    - Line 111 (`AddPlayerCriterion`): Same pattern:
      `_ = tm.AddCriterion("_Player", playerName, op) // OpContains/OpSoundex never fail`

    **filter.go changes:**
    - Line 58 (`AddFEN` in `LoadTagFile`): Capture and silently continue on error:
      ```go
      if err := gf.PositionMatcher.AddFEN(rest, ""); err != nil {
          continue // skip invalid FEN lines
      }
      ```
    - Line 61 (`ParseCriterion`): Capture and continue:
      ```go
      if err := gf.TagMatcher.ParseCriterion(line); err != nil {
          continue // skip unparseable lines
      }
      ```
    - Line 70 (`AddTagCriterion`): Change signature to return error:
      `func (gf *GameFilter) AddTagCriterion(tagName, value string, op TagOperator) error`
      Return the error from `AddCriterion`. Update all callers in cmd/ as needed
      (but since callers are in cmd/ files not touched by this plan, prefer keeping
      the void signature and using explicit discard instead).
      **Decision**: Use explicit discard `_ = gf.TagMatcher.AddCriterion(...)` since
      callers in cmd/ would need changes. The operator is caller-controlled and typically
      not OpRegex, but for safety add a comment.
    - Lines 80, 85, 90, 95, 100: Use explicit discard pattern
      `_ = gf.TagMatcher.AddCriterion(...)` with a brief comment explaining why
      the error cannot occur (hardcoded non-regex operator).
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go build ./internal/matching/ && go test ./internal/matching/...</verify>
  <done>
    - Zero nolint directives in `internal/matching/tags.go`
    - Zero nolint directives in `internal/matching/filter.go` lines 58-100
    - G304 nolint on line 34 of filter.go is preserved (Category C, justified)
    - All matching tests pass
  </done>
</task>

<task id="2" files="internal/output/json.go,internal/worker/pool.go" tdd="false">
  <action>
    **json.go changes:**
    - Line 49 (`OutputGameJSON`): Change function signature to return `error`.
      Return the error from `enc.Encode(jsonGame)`. Remove nolint.
      ```go
      func OutputGameJSON(game *chess.Game, cfg *config.Config) error {
          jsonGame := GameToJSON(game, cfg)
          enc := json.NewEncoder(cfg.OutputFile)
          enc.SetIndent("", "  ")
          return enc.Encode(jsonGame)
      }
      ```
    - Line 61 (`OutputGamesJSON`): Change function signature to return `error`.
      Return the error from `enc.Encode(...)`. Remove nolint.
    - Line 110 (`getInitialBoard`): This is Category A (known-valid FEN from game tag).
      The nil check handles the error properly. Remove the nolint -- the `_` discard
      with the nil check is idiomatic Go. Actually, check if errcheck flags this
      pattern. If it does, use explicit `_ =` or keep. The current code
      `if board, _ := engine.NewBoardFromFEN(fen); board != nil` should be changed to:
      ```go
      board, err := engine.NewBoardFromFEN(fen)
      if err == nil && board != nil {
          return board, fen
      }
      ```

    **worker/pool.go changes:**
    - Line 45: The `gi, _ := r.GameInfo.(GameInfo)` pattern is valid Go. The `_`
      is the boolean ok value. Remove the `//nolint:errcheck` comment. If errcheck
      flags type assertions, the `.golangci.yml` should already handle this. If not,
      switch to the explicit ok pattern:
      ```go
      gi, ok := r.GameInfo.(GameInfo)
      if !ok {
          return nil
      }
      return gi
      ```

    Update callers of `OutputGameJSON` and `OutputGamesJSON` to handle the returned
    error. Search for callers -- they are likely in `internal/output/` or `cmd/`.
    If callers are in `cmd/` (outside this plan's scope), consider whether the
    signature change is safe. Check callers first and adjust approach accordingly.
    If callers exist only in cmd/ files, keep void return and use explicit discard
    instead to avoid cross-plan file conflicts.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go build ./... && go test ./internal/output/... ./internal/worker/...</verify>
  <done>
    - Zero nolint directives in `internal/output/json.go`
    - Zero nolint directives in `internal/worker/pool.go`
    - `go build ./...` succeeds (no broken callers)
    - All output and worker tests pass
  </done>
</task>

<task id="3" files="" tdd="false">
  <action>
    Run full lint and test suite to verify no regressions.
    Count remaining nolint suppressions in the touched files.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && golangci-lint run ./internal/matching/ ./internal/output/ ./internal/worker/ && go test ./... && echo "--- nolint count ---" && grep -rn 'nolint' internal/matching/filter.go internal/matching/tags.go internal/output/json.go internal/worker/pool.go || echo "No nolints remain in touched files"</verify>
  <done>
    - golangci-lint passes with no new warnings
    - All tests pass
    - Only the justified G304 nolint on filter.go:34 remains in touched files
  </done>
</task>
