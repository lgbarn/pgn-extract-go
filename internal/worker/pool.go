// Package worker provides a worker pool for parallel game processing.
package worker

import (
	"sync"
	"sync/atomic"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// GameInfo is an interface for game analysis results.
// Implementations should provide information about game properties
// discovered during analysis (e.g., checkmate, stalemate, repetition).
type GameInfo interface {
	// FiftyMoveTriggered returns true if the game triggered the fifty-move rule.
	FiftyMoveTriggered() bool

	// RepetitionDetected returns true if the game has a threefold repetition.
	RepetitionDetected() bool

	// UnderpromotionFound returns true if any pawn promoted to non-queen.
	UnderpromotionFound() bool
}

// WorkItem represents a game to be processed.
type WorkItem struct {
	Game  *chess.Game
	Index int // Original index for tracking
}

// ProcessResult represents the result of processing a game.
type ProcessResult struct {
	Game         *chess.Game
	Index        int
	Matched      bool
	Board        *chess.Board // Final board position (may be nil)
	GameInfo     interface{}  // Implements GameInfo or is nil
	ShouldOutput bool         // Whether to output this game
	OutputToDup  bool         // Whether to output to duplicate file
	Error        error
}

// GetGameInfo returns the GameInfo if it implements the interface, or nil.
func (r *ProcessResult) GetGameInfo() GameInfo {
	if r.GameInfo == nil {
		return nil
	}
	if gi, ok := r.GameInfo.(GameInfo); ok {
		return gi
	}
	return nil
}

// ProcessFunc is the function signature for processing a work item.
type ProcessFunc func(item WorkItem) ProcessResult

// Pool manages a pool of workers for parallel game processing.
type Pool struct {
	numWorkers  int
	bufferSize  int
	workChan    chan WorkItem
	resultChan  chan ProcessResult
	processFunc ProcessFunc
	wg          sync.WaitGroup
	stopFlag    int32 // Atomic flag for early termination
}

// PoolOption configures a Pool.
type PoolOption func(*Pool)

// WithWorkers sets the number of worker goroutines.
func WithWorkers(n int) PoolOption {
	return func(p *Pool) {
		if n >= 1 {
			p.numWorkers = n
		}
	}
}

// WithBufferSize sets the channel buffer size.
func WithBufferSize(size int) PoolOption {
	return func(p *Pool) {
		if size >= 1 {
			p.bufferSize = size
		}
	}
}

// NewPool creates a new worker pool.
// numWorkers: number of worker goroutines
// bufferSize: channel buffer size (recommended: min(numGames, 100))
// processFunc: the function to process each work item
func NewPool(numWorkers int, bufferSize int, processFunc ProcessFunc) *Pool {
	if numWorkers < 1 {
		numWorkers = 1
	}
	if bufferSize < 1 {
		bufferSize = 1
	}
	return &Pool{
		numWorkers:  numWorkers,
		bufferSize:  bufferSize,
		workChan:    make(chan WorkItem, bufferSize),
		resultChan:  make(chan ProcessResult, bufferSize),
		processFunc: processFunc,
	}
}

// NewPoolWithOptions creates a new worker pool using functional options.
// processFunc is required; other settings have sensible defaults.
// Default: 1 worker, buffer size of 10.
func NewPoolWithOptions(processFunc ProcessFunc, opts ...PoolOption) *Pool {
	p := &Pool{
		numWorkers:  1,
		bufferSize:  10,
		processFunc: processFunc,
	}
	for _, opt := range opts {
		opt(p)
	}
	// Create channels after options are applied
	p.workChan = make(chan WorkItem, p.bufferSize)
	p.resultChan = make(chan ProcessResult, p.bufferSize)
	return p
}

// Start starts the worker goroutines.
func (p *Pool) Start() {
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// worker is the main worker goroutine loop.
func (p *Pool) worker() {
	defer p.wg.Done()

	for item := range p.workChan {
		// Check if we should stop early
		if atomic.LoadInt32(&p.stopFlag) != 0 {
			// Still drain the channel but don't process
			continue
		}

		result := p.processFunc(item)
		p.resultChan <- result
	}
}

// Submit submits a work item for processing.
// This may block if the work channel buffer is full.
func (p *Pool) Submit(item WorkItem) {
	p.workChan <- item
}

// TrySubmit attempts to submit a work item without blocking.
// Returns false if the work channel is full or the pool is stopped.
func (p *Pool) TrySubmit(item WorkItem) bool {
	if atomic.LoadInt32(&p.stopFlag) != 0 {
		return false
	}
	select {
	case p.workChan <- item:
		return true
	default:
		return false
	}
}

// Stop signals workers to stop processing new items.
// Items already in the channel will be drained but not processed.
func (p *Pool) Stop() {
	atomic.StoreInt32(&p.stopFlag, 1)
}

// IsStopped returns true if the pool has been stopped.
func (p *Pool) IsStopped() bool {
	return atomic.LoadInt32(&p.stopFlag) != 0
}

// Close closes the work channel and waits for all workers to finish.
// After calling Close, the result channel will be closed when all workers are done.
func (p *Pool) Close() {
	close(p.workChan)
	p.wg.Wait()
	close(p.resultChan)
}

// Results returns the result channel for reading processed results.
func (p *Pool) Results() <-chan ProcessResult {
	return p.resultChan
}

// NumWorkers returns the number of workers in the pool.
func (p *Pool) NumWorkers() int {
	return p.numWorkers
}
