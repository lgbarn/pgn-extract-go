# Phase 2: Concurrency Safety Fixes - Research Document

**Date:** 2026-01-31
**Status:** Research Complete
**Working Directory:** `/Users/lgbarn/Personal/Chess/pgn-extract-go`

---

## Executive Summary

This research identifies **data race conditions** in the parallel game processing pipeline, specifically around the `DuplicateDetector`, `ECOSplitWriter`, and other shared mutable state. While `go test -race ./...` currently **passes** (no races detected in existing tests), the code has **latent race conditions** that could manifest under high concurrency or stress testing. A thread-safe `ThreadSafeDuplicateDetector` already exists but is **not used** in the parallel path.

### Critical Finding

The non-thread-safe `DuplicateDetector` is shared via `ProcessingContext` and accessed from the **main result-processing goroutine** in `outputGamesParallel`. Although this is currently a **single goroutine**, the architecture makes it fragile and prevents future parallelization of the output phase.

---

## 1. DuplicateDetector Usage in Parallel Paths

### Current Implementation

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor.go`

#### ProcessingContext (Line 32-43)

```go
type ProcessingContext struct {
	cfg              *config.Config
	detector         *hashing.DuplicateDetector  // ⚠️ NON-THREAD-SAFE
	setupDetector    *hashing.SetupDuplicateDetector
	ecoClassifier    *eco.ECOClassifier
	gameFilter       *matching.GameFilter
	cqlNode          cql.Node
	variationMatcher *matching.VariationMatcher
	materialMatcher  *matching.MaterialMatcher
	ecoSplitWriter   *ECOSplitWriter
}
```

#### Detector Created in main.go (Line 192-222)

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/main.go`

```go
func setupDuplicateDetector(cfg *config.Config) *hashing.DuplicateDetector {
	if !*suppressDuplicates && *duplicateFile == "" && !*outputDupsOnly && *checkFile == "" {
		return nil
	}

	detector := hashing.NewDuplicateDetector(false)  // ⚠️ Creates non-thread-safe detector
	cfg.Duplicate.Suppress = *suppressDuplicates
	cfg.Duplicate.SuppressOriginals = *outputDupsOnly

	// Load check file for duplicate detection
	if *checkFile != "" {
		file, err := os.Open(*checkFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening check file %s: %v\n", *checkFile, err)
			os.Exit(1)
		}
		defer file.Close()

		checkGames := processInput(file, *checkFile, cfg)
		for _, game := range checkGames {
			board := replayGame(game)
			detector.CheckAndAdd(game, board)  // Sequential pre-loading is fine
		}
	}

	return detector
}
```

#### Usage in outputGamesParallel (Line 345-412)

```go
func outputGamesParallel(games []*chess.Game, ctx *ProcessingContext, numWorkers int) (int, int) {
	cfg := ctx.cfg
	outputCount := int64(0)
	duplicateCount := int64(0)

	// Worker pool processes games in parallel
	processFunc := func(item worker.WorkItem) worker.ProcessResult {
		return processGameWorker(item, ctx)  // Workers do NOT access detector
	}

	pool := worker.NewPool(numWorkers, bufferSize, processFunc)
	pool.Start()

	// Submit work (goroutine)
	go func() {
		for i, game := range games {
			pool.Submit(worker.WorkItem{Game: game, Index: i})
		}
		pool.Close()
	}()

	var jsonGames []*chess.Game

	// ⚠️ RACE CONDITION POTENTIAL: Main goroutine processes results
	for result := range pool.Results() {
		// ...
		gameInfo, _ := result.GameInfo.(*GameAnalysis)
		out, dup := handleGameOutput(result.Game, result.Board, gameInfo, ctx, &jsonGames)
		//              ^^^^^^^^^^^^^^^^^^
		//              Calls detector.CheckAndAdd inside
		atomic.AddInt64(&outputCount, int64(out))
		atomic.AddInt64(&duplicateCount, int64(dup))
	}

	return int(atomic.LoadInt64(&outputCount)), int(atomic.LoadInt64(&duplicateCount))
}
```

#### handleGameOutput Calls Non-Thread-Safe Detector (Line 290-324)

```go
func handleGameOutput(game *chess.Game, board *chess.Board, gameInfo *GameAnalysis,
                      ctx *ProcessingContext, jsonGames *[]*chess.Game) (int, int) {
	cfg := ctx.cfg
	detector := ctx.detector

	if detector == nil {
		outputGameWithECOSplit(game, cfg, gameInfo, jsonGames, ctx.ecoSplitWriter)
		atomic.AddInt64(&matchedCount, 1)
		return 1, 0
	}

	if board == nil {
		board = replayGame(game)
	}

	isDuplicate := detector.CheckAndAdd(game, board)  // ⚠️ RACE: Non-thread-safe map access
	//                      ^^^^^^^^^^^^
	//                      Called from main goroutine in parallel mode
	//                      BUT still touches shared mutable map

	if isDuplicate {
		outputDuplicateGame(game, cfg)
		if cfg.Duplicate.SuppressOriginals {
			outputGameWithECOSplit(game, cfg, gameInfo, jsonGames, ctx.ecoSplitWriter)
			atomic.AddInt64(&matchedCount, 1)
			return 1, 1
		}
		return 0, 1
	}

	if shouldOutputUnique(cfg) {
		outputGameWithECOSplit(game, cfg, gameInfo, jsonGames, ctx.ecoSplitWriter)
		atomic.AddInt64(&matchedCount, 1)
		return 1, 0
	}

	return 0, 0
}
```

### Issue Analysis

**Current Behavior:**
- Workers process games in parallel (CPU-intensive work: replay, filter, analyze)
- Results are collected in **main goroutine** which calls `handleGameOutput`
- `handleGameOutput` calls `detector.CheckAndAdd(game, board)` on the **non-thread-safe** detector

**Why No Race Detected Yet:**
- All `CheckAndAdd` calls happen in the **same goroutine** (the result consumer)
- Tests don't stress-test with enough parallelism or duplicate detection
- Current tests pass because there's no actual concurrent access *yet*

**Why This Is Still a Problem:**
1. **Architectural fragility:** If we ever parallelize the output phase (multiple output consumers), instant race condition
2. **Future-proofing:** ThreadSafeDuplicateDetector already exists but isn't used
3. **Principle violation:** Sharing non-thread-safe state in a parallel architecture is a footgun

---

## 2. Thread-Safe Detector Implementation

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/thread_safe.go`

### ThreadSafeDuplicateDetector (Line 10-51)

```go
// ThreadSafeDuplicateDetector wraps DuplicateDetector with mutex protection for concurrent access.
type ThreadSafeDuplicateDetector struct {
	detector *DuplicateDetector
	mu       sync.RWMutex
}

// NewThreadSafeDuplicateDetector creates a new thread-safe detector.
func NewThreadSafeDuplicateDetector(exactMatch bool) *ThreadSafeDuplicateDetector {
	return &ThreadSafeDuplicateDetector{
		detector: NewDuplicateDetector(exactMatch),
	}
}

// CheckAndAdd atomically checks if a game is a duplicate and adds it to the hash table.
func (d *ThreadSafeDuplicateDetector) CheckAndAdd(game *chess.Game, board *chess.Board) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.detector.CheckAndAdd(game, board)
}

// DuplicateCount returns the number of duplicates detected.
func (d *ThreadSafeDuplicateDetector) DuplicateCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.detector.DuplicateCount()
}

// UniqueCount returns the number of unique games.
func (d *ThreadSafeDuplicateDetector) UniqueCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.detector.UniqueCount()
}

// LoadFromDetector copies entries from an existing detector. Call before concurrent use.
func (d *ThreadSafeDuplicateDetector) LoadFromDetector(other *DuplicateDetector) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for hash, sigs := range other.hashTable {
		d.detector.hashTable[hash] = append(d.detector.hashTable[hash], sigs...)
	}
}
```

### Key Features

1. **Mutex Protection:** All methods use `sync.RWMutex` for safe concurrent access
2. **LoadFromDetector:** Supports pre-loading from a non-thread-safe detector (for `-c` checkfile)
3. **Same Interface:** Drop-in replacement for `DuplicateDetector`

---

## 3. DuplicateDetector Internal Structure

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/hashing.go`

### DuplicateDetector (Line 8-28)

```go
type DuplicateDetector struct {
	hashTable      map[uint64][]GameSignature  // ⚠️ Non-thread-safe map
	useExactMatch  bool
	duplicateCount int
}

type GameSignature struct {
	Hash      uint64
	MoveCount int
	WeakHash  chess.HashCode
}

func NewDuplicateDetector(exactMatch bool) *DuplicateDetector {
	return &DuplicateDetector{
		hashTable:     make(map[uint64][]GameSignature),
		useExactMatch: exactMatch,
	}
}
```

### CheckAndAdd Method (Line 30-60)

```go
func (d *DuplicateDetector) CheckAndAdd(game *chess.Game, board *chess.Board) bool {
	if board == nil {
		return false
	}

	hash := GenerateZobristHash(board)
	weakHash := WeakHash(board)
	moveCount := countMoves(game)

	sig := GameSignature{
		Hash:      hash,
		MoveCount: moveCount,
		WeakHash:  weakHash,
	}

	// Check for duplicates
	if existing, ok := d.hashTable[hash]; ok {  // ⚠️ Read from map
		for _, existingSig := range existing {
			if d.signaturesMatch(sig, existingSig) {
				d.duplicateCount++
				return true
			}
		}
	}

	// Add to hash table
	d.hashTable[hash] = append(d.hashTable[hash], sig)  // ⚠️ Write to map
	return false
}
```

**Race Hazard:**
- Map reads/writes without synchronization
- `duplicateCount` increment is not atomic

---

## 4. ECOSplitWriter Thread Safety

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor.go`

### ECOSplitWriter (Line 100-193)

```go
type ECOSplitWriter struct {
	baseName string
	level    int
	files    map[string]*os.File  // ⚠️ Shared mutable map
	cfg      *config.Config
}

func (ew *ECOSplitWriter) WriteGame(game *chess.Game) error {
	ecoCode := ew.getECOPrefix(game)
	file, err := ew.getOrCreateFile(ecoCode)  // ⚠️ May modify files map
	if err != nil {
		return err
	}

	// Temporarily redirect output to this file
	originalOutput := ew.cfg.OutputFile
	ew.cfg.OutputFile = file
	output.OutputGame(game, ew.cfg)
	ew.cfg.OutputFile = originalOutput

	return nil
}

func (ew *ECOSplitWriter) getOrCreateFile(ecoPrefix string) (*os.File, error) {
	if file, ok := ew.files[ecoPrefix]; ok {  // ⚠️ Map read
		return file, nil
	}

	filename := fmt.Sprintf("%s_%s.pgn", ew.baseName, ecoPrefix)
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	ew.files[ecoPrefix] = file  // ⚠️ Map write
	return file, nil
}
```

### Current Usage in outputGamesParallel

**Line 402:**
```go
out, dup := handleGameOutput(result.Game, result.Board, gameInfo, ctx, &jsonGames)
```

**handleGameOutput → outputGameWithECOSplit (Line 435-456):**
```go
func outputGameWithECOSplit(game *chess.Game, cfg *config.Config, gameInfo *GameAnalysis,
                            jsonGames *[]*chess.Game, ecoWriter *ECOSplitWriter) {
	// Handle split writer
	if sw, ok := cfg.OutputFile.(*SplitWriter); ok {
		defer sw.IncrementGameCount()
	}

	if cfg.Output.JSONFormat {
		*jsonGames = append(*jsonGames, game)  // ⚠️ Shared slice mutation
		return
	}

	// If ECO split writer is configured, use it
	if ecoWriter != nil {
		if err := ecoWriter.WriteGame(game); err != nil {  // ⚠️ Map mutation
			fmt.Fprintf(os.Stderr, "Error writing game to ECO file: %v\n", err)
		}
		return
	}

	output.OutputGame(game, cfg)
}
```

### Assessment

**Current Status:**
- `ECOSplitWriter` is accessed **only from the main result-processing goroutine**
- No race condition *yet* because single consumer

**Problem:**
1. **Fragile design:** Shared map without synchronization in a parallel context
2. **File handle safety:** `files` map grows dynamically, not safe for concurrent modification
3. **Config mutation:** `ew.cfg.OutputFile` is mutated temporarily (though restored)

**Recommendation:**
- Add `sync.Mutex` to protect `files` map operations
- OR ensure ECOSplitWriter is only used from single goroutine (document this requirement)

---

## 5. Other Shared Mutable State

### jsonGames Slice (processor.go:379, 402, 443)

**Line 379:**
```go
var jsonGames []*chess.Game
```

**Line 402 (in outputGamesParallel):**
```go
out, dup := handleGameOutput(result.Game, result.Board, gameInfo, ctx, &jsonGames)
```

**Line 443 (in outputGameWithECOSplit):**
```go
*jsonGames = append(*jsonGames, game)  // ⚠️ Slice append from result consumer goroutine
```

**Assessment:**
- Currently safe because **single consumer goroutine**
- If output phase is parallelized, this becomes a race condition

### SplitWriter (processor.go:45-98)

```go
type SplitWriter struct {
	baseName     string
	pattern      string
	gamesPerFile int
	currentFile  *os.File  // ⚠️ Mutable state
	fileNumber   int       // ⚠️ Mutable counter
	gameCount    int       // ⚠️ Mutable counter
}

func (sw *SplitWriter) Write(p []byte) (n int, err error) {
	if sw.currentFile == nil || sw.gameCount >= sw.gamesPerFile {
		if sw.currentFile != nil {
			sw.currentFile.Close()
			sw.fileNumber++  // ⚠️ Not atomic
		}
		filename := fmt.Sprintf(sw.pattern, sw.baseName, sw.fileNumber)
		sw.currentFile, err = os.Create(filename)
		if err != nil {
			return 0, err
		}
		sw.gameCount = 0
	}
	return sw.currentFile.Write(p)
}

func (sw *SplitWriter) IncrementGameCount() {
	sw.gameCount++  // ⚠️ Not atomic
}
```

**Assessment:**
- Used as `cfg.OutputFile` in output phase
- Currently accessed only from single goroutine
- Would require mutex if parallelized

---

## 6. Atomic Counter Usage Audit

### Global Atomic Counters (filters.go:436-439)

```go
var matchedCount int64
var gamePositionCounter int64

func IncrementMatchedCount() int64 {
	return atomic.AddInt64(&matchedCount, 1)
}

func GetMatchedCount() int64 {
	return atomic.LoadInt64(&matchedCount)
}

func IncrementGamePosition() int64 {
	return atomic.AddInt64(&gamePositionCounter, 1)
}
```

### Usage Patterns

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor.go`

#### Correct Atomic Usage (Line 233, 258, 296, 310, 319, 364, 382, 393-394, 403-404)

```go
// Sequential path
if *stopAfter > 0 && atomic.LoadInt64(&matchedCount) >= int64(*stopAfter) {
	break
}
atomic.AddInt64(&matchedCount, 1)

// Parallel path
outputCount := int64(0)
duplicateCount := int64(0)
// ...
atomic.AddInt64(&outputCount, int64(out))
atomic.AddInt64(&duplicateCount, int64(dup))
// ...
return int(atomic.LoadInt64(&outputCount)), int(atomic.LoadInt64(&duplicateCount))
```

**Assessment:** ✅ Atomic operations are used correctly for all shared counters.

---

## 7. No Interface for Detectors

### Finding

Searched for interfaces that both `DuplicateDetector` and `ThreadSafeDuplicateDetector` implement:

```bash
grep -r "type.*interface" --include="*.go" | grep -i detect
# No results
```

**Current Situation:**
- No shared interface between `DuplicateDetector` and `ThreadSafeDuplicateDetector`
- `ProcessingContext.detector` is **hardcoded** to `*hashing.DuplicateDetector`
- Cannot swap in `ThreadSafeDuplicateDetector` without changing the type

**Recommendation:**
- **Option A:** Create a `Detector` interface with `CheckAndAdd`, `DuplicateCount`, `UniqueCount` methods
- **Option B:** Change `ProcessingContext.detector` to `*hashing.ThreadSafeDuplicateDetector` (simpler)

---

## 8. Test Infrastructure

### Existing Parallel Tests

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/parallel_test.go`

Tests include:
- `TestParallelMatchesSequential` (Line 28-56): Verifies parallel produces same games as sequential
- `TestParallelDuplicateDetection` (Line 95-118): Tests duplicate detection with `--workers 4`
- `TestParallelWithECO` (Line 167-190): ECO classification with parallel workers
- `TestParallelMultipleFiles` (Line 210-234): Multiple input files
- `TestParallelWithValidation` (Line 236-252): Validation mode

### Thread-Safe Detector Tests

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/thread_safe_test.go`

```go
func TestThreadSafeDuplicateDetector_Concurrent(t *testing.T) {
	detector := NewThreadSafeDuplicateDetector(false)
	const numGames = 100
	const numWorkers = 10

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			start := workerID * gamesPerWorker
			end := start + gamesPerWorker
			for j := start; j < end; j++ {
				detector.CheckAndAdd(games[j], boards[j])
			}
		}(i)
	}
	wg.Wait()

	if detector.DuplicateCount() != 99 {
		t.Errorf("Expected 99 duplicates, got %d", detector.DuplicateCount())
	}
}

func TestThreadSafeDuplicateDetector_NoRace(t *testing.T) {
	detector := NewThreadSafeDuplicateDetector(false)
	// ... 100 goroutines concurrently calling CheckAndAdd, DuplicateCount, UniqueCount
}
```

**Assessment:**
- ✅ ThreadSafeDuplicateDetector has comprehensive concurrency tests
- ✅ Tests pass with `-race` flag
- ⚠️ Integration tests don't stress-test actual parallel duplicate detection with high worker count

### Race Detector Status

```bash
go test -race ./...
# All tests pass, no races detected (as of 2026-01-31)
```

**Why?**
- Current architecture has **single consumer** goroutine accessing detector
- No actual concurrent `CheckAndAdd` calls in production code
- Tests use `--workers 4` but still have single output consumer

---

## 9. Assessment and Recommendations

### Current Race Condition Summary

| Component | Thread-Safe? | Current Usage | Risk Level | Fix Required |
|-----------|--------------|---------------|------------|--------------|
| `DuplicateDetector` | ❌ No | Single consumer goroutine | 🟡 Medium | Yes - swap to `ThreadSafeDuplicateDetector` |
| `ECOSplitWriter.files` map | ❌ No | Single consumer goroutine | 🟡 Medium | Yes - add mutex or document single-consumer requirement |
| `jsonGames` slice | ❌ No | Single consumer goroutine | 🟡 Medium | No - but document requirement |
| `SplitWriter` counters | ❌ No | Single consumer goroutine | 🟡 Medium | No - but document requirement |
| `matchedCount` | ✅ Yes | `atomic.AddInt64` | 🟢 Low | No |
| `gamePositionCounter` | ✅ Yes | `atomic.AddInt64` | 🟢 Low | No |
| `outputCount` (local) | ✅ Yes | `atomic.AddInt64` | 🟢 Low | No |
| `duplicateCount` (local) | ✅ Yes | `atomic.AddInt64` | 🟢 Low | No |

### Why Tests Pass

Current tests pass `-race` because:
1. **Worker pool processes games in parallel** (CPU work)
2. **Single consumer goroutine** collects results and handles output
3. No actual concurrent access to detector/ECOSplitWriter

### Fix Approach Recommendation

**Priority 1: Swap in ThreadSafeDuplicateDetector**

1. **Create interface (optional but recommended):**
   ```go
   type GameDuplicateDetector interface {
       CheckAndAdd(game *chess.Game, board *chess.Board) bool
       DuplicateCount() int
       UniqueCount() int
   }
   ```

2. **Update ProcessingContext:**
   ```go
   type ProcessingContext struct {
       detector *hashing.ThreadSafeDuplicateDetector  // Or interface
       // ... rest unchanged
   }
   ```

3. **Update setupDuplicateDetector in main.go:**
   ```go
   func setupDuplicateDetector(cfg *config.Config) *hashing.ThreadSafeDuplicateDetector {
       if !*suppressDuplicates && *duplicateFile == "" && !*outputDupsOnly && *checkFile == "" {
           return nil
       }

       detector := hashing.NewThreadSafeDuplicateDetector(false)
       cfg.Duplicate.Suppress = *suppressDuplicates
       cfg.Duplicate.SuppressOriginals = *outputDupsOnly

       if *checkFile != "" {
           // Load check file games into a temporary non-thread-safe detector
           tempDetector := hashing.NewDuplicateDetector(false)
           // ... populate tempDetector ...
           // Load into thread-safe detector
           detector.LoadFromDetector(tempDetector)
       }

       return detector
   }
   ```

4. **No changes needed in processor.go** — `CheckAndAdd` signature is identical

**Priority 2: ECOSplitWriter Thread Safety**

**Option A:** Add mutex protection
```go
type ECOSplitWriter struct {
	baseName string
	level    int
	files    map[string]*os.File
	cfg      *config.Config
	mu       sync.Mutex  // Add mutex
}

func (ew *ECOSplitWriter) getOrCreateFile(ecoPrefix string) (*os.File, error) {
	ew.mu.Lock()
	defer ew.mu.Unlock()

	if file, ok := ew.files[ecoPrefix]; ok {
		return file, nil
	}
	// ... create file ...
	ew.files[ecoPrefix] = file
	return file, nil
}
```

**Option B:** Document single-consumer requirement (simpler, acceptable for current architecture)
```go
// ECOSplitWriter writes games to different files based on ECO code.
// IMPORTANT: This type is NOT thread-safe. It must only be accessed from a single goroutine.
type ECOSplitWriter struct {
	// ...
}
```

**Recommendation:** Choose **Option B** for now (document requirement), add mutex later if output phase is parallelized.

**Priority 3: Audit and Document Other Shared State**

Add comments to document thread-safety requirements:
- `jsonGames` slice append: single consumer only
- `SplitWriter`: single consumer only

---

## 10. Implementation Checklist

### Phase 2 Fix Plan

- [ ] Create `GameDuplicateDetector` interface (optional)
- [ ] Update `ProcessingContext.detector` type to `*ThreadSafeDuplicateDetector`
- [ ] Update `setupDuplicateDetector` to return `*ThreadSafeDuplicateDetector`
- [ ] Handle check file loading with `LoadFromDetector`
- [ ] Update `reportStatistics` function signature if needed
- [ ] Add thread-safety comments to `ECOSplitWriter`
- [ ] Add thread-safety comments to `SplitWriter`
- [ ] Add thread-safety comments to `jsonGames` usage
- [ ] Run `go test -race ./...` to verify
- [ ] Add stress test with high worker count and duplicate detection

### Verification Steps

```bash
# 1. Run race detector
go test -race ./...

# 2. Run parallel tests specifically
go test -race -v ./cmd/pgn-extract -run Parallel

# 3. Test duplicate detection with high parallelism
pgn-extract -D --workers 8 large-file.pgn large-file.pgn

# 4. Test ECO split with parallelism
pgn-extract --ecosplit 3 --workers 8 -e eco.pgn large-file.pgn
```

---

## 11. Relevant File Paths

### Core Implementation Files

- `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/main.go` (lines 192-222: `setupDuplicateDetector`)
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor.go` (lines 32-43: `ProcessingContext`, 290-324: `handleGameOutput`, 345-412: `outputGamesParallel`)
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/hashing.go` (lines 8-83: `DuplicateDetector`)
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/thread_safe.go` (lines 10-51: `ThreadSafeDuplicateDetector`)

### Test Files

- `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/parallel_test.go` (parallel processing tests)
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/thread_safe_test.go` (thread-safe detector tests)

### Supporting Files

- `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/filters.go` (lines 436-453: atomic counter functions)
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/flags.go` (line 139: `workers` flag definition)
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/worker/pool.go` (worker pool implementation)

---

## 12. Conclusion

The pgn-extract-go project has **latent race conditions** around the `DuplicateDetector` that don't manifest in current tests because the output phase uses a **single consumer goroutine**. However, best practices dictate using thread-safe primitives in concurrent contexts.

**Key Actions:**
1. ✅ Swap `DuplicateDetector` → `ThreadSafeDuplicateDetector` (thread-safe wrapper already exists)
2. ✅ Document single-consumer requirements for `ECOSplitWriter`, `SplitWriter`, `jsonGames`
3. ✅ Verify with `go test -race ./...`
4. ✅ Add stress test for high-concurrency duplicate detection

This phase eliminates architectural fragility and ensures `go test -race ./...` will continue to pass even as the codebase evolves.
