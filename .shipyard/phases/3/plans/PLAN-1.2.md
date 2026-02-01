---
phase: memory-management
plan: "1.2"
wave: 1
dependencies: []
must_haves:
  - ECOSplitWriter limits open file handles to configurable maxHandles
  - Default maxHandles 128
  - LRU eviction closes least-recently-used file handle when limit reached
  - Evicted files are reopened in append mode on next access
  - CLI flag -eco-max-handles wired to config
  - Existing ECO split output is byte-identical (append preserves content)
files_touched:
  - cmd/pgn-extract/processor.go
  - cmd/pgn-extract/processor_test.go
  - cmd/pgn-extract/flags.go
  - cmd/pgn-extract/main.go
  - internal/config/output.go
tdd: true
---

# Plan 1.2 -- LRU ECOSplitWriter

## Context

`ECOSplitWriter` opens one `*os.File` per ECO prefix and never closes them until
`Close()` is called. At level 3 (full ECO codes A00-E99), this can mean up to ~500
open file descriptors plus an "unknown" bucket. On systems with low `ulimit -n` (e.g.,
256 on some macOS defaults), this exhausts file descriptors.

The fix is an LRU cache of file handles using `container/list` from the standard library.
When the number of open handles reaches `maxHandles`, the least-recently-used file is
closed. On the next write to that ECO code, the file is reopened in append mode
(`os.O_APPEND|os.O_CREATE|os.O_WRONLY`).

This plan is independent of Plan 1.1 (Bounded DuplicateDetector) and can execute in parallel.

## Tasks

<task id="1" files="cmd/pgn-extract/processor.go" tdd="true">
  <action>
    Refactor `ECOSplitWriter` to add LRU file handle management. Add these fields:

    ```go
    type ECOSplitWriter struct {
        baseName   string
        level      int
        cfg        *config.Config
        maxHandles int                          // max open file descriptors
        files      map[string]*lruFileEntry     // eco prefix -> entry
        lruList    *list.List                    // front = most recent, back = least recent
    }

    type lruFileEntry struct {
        ecoPrefix string
        file      *os.File
        element   *list.Element  // pointer into lruList for O(1) move-to-front
    }
    ```

    Import `container/list` (stdlib only).

    Update `NewECOSplitWriter` to accept `maxHandles int` and initialize the LRU list:
    ```go
    func NewECOSplitWriter(baseName string, level int, cfg *config.Config, maxHandles int) *ECOSplitWriter {
        if maxHandles <= 0 {
            maxHandles = 128
        }
        return &ECOSplitWriter{
            baseName:   baseName,
            level:      level,
            cfg:        cfg,
            maxHandles: maxHandles,
            files:      make(map[string]*lruFileEntry),
            lruList:    list.New(),
        }
    }
    ```

    Rewrite `getOrCreateFile` with LRU logic:
    1. If the entry exists in `files` map and its `file` is not nil, move its element
       to the front of `lruList` and return the file.
    2. If the entry exists but `file` is nil (was evicted), reopen in append mode:
       `os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)`.
       Move to front.
    3. If the entry does not exist, create a new file with `os.Create`.
       Add to map and push to front of `lruList`.
    4. After steps 2 or 3, if `lruList.Len() > maxHandles`, evict from back:
       remove the back element, close its file, set `entry.file = nil` but keep
       the map entry (so we know the filename and can reopen).

    Update `Close()` to iterate the map and close all non-nil file handles.

    Update `FileCount()` to return `len(ew.files)` (total ECO codes seen, not just open handles).

    Add a new method `OpenHandleCount() int` that returns `ew.lruList.Len()`.

    Update the single call site in `cmd/pgn-extract/main.go` where `NewECOSplitWriter`
    is called -- pass the maxHandles value from config (to be wired in Task 3).
    For now, pass `128` as the literal default so compilation succeeds.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go build ./cmd/pgn-extract/</verify>
  <done>ECOSplitWriter uses LRU eviction. Compiles successfully. Old files map replaced with LRU-managed entries.</done>
</task>

<task id="2" files="cmd/pgn-extract/processor_test.go" tdd="true">
  <action>
    Add tests for the LRU ECOSplitWriter behavior:

    1. `TestECOSplitWriter_LRU_EvictsOldestHandle`:
       - Create writer with maxHandles=3, level=3.
       - Write games with ECO codes A00, B00, C00, D00 (4 distinct codes).
       - After writing D00, verify OpenHandleCount() == 3.
       - Verify FileCount() == 4 (all codes tracked).
       - Verify all 4 output files exist on disk and contain valid PGN content.

    2. `TestECOSplitWriter_LRU_ReopensEvictedFile`:
       - Create writer with maxHandles=2, level=3.
       - Write game with ECO A00, then B00 (fills cache), then C00 (evicts A00).
       - Write another game with ECO A00 (reopens in append mode).
       - Verify the A00 file contains both games (2 games worth of content).
       - Verify OpenHandleCount() == 2 (B00 was evicted when A00 reopened after C00).

    3. `TestECOSplitWriter_LRU_UnlimitedWhenHigh`:
       - Create writer with maxHandles=1000, level=3.
       - Write games with 10 distinct ECO codes.
       - Verify OpenHandleCount() == 10 (no eviction needed).

    Helper: create a minimal `*chess.Game` with a specific ECO tag for testing.
    Use `t.TempDir()` for file output to avoid polluting the working directory.
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go test -run "TestECOSplitWriter_LRU" ./cmd/pgn-extract/ -v</verify>
  <done>LRU eviction, reopening in append mode, and no-eviction scenarios all pass. File contents verified on disk.</done>
</task>

<task id="3" files="internal/config/output.go,cmd/pgn-extract/flags.go,cmd/pgn-extract/main.go" tdd="false">
  <action>
    1. In `internal/config/output.go`, add field to OutputConfig:
       ```go
       // ECOMaxHandles limits open file handles for ECO split writing.
       // Default 128. Only relevant when ECO splitting is enabled (-E flag).
       ECOMaxHandles int
       ```
       Update `NewOutputConfig()` to set `ECOMaxHandles: 128`.

    2. In `cmd/pgn-extract/flags.go`, add a new flag near the ECO section:
       ```go
       ecoMaxHandles = flag.Int("eco-max-handles", 128,
           "Max open file handles for ECO split output (default 128)")
       ```

    3. In `cmd/pgn-extract/flags.go`, in `applyFlags` or the appropriate sub-function,
       wire: `cfg.Output.ECOMaxHandles = *ecoMaxHandles`

    4. In `cmd/pgn-extract/main.go`, update the `NewECOSplitWriter` call to pass
       `cfg.Output.ECOMaxHandles` instead of the hardcoded `128` from Task 1.

    5. Run the full test suite: `go test ./... && go vet ./...`
  </action>
  <verify>cd /Users/lgbarn/Personal/Chess/pgn-extract-go && go build ./cmd/pgn-extract/ && go test ./... && go vet ./...</verify>
  <done>CLI flag -eco-max-handles wired through config to ECOSplitWriter. Default 128. Full test suite passes.</done>
</task>

## Verification

```bash
cd /Users/lgbarn/Personal/Chess/pgn-extract-go

# Unit tests
go test -run "TestECOSplitWriter" ./cmd/pgn-extract/ -v

# Full suite with race detector
go test -race ./...

# Verify flag exists
go run ./cmd/pgn-extract/ -h 2>&1 | grep eco-max-handles

# Manual smoke test: ECO split with low handle limit
# (requires an ECO file and PGN input)
go run ./cmd/pgn-extract/ -e eco.pgn -E 3 -eco-max-handles 4 input.pgn
```
