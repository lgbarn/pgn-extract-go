# Phase 6 Research: Nolint Suppressions and Global Config Cleanup

**Research Date:** 2026-02-01
**Goal:** Reduce nolint suppressions by at least 50% (from 50 to 25 or fewer) and begin reducing reliance on global config.

---

## Executive Summary

**Current State:**
- **Total nolint directives in source files (non-test):** 45
- **Total nolint directives in test files:** 5
- **Total across all Go files:** 50

**Goal Achievement:**
- Target: Reduce from 50 to 25 or fewer in source files
- **Current source file count:** 45 (excludes test files already)
- **Required reduction:** ~20 suppressions to reach target of 25

**Feasibility Assessment:** **ACHIEVABLE**
The majority of suppressions fall into a small number of categories where systematic fixes can be applied.

---

## 1. Nolint Suppressions by Category

### 1.1 Source Files (45 total)

#### Category A: `errcheck` on known-valid FEN parsing (8 instances)
**Pattern:** `NewBoardFromFEN(InitialFEN)` where InitialFEN is a constant

**Locations:**
1. `internal/engine/fen.go:300` - `board, _ := NewBoardFromFEN(InitialFEN)`
2. `internal/eco/eco.go:84` - `board, _ := engine.NewBoardFromFEN(engine.InitialFEN)`
3. `internal/eco/eco.go:183` - `board, _ := engine.NewBoardFromFEN(engine.InitialFEN)`
4. `cmd/pgn-extract/filters.go:347` - `board, _ := engine.NewBoardFromFEN(engine.InitialFEN)`
5. `internal/matching/position.go:126` - `board, _ := engine.NewBoardFromFEN(engine.InitialFEN)`
6. `internal/matching/variation.go:150` - `board, _ := engine.NewBoardFromFEN(engine.InitialFEN)`
7. `internal/matching/material.go:75` - `board, _ := engine.NewBoardFromFEN(engine.InitialFEN)`
8. `internal/processing/analyzer.go:152` - `board, _ = engine.NewBoardFromFEN(engine.InitialFEN)`

**Current Justification:** "InitialFEN is known valid"

**Fixability:** **EASY - Extract helper function**
**Recommended Approach:** Create `MustBoardFromFEN(fen string) *Board` helper in `internal/engine/fen.go` that panics on error (acceptable since InitialFEN is compile-time constant). Replace all 8 instances with this helper.

**Impact:** Removes 8 suppressions (-17.8%)

---

#### Category B: `errcheck` on `fmt.Sscanf` with acceptable defaults (3 instances)
**Pattern:** `fmt.Sscanf` where default zero value is acceptable

**Locations:**
1. `internal/engine/fen.go:206` - `fmt.Sscanf(parts[4], "%d", &board.HalfmoveClock)` - halfmove clock
2. `internal/engine/fen.go:209` - `fmt.Sscanf(parts[5], "%d", &board.MoveNumber)` - move number
3. `internal/parser/lexer.go:545` - `fmt.Sscanf(numStr, "%d", &moveNum)` - move number parsing

**Current Justification:** "default 0 acceptable for invalid input"

**Fixability:** **KEEP WITH BETTER DOCUMENTATION**
**Recommended Approach:** These are legitimate cases where the zero value on parse failure is the desired behavior. The `.golangci.yml` already excludes `fmt.Sscanf` from errcheck (line 66), so these suppressions are actually redundant.

**Impact:** Remove suppressions (already covered by config) - Removes 3 suppressions (-6.7%)

---

#### Category C: `gosec G304` - User-specified file paths (7 instances)
**Pattern:** Opening files from user-provided paths (expected behavior for CLI tool)

**Locations:**
1. `internal/eco/eco.go:50` - ECO file
2. `cmd/pgn-extract/processor.go:80` - Split output file creation
3. `cmd/pgn-extract/processor.go:196` - Append mode output file
4. `cmd/pgn-extract/processor.go:208` - Create output file
5. `cmd/pgn-extract/main.go:383` - Open input PGN file
6. `cmd/pgn-extract/main.go:439` - Open input file (variation matcher)
7. `cmd/pgn-extract/main.go:500` - Open input file (check file)
8. `internal/matching/variation.go:31` - Variation file
9. `internal/matching/variation.go:57` - Position file
10. `internal/matching/filter.go:34` - Filter file

**Current Justification:** "CLI tool opens user-specified files"

**Fixability:** **KEEP - Legitimate CLI behavior**
**Recommended Approach:** These are genuine use cases. pgn-extract is a file processing tool that must open user-specified files. G304 warnings are false positives here.

**Note:** Consider adding a security comment in documentation about file path validation if running in server mode (not current use case).

**Impact:** Keep all 10 (-0 suppressions)

---

#### Category D: `gosec G302` - File permissions 0644 (2 instances)
**Pattern:** Creating user output files with standard read-write permissions

**Locations:**
1. `cmd/pgn-extract/main.go:146` - Log file creation with 0644
2. `cmd/pgn-extract/main.go:165` - Output file creation with 0644

**Current Justification:** "0644 is appropriate for user-created output/log files"

**Fixability:** **KEEP - Correct permissions**
**Recommended Approach:** 0644 (rw-r--r--) is the standard and appropriate permission for user-created output files. This is not a security issue.

**Impact:** Keep all 2 (-0 suppressions)

---

#### Category E: `errcheck` on `Close()` during cleanup (7 instances)
**Pattern:** Ignoring Close() errors during cleanup/exit paths

**Locations:**
1. `cmd/pgn-extract/processor.go:76` - Close before creating new split file
2. `cmd/pgn-extract/processor.go:241` - Close on LRU eviction
3. `cmd/pgn-extract/main.go:395` - Close input file on exit
4. `cmd/pgn-extract/main.go:400` - Close split writer on exit
5. `cmd/pgn-extract/main.go:405` - Close ECO split writer on exit

**Current Justification:** "cleanup before creating new file" or "cleanup on exit"

**Fixability:** **MIXED - Some fixable, some acceptable**

**Recommended Approach:**
- **Exit path closes (3 instances):** Can LOG errors but not much else can be done. The `.golangci.yml` already excludes `(*os.File).Close` (line 50), so these are redundant.
- **Pre-create closes (2 instances):** Could check and log error before proceeding to create new file.
- **LRU eviction (1 instance):** Could log error but eviction must proceed.

**Conservative estimate:** Remove 3 redundant suppressions (covered by config), fix 2 by logging errors.

**Impact:** Removes 5 suppressions (-11.1%)

---

#### Category F: Type assertions with ok pattern (2 instances)
**Pattern:** Type assertions where the ok value is checked or zero value is acceptable

**Locations:**
1. `internal/worker/pool.go:45` - `gi, _ := r.GameInfo.(GameInfo)`
2. `cmd/pgn-extract/processor.go:485` - `gameInfo, _ := result.GameInfo.(*GameAnalysis)`

**Current Justification:** "type assertion with ok returns zero value"

**Fixability:** **EASY - Use ok variable**
**Recommended Approach:** Change to `gi, ok := r.GameInfo.(GameInfo)` and handle appropriately. The `.golangci.yml` has `check-type-assertions: true` (line 44), so these should be fixed.

**Impact:** Removes 2 suppressions (-4.4%)

---

#### Category G: `errcheck` on errors handled elsewhere (5 instances)
**Pattern:** Encoding/output operations where errors are handled by the writer

**Locations:**
1. `internal/matching/tags.go:101` - `AddCriterion` - errors not expected
2. `internal/matching/tags.go:111` - `AddCriterion` - errors not expected
3. `internal/matching/filter.go:58` - `AddFEN` - errors logged internally
4. `internal/matching/filter.go:61` - `ParseCriterion` - parsing errors tolerated
5. `internal/matching/filter.go:70` - `AddCriterion` - errors not expected
6. `internal/matching/filter.go:80` - `AddCriterion` - errors not expected
7. `internal/matching/filter.go:85` - `AddCriterion` - errors not expected
8. `internal/matching/filter.go:90` - `AddCriterion` - errors not expected
9. `internal/matching/filter.go:95` - `AddCriterion` - errors not expected
10. `internal/matching/filter.go:100` - `AddCriterion` - errors not expected
11. `internal/output/json.go:49` - `enc.Encode` - output errors handled by writer
12. `internal/output/json.go:61` - `enc.Encode` - output errors handled by writer
13. `internal/output/json.go:110` - FEN parsing with nil check

**Current Justification:** Various - "errors not expected", "errors logged internally", "output errors handled by writer"

**Fixability:** **MIXED**

**Recommended Approach:**
- **JSON encoding (2 instances):** Already excluded in `.golangci.yml` line 64 - suppressions redundant. Remove suppressions.
- **AddCriterion calls (9 instances):** These are programming errors if they fail. Should either: (a) return error to caller, or (b) panic if truly impossible. **Refactor to return errors.**
- **AddFEN/ParseCriterion (2 instances):** If errors are logged internally and failure is acceptable, this is fine. Keep.

**Impact:** Remove 2 redundant, refactor 9 to return errors = Removes 11 suppressions (-24.4%)

---

#### Category H: Test helper with acceptable panic (1 instance)
**Pattern:** Test utility function that can panic

**Locations:**
1. `internal/testutil/game.go:31` - `games, _ := p.ParseAllGames()` - test helper

**Current Justification:** "test helper, panics not expected"

**Fixability:** **FIX - Check error or use Must pattern**
**Recommended Approach:** This is in test utility code. Either check the error and panic explicitly with a helpful message, or rename to `MustParseAllGames()` to signal panicking behavior.

**Impact:** Removes 1 suppression (-2.2%)

---

### 1.2 Test Files (5 total)

**Locations:**
1. `cmd/pgn-extract/clock_test.go:15` - G306: test file permissions (0644 is fine for tests)
2. `cmd/pgn-extract/golden_test.go:49` - G204,noctx: test builds binary (expected)
3. `cmd/pgn-extract/golden_test.go:68` - G204,noctx: test runs binary (expected)
4. `cmd/pgn-extract/golden_test.go:337` - G304: test reads temp file (safe)
5. `internal/output/writer_test.go:138` - G104: test code ignoring error (acceptable)

**Recommended Approach:** **KEEP ALL**
These are test files. The `.golangci.yml` already excludes errcheck and gosec from test files (lines 138-141). These suppressions may be redundant but are harmless. Low priority.

**Impact:** 0 (not counted toward goal)

---

## 2. Suppression Reduction Plan

### Summary Table

| Category | Count | Fixable | Keep | Reduction |
|----------|-------|---------|------|-----------|
| A. Known-valid FEN | 8 | 8 | 0 | -8 |
| B. Sscanf defaults | 3 | 3 (remove) | 0 | -3 |
| C. User file paths (G304) | 10 | 0 | 10 | 0 |
| D. File permissions (G302) | 2 | 0 | 2 | 0 |
| E. Close() cleanup | 7 | 5 | 2 | -5 |
| F. Type assertions | 2 | 2 | 0 | -2 |
| G. Error handling | 13 | 11 | 2 | -11 |
| H. Test helper | 1 | 1 | 0 | -1 |
| **TOTAL (Source)** | **45** | **30** | **16** | **-30** |
| **Test files** | **5** | **0** | **5** | **0** |

### Projected Outcome
- **Current:** 45 source file suppressions
- **After fixes:** 15 source file suppressions
- **Reduction:** 66.7% (exceeds 50% goal)

---

## 3. Implementation Roadmap

### Phase 6.1: Low-Hanging Fruit (Quick Wins) — 14 suppressions removed

**Priority: HIGH | Effort: LOW | Risk: LOW**

1. **Create `MustBoardFromFEN` helper** (8 suppressions)
   - Add function to `internal/engine/fen.go`:
     ```go
     func MustBoardFromFEN(fen string) *Board {
         board, err := NewBoardFromFEN(fen)
         if err != nil {
             panic(fmt.Sprintf("invalid FEN (should be impossible): %v", err))
         }
         return board
     }
     ```
   - Replace all 8 `board, _ := NewBoardFromFEN(InitialFEN) //nolint` instances
   - **Validation:** All unit tests should pass (InitialFEN is constant and valid)

2. **Remove redundant Sscanf suppressions** (3 suppressions)
   - These are already excluded in `.golangci.yml` line 66
   - Simply delete the `//nolint:errcheck,gosec` comments
   - **Validation:** `golangci-lint run` should not flag these lines

3. **Fix type assertions** (2 suppressions)
   - `internal/worker/pool.go:45`: Use ok variable and handle zero value case
   - `cmd/pgn-extract/processor.go:485`: Use ok variable and handle zero value case
   - **Validation:** Unit tests for worker pool and processor

4. **Fix test helper** (1 suppression)
   - Rename `internal/testutil/game.go` function to `MustParseAllGames()` and panic explicitly on error
   - **Validation:** Test suite runs successfully

---

### Phase 6.2: Error Handling Refactor — 11 suppressions removed

**Priority: MEDIUM | Effort: MEDIUM | Risk: MEDIUM**

5. **Refactor `AddCriterion` calls to return errors** (9 suppressions)
   - Files affected:
     - `internal/matching/tags.go` (2 instances)
     - `internal/matching/filter.go` (7 instances)
   - Change calling code to check and handle errors
   - Consider whether errors should be logged or propagated
   - **Validation:**
     - Unit tests for matching package
     - Integration tests with malformed filter inputs

6. **Remove redundant JSON encoding suppressions** (2 suppressions)
   - `internal/output/json.go:49` and `json.go:61`
   - Already excluded in `.golangci.yml` line 64
   - Simply delete the `//nolint:errcheck,gosec` comments
   - **Validation:** `golangci-lint run` should not flag these

---

### Phase 6.3: Close() Error Handling — 5 suppressions removed

**Priority: LOW | Effort: LOW | Risk: LOW**

7. **Add error logging for Close() in file rotation** (2 suppressions)
   - `cmd/pgn-extract/processor.go:76` - log error before creating new file
   - `cmd/pgn-extract/processor.go:241` - log error on LRU eviction
   - Use log.Printf to stderr for diagnostic purposes

8. **Remove redundant exit-path Close() suppressions** (3 suppressions)
   - Already excluded in `.golangci.yml` line 50
   - Delete nolint comments from exit path cleanup code
   - **Validation:** `golangci-lint run` should not flag these

---

## 4. Global Config Usage Analysis

### 4.1 Current Architecture

**Global Config State:**
- `internal/config/config.go:150` - `var GlobalConfig *Config`
- Initialized in `init()` function at line 177
- **Usage:** Currently **NOT directly referenced** anywhere in codebase

**Flag Architecture:**
- All flags defined as package-level variables in `cmd/pgn-extract/flags.go` (165 total flags)
- Flags are read directly by filter/processing functions
- Configuration is passed as `*config.Config` parameter to most functions

### 4.2 Global Flag Usage Patterns

**Direct flag references found in:**

1. **`cmd/pgn-extract/filters.go`** - Extensive direct flag reads:
   - Line 27-38: Flag parsing in `initSelectionSets()`
   - Line 79-255: Filter application functions read ~40 flags directly:
     - `*fixableMode`, `*strictMode`, `*validateMode`
     - `*negateMatch`, `*exactPly`, `*minPly`, `*maxPly`
     - `*exactMove`, `*minMoves`, `*maxMoves`
     - `*checkmateFilter`, `*stalemateFilter`, `*fiftyMoveFilter`
     - `*repetitionFilter`, `*underpromotionFilter`, `*commentedFilter`
     - `*higherRatedWinner`, `*lowerRatedWinner`
     - `*seventyFiveMoveFilter`, `*fiveFoldRepFilter`
     - `*insufficientFilter`, `*materialOddsFilter`
     - `*pieceCount`, `*noSetupTags`, `*onlySetupTags`
     - `*dropPly`, `*startPly`, `*plyLimit`, `*dropBefore`

2. **`cmd/pgn-extract/main.go`** - Setup and initialization:
   - Flags read during setup: `*outputFile`, `*appendOutput`, `*logFile`, `*appendLog`
   - Flags read during processing: `*splitGames`, `*ecoSplit`, `*negateMatch`

3. **`cmd/pgn-extract/processor.go`** - Limited flag usage:
   - `*splitPattern` for filename generation

4. **`cmd/pgn-extract/main_test.go`** - Test code directly modifies flags:
   - Lines 158-228: Modifies `*playerFilter`, `*whiteFilter`, `*blackFilter`, etc. for testing

### 4.3 Good News: Most Configuration is Already Parameterized

**Functions that accept `*config.Config`:**
- `applyFlags(cfg *config.Config)` - applies flags to config
- `withOutputFile(cfg *config.Config, w io.Writer, fn func())`
- `ProcessingContext.cfg *config.Config`
- Most matching and filtering functions accept config

**Current pattern:**
```go
cfg := config.NewConfig()
applyFlags(cfg)
// cfg is passed to processing functions
```

### 4.4 Global Config Reduction Opportunities

#### Easy Wins (Straightforward refactoring)

**Opportunity 1: Move flag variables into a FlagSet struct**
- **Impact:** Eliminates package-level flag variables
- **Effort:** Medium (need to update all references)
- **Risk:** Medium (affects test code that modifies flags)
- **Files affected:** `flags.go`, `filters.go`, `main.go`, all test files
- **Note:** This would make testing easier and eliminate global mutable state

**Opportunity 2: Expand `config.Filter` to include all filter flags**
- **Current:** Many filter flags are read directly instead of from `cfg.Filter`
- **Missing from config.Filter:**
  - Exact ply/move matching (`exactPly`, `exactMove`)
  - Ply/move ranges (`plyRange`, `moveRange`)
  - Game position selection (`selectOnly`, `skipMatching`)
  - Ending filters (`checkmateFilter`, `stalemateFilter`)
  - Feature filters (50-move, repetition, underpromotion, etc.)
  - Rating-based filters (`higherRatedWinner`, `lowerRatedWinner`)
  - Piece count filter
  - Setup tag filters
- **Benefit:** Filter functions could operate purely on `cfg.Filter` instead of reading global flags
- **Effort:** Medium (add fields, update `applyFlags()`, update filter functions)
- **Risk:** Low (config is already passed around, just expanding it)

**Opportunity 3: Move validation flags to config.Validation**
- **Current flags:** `*strictMode`, `*validateMode`, `*fixableMode`
- **Create:** `config.ValidationConfig` struct
- **Benefit:** Clearer separation of concerns
- **Effort:** Low
- **Risk:** Low

**Opportunity 4: Move output truncation to config.Output**
- **Current flags:** `*dropPly`, `*startPly`, `*plyLimit`, `*dropBefore`
- **Move to:** `config.Output` struct (already exists)
- **Benefit:** Consolidate output-related settings
- **Effort:** Low
- **Risk:** Low

#### Harder Refactorings (Require more thought)

**Opportunity 5: Eliminate global counters**
- **Current:** `matchedCount` and `gamePositionCounter` are package-level `atomic.Int64`
- **Better:** Pass as part of processing context or return from functions
- **Benefit:** Eliminates mutable global state, easier testing
- **Effort:** Medium (need to thread through call chain)
- **Risk:** Medium (affects concurrent processing logic)

**Opportunity 6: Encapsulate selection sets**
- **Current:** `selectOnlySet`, `skipMatchingSet`, `parsedPlyRange`, `parsedMoveRange` are package-level
- **Better:** Move to `config.Filter` or create `SelectionConfig`
- **Benefit:** Easier to test, no global state
- **Effort:** Low-Medium
- **Risk:** Low

---

## 5. Recommended Approach for Phase 6

### Goal Scoping

Given the Phase 6 timeline and risk profile, recommend focusing on **nolint reduction only** and defer global config cleanup to Phase 7 or future work.

**Rationale:**
1. **Nolint reduction is clear-cut:** 30 suppressions can be removed with low risk
2. **Global config cleanup is more invasive:** Requires touching many files and coordinating changes
3. **Test impact:** Flag refactoring affects test infrastructure significantly
4. **Incremental approach:** Better to do nolint cleanup cleanly in Phase 6, then address global state in dedicated refactoring phase

### Phase 6 Deliverables

**Primary Goal: Reduce nolint suppressions from 45 to 15 (66% reduction)**

**Implementation Plan:**
1. Week 1: Phase 6.1 — Low-hanging fruit (14 suppressions removed)
2. Week 2: Phase 6.2 — Error handling refactor (11 suppressions removed)
3. Week 3: Phase 6.3 — Close() improvements (5 suppressions removed)
4. Week 4: Testing, validation, documentation

**Success Criteria:**
- Source file suppressions: 15 or fewer (from 45)
- All tests pass
- `golangci-lint run` passes with no new violations
- Documentation updated for remaining suppressions

---

## 6. Potential Risks and Mitigations

### Risk 1: MustBoardFromFEN panic in production
**Likelihood:** Very Low
**Impact:** High (program crash)
**Mitigation:** InitialFEN is a compile-time constant and verified by tests. Add explicit test that validates InitialFEN parses successfully.

### Risk 2: Error propagation changes behavior
**Likelihood:** Medium
**Impact:** Medium (filter behavior changes)
**Mitigation:**
- Add tests for error cases in `AddCriterion` before refactoring
- Review all call sites to ensure error handling is appropriate
- Consider backward compatibility if errors were silently ignored before

### Risk 3: Removing suppressions reveals real issues
**Likelihood:** Low
**Impact:** Medium (need to fix underlying issues)
**Mitigation:**
- Run `golangci-lint run` after each category of changes
- Fix any newly revealed issues before removing suppressions
- Have rollback plan if issues are severe

### Risk 4: Test instability from flag modifications
**Likelihood:** Low
**Impact:** Low (test-only)
**Mitigation:** Test files are not part of reduction goal. If refactoring causes issues, test suppressions can remain.

---

## 7. Golangci-lint Configuration Review

**File:** `.golangci.yml`

**Enabled Linters (17 total):**
- errcheck, govet, ineffassign, staticcheck, unused (defaults)
- bodyclose, durationcheck, nilerr, noctx (bug detection)
- misspell (code style)
- cyclop, gocognit, nakedret (complexity)
- errorlint (error handling)
- prealloc (performance)
- gosec (security)

**Relevant Exclusions:**
- Line 43-67: `errcheck.exclude-functions` - Already excludes:
  - `(*os.File).Close`
  - `fmt.Sscanf`
  - `(*encoding/json.Encoder).Encode`
  - Many others
- Line 138-141: Excludes errcheck and gosec from `*_test.go` files
- Line 143-145: Excludes cyclop from `internal/parser/lexer.go`

**Configuration Quality:** Good
The configuration is well-tuned for this codebase. Many of the current nolint suppressions are actually redundant because the checks are already excluded in the config.

**Recommendation:** No changes needed to `.golangci.yml`. The configuration is appropriate. Focus on fixing code rather than loosening linter rules.

---

## 8. Documentation Links

### Relevant Documentation

1. **golangci-lint Configuration:**
   - https://golangci-lint.run/usage/configuration/
   - https://golangci-lint.run/usage/linters/

2. **Linter-Specific Docs:**
   - errcheck: https://github.com/kisielk/errcheck
   - gosec: https://github.com/securego/gosec
   - staticcheck: https://staticcheck.dev/docs/

3. **Go Error Handling Best Practices:**
   - https://go.dev/blog/error-handling-and-go
   - https://github.com/uber-go/guide/blob/master/style.md#error-handling

4. **Testing Patterns:**
   - Table-driven tests: https://go.dev/wiki/TableDrivenTests
   - Test fixtures: https://go.dev/wiki/TestFixtures

### Related GitHub Issues/Discussions

No specific upstream issues are relevant. This is internal code quality work.

---

## 9. Implementation Considerations

### Integration Points

**Files requiring changes (by phase):**
- **Phase 6.1:**
  - `internal/engine/fen.go` (add helper, modify 1 call site)
  - `internal/eco/eco.go` (2 call sites)
  - `cmd/pgn-extract/filters.go` (1 call site)
  - `internal/matching/position.go` (1 call site)
  - `internal/matching/variation.go` (1 call site)
  - `internal/matching/material.go` (1 call site)
  - `internal/processing/analyzer.go` (1 call site)
  - `internal/parser/lexer.go` (remove 1 suppression)
  - `internal/worker/pool.go` (fix type assertion)
  - `cmd/pgn-extract/processor.go` (fix type assertion)
  - `internal/testutil/game.go` (rename function)

- **Phase 6.2:**
  - `internal/matching/tags.go` (refactor error handling)
  - `internal/matching/filter.go` (refactor error handling)
  - `internal/output/json.go` (remove 2 suppressions)

- **Phase 6.3:**
  - `cmd/pgn-extract/processor.go` (add error logging)
  - `cmd/pgn-extract/main.go` (remove 3 suppressions)

### Migration Concerns

**No breaking API changes expected.** All changes are internal implementations.

**Potential behavior changes:**
- `AddCriterion` will return errors instead of silently failing
- Type assertions will handle nil/zero cases explicitly
- Close() errors will be logged (new output to stderr)

### Performance Implications

**Negligible impact.** Changes are primarily about error handling, not algorithmic changes.

**Possible micro-optimizations:**
- `MustBoardFromFEN` avoids error check at call sites (tiny speedup)
- Error propagation adds branches (tiny slowdown)
- Net effect: ~0%

### Testing Strategy

**Unit Tests:**
- Add test for `MustBoardFromFEN` with valid and invalid FEN
- Add tests for error cases in `AddCriterion` refactor
- Verify type assertion handling with nil values

**Integration Tests:**
- Run existing golden tests: `cmd/pgn-extract/golden_test.go`
- Verify filter behavior unchanged with error propagation changes
- Test file processing end-to-end

**Regression Prevention:**
- Run full test suite after each phase: `go test ./...`
- Run golangci-lint after each category: `golangci-lint run`
- Verify no new suppressions added: `grep -r "//nolint" --include="*.go" | wc -l`

**Benchmark Testing:**
- Run existing benchmarks to verify no performance regression
- `go test -bench=. -benchmem ./...`

---

## 10. Summary

### What Can Realistically Be Achieved

**Nolint Suppressions:**
- **Baseline:** 45 suppressions in source files
- **Achievable target:** 15 suppressions (66% reduction)
- **Exceeds goal:** 50% reduction goal → 66% actual reduction
- **Timeline:** 3-4 weeks of focused work
- **Risk:** Low (mostly mechanical refactoring)

**Global Config Cleanup:**
- **Current state:** No direct GlobalConfig usage, but extensive global flag reads
- **Achievable in Phase 6:** Limited scope (defer to Phase 7)
- **Recommended:**
  - Document current architecture
  - Identify refactoring opportunities
  - Plan for future Phase 7+ work
- **Rationale:** Global config cleanup is more invasive and should be tackled separately

### Confidence Level

**Nolint Reduction: HIGH CONFIDENCE (90%)**
- Clear categories with known fixes
- Low interdependencies between changes
- Good test coverage exists
- Incremental approach with validation at each step

**Global Config Elimination: MEDIUM CONFIDENCE (60%)**
- Requires touching many files
- Test infrastructure depends on flag mutation
- Need careful coordination of changes
- Better suited for dedicated refactoring phase

### Recommendation

**PROCEED** with nolint reduction in Phase 6.
**DEFER** global config cleanup to Phase 7 or later dedicated effort.

This approach delivers clear wins (66% nolint reduction) with manageable risk while avoiding scope creep into larger architectural changes.
