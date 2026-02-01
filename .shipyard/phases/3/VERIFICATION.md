# Phase 3 Plan Review Verification
**Phase:** Memory Management
**Date:** 2026-01-31
**Type:** plan-review

## Executive Summary

Phase 3 plans comprehensively address all four milestone requirements with clear, testable acceptance criteria. The three-plan structure (1.1, 1.2, 2.1) enables Wave 1 parallelization while maintaining correct dependencies in Wave 2. No structural or file conflicts identified.

---

## Results

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | All 4 phase requirements covered by plans | PASS | Each requirement has explicit coverage mapping to specific plan tasks |
| 2 | Requirement 1: DuplicateDetector bounded hash table | PASS | Plan 1.1 Tasks 1-2 + Plan 2.1 Task 1 (benchmark with 100K games) |
| 3 | Requirement 2: ECOSplitWriter file handle limits | PASS | Plan 1.2 Tasks 1-3 + Plan 2.1 Task 2 (integration with maxHandles=5) |
| 4 | Requirement 3: Behavior unchanged below capacity | PASS | Plan 2.1 Task 3 + Plan 1.1 unit tests verify duplicate detection at capacity |
| 5 | Requirement 4: Bounded memory benchmark | PASS | Plan 2.1 Task 1 includes BenchmarkDuplicateDetector_BoundedMemory (100K games, cap=1000) |
| 6 | No task exceeds 3 tasks per plan | PASS | Plan 1.1: 3 tasks, Plan 1.2: 3 tasks, Plan 2.1: 3 tasks |
| 7 | Wave 1 can execute in parallel | PASS | Plans 1.1 and 1.2 have no dependencies; both independent |
| 8 | Wave 2 correctly depends on Wave 1 | PASS | Plan 2.1 dependencies: ["1.1", "1.2"] correctly specified |
| 9 | No file conflicts between parallel plans | PASS | Shared files (flags.go, main.go) modified in different sections |
| 10 | cmd/pgn-extract/flags.go: no duplicate modifications | PASS | Plan 1.1 adds `-duplicate-capacity` flag; Plan 1.2 adds `-eco-max-handles` flag (different sections) |
| 11 | cmd/pgn-extract/main.go: no duplicate modifications | PASS | Plan 1.1 modifies setupDuplicateDetector(); Plan 1.2 modifies NewECOSplitWriter() call (different functions) |
| 12 | All acceptance criteria are testable | PASS | 17 distinct test/verification points all have concrete, measurable acceptance criteria |
| 13 | Plan 1.1 must-haves addressed | PASS | 6/6 must-haves map to specific tasks (maxCapacity field, ThreadSafe propagation, CLI flag, IsFull() method) |
| 14 | Plan 1.2 must-haves addressed | PASS | 6/6 must-haves map to specific tasks (LRU eviction, append-mode reopening, default 128, OpenHandleCount()) |
| 15 | Plan 2.1 must-haves addressed | PASS | 4/4 must-haves map to specific tasks (bounded benchmarks, integration test, regression suite) |
| 16 | Unit test count adequate | PASS | Plan 1.1: 5 unit tests; Plan 1.2: 3 integration tests; Plan 2.1: 3 integration tests + 2 benchmarks |
| 17 | Thread safety verified in plans | PASS | Plan 1.1 Task 2 includes concurrent capacity test; Plan 2.1 Task 3 runs full race detector suite |

---

## Detailed Plan Coverage

### Plan 1.1: Bounded DuplicateDetector

**Tasks:** 3 (TDD-first design)

1. **Task 1:** Add `maxCapacity` field to DuplicateDetector
   - Modification: `CheckAndAdd()` gates insertion when capacity reached
   - New method: `IsFull() bool`
   - Unit tests: 4 tests covering zero/bounded/detection/full scenarios
   - Files: `internal/hashing/hashing.go`, `hashing_test.go`

2. **Task 2:** Propagate through ThreadSafeDuplicateDetector
   - Propagation: maxCapacity passed through to inner DuplicateDetector
   - New method: `IsFull()` on thread-safe wrapper
   - Concurrent test: 10 goroutines, 100 games each, verifies UniqueCount <= 50
   - Files: `internal/hashing/thread_safe.go`, `thread_safe_test.go`

3. **Task 3:** Wire CLI flag and config
   - Config field: `DuplicateConfig.MaxCapacity`
   - Flag: `-duplicate-capacity` (default 0 = unlimited)
   - Integration: setupDuplicateDetector() reads flag and passes to constructors
   - Files: `internal/config/duplicate.go`, `cmd/pgn-extract/flags.go`, `main.go`

**Requirement Coverage:** ✓ Fully addresses Requirement 1 (bounded hash table)

---

### Plan 1.2: LRU ECOSplitWriter

**Tasks:** 3 (TDD-first design)

1. **Task 1:** Refactor ECOSplitWriter with LRU cache
   - New data structure: `lruFileEntry` with file pointer and LRU element
   - LRU implementation: `container/list` (stdlib only, no external deps)
   - Methods: `getOrCreateFile()` with move-to-front logic, `OpenHandleCount()`
   - Eviction: Closes least-recent file when limit reached; reopens in append on next access
   - Files: `cmd/pgn-extract/processor.go`

2. **Task 2:** LRU behavior tests
   - Test 1: EvictsOldestHandle - 4 ECO codes, maxHandles=3, verifies count and file existence
   - Test 2: ReopensEvictedFile - verifies append mode preserves prior content
   - Test 3: UnlimitedWhenHigh - 10 codes with maxHandles=1000, no eviction
   - Files: `cmd/pgn-extract/processor_test.go`

3. **Task 3:** Wire CLI flag and config
   - Config field: `OutputConfig.ECOMaxHandles` (default 128)
   - Flag: `-eco-max-handles 128`
   - Integration: main.go NewECOSplitWriter call uses cfg.Output.ECOMaxHandles
   - Files: `internal/config/output.go`, `cmd/pgn-extract/flags.go`, `main.go`

**Requirement Coverage:** ✓ Fully addresses Requirement 2 (bounded file handles)

---

### Plan 2.1: Benchmarks and Integration Verification

**Tasks:** 3 (Integration-level, post-implementation)

1. **Task 1:** DuplicateDetector bounded memory benchmarks
   - Benchmark 1: 100K games with cap=1000, verifies UniqueCount <= 1000
   - Benchmark 2: Multiple capacities (0, 100, 1000, 5000), reports unique_games metric
   - Verifies: Memory stays bounded under heavy load
   - Files: `internal/hashing/benchmark_test.go`

2. **Task 2:** ECOSplitWriter handle count bounded integration
   - Test: 20 distinct ECO codes with maxHandles=5
   - Verifies: OpenHandleCount() never exceeds 5; all 20 files created and contain correct content
   - Uses: t.TempDir() for clean isolation
   - Files: `cmd/pgn-extract/processor_test.go`

3. **Task 3:** Behavior preservation regression suite
   - Test 1: 100 unique games, cap=1000, verifies UniqueCount == 100, all duplicates detected
   - Test 2: 500 unique games, cap=0 (unlimited), verifies behavior identical to unlimited mode
   - Full suite: `go test -race ./...` verifies no regressions
   - Files: `internal/hashing/hashing_test.go`

**Requirement Coverage:** ✓ Fully addresses Requirements 3 and 4 (unchanged behavior and benchmarks)

---

## File Conflict Analysis

### Shared Files in Wave 1 (Plans 1.1 and 1.2)

**File: cmd/pgn-extract/flags.go**
- Plan 1.1 adds: `-duplicate-capacity` flag in duplicate detection section
- Plan 1.2 adds: `-eco-max-handles` flag in ECO section
- **Conflict Risk:** None. Different logical sections, different flag names. Merge order irrelevant.

**File: cmd/pgn-extract/main.go**
- Plan 1.1 modifies: `setupDuplicateDetector()` function
- Plan 1.2 modifies: `NewECOSplitWriter()` call location
- **Conflict Risk:** None. Different functions. Merge order irrelevant.

**Recommendation:** Both plans can be executed simultaneously; if serialized, either order works.

---

## Test Coverage Summary

| Component | Unit Tests | Integration Tests | Benchmarks | Total |
|-----------|------------|-------------------|------------|-------|
| DuplicateDetector | 5 (Plan 1.1) | 2 (Plan 2.1) | 2 (Plan 2.1) | 9 |
| ThreadSafeDuplicateDetector | 1 (Plan 1.1) | - | - | 1 |
| ECOSplitWriter | - | 3 (Plan 1.2) + 1 (Plan 2.1) | - | 4 |
| **Total** | **6** | **6** | **2** | **14** |

All tests include `go test -race` verification where applicable.

---

## Verification Commands

The following commands are provided in each plan and can be executed sequentially to verify all requirements:

```bash
# Plan 1.1 - DuplicateDetector
go test -race -run "TestDuplicateDetector" ./internal/hashing/ -v
go test -race -run "TestThreadSafe" ./internal/hashing/ -v
go build ./cmd/pgn-extract/
go run ./cmd/pgn-extract/ -h 2>&1 | grep duplicate-capacity

# Plan 1.2 - ECOSplitWriter
go test -run "TestECOSplitWriter_LRU" ./cmd/pgn-extract/ -v
go build ./cmd/pgn-extract/
go run ./cmd/pgn-extract/ -h 2>&1 | grep eco-max-handles

# Plan 2.1 - Benchmarks and Integration
go test -bench "BenchmarkDuplicateDetector_Bounded" -benchtime 3x ./internal/hashing/ -v
go test -race -run "TestECOSplitWriter_LRU_HandleCountBounded|TestDuplicateDetector_Behavior" -v

# Full regression suite
go test -race ./...
go vet ./...
```

---

## Gaps

**None identified.** All phase requirements are explicitly covered by plan tasks, all acceptance criteria are concrete and testable, and all must-haves are addressed.

---

## Recommendations

**Pre-Execution Checklist:**
1. Clone Phase 3 plan descriptions for distribution to implementers
2. Confirm Wave 1 (Plans 1.1 and 1.2) can execute in parallel in your CI/CD pipeline
3. Schedule Wave 2 (Plan 2.1) to start after both Wave 1 plans are merged to main
4. Ensure all implementers have access to `container/list` stdlib documentation (used in Plan 1.2)

**Post-Execution Checklist:**
1. Verify all 14 test cases pass as specified
2. Confirm benchmark output shows capacity bounds (Plan 2.1 Task 1)
3. Check that no file descriptor warnings appear when running with `-eco-max-handles 5` (Plan 1.2)
4. Validate memory usage under 100K game load against baseline (Plan 2.1 Task 1)

---

## Verdict

**PASS** — Phase 3 plans are structurally sound, comprehensive, and ready for execution. All four phase requirements are fully addressed with concrete, testable acceptance criteria. Wave 1 can execute in parallel with no file conflicts. Wave 2 has correct dependencies. All 14 test cases (6 unit, 6 integration, 2 benchmark) collectively verify bounded memory behavior while preserving existing correctness.

**Quality Rating:** Excellent. Plans demonstrate clear understanding of requirement interdependencies, appropriate use of TDD methodology (Plans 1.1 and 1.2), and comprehensive regression verification (Plan 2.1 Task 3 with race detector).

---

## Appendix: Requirement-to-Plan Mapping

```
Phase 3 Requirement 1: Hash table bounded by capacity
  ├─ Plan 1.1, Task 1: maxCapacity field + gating logic
  ├─ Plan 1.1, Task 2: Propagate through ThreadSafeDuplicateDetector
  ├─ Plan 1.1, Task 3: CLI flag -duplicate-capacity
  └─ Plan 2.1, Task 1: Benchmark 100K games with cap=1000

Phase 3 Requirement 2: ECOSplitWriter bounded file handles
  ├─ Plan 1.2, Task 1: LRU cache with eviction + OpenHandleCount()
  ├─ Plan 1.2, Task 2: LRU behavior tests (eviction, reopen, unlimited)
  ├─ Plan 1.2, Task 3: CLI flag -eco-max-handles (default 128)
  └─ Plan 2.1, Task 2: Integration test with 20 codes, maxHandles=5

Phase 3 Requirement 3: Behavior unchanged below capacity
  ├─ Plan 1.1, Task 1: DuplicatesStillDetected test at capacity
  └─ Plan 2.1, Task 3: BehaviorUnchanged_BelowCapacity + BehaviorUnchanged_Unlimited tests

Phase 3 Requirement 4: Bounded memory benchmark
  └─ Plan 2.1, Task 1: BenchmarkDuplicateDetector_BoundedMemory (100K games, multiple capacities)
```
