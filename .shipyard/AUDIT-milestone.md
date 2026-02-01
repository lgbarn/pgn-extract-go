# Security Audit Report

**Project:** pgn-extract-go
**Phase:** Full Milestone (Phases 1-7)
**Date:** 2026-02-01
**Scope:** 49 files changed across 44 commits, 239 insertions / 72 deletions in production code (excluding tests, docs, PGN data)

## Summary

**Verdict:** PASS

**Critical findings:** 0
**Important findings:** 2
**Advisory findings:** 5

---

## Critical Findings

None.

---

## Important Findings

### IMP-1: Compiled binary committed to repository

- **Location:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/pgn-extract-new` (Mach-O 64-bit arm64 executable, 3.9 MB)
- **Category:** Supply Chain / Repository Hygiene
- **Description:** A compiled Go binary (`pgn-extract-new`) is tracked as an untracked file and appears in `git diff --stat` as a new file. The `.gitignore` excludes `/pgn-extract` and `/pgn-extract-go` but does not exclude `pgn-extract-new`. If committed, this binary bloats the repository, cannot be audited for content, and could be replaced with a malicious version without detection.
- **Risk:** Binaries in version control bypass all code review. An attacker with repository write access could replace it with a trojaned binary. It also inflates clone/fetch sizes permanently.
- **Remediation:** Add `pgn-extract-new` (or `/pgn-extract*`) to `.gitignore`. Remove the binary from the working tree or ensure it is never committed. If distribution is needed, use GitHub Releases.
- **Reference:** CWE-506 (Embedded Malicious Code), SLSA Supply Chain best practices

### IMP-2: Large PGN data files in repository

- **Location:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/cancer0707.pgn`, `g.pgn`, `j.pgn`, `k.pgn`, `l.pgn`, `m.pgn` (2.8 MB each, ~294 MB total)
- **Category:** Repository Hygiene
- **Description:** Six large PGN files appear as untracked files in the diff stats. These are likely test/benchmark data but are not gitignored. If committed, they permanently inflate the repository.
- **Risk:** No direct security risk, but large files in git history cannot be easily removed. They increase attack surface for social engineering (hiding malicious payloads in large binary-like files).
- **Remediation:** Add `*.pgn` to `.gitignore` or place test data in a separate location. If these are needed for CI testing, use Git LFS or download them in CI.

---

## Advisory Findings

### ADV-1: User-controlled format string in SplitWriter

- **Location:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor.go:79`
- **Description:** The `-splitpattern` flag (default `%s_%d.pgn`) is passed directly to `fmt.Sprintf(sw.pattern, sw.baseName, sw.fileNumber)`. A user could provide a pattern with extra format verbs (e.g., `%s_%d_%x.pgn`) which would cause a runtime panic or unexpected output. This is a CLI tool where the user controls their own input, so this is not exploitable by a third party.
- **Remediation:** Validate that the pattern contains exactly one `%s` and one `%d` verb, or use `strings.Replace` instead of `fmt.Sprintf`.
- **Reference:** CWE-134 (Use of Externally-Controlled Format String)

### ADV-2: User-supplied regex compiled without complexity bounds

- **Location:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/matching/tags.go:78`
- **Description:** When a user supplies a tag filter with the `~` operator (regex), the pattern is compiled via `regexp.Compile(value)` without any length or complexity limits. Go's `regexp` package uses RE2 semantics (linear-time matching), which mitigates catastrophic backtracking (ReDoS). However, extremely large or deeply nested patterns could still consume significant memory during compilation.
- **Remediation:** Consider adding a maximum pattern length check (e.g., 1024 characters) before compilation. Low priority since Go's RE2 engine prevents exponential blowup.
- **Reference:** CWE-1333 (Inefficient Regular Expression Complexity)

### ADV-3: MustBoardFromFEN panics on invalid input

- **Location:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/engine/fen.go:300-306`
- **Description:** `MustBoardFromFEN` calls `panic()` on invalid FEN strings. All current call sites pass the constant `InitialFEN` or are in test code, which is safe. The function's docstring correctly states "Use only with known-valid FEN constants." This is well-documented and the naming convention (`Must*`) is idiomatic Go.
- **Remediation:** No action needed; this is a style note. All call sites were verified to use known-valid constants only.
- **Reference:** Go standard library convention (e.g., `regexp.MustCompile`)

### ADV-4: G304 (file inclusion) suppressions are appropriate but numerous

- **Locations:** 8 `//nolint:gosec // G304` annotations across:
  - `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/main.go` (lines 383, 439, 500)
  - `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor.go` (lines 80, 196, 208)
  - `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/matching/filter.go` (line 34)
  - `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/matching/variation.go` (lines 31, 57)
  - `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/eco/eco.go` (line 50)
- **Description:** All G304 suppressions are for a CLI tool opening user-specified files from command-line arguments. This is the correct and expected behavior for a command-line file processing tool. Each suppression includes a justifying comment. No file paths come from network input, environment variables, or untrusted configuration files.
- **Remediation:** No action needed. The suppressions are appropriate and well-documented.

### ADV-5: errcheck exclusions for fmt.Sscanf and strconv.Atoi

- **Location:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/.golangci.yml:66-67`
- **Description:** The linter configuration globally excludes error checking for `fmt.Sscanf` and `strconv.Atoi`. While the current usage sites (FEN clock parsing at `fen.go:206-209`) have acceptable defaults on failure (0), a global exclusion could mask real bugs if these functions are used in new code paths where errors matter.
- **Remediation:** Consider removing the global exclusions and using targeted `//nolint` comments at the specific call sites instead, which was already done for the `gosec` suppressions.

---

## Dependency Status

| Package | Version | Known CVEs | Status |
|---------|---------|-----------|--------|
| (none -- pure Go standard library) | Go 1.23 | N/A | OK |

This project has **zero external dependencies**. The `go.mod` file contains only the module declaration and Go version. This eliminates an entire class of supply chain vulnerabilities.

---

## IaC Status

Not applicable. No Terraform, Ansible, Docker, or other IaC files are present in the changed files.

---

## Docker Security

Not applicable. No Dockerfiles or container configurations are present.

---

## Configuration Security

| Check | Status | Notes |
|-------|--------|-------|
| Debug mode | PASS | No debug flags or verbose error output to users |
| Error messages | PASS | Errors go to stderr, not to output streams |
| CORS/Headers | N/A | CLI tool, not a web service |
| Logging | PASS | Log output goes to user-specified file; no sensitive data logged |

---

## Cross-Task Observations

### Concurrency Safety (Phase 2 + Phase 3)

The `ThreadSafeDuplicateDetector` (Phase 2) correctly wraps the non-thread-safe `DuplicateDetector` with a `sync.RWMutex`. The lock is acquired with `Lock()` for `CheckAndAdd` (write) and `RLock()` for read-only methods. The `LoadFromDetector` method correctly uses `Lock()` and is documented as "call before concurrent use."

The parallel processing model in `outputGamesParallel` (`processor.go:432-502`) is well-designed:
- **Worker goroutines** only call `applyFilters`, which is read-only on shared state.
- **Single consumer goroutine** handles all mutations: `jsonGames` slice appends, `ECOSplitWriter` writes, `SplitWriter` writes, and `handleGameOutput` calls.
- Both `SplitWriter` and `ECOSplitWriter` are correctly documented as "NOT thread-safe: Only accessed from the single result-consumer goroutine."

The `withOutputFile` function (`processor.go:26-31`) temporarily mutates `cfg.OutputFile`, which would be a race condition if called from multiple goroutines. However, it is only called from `outputNonMatchingGame` and `outputDuplicateGame`, both of which execute in the single consumer goroutine. This is safe but fragile -- the invariant is documented only in comments.

### Resource Exhaustion (Phase 3)

The bounded `DuplicateDetector` (maxCapacity) and LRU file handle cache in `ECOSplitWriter` (maxHandles) provide effective protection against resource exhaustion. The `maxHandles` default of 128 is conservative and appropriate for most systems.

### Input Validation Pipeline (Phase 4 + Phase 5)

FEN parsing (`NewBoardFromFEN`) validates:
- Non-empty input
- Valid piece characters
- Board bounds (col > 'h' or rank < '1')
- Valid side-to-move character

Tag matching (`ParseCriterion`) handles edge cases:
- Empty lines and comments are skipped
- Regex compilation errors are returned to caller
- Date parsing validates year range (100-3000)

### Type Assertions (Phase 6)

The comma-ok pattern is correctly used in the critical path:
- `processor.go:239` -- `entry, ok := back.Value.(*lruFileEntry)` in LRU eviction
- `processor.go:488-490` -- `gameInfo, ok := result.GameInfo.(*GameAnalysis)` in parallel processing
- `worker/pool.go:44-49` -- `gi, ok := r.GameInfo.(GameInfo)` in GetGameInfo

### Secrets Scanning

Full scan of all changed files (Go source, YAML, markdown, test files) found:
- **No API keys, tokens, passwords, or connection strings**
- **No private keys or certificates**
- **No base64-encoded credentials**
- **No `.env` files**
- No hardcoded credentials in test fixtures

The `password`/`secret`/`token` grep matches were all false positives (lexer token types like `NextToken`, `EOFToken`).

---

## Conclusion

This project demonstrates strong security practices for a CLI tool:

1. **Zero dependencies** eliminates supply chain risk entirely.
2. **gosec is enabled** with comprehensive rule coverage (G101-G505).
3. **All nolint suppressions are justified** with inline comments explaining why each suppression is appropriate.
4. **Concurrency model is sound** with clear documentation of thread-safety invariants.
5. **Resource bounds are enforced** for both memory (hash table capacity) and file handles (LRU cache).
6. **Input validation is thorough** for FEN parsing, tag matching, and file operations.

The two Important findings (binary and PGN files in repository) are hygiene issues that should be addressed before shipping but do not represent exploitable vulnerabilities in the application code itself.

**Verdict: PASS** -- No critical findings. The codebase is ready to ship after addressing the Important findings.
