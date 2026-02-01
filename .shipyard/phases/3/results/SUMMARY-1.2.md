# Plan 1.2 Summary: LRU ECOSplitWriter

**Status:** ✅ Complete
**Branch:** main
**Date:** 2026-01-31

## Overview

Successfully implemented LRU (Least Recently Used) file handle caching for `ECOSplitWriter` to prevent file descriptor exhaustion when splitting output by ECO codes. At level 3 (A00-E99), this feature can create up to 500 files, which previously meant 500 open file handles. The LRU cache limits open handles to a configurable maximum (default 128), automatically closing and reopening files as needed.

## Tasks Completed

### Task 1: Refactor ECOSplitWriter with LRU ✅
**Files Modified:**
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor.go`
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/main.go`

**Changes:**
1. Added `container/list` import for stdlib LRU implementation
2. Created `lruFileEntry` struct to track ECO prefix, file handle, and LRU list element
3. Refactored `ECOSplitWriter` structure:
   - Changed `files` from `map[string]*os.File` to `map[string]*lruFileEntry`
   - Added `lruList *list.List` for LRU tracking
   - Added `maxHandles int` field
4. Updated `NewECOSplitWriter` to accept `maxHandles int` parameter (defaults to 128 if ≤ 0)
5. Rewrote `getOrCreateFile` with LRU logic:
   - If entry exists and file is open: move to front of LRU list, return file
   - If entry exists but file is nil (evicted): reopen in append mode, move to front
   - If new entry: create file, add to front of LRU list
   - After any open/create: call `evictIfNeeded()` to enforce maxHandles limit
6. Added `evictIfNeeded()` method that closes least recently used file when limit exceeded
7. Updated `Close()` to iterate over entries and close all non-nil files
8. Updated `FileCount()` to return `len(ew.files)` (total files created)
9. Added `OpenHandleCount()` method returning `ew.lruList.Len()` (currently open handles)
10. Updated `NewECOSplitWriter` call in main.go to pass 128 as initial literal value

**Commit:** `db4b7fb` - shipyard(phase-3): add LRU file handle cache to ECOSplitWriter

### Task 2: Add LRU tests ✅
**Files Modified:**
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor_test.go`

**Changes:**
1. Added imports for `os`, `path/filepath`, `strings`, and `config`
2. Created `makeMinimalGame(eco string)` helper function to generate test games with ECO tags
3. Added three comprehensive test functions:
   - `TestECOSplitWriter_LRU_EvictsOldestHandle`: Verifies eviction behavior
     - maxHandles=3, writes 4 ECO codes
     - Confirms OpenHandleCount()==3, FileCount()==4
     - Verifies all 4 files exist on disk
   - `TestECOSplitWriter_LRU_ReopensEvictedFile`: Verifies append-mode reopening
     - maxHandles=2, writes A00→B00→C00 (evicts A00)
     - Writes A00 again
     - Confirms A00 file contains both games
   - `TestECOSplitWriter_LRU_UnlimitedWhenHigh`: Verifies no eviction with high limit
     - maxHandles=1000, writes 10 codes
     - Confirms OpenHandleCount()==10 (no eviction)

All tests use `t.TempDir()` for isolation and pass cleanly.

**Commit:** `b9f0e05` - shipyard(phase-3): add LRU ECOSplitWriter tests

### Task 3: Wire CLI flag and config ✅
**Files Modified:**
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/config/output.go`
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/flags.go`
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/main.go`
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor_test.go`

**Changes:**
1. Added `ECOMaxHandles int` field to `OutputConfig` struct
2. Updated `NewOutputConfig()` to set `ECOMaxHandles: 128` as default
3. Added `-eco-max-handles` CLI flag with default value 128
4. Updated `applyContentFlags()` to wire flag value: `cfg.Output.ECOMaxHandles = *ecoMaxHandles`
5. Updated `NewECOSplitWriter` call in main.go to pass `cfg.Output.ECOMaxHandles`
6. Updated test calls to `NewThreadSafeDuplicateDetector` to include `maxCapacity` parameter (0 for unbounded) - required for compatibility with Plan 1.1

**Commit:** `a4ee37f` - shipyard(phase-3): wire -eco-max-handles CLI flag

## Verification

All verification steps passed successfully:

### Build Verification
```bash
go build ./cmd/pgn-extract/
# Build successful with no errors
```

### Test Suite
```bash
go test ./...
# All packages pass:
# - cmd/pgn-extract (including new LRU tests)
# - All internal packages
# Total: 14/14 packages passed
```

### Vet Check
```bash
go vet ./...
# No issues reported
```

### Specific LRU Tests
```bash
go test -run "TestECOSplitWriter_LRU" ./cmd/pgn-extract/ -v
# === RUN   TestECOSplitWriter_LRU_EvictsOldestHandle
# --- PASS: TestECOSplitWriter_LRU_EvictsOldestHandle (0.00s)
# === RUN   TestECOSplitWriter_LRU_ReopensEvictedFile
# --- PASS: TestECOSplitWriter_LRU_ReopensEvictedFile (0.00s)
# === RUN   TestECOSplitWriter_LRU_UnlimitedWhenHigh
# --- PASS: TestECOSplitWriter_LRU_UnlimitedWhenHigh (0.00s)
# PASS
```

## Deviations from Plan

### Minor Deviations
1. **Plan 1.1 Integration**: Plan 1.2 was executed in parallel with Plan 1.1 (Bounded DuplicateDetector). Plan 1.1 modified the signature of `NewThreadSafeDuplicateDetector` to add a `maxCapacity int` parameter. This required updating test calls in `processor_test.go` to pass `0` (unbounded) as the second parameter. This was a necessary integration adjustment and did not affect the core LRU functionality.

2. **File modifications**: During execution, main.go was automatically updated by Plan 1.1's changes (detected by linter/formatter). The necessary integration was seamless.

### No Other Deviations
All other aspects of the plan were followed exactly:
- Used `container/list` from stdlib (no external dependencies)
- Implemented exact LRU eviction logic as specified
- Created all three required test cases
- Wired config through OutputConfig as planned
- Used default value of 128 for maxHandles

## Integration with Plan 1.1

This plan ran in parallel with Plan 1.1 (Bounded DuplicateDetector). The integration points were:
- Both plans touched `flags.go` but in different sections (no conflicts)
- Both plans touched `main.go` but in different sections (no conflicts)
- Plan 1.1 added `maxCapacity` parameter to `NewThreadSafeDuplicateDetector`, requiring updates to test calls

The parallel execution was successful with no merge conflicts or functional issues.

## Technical Implementation Details

### LRU Cache Strategy
The implementation uses `container/list` (stdlib) for O(1) move-to-front and eviction operations:
- **Map**: Stores all entries (both open and evicted) for quick lookup
- **List**: Tracks only open handles in LRU order (front = most recent, back = least recent)
- **Eviction**: When `lruList.Len() > maxHandles`, removes from back, closes file, sets entry.file = nil
- **Reopening**: When accessing evicted entry, reopens in append mode (`os.O_APPEND|os.O_CREATE|os.O_WRONLY`)

### Memory Efficiency
- Map entries are never deleted (small metadata overhead for ~500 ECO codes)
- Only open file handles consume significant memory
- LRU list only contains open handles, keeping memory proportional to maxHandles

### File Handle Management
- Default limit: 128 open handles (configurable via `-eco-max-handles`)
- Automatic eviction prevents hitting OS file descriptor limits
- Files reopened transparently when needed
- All files properly closed on `Close()`

## Files Modified

1. `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor.go`
   - Added LRU cache infrastructure
   - Rewrote file handle management logic

2. `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor_test.go`
   - Added comprehensive LRU test suite
   - Updated test compatibility with Plan 1.1

3. `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/config/output.go`
   - Added ECOMaxHandles configuration field

4. `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/flags.go`
   - Added -eco-max-handles CLI flag
   - Wired flag to config

5. `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/main.go`
   - Updated NewECOSplitWriter call to use config value

## Commits

1. `db4b7fb` - shipyard(phase-3): add LRU file handle cache to ECOSplitWriter
2. `b9f0e05` - shipyard(phase-3): add LRU ECOSplitWriter tests
3. `a4ee37f` - shipyard(phase-3): wire -eco-max-handles CLI flag

## Conclusion

Plan 1.2 successfully implemented LRU file handle caching for ECOSplitWriter, solving the file descriptor exhaustion problem when splitting output by full ECO codes (A00-E99). The implementation:

- Uses stdlib `container/list` for efficient LRU operations
- Defaults to 128 open handles (configurable)
- Automatically evicts least recently used files
- Transparently reopens files in append mode
- Includes comprehensive test coverage
- Integrates cleanly with Plan 1.1 changes

All verification steps passed, and the feature is production-ready.
