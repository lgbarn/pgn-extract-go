# Plan 1.1: Bump Go Version to 1.23

## Context
Update the project's minimum Go version from 1.21 to 1.23. This is a foundational change that unblocks use of Go 1.22/1.23 language features in later phases. The CI build matrix currently tests Go 1.21, 1.22, and 1.23 — it needs updating to reflect the new minimum.

## Dependencies
None

## Tasks

### Task 1: Update go.mod and CI configuration
**Files:** `go.mod`, `.github/workflows/ci.yml`
**Action:** modify
**Description:**
1. Edit `go.mod` to change `go 1.21` to `go 1.23`
2. Run `go mod tidy` to regenerate module metadata
3. Update `.github/workflows/ci.yml` build matrix:
   - Remove Go `"1.21"` and `"1.22"` from `matrix.go`
   - Keep `"1.23"` as the only version (or add `"1.24"` if forward-looking)
   - Remove the Windows exclusion entries for 1.21 and 1.22 (no longer needed since those versions are dropped)
   - The `env.GO_VERSION: "1.23"` stays as-is

**Acceptance Criteria:**
- `go.mod` contains `go 1.23`
- `go mod tidy` exits cleanly with no errors
- CI build matrix only tests Go 1.23+
- Windows exclusion rules are cleaned up

### Task 2: Verify all tests and checks pass
**Files:** none (verification only)
**Action:** test
**Description:**
1. Run `go vet ./...` to verify no new issues
2. Run `go test ./...` to verify all existing tests pass
3. Run `go test -race ./...` to verify race detector still works (baseline)

**Acceptance Criteria:**
- `go vet ./...` reports zero issues
- `go test ./...` passes with zero failures
- `go test -race ./...` passes (races may exist — this is just a baseline, Phase 2 will fix them)

## Verification
```bash
grep 'go 1.23' go.mod
go mod tidy
go vet ./...
go test ./...
```
