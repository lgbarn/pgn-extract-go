---
phase: memory-management
plan: "2.1"
wave: 2
dependencies: ["1.1", "1.2"]
must_haves:
  - Benchmark proving DuplicateDetector memory is bounded under 100K+ game load
  - Benchmark proving ECOSplitWriter open handles stay within configured limit
  - Integration test confirming duplicate detection behavior unchanged at low volume
  - All existing tests continue to pass
files_touched:
  - internal/hashing/benchmark_test.go
  - cmd/pgn-extract/processor_test.go
tdd: false
---

# Plan 2.1 -- Bounded memory benchmarks and integration verification

## Context

Plans 1.1 and 1.2 add capacity bounds to `DuplicateDetector` and `ECOSplitWriter`.
This plan adds benchmark tests that prove memory stays bounded under load, and an
integration-level test confirming that duplicate detection behavior is unchanged when
the capacity is not exceeded (i.e., the common case).

## Tasks

<task id="1" files="internal/hashing/benchmark_test.go" tdd="false">
  <action>
    Add benchmark `BenchmarkDuplicateDetector_BoundedMemory` to `internal/hashing/benchmark_test.go`:

    ```go
    func BenchmarkDuplicateDetector_BoundedMemory(b *testing.B) {
        // Test that bounded detector memory does not grow beyond capacity
        const capacity = 1000
        const totalGames = 100_000

        b.Run("Bounded", func(b *testing.B) {
            for n := 0; n < b.N; n++ {
                dd := NewDuplicateDetector(false, capacity)
                for i := 0; i < totalGames; i++ {
                    board := chess.NewBoard()
                    board.SetupInitialPosition()
                    // Vary position to create unique games
                    board.Set(chess.Col('a'+i%8), chess.Rank('1'+i/8%8), chess.Empty)
                    game := &chess.Game{Tags: make(map[string]string)}
                    dd.CheckAndAdd(game, board)
                }
                if dd.UniqueCount() > capacity {
                    b.Fatalf("UniqueCount %d exceeds capacity %d", dd.UniqueCount(), capacity)
                }
                if !dd.IsFull() {
                    b.Fatal("Expected detector to be full")
                }
            }
        })

        b.Run("Unlimited", func(b *testing.B) {
            for n := 0; n < b.N; n++ {
                dd := NewDuplicateDetector(false, 0)
                for i := 0; i < 1000; i++ {
                    board := chess.NewBoard()
                    board.SetupInitialPosition()
                    board.Set(chess.Col('a'+i%8), chess.Rank('1'+i/8%8), chess.Empty)
                    game := &chess.Game{Tags: make(map[string]string)}
                    dd.CheckAndAdd(game, board)
                }
            }
        })
    }
    ```

    Also add `BenchmarkDuplicateDetector_BoundedVsUnlimited` that runs both variants
    and reports `b.ReportMetric` for unique count to make the bound visible in output:

    ```go
    func BenchmarkDuplicateDetector_BoundedVsUnlimited(b *testing.B) {
        const games = 10_000
        for _, cap := range []int{0, 100, 1000, 5000} {
            name := fmt.Sprintf("cap=%d", cap)
            b.Run(name, func(b *testing.B) {
                for n := 0; n < b.N; n++ {
                    dd := NewDuplicateDetector(false, cap)
                    for i := 0; i < games; i++ {
                        board := chess.NewBoard()
                        board.SetupInitialPosition()
                        board.Set(chess.Col('a'+i%8), chess.Rank('1'+i/8%8), chess.Empty)
                        game := &chess.Game{Tags: make(map[string]string)}
                        dd.CheckAndAdd(game, board)
                    }
                    b.ReportMetric(float64(dd.UniqueCount()), "unique_games")
                }
            })
        }
    }
    ```

    Import `fmt` if not already imported.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -bench "BenchmarkDuplicateDetector_Bounded" -benchtime 1x ./internal/hashing/ -v</verify>
  <done>Benchmark demonstrates bounded detector stays within capacity. Unlimited detector grows as expected. Both variants complete without error.</done>
</task>

<task id="2" files="cmd/pgn-extract/processor_test.go" tdd="false">
  <action>
    Add integration test `TestECOSplitWriter_LRU_HandleCountBounded` to
    `cmd/pgn-extract/processor_test.go`:

    - Create ECOSplitWriter with maxHandles=5 and level=3.
    - Generate games covering 20 distinct ECO codes (A00-A19 or similar).
    - Write all games through the writer.
    - Assert `OpenHandleCount() <= 5` at every step after the 5th write.
    - Assert all 20 files exist on disk after `Close()`.
    - Read back file contents and verify each file contains the expected games.

    Use `t.TempDir()` for output directory.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -run "TestECOSplitWriter_LRU_HandleCountBounded" ./cmd/pgn-extract/ -v</verify>
  <done>Integration test proves handle count stays bounded. All output files created and contain correct content.</done>
</task>

<task id="3" files="internal/hashing/hashing_test.go" tdd="false">
  <action>
    Add integration test `TestDuplicateDetector_BehaviorUnchanged_BelowCapacity` to
    `internal/hashing/hashing_test.go`:

    - Create detector with maxCapacity=1000.
    - Add 100 unique games, verify UniqueCount() == 100 and IsFull() == false.
    - Add duplicates of each game, verify all 100 are detected as duplicates.
    - Verify DuplicateCount() == 100.
    - This confirms that when capacity is not exceeded, behavior is identical
      to the unlimited case.

    Also add `TestDuplicateDetector_BehaviorUnchanged_Unlimited`:
    - Create detector with maxCapacity=0.
    - Add 500 unique games, verify UniqueCount() == 500.
    - Add duplicates of each, verify DuplicateCount() == 500.
    - This confirms zero means unlimited (regression test for default behavior).

    Run the full test suite with race detector:
    ```bash
    go test -race ./...
    go vet ./...
    ```
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race ./... && go vet ./...</verify>
  <done>Behavior-preservation tests pass. Full suite green with race detector. No regressions from Phase 3 changes.</done>
</task>

## Verification

```bash
cd /Users/lgbarn/Personal/Chess/pgn-extract-go

# All benchmarks
go test -bench "BenchmarkDuplicateDetector_Bounded" -benchtime 3x ./internal/hashing/ -v

# All Phase 3 tests
go test -race -run "TestDuplicateDetector_MaxCapacity|TestDuplicateDetector_Behavior|TestThreadSafe.*MaxCapacity|TestECOSplitWriter_LRU" ./internal/hashing/ ./cmd/pgn-extract/ -v

# Full regression suite
go test -race ./...
go vet ./...
```
