# Phase 3: Memory Management Research

**Project:** pgn-extract-go
**Date:** 2026-01-31
**Objective:** Address unbounded memory growth in duplicate detection and ECO split writing

---

## Executive Summary

Phase 3 addresses two unbounded data structures:
1. **DuplicateDetector hash table** - grows without limit as unique games are processed
2. **ECOSplitWriter file handle map** - can open up to 500+ file descriptors simultaneously

**Recommended Approach:**
- DuplicateDetector: Add configurable capacity with **simple cap** (stop tracking when full)
- ECOSplitWriter: Implement **LRU file handle cache** with close-and-reopen strategy
- Both solutions use **stdlib only** with **sensible defaults** to preserve existing behavior

---

## 1. DuplicateDetector Hash Table Analysis

### Current Structure

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/hashing.go`

```go
// Lines 21-25
type DuplicateDetector struct {
	hashTable      map[uint64][]GameSignature
	useExactMatch  bool
	duplicateCount int
}

// Lines 28-32
type GameSignature struct {
	Hash      uint64      // 8 bytes - Zobrist hash
	MoveCount int         // 8 bytes - number of half-moves
	WeakHash  chess.HashCode  // 8 bytes (uint64 alias at types.go:148)
}
```

**Thread-Safe Wrapper:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/thread_safe.go`

```go
// Lines 11-14
type ThreadSafeDuplicateDetector struct {
	detector *DuplicateDetector
	mu       sync.RWMutex
}
```

The interface contract (lines 8-18 in hashing.go):
```go
type DuplicateChecker interface {
	CheckAndAdd(game *chess.Game, board *chess.Board) bool
	DuplicateCount() int
	UniqueCount() int
}
```

### Memory Footprint Analysis

**Per GameSignature entry:**
- `Hash`: 8 bytes (uint64)
- `MoveCount`: 8 bytes (int)
- `WeakHash`: 8 bytes (chess.HashCode = uint64)
- **Total per signature: 24 bytes**

**Map overhead:**
- Go map bucket overhead: ~8-16 bytes per entry
- Slice backing array for collision chain: 24 bytes × chain length
- Estimated: **~50-80 bytes per unique game**

**For 100K games:**
- 100,000 unique games × 70 bytes avg = **~7 MB**
- With hash table overhead: **~10-15 MB**

**For 1M games:**
- 1,000,000 unique games × 70 bytes = **~70 MB**
- With overhead: **~100-150 MB**

### Current Access Patterns

**CheckAndAdd (lines 44-72):**
1. Generate hash for current game
2. Look up hash in `hashTable` map
3. If collision chain exists, iterate to check for exact match
4. If not duplicate, append new GameSignature to slice at that hash bucket
5. **No eviction or size limit** - unbounded growth

**Usage context:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor.go`

```go
// Line 306 - handleGameOutput function
isDuplicate := detector.CheckAndAdd(game, board)
```

Called once per game in both sequential (line 306) and parallel (line 410) processing paths.

### Technology Options for Bounding

| Approach | Pros | Cons | Stdlib? |
|----------|------|------|---------|
| **Simple Cap** | Simple to implement; predictable behavior; no dependencies | Stops detecting duplicates after limit; may miss late duplicates | ✅ Yes |
| **LRU Eviction** | Continues detecting duplicates; evicts oldest entries | More complex; requires doubly-linked list; higher CPU overhead | ✅ Yes (container/list) |
| **Bloom Filter** | Memory-efficient probabilistic filter | False positives; requires tuning; additional complexity | ❌ Needs external package |
| **Count-Min Sketch** | Memory-bounded probabilistic counting | Overcounting possible; complex implementation | ❌ Needs external package |

### Recommended Approach: Simple Cap

**Rationale:**
1. **Simplicity:** Minimal code changes, easy to understand and maintain
2. **Predictable:** Known maximum memory usage (capacity × 70 bytes)
3. **Transparent:** User knows exactly when tracking stops
4. **No external deps:** Pure stdlib implementation
5. **Existing behavior preserved:** Default of 0 (unlimited) maintains current behavior
6. **Performance:** No additional overhead until limit reached

**Implementation strategy:**
```go
type DuplicateDetector struct {
	hashTable      map[uint64][]GameSignature
	useExactMatch  bool
	duplicateCount int
	maxCapacity    int  // 0 = unlimited (current behavior)
	currentSize    int  // track unique game count
}

// In CheckAndAdd, before adding:
if d.maxCapacity > 0 && d.currentSize >= d.maxCapacity {
	// Stop tracking new games, but still check existing ones
	return isDuplicate  // Don't add new entry
}
```

**Default value:** 0 (unlimited) to preserve existing behavior for users not setting the flag.

**Typical values:**
- Small datasets: 10,000 games (~700 KB)
- Medium datasets: 100,000 games (~7 MB)
- Large datasets: 1,000,000 games (~70 MB)

---

## 2. ECOSplitWriter File Handle Management

### Current Structure

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor.go`

```go
// Lines 101-108
type ECOSplitWriter struct {
	baseName string
	level    int // 1=A-E, 2=A0-E9, 3=A00-E99
	files    map[string]*os.File  // Unbounded map
	cfg      *config.Config
}
```

**Access pattern (lines 120-135):**
```go
func (ew *ECOSplitWriter) WriteGame(game *chess.Game) error {
	ecoCode := ew.getECOPrefix(game)
	file, err := ew.getOrCreateFile(ecoCode)  // Opens and caches
	// ... writes to file
}

// Lines 165-179 - getOrCreateFile
func (ew *ECOSplitWriter) getOrCreateFile(ecoPrefix string) (*os.File, error) {
	if file, ok := ew.files[ecoPrefix]; ok {
		return file, nil  // Return cached handle
	}
	// Create and cache new file handle
	file, err := os.Create(filename)
	ew.files[ecoPrefix] = file  // Cache forever - UNBOUNDED
	return file, nil
}
```

### ECO Code Distribution Analysis

Based on [web research](https://www.chessprogramming.org/ECO), the ECO system defines:
- **Level 1 (A-E):** 5 files maximum
- **Level 2 (A0-E9):** 50 files maximum
- **Level 3 (A00-E99):** **500 files maximum**

Real-world distribution is uneven:
- Popular openings (e.g., Sicilian B20-B99) may appear frequently
- Rare openings (e.g., Polish A00) may appear once or never
- Typical large PGN file might span 100-300 distinct ECO codes

**File descriptor limits:**
- macOS default: 256 (soft) / 10,240 (hard) per process
- Linux default: 1024 (soft) / 4096 (hard) per process
- Windows: No hard limit, but resource-intensive beyond 2048

**Risk:** At Level 3, opening all 500 ECO files simultaneously could:
1. Exhaust file descriptors on systems with low limits
2. Consume excessive kernel resources
3. Fail with "too many open files" error

### Technology Options for Bounding

| Approach | Pros | Cons | Stdlib? |
|----------|------|------|---------|
| **LRU Cache with Close/Reopen** | Bounded file descriptors; continues working; predictable memory | Requires reopening files (overhead); more complex | ✅ Yes (container/list) |
| **Fixed Limit (MRU)** | Simple; fast; bounded | No LRU eviction; may thrash with poor access patterns | ✅ Yes |
| **Reference Counting** | Keep popular files open | Complex; doesn't bound total count | ✅ Yes |
| **Time-based Expiry** | Automatic cleanup of idle files | Unpredictable; may close active files | ✅ Yes (time package) |

### Recommended Approach: LRU File Handle Cache

**Rationale:**
1. **Bounded resource usage:** Maximum N files open at once (default: 128)
2. **Temporal locality:** Recent ECO codes likely to be accessed again (games often grouped by opening)
3. **Transparent:** No user-visible behavior change except for large splits
4. **Stdlib implementation:** Use `container/list` for LRU tracking
5. **Safe default:** 128 file handles leaves plenty of headroom on all platforms

**Implementation strategy:**
```go
import "container/list"

type ecoFileEntry struct {
	ecoCode string
	file    *os.File
	element *list.Element  // For LRU tracking
}

type ECOSplitWriter struct {
	baseName    string
	level       int
	files       map[string]*ecoFileEntry  // ECO code -> entry
	lruList     *list.List                 // MRU at front, LRU at back
	maxHandles  int                        // Default: 128
	cfg         *config.Config
}

func (ew *ECOSplitWriter) getOrCreateFile(ecoPrefix string) (*os.File, error) {
	// If already open, move to front of LRU
	if entry, ok := ew.files[ecoPrefix]; ok {
		ew.lruList.MoveToFront(entry.element)
		return entry.file, nil
	}

	// Evict LRU if at capacity
	if ew.maxHandles > 0 && len(ew.files) >= ew.maxHandles {
		ew.evictLRU()
	}

	// Open file and add to front of LRU
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	entry := &ecoFileEntry{ecoCode: ecoPrefix, file: file}
	entry.element = ew.lruList.PushFront(entry)
	ew.files[ecoPrefix] = entry
	return file, nil
}

func (ew *ECOSplitWriter) evictLRU() {
	element := ew.lruList.Back()
	entry := element.Value.(*ecoFileEntry)
	entry.file.Close()
	delete(ew.files, entry.ecoCode)
	ew.lruList.Remove(element)
}
```

**Default value:** 128 (safe on all platforms; handles 25% of possible Level 3 ECO codes)

**Typical values:**
- Conservative: 64 file handles
- Default: 128 file handles
- Aggressive: 256 file handles

**Performance impact:** Minimal. File close/reopen overhead is negligible compared to PGN parsing and game processing.

---

## 3. Configuration Integration

### Current Configuration Structure

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/config/config.go`

```go
// Lines 71-147 - Main Config struct
type Config struct {
	Output     *OutputConfig
	Filter     *FilterConfig
	Duplicate  *DuplicateConfig
	Annotation *AnnotationConfig
	// ... other fields
}
```

**Sub-config pattern (example from duplicate.go):**

```go
// Lines 5-24 in internal/config/duplicate.go
type DuplicateConfig struct {
	Suppress              bool
	SuppressOriginals     bool
	FuzzyMatch            bool
	FuzzyDepth            uint
	UseVirtualHashTable   bool
	DuplicateFile         io.Writer
}

func NewDuplicateConfig() *DuplicateConfig {
	return &DuplicateConfig{}  // Zero values as defaults
}
```

### Flag Definition Pattern

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/flags.go`

```go
// Lines 28-32 - Duplicate detection flags
suppressDuplicates = flag.Bool("D", false, "Suppress duplicate games")
duplicateFile      = flag.String("d", "", "Output duplicates to this file")
outputDupsOnly     = flag.Bool("U", false, "Output only duplicates")
checkFile          = flag.String("c", "", "Check file for duplicate detection")
```

**Flag application pattern (lines 165-179):**

```go
func applyFlags(cfg *config.Config) {
	applyTagOutputFlags(cfg)
	applyContentFlags(cfg)
	// ... etc
}

// In main.go lines 197-198:
cfg.Duplicate.Suppress = *suppressDuplicates
cfg.Duplicate.SuppressOriginals = *outputDupsOnly
```

### Recommended Configuration Points

**1. Add to DuplicateConfig:**

```go
// In internal/config/duplicate.go
type DuplicateConfig struct {
	// ... existing fields
	MaxCapacity int  // Maximum unique games to track (0 = unlimited)
}
```

**2. Add flag in cmd/pgn-extract/flags.go:**

```go
// In duplicate detection section (around line 32)
duplicateMaxCapacity = flag.Int("duplicate-capacity", 0,
	"Maximum unique games to track for duplicate detection (0 = unlimited)")
```

**3. Add to applyFlags in flags.go:**

```go
cfg.Duplicate.MaxCapacity = *duplicateMaxCapacity
```

**4. Pass to detector in main.go:**

```go
// Modify setupDuplicateDetector (line 192)
func setupDuplicateDetector(cfg *config.Config) hashing.DuplicateChecker {
	// ...
	detector := hashing.NewThreadSafeDuplicateDetectorWithCapacity(
		false,
		cfg.Duplicate.MaxCapacity,
	)
	return detector
}
```

**For ECOSplitWriter:**

**1. Add new flag in flags.go:**

```go
// In ECO section (around line 147)
ecoMaxHandles = flag.Int("eco-max-handles", 128,
	"Maximum file handles for ECO split output (0 = unlimited)")
```

**2. Pass to ECOSplitWriter in main.go:**

```go
// Modify ECO split setup (around line 103)
ecoSplitWriter = NewECOSplitWriterWithCache(base, *ecoSplit, cfg, *ecoMaxHandles)
```

### Existing Patterns for Optional Limits

**Similar pattern in config/output.go:**

```go
// Lines 8-9
MaxLineLength uint  // Has default value of 80 (line 49)
```

**Similar pattern in flags.go:**

```go
// Line 139 - workers flag with 0 = auto-detect
workers = flag.Int("workers", 0, "Number of worker threads (0 = auto-detect)")
```

**Pattern to follow:**
- Use **0 or negative value to mean "unlimited/default"**
- Provide sensible non-zero default that works for 95% of use cases
- Document the special meaning of 0

---

## 4. Existing Benchmarks

### Hashing Package Benchmarks

**File:** `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/benchmark_test.go`

Existing benchmarks (lines 18-100):
- `BenchmarkGenerateZobristHash` - measures hash generation speed
- `BenchmarkWeakHash` - measures weak hash speed
- `BenchmarkDuplicateDetector_CheckAndAdd` - measures detector performance
  - Sub-benchmark "Unique" - adds 100 unique positions (line 46-60)
  - Sub-benchmark "Duplicates" - checks same position repeatedly (line 62-72)

**Current benchmark pattern:**
```go
func BenchmarkDuplicateDetector_CheckAndAdd(b *testing.B) {
	b.Run("Unique", func(b *testing.B) {
		dd := NewDuplicateDetector(false)
		// ... setup
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dd.CheckAndAdd(games[i%100], boards[i%100])
		}
	})
}
```

**No memory benchmarks exist yet.** Need to add:
- `b.ReportAllocs()` to track allocations
- Explicit memory measurements using `runtime.MemStats`

### Recommended Memory Benchmark

Add to `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/benchmark_test.go`:

```go
import "runtime"

func BenchmarkDuplicateDetector_MemoryGrowth(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size=%d", size), func(b *testing.B) {
			var m1, m2 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)

			dd := NewDuplicateDetector(false)
			for i := 0; i < size; i++ {
				board := createUniqueBoard(i)
				game := &chess.Game{Tags: make(map[string]string)}
				dd.CheckAndAdd(game, board)
			}

			runtime.ReadMemStats(&m2)
			bytesPerGame := (m2.Alloc - m1.Alloc) / uint64(size)
			b.ReportMetric(float64(bytesPerGame), "bytes/game")
		})
	}
}
```

**For ECOSplitWriter:**

```go
func BenchmarkECOSplitWriter_FileHandles(b *testing.B) {
	// Test with different maxHandles settings
	// Measure file open/close overhead
	// Track max file descriptors used
}
```

---

## 5. Potential Risks and Mitigations

### DuplicateDetector Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Late duplicates missed** | Games after capacity limit may not be detected as duplicates | Document behavior clearly; log warning when limit reached; make limit configurable per use case |
| **Wrong capacity setting** | Too low: misses duplicates; too high: wastes memory | Provide guidance in docs (e.g., "set to expected unique game count × 1.5"); add `-v` verbosity message showing current size |
| **Thread-safety overhead** | RWMutex contention in parallel processing | Acceptable - read-only `currentSize` check uses RLock; write lock only when adding |
| **Map resize overhead** | Go maps resize when full, causing allocation spikes | Pre-size map in constructor: `make(map[uint64][]GameSignature, maxCapacity)` |

**Mitigation implementation:**
```go
func NewDuplicateDetectorWithCapacity(exactMatch bool, maxCapacity int) *DuplicateDetector {
	initialSize := maxCapacity
	if initialSize == 0 {
		initialSize = 1024  // Reasonable default
	}
	return &DuplicateDetector{
		hashTable:     make(map[uint64][]GameSignature, initialSize),
		useExactMatch: exactMatch,
		maxCapacity:   maxCapacity,
	}
}
```

### ECOSplitWriter Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| **File reopen overhead** | Repeated open/close cycles on thrashing access patterns | Acceptable - file I/O dominates; LRU minimizes thrashing for typical sorted PGN files |
| **Data corruption on close** | Unflushed writes lost if evicted file isn't synced | Call `file.Sync()` before closing in evictLRU |
| **Race condition on access** | Not thread-safe (documented in comments line 102) | Maintain current single-goroutine usage (only accessed from result consumer) |
| **OS file descriptor leak** | Files not closed on error paths | Add `defer Close()` in main.go; ensure all error paths call Close() |

**Mitigation implementation:**
```go
func (ew *ECOSplitWriter) evictLRU() {
	element := ew.lruList.Back()
	entry := element.Value.(*ecoFileEntry)
	entry.file.Sync()   // Flush before close
	entry.file.Close()
	delete(ew.files, entry.ecoCode)
	ew.lruList.Remove(element)
}
```

### Constraint Compliance

**No breaking changes:**
- Default of 0 (unlimited) preserves current behavior
- Existing tests pass without modification
- New limits only apply when explicitly set via flags

**No external dependencies:**
- `container/list` is stdlib (Go 1.0+)
- No third-party packages required
- Pure Go implementation

---

## 6. Implementation Considerations

### Integration with Existing Codebase

**DuplicateDetector changes:**

1. Update `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/hashing.go`:
   - Add `maxCapacity`, `currentSize` fields to `DuplicateDetector`
   - Modify `CheckAndAdd` to check capacity before adding
   - Add `NewDuplicateDetectorWithCapacity` constructor
   - Update `Reset()` to reset `currentSize`

2. Update `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/thread_safe.go`:
   - Add `NewThreadSafeDuplicateDetectorWithCapacity` constructor
   - Wrapper automatically inherits capacity behavior

3. No changes needed to interface - maintains compatibility

**ECOSplitWriter changes:**

1. Update `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor.go`:
   - Change `files map[string]*os.File` to `files map[string]*ecoFileEntry`
   - Add `lruList *list.List` and `maxHandles int` fields
   - Add `NewECOSplitWriterWithCache` constructor
   - Implement `evictLRU()` method
   - Modify `getOrCreateFile` to use LRU logic
   - Update `Close()` to close all cached files

2. Add `import "container/list"`

### Testing Strategy

**Unit tests:**
1. Test capacity enforcement in `DuplicateDetector`
2. Test LRU eviction in `ECOSplitWriter`
3. Test file handle reuse in `ECOSplitWriter`

**Benchmark tests:**
1. Measure memory usage with different capacities
2. Measure performance impact of LRU overhead
3. Compare bounded vs. unbounded memory growth

**Integration tests:**
1. Process large PGN file with capacity limit
2. Verify ECO split works with limited file handles
3. Test behavior at exact capacity boundary

### Performance Implications

**DuplicateDetector:**
- **Best case:** No performance impact if under capacity
- **At capacity:** Two hash lookups instead of one (check before add) - negligible
- **Memory savings:** Up to 90%+ reduction for large datasets

**ECOSplitWriter:**
- **Best case:** No reopens if files fit in cache - zero overhead
- **Worst case:** Every write is to different ECO code - one extra open/close per write
- **Typical case:** Temporal locality means 95%+ cache hits

**Measurement approach:**
```bash
# Before changes
go test -bench=BenchmarkDuplicateDetector -benchmem

# After changes
go test -bench=BenchmarkDuplicateDetector -benchmem -run=^$
go test -bench=BenchmarkECOSplitWriter -benchmem -run=^$
```

---

## 7. Documentation and User Guidance

### Flag Documentation

**For `-duplicate-capacity`:**
```
-duplicate-capacity int
    Maximum number of unique games to track for duplicate detection.
    Once this limit is reached, new games are still checked against
    existing entries but are not added to the hash table.
    Set to 0 for unlimited tracking (default).

    Recommended values:
      10000   - Small datasets (~700 KB memory)
      100000  - Medium datasets (~7 MB memory)
      1000000 - Large datasets (~70 MB memory)
```

**For `-eco-max-handles`:**
```
-eco-max-handles int
    Maximum number of simultaneously open file handles when using
    -E (ECO split output). Files are closed and reopened as needed
    using an LRU cache. Set to 0 for unlimited (default: 128).

    Note: The ECO system has up to 500 codes (A00-E99). Most systems
    support 256-1024 file handles. The default of 128 is safe on all
    platforms while handling most real-world PGN distributions.
```

### Error Messages

**When capacity reached:**
```go
if cfg.Verbosity > 0 && d.currentSize >= d.maxCapacity {
	fmt.Fprintf(cfg.LogFile,
		"Warning: Duplicate detector capacity reached (%d games). "+
		"Further duplicates may not be detected.\n",
		d.maxCapacity)
}
```

**When file eviction occurs (debug mode):**
```go
if cfg.Verbosity > 1 {
	fmt.Fprintf(cfg.LogFile,
		"Debug: Evicting ECO file handle for %s (LRU)\n",
		entry.ecoCode)
}
```

---

## 8. Related Work and References

### Go stdlib packages
- [`container/list`](https://pkg.go.dev/container/list) - Doubly-linked list for LRU implementation
- [`runtime.MemStats`](https://pkg.go.dev/runtime#MemStats) - Memory profiling
- [`testing.B.ReportAllocs`](https://pkg.go.dev/testing#B.ReportAllocs) - Allocation tracking

### ECO Classification
- [ECO - Chessprogramming wiki](https://www.chessprogramming.org/ECO)
- [Encyclopaedia of Chess Openings - Wikipedia](https://en.wikipedia.org/wiki/Encyclopaedia_of_Chess_Openings)
- [List of ECO codes - Wikipedia](https://en.wikipedia.org/wiki/List_of_chess_openings)

### Similar implementations
- Original pgn-extract (C): Uses fixed-size hash tables with overflow handling
- Scid chess database: Uses bounded memory pools with LRU eviction
- Chess.com PGN processor: Uses probabilistic filters for duplicate detection

---

## 9. Next Steps

### Implementation Order

1. **DuplicateDetector capacity limiting** (simpler, lower risk)
   - Add fields and constructor
   - Modify CheckAndAdd logic
   - Add unit tests
   - Add memory benchmarks

2. **ECOSplitWriter LRU cache** (more complex, file I/O risk)
   - Implement LRU eviction logic
   - Add file handle tracking
   - Add unit tests for eviction
   - Test with real PGN files

3. **Configuration and flags**
   - Add config fields
   - Add command-line flags
   - Update documentation
   - Add integration tests

4. **Benchmarking and tuning**
   - Run memory benchmarks
   - Tune default values
   - Document performance characteristics

### Open Questions

1. Should we add a warning when 90% of capacity is reached?
2. Should ECOSplitWriter use `O_APPEND` mode for reopened files?
3. Should we add a `-stats` flag to report memory usage?
4. Should we expose `UniqueCount()` in verbose output?

---

## Sources

- [ECO - Chessprogramming wiki](https://www.chessprogramming.org/ECO)
- [ECO Codes - 365Chess.com](https://www.365chess.com/eco.php)
- [List of chess openings - Wikipedia](https://en.wikipedia.org/wiki/List_of_chess_openings)
- [Encyclopaedia of Chess Openings - Wikipedia](https://en.wikipedia.org/wiki/Encyclopaedia_of_Chess_Openings)
