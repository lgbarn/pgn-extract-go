# pgn-extract-go Quality Improvement Roadmap

## Milestone: Production-Quality Hardening

Bring pgn-extract-go to production quality by fixing concurrency bugs, bounding memory usage, improving test coverage, and cleaning up technical debt -- all while maintaining full backward compatibility and zero external dependencies.

### Success Criteria (Milestone-Level)

1. `go test -race ./...` passes with zero race conditions
2. No unbounded memory growth when processing 100K+ game files
3. Matching package test coverage >70%
4. Processing package test coverage >70%
5. Nolint suppressions reduced by at least 50% (from 50 to 25 or fewer in .go files)
6. `go.mod` specifies Go 1.23 as minimum version
7. All existing CLI behaviors preserved (backward compatible)

---

### Phase 1: Go Version Bump

**Complexity:** S

**Dependencies:** None

**Description:**
Update the minimum Go version from 1.21 to 1.23 in `go.mod`. Run `go mod tidy` to regenerate module metadata. Verify that all existing tests pass under Go 1.23. This is purely foundational -- no code changes beyond `go.mod`, and it unblocks use of Go 1.22/1.23 language features (range-over-int, improved standard library APIs) in later phases.

**Files affected:**
- `go.mod`

**Success Criteria:**
- `go.mod` contains `go 1.23`
- `go mod tidy` exits cleanly
- `go test ./...` passes with zero failures
- `go vet ./...` reports no new issues

---

### Phase 2: Concurrency Safety Fixes

**Complexity:** M

**Dependencies:** Phase 1

**Description:**
Fix the critical data race in parallel game processing. The `DuplicateDetector` (non-thread-safe) is currently used in `outputGamesParallel` (`cmd/pgn-extract/processor.go:346`), while a `ThreadSafeDuplicateDetector` already exists but is not wired in. This phase swaps in the thread-safe variant for all parallel code paths, audits atomic counter usage for correctness, and ensures `go test -race ./...` passes cleanly. Also audit the ECO split writer and any other shared mutable state accessed from worker goroutines.

**Files affected:**
- `cmd/pgn-extract/processor.go`
- `internal/hashing/hashing.go`
- `internal/hashing/thread_safe.go`
- `cmd/pgn-extract/filters.go` (atomic counter audit)

**Success Criteria:**
- `go test -race ./...` passes with zero data race reports
- Parallel duplicate detection produces correct counts (identical results to single-threaded mode on the same input)
- No behavioral change for single-threaded execution paths

---

### Phase 3: Memory Management

**Complexity:** M

**Dependencies:** Phase 2

**Description:**
Address unbounded memory growth in duplicate detection and ECO split writing. For `DuplicateDetector` and `ThreadSafeDuplicateDetector`, add a configurable maximum capacity with either an eviction policy (LRU) or a simple cap that stops tracking once the limit is reached. For the `ECOSplitWriter`, implement an LRU file handle cache so that the number of simultaneously open file descriptors is bounded (default to a safe limit like 128). Measure memory usage with a benchmark that processes a large number of synthetic games to verify the bounds hold.

**Files affected:**
- `internal/hashing/hashing.go`
- `internal/hashing/thread_safe.go`
- `cmd/pgn-extract/processor.go` (ECOSplitWriter)

**Success Criteria:**
- Hash table size is bounded: processing 100K+ games does not grow the hash table beyond the configured limit
- ECO split writer holds at most N file handles open simultaneously (configurable, default 128)
- Existing duplicate detection behavior unchanged when capacity is not exceeded
- New benchmark test demonstrates bounded memory under load

---

### Phase 4: Test Coverage -- Matching Package

**Complexity:** M

**Dependencies:** Phase 2

**Description:**
Raise the `internal/matching` package test coverage from 34.6% to above 70%. Focus on under-tested areas: `filter.go` (9 nolint suppressions suggest complex untested paths), `position.go`, `variation.go`, `material.go`, and `tags.go`. Write unit tests that exercise each filter type with both matching and non-matching inputs. Use table-driven test patterns consistent with the existing codebase style. Tests should also cover edge cases: nil games, empty tag sets, invalid patterns, and boundary conditions.

**Files affected:**
- `internal/matching/filter_test.go` (new or extended)
- `internal/matching/position_test.go` (new or extended)
- `internal/matching/variation_test.go` (new or extended)
- `internal/matching/material_test.go` (new or extended)
- `internal/matching/tags_test.go` (new or extended)

**Success Criteria:**
- `go test -cover ./internal/matching/` reports >70% coverage
- All new tests pass with `go test -race`
- No changes to production code in this phase (tests only)

---

### Phase 5: Test Coverage -- Processing Package

**Complexity:** M

**Dependencies:** Phase 2, Phase 3

**Description:**
Raise the `cmd/pgn-extract` (processing) package test coverage from 36.1% to above 70%. Focus on `processor.go` (game output, split writing, ECO splitting, parallel processing), `filters.go` (filter application pipeline), and `main.go` (CLI argument handling, file processing loop). Write integration-style tests that exercise the full pipeline with synthetic PGN input. Cover both single-threaded and parallel code paths. Test error conditions: invalid files, permission errors, split writer edge cases.

**Files affected:**
- `cmd/pgn-extract/processor_test.go` (new or extended)
- `cmd/pgn-extract/filters_test.go` (new or extended)
- `cmd/pgn-extract/main_test.go` (extended)

**Success Criteria:**
- `go test -cover ./cmd/pgn-extract/` reports >70% coverage
- All new tests pass with `go test -race`
- Tests cover both single-threaded and parallel processing paths

---

### Phase 6: Code Cleanup -- Nolint Suppressions and Global Config

**Complexity:** M

**Dependencies:** Phase 4, Phase 5

**Description:**
Reduce nolint suppressions by at least 50% (from 50 to 25 or fewer in .go source files). For each suppression: (a) fix the underlying issue if feasible (e.g., check error returns from `Close()`, use `filepath.Clean` for G304 where appropriate), (b) keep the suppression with an improved justification comment if it is genuinely acceptable (e.g., CLI tool intentionally opens user-specified files), or (c) restructure code to avoid the linter complaint. Separately, begin reducing reliance on `GlobalConfig` by passing `*config.Config` explicitly where it is straightforward to do so. Full removal of `GlobalConfig` is not required, but usage should be reduced.

**Files affected:**
- `cmd/pgn-extract/main.go`
- `cmd/pgn-extract/processor.go`
- `cmd/pgn-extract/filters.go`
- `cmd/pgn-extract/golden_test.go`
- `internal/matching/filter.go`
- `internal/matching/position.go`
- `internal/matching/variation.go`
- `internal/matching/tags.go`
- `internal/matching/material.go`
- `internal/output/json.go`
- `internal/output/output.go`
- `internal/eco/eco.go`
- `internal/engine/fen.go`
- `internal/parser/lexer.go`
- `internal/worker/pool.go`
- `internal/config/config.go`

**Success Criteria:**
- 25 or fewer `//nolint` directives remain in `.go` source files
- Every remaining `//nolint` has a clear justification comment
- `go test ./...` passes (no regressions from cleanup)
- `golangci-lint run` passes (or produces fewer warnings than before)
- At least some `GlobalConfig` usage replaced with explicit parameter passing

---

### Phase 7: Final Verification and Polish

**Complexity:** S

**Dependencies:** Phase 3, Phase 4, Phase 5, Phase 6

**Description:**
End-to-end verification that all milestone success criteria are met. Run the full test suite with the race detector. Confirm coverage numbers. Count remaining nolint suppressions. Verify Go version in `go.mod`. Run a manual smoke test processing a large PGN file to confirm no regressions in CLI behavior or output format. Address any remaining issues discovered during verification. Update CI configuration if needed to match the new Go 1.23 minimum.

**Files affected:**
- `.github/workflows/ci.yml` (if CI Go version needs updating)
- Any files with issues discovered during verification

**Success Criteria:**
- `go test -race ./...` passes with zero failures and zero race reports
- `go test -cover ./internal/matching/` reports >70%
- `go test -cover ./cmd/pgn-extract/` reports >70%
- 25 or fewer `//nolint` directives in `.go` source files
- `go.mod` specifies `go 1.23`
- CLI output for a representative PGN file is identical before and after the project (backward compatibility)
- All 7 milestone success criteria confirmed met

---

### Phase Dependency Graph

```
Phase 1 (Go Version)
  |
  v
Phase 2 (Concurrency) --------+------------------+
  |                            |                  |
  v                            v                  |
Phase 3 (Memory)          Phase 4 (Matching)      |
  |                            |                  |
  |                            v                  v
  +--------------------> Phase 5 (Processing)     |
                               |                  |
                               v                  |
                         Phase 6 (Cleanup) <------+
                               |
                               v
                         Phase 7 (Verification)
```

### Complexity Summary

| Phase | Title                  | Complexity | Est. Tasks |
|-------|------------------------|------------|------------|
| 1     | Go Version Bump        | S          | 1-2        |
| 2     | Concurrency Safety     | M          | 2-3        |
| 3     | Memory Management      | M          | 2-3        |
| 4     | Test Coverage: Matching | M          | 2-3       |
| 5     | Test Coverage: Processing | M       | 2-3        |
| 6     | Nolint and Global Config Cleanup | M | 2-3       |
| 7     | Final Verification     | S          | 1-2        |
