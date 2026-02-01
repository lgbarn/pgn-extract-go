// processor.go - Game processing and output functions
package main

import (
	"container/list"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/cql"
	"github.com/lgbarn/pgn-extract-go/internal/eco"
	"github.com/lgbarn/pgn-extract-go/internal/hashing"
	"github.com/lgbarn/pgn-extract-go/internal/matching"
	"github.com/lgbarn/pgn-extract-go/internal/output"
	"github.com/lgbarn/pgn-extract-go/internal/parser"
	"github.com/lgbarn/pgn-extract-go/internal/worker"
)

// withOutputFile temporarily redirects output to a different writer, executes fn,
// then restores the original output file. This eliminates the repeated pattern of
// saving, swapping, and restoring the output file.
func withOutputFile(cfg *config.Config, w io.Writer, fn func()) {
	original := cfg.OutputFile
	cfg.OutputFile = w
	fn()
	cfg.OutputFile = original
}

// ProcessingContext holds all processing state
type ProcessingContext struct {
	cfg              *config.Config
	detector         hashing.DuplicateChecker
	setupDetector    *hashing.SetupDuplicateDetector
	ecoClassifier    *eco.ECOClassifier
	gameFilter       *matching.GameFilter
	cqlNode          cql.Node
	variationMatcher *matching.VariationMatcher
	materialMatcher  *matching.MaterialMatcher
	ecoSplitWriter   *ECOSplitWriter
}

// SplitWriter handles writing to multiple output files.
// NOT thread-safe: Only accessed from the single result-consumer goroutine in outputGamesParallel.
type SplitWriter struct {
	baseName     string
	pattern      string // filename pattern with %s for base and %d for number
	gamesPerFile int
	currentFile  *os.File
	fileNumber   int
	gameCount    int
}

// NewSplitWriter creates a new split writer with default pattern
func NewSplitWriter(baseName string, gamesPerFile int) *SplitWriter {
	return NewSplitWriterWithPattern(baseName, gamesPerFile, "%s_%d.pgn")
}

// NewSplitWriterWithPattern creates a new split writer with a custom filename pattern
func NewSplitWriterWithPattern(baseName string, gamesPerFile int, pattern string) *SplitWriter {
	return &SplitWriter{
		baseName:     baseName,
		pattern:      pattern,
		gamesPerFile: gamesPerFile,
		fileNumber:   1,
	}
}

// Write implements io.Writer
func (sw *SplitWriter) Write(p []byte) (n int, err error) {
	if sw.currentFile == nil || sw.gameCount >= sw.gamesPerFile {
		if sw.currentFile != nil {
			_ = sw.currentFile.Close() // cleanup before creating new file
			sw.fileNumber++
		}
		filename := fmt.Sprintf(sw.pattern, sw.baseName, sw.fileNumber)
		sw.currentFile, err = os.Create(filename) //nolint:gosec // G304: filename is derived from user-specified base name
		if err != nil {
			return 0, err
		}
		sw.gameCount = 0
	}
	return sw.currentFile.Write(p)
}

// IncrementGameCount should be called after each game is written
func (sw *SplitWriter) IncrementGameCount() {
	sw.gameCount++
}

// Close closes the current file
func (sw *SplitWriter) Close() error {
	if sw.currentFile != nil {
		return sw.currentFile.Close()
	}
	return nil
}

// lruFileEntry represents an entry in the LRU file handle cache.
type lruFileEntry struct {
	ecoPrefix string
	file      *os.File
	element   *list.Element
}

// ECOSplitWriter writes games to different files based on ECO code.
// NOT thread-safe: Only accessed from the single result-consumer goroutine in outputGamesParallel.
type ECOSplitWriter struct {
	baseName   string
	level      int // 1=A-E, 2=A0-E9, 3=A00-E99
	files      map[string]*lruFileEntry
	cfg        *config.Config
	lruList    *list.List
	maxHandles int
}

// NewECOSplitWriter creates a new ECO-based split writer.
func NewECOSplitWriter(baseName string, level int, cfg *config.Config, maxHandles int) *ECOSplitWriter {
	if maxHandles <= 0 {
		maxHandles = 128
	}
	return &ECOSplitWriter{
		baseName:   baseName,
		level:      level,
		files:      make(map[string]*lruFileEntry),
		cfg:        cfg,
		lruList:    list.New(),
		maxHandles: maxHandles,
	}
}

// WriteGame writes a game to the appropriate ECO-based file.
func (ew *ECOSplitWriter) WriteGame(game *chess.Game) error {
	ecoCode := ew.getECOPrefix(game)
	file, err := ew.getOrCreateFile(ecoCode)
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

// getECOPrefix extracts the ECO prefix based on the configured level.
func (ew *ECOSplitWriter) getECOPrefix(game *chess.Game) string {
	eco := game.ECO()
	if eco == "" {
		return "unknown"
	}

	switch ew.level {
	case 1:
		// Just the letter: A, B, C, D, E
		if len(eco) >= 1 {
			return string(eco[0])
		}
	case 2:
		// Letter + first digit: A0, A1, ..., E9
		if len(eco) >= 2 {
			return eco[:2]
		}
	case 3:
		// Full code: A00, A01, ..., E99
		if len(eco) >= 3 {
			return eco[:3]
		}
	}

	return eco
}

// getOrCreateFile gets an existing file or creates a new one for the given ECO prefix.
// Uses LRU cache to limit open file handles.
func (ew *ECOSplitWriter) getOrCreateFile(ecoPrefix string) (*os.File, error) {
	entry, exists := ew.files[ecoPrefix]

	// Case 1: Entry exists and file is open
	if exists && entry.file != nil {
		// Move to front (most recently used)
		ew.lruList.MoveToFront(entry.element)
		return entry.file, nil
	}

	filename := fmt.Sprintf("%s_%s.pgn", ew.baseName, ecoPrefix)

	// Case 2: Entry exists but file was evicted (closed) - reopen in append mode
	if exists && entry.file == nil {
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //nolint:gosec // G304: filename is derived from user-specified base name, G302: 0644 is appropriate for user-created output files
		if err != nil {
			return nil, err
		}
		entry.file = file
		// Re-add to LRU list (element was removed during eviction)
		entry.element = ew.lruList.PushFront(entry)
		ew.evictIfNeeded()
		return file, nil
	}

	// Case 3: New entry - create file
	file, err := os.Create(filename) //nolint:gosec // G304: filename is derived from user-specified base name
	if err != nil {
		return nil, err
	}

	// Create new entry and add to front of LRU list
	newEntry := &lruFileEntry{
		ecoPrefix: ecoPrefix,
		file:      file,
	}
	newEntry.element = ew.lruList.PushFront(newEntry)
	ew.files[ecoPrefix] = newEntry

	// Evict least recently used if we've exceeded maxHandles
	ew.evictIfNeeded()

	return file, nil
}

// evictIfNeeded evicts the least recently used file handle if we've exceeded maxHandles.
func (ew *ECOSplitWriter) evictIfNeeded() {
	if ew.lruList.Len() <= ew.maxHandles {
		return
	}

	// Evict from back (least recently used)
	back := ew.lruList.Back()
	if back == nil {
		return
	}

	entry, ok := back.Value.(*lruFileEntry)
	if !ok {
		return
	}
	if entry.file != nil {
		_ = entry.file.Close() // cleanup on eviction
		entry.file = nil
	}

	// Remove from LRU list but keep entry in map for potential reopen
	ew.lruList.Remove(back)
	entry.element = nil // Defensive: element is no longer in the list
}

// Close closes all open files.
func (ew *ECOSplitWriter) Close() error {
	var lastErr error
	for _, entry := range ew.files {
		if entry.file != nil {
			if err := entry.file.Close(); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

// FileCount returns the number of files created.
func (ew *ECOSplitWriter) FileCount() int {
	return len(ew.files)
}

// OpenHandleCount returns the number of currently open file handles.
func (ew *ECOSplitWriter) OpenHandleCount() int {
	return ew.lruList.Len()
}

// processInput parses games from a reader
func processInput(r io.Reader, name string, cfg *config.Config) []*chess.Game {
	cfg.CurrentInputFile = name

	p := parser.NewParser(r, cfg)
	games, err := p.ParseAllGames()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", name, err)
	}

	return games
}

// outputGamesWithProcessing outputs games with optional filtering, ECO classification, and duplicate detection.
// Returns the number of games output and the number of duplicates found.
func outputGamesWithProcessing(games []*chess.Game, ctx *ProcessingContext) (int, int) {
	numWorkers := *workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	// Use parallel processing for multiple workers and enough games
	if numWorkers > 1 && len(games) > 2 {
		return outputGamesParallel(games, ctx, numWorkers)
	}

	return outputGamesSequential(games, ctx)
}

// outputGamesSequential processes games sequentially (single-threaded).
func outputGamesSequential(games []*chess.Game, ctx *ProcessingContext) (int, int) {
	cfg := ctx.cfg
	outputCount := 0
	duplicateCount := 0

	var jsonGames []*chess.Game

	for _, game := range games {
		if *stopAfter > 0 && atomic.LoadInt64(&matchedCount) >= int64(*stopAfter) {
			break
		}

		// Track game position (1-indexed) and check if it should be processed
		position := int(IncrementGamePosition())
		if !checkGamePosition(position) {
			continue
		}

		filterResult := applyFilters(game, ctx)

		if filterResult.SkipOutput {
			if !*quiet && filterResult.ErrorMessage != "" {
				fmt.Fprintf(os.Stderr, "Skipping game: %s\n", filterResult.ErrorMessage)
			}
			continue
		}

		if !filterResult.Matched {
			outputNonMatchingGame(game, cfg)
			continue
		}

		if *reportOnly {
			atomic.AddInt64(&matchedCount, 1)
			outputCount++
			continue
		}

		// Apply move truncation before output
		truncateMoves(game)

		out, dup := handleGameOutput(game, filterResult.Board, filterResult.GameInfo, ctx, &jsonGames)
		outputCount += out
		duplicateCount += dup
	}

	if cfg.Output.JSONFormat && len(jsonGames) > 0 {
		output.OutputGamesJSON(jsonGames, cfg, cfg.OutputFile)
	}

	return outputCount, duplicateCount
}

// outputNonMatchingGame outputs a game to the non-matching file if configured.
func outputNonMatchingGame(game *chess.Game, cfg *config.Config) {
	if cfg.NonMatchingFile == nil {
		return
	}
	withOutputFile(cfg, cfg.NonMatchingFile, func() {
		output.OutputGame(game, cfg)
	})
}

// handleGameOutput handles duplicate detection and game output.
// Returns (output count, duplicate count).
func handleGameOutput(game *chess.Game, board *chess.Board, gameInfo *GameAnalysis, ctx *ProcessingContext, jsonGames *[]*chess.Game) (int, int) {
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

	isDuplicate := detector.CheckAndAdd(game, board)

	if isDuplicate {
		outputDuplicateGame(game, cfg)
		if cfg.Duplicate.SuppressOriginals {
			outputGameWithECOSplit(game, cfg, gameInfo, jsonGames, ctx.ecoSplitWriter)
			atomic.AddInt64(&matchedCount, 1)
			return 1, 1
		}
		return 0, 1
	}

	// Not a duplicate - output if not suppressing or if not outputting only duplicates
	if shouldOutputUnique(cfg) {
		outputGameWithECOSplit(game, cfg, gameInfo, jsonGames, ctx.ecoSplitWriter)
		atomic.AddInt64(&matchedCount, 1)
		return 1, 0
	}

	return 0, 0
}

// shouldOutputUnique returns true if unique (non-duplicate) games should be output.
func shouldOutputUnique(cfg *config.Config) bool {
	return !cfg.Duplicate.Suppress || !cfg.Duplicate.SuppressOriginals
}

// outputDuplicateGame outputs a game to the duplicate file if configured.
func outputDuplicateGame(game *chess.Game, cfg *config.Config) {
	if cfg.Duplicate.DuplicateFile == nil {
		return
	}
	withOutputFile(cfg, cfg.Duplicate.DuplicateFile, func() {
		if cfg.Output.JSONFormat {
			output.OutputGameJSON(game, cfg)
		} else {
			output.OutputGame(game, cfg)
		}
	})
}

// outputGamesParallel processes games using a worker pool for parallel execution.
//
// Concurrency model: Multiple worker goroutines process games in parallel, but all results
// are consumed by a single goroutine (the main function body below). This ensures that
// non-thread-safe components (jsonGames slice, ECOSplitWriter, SplitWriter) are only
// accessed from one goroutine, avoiding data races without requiring synchronization.
func outputGamesParallel(games []*chess.Game, ctx *ProcessingContext, numWorkers int) (int, int) {
	cfg := ctx.cfg
	outputCount := int64(0)
	duplicateCount := int64(0)

	processFunc := func(item worker.WorkItem) worker.ProcessResult {
		return processGameWorker(item, ctx)
	}

	bufferSize := len(games)
	if bufferSize > 100 {
		bufferSize = 100
	}
	pool := worker.NewPool(numWorkers, bufferSize, processFunc)
	pool.Start()

	go func() {
		for i, game := range games {
			if *stopAfter > 0 && atomic.LoadInt64(&matchedCount) >= int64(*stopAfter) {
				break
			}

			// Track game position (1-indexed) and check if it should be processed
			position := int(IncrementGamePosition())
			if !checkGamePosition(position) {
				continue
			}

			pool.Submit(worker.WorkItem{Game: game, Index: i})
		}
		pool.Close()
	}()

	// jsonGames is only appended to from this single consumer goroutine (not thread-safe).
	var jsonGames []*chess.Game

	for result := range pool.Results() {
		if *stopAfter > 0 && atomic.LoadInt64(&matchedCount) >= int64(*stopAfter) {
			pool.Stop()
			continue
		}

		if !result.Matched {
			outputNonMatchingGame(result.Game, cfg)
			continue
		}

		if *reportOnly {
			atomic.AddInt64(&matchedCount, 1)
			atomic.AddInt64(&outputCount, 1)
			continue
		}

		// Apply move truncation before output
		truncateMoves(result.Game)

		gameInfo, ok := result.GameInfo.(*GameAnalysis)
		if !ok {
			gameInfo = nil
		}
		out, dup := handleGameOutput(result.Game, result.Board, gameInfo, ctx, &jsonGames)
		atomic.AddInt64(&outputCount, int64(out))
		atomic.AddInt64(&duplicateCount, int64(dup))
	}

	if cfg.Output.JSONFormat && len(jsonGames) > 0 {
		output.OutputGamesJSON(jsonGames, cfg, cfg.OutputFile)
	}

	return int(atomic.LoadInt64(&outputCount)), int(atomic.LoadInt64(&duplicateCount))
}

// processGameWorker processes a single game in a worker goroutine.
// This does all the CPU-intensive work that can be safely parallelized.
func processGameWorker(item worker.WorkItem, ctx *ProcessingContext) worker.ProcessResult {
	game := item.Game
	result := worker.ProcessResult{
		Game:  game,
		Index: item.Index,
	}

	// Apply all filters using shared logic
	filterResult := applyFilters(game, ctx)

	// Map FilterResult to ProcessResult
	result.Matched = filterResult.Matched && !filterResult.SkipOutput
	result.Board = filterResult.Board
	result.GameInfo = filterResult.GameInfo
	result.ShouldOutput = filterResult.Matched && !filterResult.SkipOutput && !*reportOnly

	return result
}

// outputGameWithECOSplit outputs a game with optional annotations and ECO-based splitting.
func outputGameWithECOSplit(game *chess.Game, cfg *config.Config, gameInfo *GameAnalysis, jsonGames *[]*chess.Game, ecoWriter *ECOSplitWriter) {
	// Handle split writer
	if sw, ok := cfg.OutputFile.(*SplitWriter); ok {
		defer sw.IncrementGameCount()
	}

	if cfg.Output.JSONFormat {
		*jsonGames = append(*jsonGames, game)
		return
	}

	// If ECO split writer is configured, use it
	if ecoWriter != nil {
		if err := ecoWriter.WriteGame(game); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing game to ECO file: %v\n", err)
		}
		return
	}

	output.OutputGame(game, cfg)
}
