---
phase: memory-management
plan: "1.1"
wave: 1
dependencies: []
must_haves:
  - DuplicateDetector hash table bounded by configurable maxCapacity
  - Default maxCapacity 0 preserves existing unlimited behavior
  - When capacity reached, new entries are not added but existing duplicates still detected
  - ThreadSafeDuplicateDetector passes maxCapacity through to inner DuplicateDetector
  - CLI flag -duplicate-capacity wired to config
  - UniqueCount and DuplicateCount remain accurate
files_touched:
  - internal/hashing/hashing.go
  - internal/hashing/thread_safe.go
  - internal/hashing/hashing_test.go
  - internal/hashing/thread_safe_test.go
  - internal/config/duplicate.go
  - cmd/pgn-extract/flags.go
  - cmd/pgn-extract/main.go
tdd: true
---

# Plan 1.1 -- Bounded DuplicateDetector

## Context

The `DuplicateDetector.hashTable` map grows without bound. For workloads of 1M+ games this
can consume 70-150 MB. The fix is a simple capacity cap: once `len(hashTable)` reaches
`maxCapacity`, stop inserting new entries. Lookups against existing entries continue
normally, so duplicates of already-seen games are still detected. When `maxCapacity` is 0
(the default), behavior is identical to today -- no bound is enforced.

This plan is independent of Plan 1.2 (LRU ECOSplitWriter) and can execute in parallel.

## Tasks

<task id="1" files="internal/hashing/hashing.go,internal/hashing/hashing_test.go" tdd="true">
  <action>
    Add a `maxCapacity int` field to `DuplicateDetector`. Update `NewDuplicateDetector`
    to accept a second parameter `maxCapacity int` (0 = unlimited).

    In `CheckAndAdd`, after the duplicate-check block (lines 60-67), gate the insertion
    on capacity:

    ```go
    // Add to hash table only if capacity allows
    if d.maxCapacity <= 0 || len(d.hashTable) < d.maxCapacity {
        d.hashTable[hash] = append(d.hashTable[hash], sig)
    }
    ```

    Note: `len(d.hashTable)` counts distinct hash buckets, not total signatures. This is
    the correct granularity because each bucket is one map entry (~70 bytes overhead).

    Add an `IsFull() bool` method that returns `d.maxCapacity > 0 && len(d.hashTable) >= d.maxCapacity`.

    Write tests in `hashing_test.go`:
    - `TestDuplicateDetector_MaxCapacity_Zero_Unlimited`: maxCapacity=0, add 1000 unique
      games, all are stored (UniqueCount == 1000).
    - `TestDuplicateDetector_MaxCapacity_Bounded`: maxCapacity=5, add 10 unique games.
      UniqueCount <= 5. The first 5 are stored, subsequent unique games are silently dropped.
    - `TestDuplicateDetector_MaxCapacity_DuplicatesStillDetected`: maxCapacity=2, add 2
      unique games, then add duplicates of game 1 -- duplicate is still detected (returns true).
    - `TestDuplicateDetector_MaxCapacity_IsFull`: verify IsFull() returns false when under
      capacity and true when at capacity.

    Update ALL existing callers of `NewDuplicateDetector` to pass `0` as the second
    argument so existing behavior is preserved. Affected call sites:
    - `internal/hashing/hashing_test.go` (TestDuplicateDetector_* functions)
    - `internal/hashing/benchmark_test.go` (BenchmarkDuplicateDetector_CheckAndAdd)
    - `cmd/pgn-extract/main.go` line 210 (tempDetector in setupDuplicateDetector)
    - `cmd/pgn-extract/processor_test.go` line 147 and 258
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -run "TestDuplicateDetector" ./internal/hashing/ -v</verify>
  <done>DuplicateDetector respects maxCapacity. Zero means unlimited. Tests pass for bounded, unlimited, and duplicate-detection-after-full scenarios.</done>
</task>

<task id="2" files="internal/hashing/thread_safe.go,internal/hashing/thread_safe_test.go" tdd="true">
  <action>
    Update `NewThreadSafeDuplicateDetector` to accept `maxCapacity int` and pass it
    through to `NewDuplicateDetector(exactMatch, maxCapacity)`.

    Add an `IsFull() bool` method to `ThreadSafeDuplicateDetector` that acquires RLock
    and delegates to the inner detector.

    Update ALL existing callers of `NewThreadSafeDuplicateDetector` to pass `0` as the
    second argument:
    - `internal/hashing/thread_safe_test.go` (all test functions)
    - `cmd/pgn-extract/main.go` lines 222 and 228
    - `cmd/pgn-extract/processor_test.go` line 154 and 279

    Add test `TestThreadSafeDuplicateDetector_MaxCapacity_Concurrent`:
    - Create detector with maxCapacity=50.
    - Launch 10 goroutines each adding 100 unique games (1000 total unique positions).
    - After all complete, verify UniqueCount() <= 50 and IsFull() == true.
    - Verify no data race with `go test -race`.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -race -run "TestThreadSafe" ./internal/hashing/ -v</verify>
  <done>ThreadSafeDuplicateDetector passes maxCapacity through. Concurrent test verifies bounded behavior under load with no races.</done>
</task>

<task id="3" files="internal/config/duplicate.go,cmd/pgn-extract/flags.go,cmd/pgn-extract/main.go" tdd="false">
  <action>
    1. In `internal/config/duplicate.go`, add field to DuplicateConfig:
       ```go
       // MaxCapacity limits the number of unique hash buckets tracked.
       // 0 means unlimited (default, preserving existing behavior).
       MaxCapacity int
       ```

    2. In `cmd/pgn-extract/flags.go`, add a new flag in the "Duplicate detection" section:
       ```go
       duplicateCapacity = flag.Int("duplicate-capacity", 0,
           "Max unique games to track for duplicate detection (0 = unlimited)")
       ```

    3. In `cmd/pgn-extract/main.go`, update `setupDuplicateDetector`:
       - After setting `cfg.Duplicate.Suppress` and `cfg.Duplicate.SuppressOriginals`,
         add: `cfg.Duplicate.MaxCapacity = *duplicateCapacity`
       - Pass `cfg.Duplicate.MaxCapacity` as the second argument to both
         `NewDuplicateDetector` and `NewThreadSafeDuplicateDetector` calls.

    4. Run the full test suite: `go test ./... && go vet ./...`
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go build ./cmd/pgn-extract/ && go test ./... && go vet ./...</verify>
  <done>CLI flag -duplicate-capacity is wired through config to both detector constructors. Full test suite passes. Default 0 preserves backward compatibility.</done>
</task>

## Verification

```bash
cd /Users/lgbarn/Personal/Chess/pgn-extract-go

# Unit tests with race detector
go test -race ./internal/hashing/ -v

# Full suite
go test -race ./...

# Verify flag exists
go run ./cmd/pgn-extract/ -h 2>&1 | grep duplicate-capacity

# Verify default behavior unchanged (no capacity flag = unlimited)
go run ./cmd/pgn-extract/ -D testdata/simple.pgn > /dev/null
```
