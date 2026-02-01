---
phase: concurrency-safety
plan: "1.1"
wave: 1
dependencies: []
must_haves:
  - DuplicateChecker interface unifying DuplicateDetector and ThreadSafeDuplicateDetector
  - ProcessingContext.detector uses DuplicateChecker interface
  - setupDuplicateDetector returns ThreadSafeDuplicateDetector for concurrency safety
  - reportStatistics accepts DuplicateChecker instead of concrete type
  - No behavioral change for single-threaded execution paths
files_touched:
  - internal/hashing/hashing.go
  - cmd/pgn-extract/processor.go
  - cmd/pgn-extract/main.go
tdd: false
---

# Plan 1.1 -- Interface extraction and ThreadSafeDuplicateDetector swap

## Context

`ProcessingContext.detector` is typed as `*hashing.DuplicateDetector` (non-thread-safe).
In `outputGamesParallel`, the consumer goroutine calls `detector.CheckAndAdd` which is
currently safe only because the consumer is single-threaded. However, the code is fragile
and the type should be thread-safe to match the concurrent context it lives in.

Both `DuplicateDetector` and `ThreadSafeDuplicateDetector` expose `CheckAndAdd`,
`DuplicateCount`, and `UniqueCount` but share no interface. We need an interface so
`ProcessingContext` and `reportStatistics` can accept either implementation.

## Tasks

<task id="1" files="internal/hashing/hashing.go" tdd="false">
  <action>
    Define a `DuplicateChecker` interface in `internal/hashing/hashing.go` with three methods:
    - `CheckAndAdd(game *chess.Game, board *chess.Board) bool`
    - `DuplicateCount() int`
    - `UniqueCount() int`

    Both `DuplicateDetector` and `ThreadSafeDuplicateDetector` already satisfy this interface
    implicitly. Place the interface definition near the top of hashing.go, after the package
    doc comment and imports.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go build ./internal/hashing/</verify>
  <done>DuplicateChecker interface exists. Both DuplicateDetector and ThreadSafeDuplicateDetector satisfy it (verified by go build).</done>
</task>

<task id="2" files="cmd/pgn-extract/processor.go,cmd/pgn-extract/main.go" tdd="false">
  <action>
    Update ProcessingContext and all consuming code to use the DuplicateChecker interface:

    1. In `cmd/pgn-extract/processor.go` line 35, change:
       `detector *hashing.DuplicateDetector` -> `detector hashing.DuplicateChecker`

    2. In `cmd/pgn-extract/main.go`:
       a. `setupDuplicateDetector` (line 192): change return type from
          `*hashing.DuplicateDetector` to `hashing.DuplicateChecker`.
          Inside the function body, replace `hashing.NewDuplicateDetector(false)` with
          `hashing.NewThreadSafeDuplicateDetector(false)`.
          For the checkfile loading path (lines 202-218), first build a temporary
          `hashing.NewDuplicateDetector(false)`, load games into it, then call
          `tsDetector.LoadFromDetector(tempDetector)` on the thread-safe instance.
       b. `reportStatistics` (line 405): change parameter from
          `detector *hashing.DuplicateDetector` to `detector hashing.DuplicateChecker`.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go build ./cmd/pgn-extract/</verify>
  <done>ProcessingContext.detector is typed as hashing.DuplicateChecker. setupDuplicateDetector returns a ThreadSafeDuplicateDetector. reportStatistics accepts the interface. Full binary compiles.</done>
</task>

<task id="3" files="cmd/pgn-extract/processor.go,cmd/pgn-extract/main.go" tdd="false">
  <action>
    Run the existing test suite including the race detector to confirm no regressions:
    - `go test -race ./...`
    - `go vet ./...`

    Fix any compilation or test failures discovered. Confirm that the existing
    `TestThreadSafeDuplicateDetector_*` tests in `internal/hashing/` still pass.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race ./... && go vet ./...</verify>
  <done>`go test -race ./...` passes with zero failures and zero race reports. `go vet ./...` clean.</done>
</task>
