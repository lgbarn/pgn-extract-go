---
phase: concurrency-safety
plan: "2.1"
wave: 2
dependencies: ["1.1"]
must_haves:
  - Concurrency test proving parallel duplicate detection matches single-threaded results
  - Race detector passes on the new test
  - Documentation comments on single-consumer components (ECOSplitWriter, SplitWriter, jsonGames)
files_touched:
  - cmd/pgn-extract/processor_test.go
  - cmd/pgn-extract/processor.go
tdd: true
---

# Plan 2.1 -- Concurrency verification tests and safety documentation

## Context

After Plan 1.1 swaps in `ThreadSafeDuplicateDetector`, we need to verify that parallel
duplicate detection produces results identical to single-threaded mode. We also need to
document the single-consumer goroutine invariant on components that are not thread-safe
but are used in the parallel code path (`ECOSplitWriter`, `SplitWriter`, `jsonGames`).

## Tasks

<task id="1" files="cmd/pgn-extract/processor_test.go" tdd="true">
  <action>
    Create or extend `cmd/pgn-extract/processor_test.go` with a test named
    `TestParallelDuplicateDetection_MatchesSequential` that:

    1. Creates a set of test games (mix of unique and duplicate positions, at least 20 games).
    2. Runs duplicate detection sequentially using `DuplicateDetector` directly, recording
       which games are duplicates and the final DuplicateCount/UniqueCount.
    3. Runs the same games through `ThreadSafeDuplicateDetector` with 4+ goroutines
       submitting games concurrently.
    4. Asserts both detectors produce identical DuplicateCount and UniqueCount.

    This test MUST be run with `-race` to verify no data races.

    Also add a test `TestParallelDuplicateDetection_WithCheckFile` that:
    1. Pre-loads games into a DuplicateDetector (simulating checkfile).
    2. Calls LoadFromDetector on a ThreadSafeDuplicateDetector.
    3. Runs additional games concurrently through the thread-safe detector.
    4. Asserts correct duplicate/unique counts.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run TestParallelDuplicateDetection ./cmd/pgn-extract/ -v</verify>
  <done>Both tests pass under -race with correct counts matching sequential execution.</done>
</task>

<task id="2" files="cmd/pgn-extract/processor.go" tdd="false">
  <action>
    Add documentation comments noting the single-consumer goroutine safety invariant:

    1. On `ECOSplitWriter` struct (line 101): Add comment:
       "// SAFETY: ECOSplitWriter is NOT thread-safe. In the parallel processing path,
       // it is only accessed from the single result-consumer goroutine in outputGamesParallel.
       // Do not access from worker goroutines."

    2. On `SplitWriter` struct (line 46): Add similar comment:
       "// SAFETY: SplitWriter is NOT thread-safe. In the parallel processing path,
       // it is only accessed from the single result-consumer goroutine in outputGamesParallel."

    3. On the `jsonGames` variable in `outputGamesParallel` (line 379): Add comment:
       "// jsonGames is only appended to from this consumer goroutine -- not shared with workers."

    4. On `outputGamesParallel` function (line 346): Extend the doc comment to note:
       "// The result consumer goroutine (this function's main loop) is the sole writer to
       // ECOSplitWriter, SplitWriter, jsonGames, and cfg.OutputFile. Workers only return
       // ProcessResult values through the channel -- they do not write output directly."
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go build ./cmd/pgn-extract/ && go vet ./cmd/pgn-extract/</verify>
  <done>All safety documentation comments added. Code compiles and passes vet.</done>
</task>

<task id="3" files="" tdd="false">
  <action>
    Run the full test suite with race detector as the final gate:
    - `go test -race ./...`

    Confirm zero race reports and zero test failures across the entire project.
    This is the acceptance gate for Phase 2.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race ./... 2>&1 | tail -20</verify>
  <done>`go test -race ./...` exits 0 with zero data race reports. Phase 2 success criteria met.</done>
</task>
