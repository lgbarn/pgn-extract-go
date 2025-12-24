// processor.go - Game processing and output functions
package main

import (
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
	detector         *hashing.DuplicateDetector
	ecoClassifier    *eco.ECOClassifier
	gameFilter       *matching.GameFilter
	cqlNode          cql.Node
	variationMatcher *matching.VariationMatcher
	materialMatcher  *matching.MaterialMatcher
}

// SplitWriter handles writing to multiple output files
type SplitWriter struct {
	baseName     string
	gamesPerFile int
	currentFile  *os.File
	fileNumber   int
	gameCount    int
}

// NewSplitWriter creates a new split writer
func NewSplitWriter(baseName string, gamesPerFile int) *SplitWriter {
	return &SplitWriter{
		baseName:     baseName,
		gamesPerFile: gamesPerFile,
		fileNumber:   1,
	}
}

// Write implements io.Writer
func (sw *SplitWriter) Write(p []byte) (n int, err error) {
	if sw.currentFile == nil || sw.gameCount >= sw.gamesPerFile {
		if sw.currentFile != nil {
			sw.currentFile.Close()
			sw.fileNumber++
		}
		filename := fmt.Sprintf("%s_%d.pgn", sw.baseName, sw.fileNumber)
		sw.currentFile, err = os.Create(filename)
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
	detector := ctx.detector

	outputCount := 0
	duplicateCount := 0

	// For JSON output, collect games first then output all at once
	var jsonGames []*chess.Game

	for _, game := range games {
		// Check if we should stop
		if *stopAfter > 0 && atomic.LoadInt64(&matchedCount) >= int64(*stopAfter) {
			break
		}

		// Apply all filters using shared logic
		filterResult := applyFilters(game, ctx)

		// Handle validation failures (skip entirely)
		if filterResult.SkipOutput {
			if !*quiet && filterResult.ErrorMessage != "" {
				fmt.Fprintf(os.Stderr, "Skipping game: %s\n", filterResult.ErrorMessage)
			}
			continue
		}

		// If not matched, skip or output to non-matching file
		if !filterResult.Matched {
			if cfg.NonMatchingFile != nil {
				withOutputFile(cfg, cfg.NonMatchingFile, func() {
					output.OutputGame(game, cfg)
				})
			}
			continue
		}

		// Report-only mode - don't output games
		if *reportOnly {
			atomic.AddInt64(&matchedCount, 1)
			outputCount++
			continue
		}

		// Handle duplicate detection
		if detector != nil {
			board := filterResult.Board
			if board == nil {
				board = replayGame(game)
			}

			isDuplicate := detector.CheckAndAdd(game, board)

			if isDuplicate {
				duplicateCount++
				// Output to duplicate file if configured
				if cfg.Duplicate.DuplicateFile != nil {
					withOutputFile(cfg, cfg.Duplicate.DuplicateFile, func() {
						if cfg.Output.JSONFormat {
							output.OutputGameJSON(game, cfg)
						} else {
							output.OutputGame(game, cfg)
						}
					})
				}
				// If outputting only duplicates
				if cfg.Duplicate.SuppressOriginals {
					outputGameWithAnnotations(game, cfg, filterResult.GameInfo, &jsonGames)
					atomic.AddInt64(&matchedCount, 1)
					outputCount++
				}
			} else {
				// Not a duplicate - output normally unless suppressing duplicates
				if !cfg.Duplicate.Suppress {
					outputGameWithAnnotations(game, cfg, filterResult.GameInfo, &jsonGames)
					atomic.AddInt64(&matchedCount, 1)
					outputCount++
				} else if !cfg.Duplicate.SuppressOriginals {
					outputGameWithAnnotations(game, cfg, filterResult.GameInfo, &jsonGames)
					atomic.AddInt64(&matchedCount, 1)
					outputCount++
				}
			}
		} else {
			// No duplicate detection
			outputGameWithAnnotations(game, cfg, filterResult.GameInfo, &jsonGames)
			atomic.AddInt64(&matchedCount, 1)
			outputCount++
		}
	}

	// Output JSON array if JSON mode
	if cfg.Output.JSONFormat && len(jsonGames) > 0 {
		output.OutputGamesJSON(jsonGames, cfg, cfg.OutputFile)
	}

	return outputCount, duplicateCount
}

// outputGamesParallel processes games using a worker pool for parallel execution.
func outputGamesParallel(games []*chess.Game, ctx *ProcessingContext, numWorkers int) (int, int) {
	cfg := ctx.cfg
	detector := ctx.detector // Use the shared detector directly (result collection is single-threaded)

	outputCount := int64(0)
	duplicateCount := int64(0)

	// Create the processing function that will run in worker goroutines
	processFunc := func(item worker.WorkItem) worker.ProcessResult {
		return processGameWorker(item, ctx)
	}

	// Create and start pool
	bufferSize := len(games)
	if bufferSize > 100 {
		bufferSize = 100
	}
	pool := worker.NewPool(numWorkers, bufferSize, processFunc)
	pool.Start()

	// Submit all games in a separate goroutine
	go func() {
		for i, game := range games {
			// Check stopAfter before submitting
			if *stopAfter > 0 && atomic.LoadInt64(&matchedCount) >= int64(*stopAfter) {
				break
			}
			pool.Submit(worker.WorkItem{Game: game, Index: i})
		}
		pool.Close()
	}()

	// For JSON output, collect games first
	var jsonGames []*chess.Game

	// Collect results from the result channel (single goroutine - serializes output)
	for result := range pool.Results() {
		// Check stopAfter
		if *stopAfter > 0 && atomic.LoadInt64(&matchedCount) >= int64(*stopAfter) {
			pool.Stop()
			continue // Drain remaining results
		}

		// Handle non-matching games
		if !result.Matched {
			if cfg.NonMatchingFile != nil {
				withOutputFile(cfg, cfg.NonMatchingFile, func() {
					output.OutputGame(result.Game, cfg)
				})
			}
			continue
		}

		// Report-only mode
		if *reportOnly {
			atomic.AddInt64(&matchedCount, 1)
			atomic.AddInt64(&outputCount, 1)
			continue
		}

		// Handle duplicate detection (single-threaded in result collection)
		if detector != nil {
			board := result.Board
			if board == nil {
				board = replayGame(result.Game)
			}

			isDuplicate := detector.CheckAndAdd(result.Game, board)

			if isDuplicate {
				atomic.AddInt64(&duplicateCount, 1)
				// Output to duplicate file if configured
				if cfg.Duplicate.DuplicateFile != nil {
					withOutputFile(cfg, cfg.Duplicate.DuplicateFile, func() {
						if cfg.Output.JSONFormat {
							output.OutputGameJSON(result.Game, cfg)
						} else {
							output.OutputGame(result.Game, cfg)
						}
					})
				}
				// If outputting only duplicates
				if cfg.Duplicate.SuppressOriginals {
					gameInfo, _ := result.GameInfo.(*GameAnalysis)
					outputGameWithAnnotations(result.Game, cfg, gameInfo, &jsonGames)
					atomic.AddInt64(&matchedCount, 1)
					atomic.AddInt64(&outputCount, 1)
				}
			} else {
				// Not a duplicate - output normally unless suppressing duplicates
				if !cfg.Duplicate.Suppress {
					gameInfo, _ := result.GameInfo.(*GameAnalysis)
					outputGameWithAnnotations(result.Game, cfg, gameInfo, &jsonGames)
					atomic.AddInt64(&matchedCount, 1)
					atomic.AddInt64(&outputCount, 1)
				} else if !cfg.Duplicate.SuppressOriginals {
					gameInfo, _ := result.GameInfo.(*GameAnalysis)
					outputGameWithAnnotations(result.Game, cfg, gameInfo, &jsonGames)
					atomic.AddInt64(&matchedCount, 1)
					atomic.AddInt64(&outputCount, 1)
				}
			}
		} else {
			// No duplicate detection
			gameInfo, _ := result.GameInfo.(*GameAnalysis)
			outputGameWithAnnotations(result.Game, cfg, gameInfo, &jsonGames)
			atomic.AddInt64(&matchedCount, 1)
			atomic.AddInt64(&outputCount, 1)
		}
	}

	// Output JSON array if JSON mode
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

// outputGameWithAnnotations outputs a game with optional annotations
func outputGameWithAnnotations(game *chess.Game, cfg *config.Config, gameInfo *GameAnalysis, jsonGames *[]*chess.Game) {
	// Handle split writer
	if sw, ok := cfg.OutputFile.(*SplitWriter); ok {
		defer sw.IncrementGameCount()
	}

	if cfg.Output.JSONFormat {
		*jsonGames = append(*jsonGames, game)
	} else {
		output.OutputGame(game, cfg)
	}
}
