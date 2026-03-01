// Package worker provides a worker pool for parallel game processing.
package worker

import (
	"sync"
	"sync/atomic"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

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
	GameInfo     interface{}  // Opaque analysis payload; typed by consumer
	ShouldOutput bool         // Whether to output this game
	OutputToDup  bool         // Whether to output to duplicate file
	Error        error
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

// NewPool creates a new worker pool with the specified number of workers and buffer size.
func NewPool(numWorkers, bufferSize int, processFunc ProcessFunc) *Pool {
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

// worker processes items from the work channel until it is closed.
func (p *Pool) worker() {
	defer p.wg.Done()

	for item := range p.workChan {
		if p.IsStopped() {
			continue // Drain channel without processing
		}
		p.resultChan <- p.processFunc(item)
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
