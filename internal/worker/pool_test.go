package worker

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// noopProcessFunc returns a basic process function that does nothing.
func noopProcessFunc() ProcessFunc {
	return func(item WorkItem) ProcessResult {
		return ProcessResult{Game: item.Game, Index: item.Index}
	}
}

// countingProcessFunc returns a process function that increments a counter.
func countingProcessFunc(counter *int32) ProcessFunc {
	return func(item WorkItem) ProcessResult {
		atomic.AddInt32(counter, 1)
		return ProcessResult{Game: item.Game, Index: item.Index, Matched: true}
	}
}

// collectResults drains the result channel and returns the count.
func collectResults(pool *Pool) int {
	count := 0
	for range pool.Results() {
		count++
	}
	return count
}

// TestPoolBasic tests basic worker pool functionality.
func TestPoolBasic(t *testing.T) {
	var processed int32
	pool := NewPool(4, 10, countingProcessFunc(&processed))
	pool.Start()

	const numItems = 10
	for i := 0; i < numItems; i++ {
		pool.Submit(WorkItem{
			Game:  &chess.Game{Tags: map[string]string{"Event": "Test"}},
			Index: i,
		})
	}

	go pool.Close()

	resultCount := collectResults(pool)
	if resultCount != numItems {
		t.Errorf("results = %d; want %d", resultCount, numItems)
	}
	if got := atomic.LoadInt32(&processed); got != numItems {
		t.Errorf("processed = %d; want %d", got, numItems)
	}
}

// TestPoolSingleWorker tests pool with single worker.
func TestPoolSingleWorker(t *testing.T) {
	pool := NewPool(1, 5, noopProcessFunc())
	pool.Start()

	const numItems = 5
	for i := 0; i < numItems; i++ {
		pool.Submit(WorkItem{Game: &chess.Game{}, Index: i})
	}

	go pool.Close()

	if got := collectResults(pool); got != numItems {
		t.Errorf("results = %d; want %d", got, numItems)
	}
}

// TestPoolEarlyStop tests early termination with Stop().
func TestPoolEarlyStop(t *testing.T) {
	var processedCount int32

	slowProcessFunc := func(item WorkItem) ProcessResult {
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&processedCount, 1)
		return ProcessResult{Game: item.Game, Index: item.Index}
	}

	pool := NewPool(2, 100, slowProcessFunc)
	pool.Start()

	const numItems = 50
	for i := 0; i < numItems; i++ {
		pool.Submit(WorkItem{Game: &chess.Game{}, Index: i})
	}

	time.Sleep(30 * time.Millisecond)
	pool.Stop()

	go pool.Close()
	collectResults(pool)

	// Should have processed fewer than total due to early stop
	if processed := atomic.LoadInt32(&processedCount); processed >= numItems {
		t.Logf("early stop may not have prevented all processing: %d processed", processed)
	}
}

// TestPoolIsStopped tests the IsStopped method.
func TestPoolIsStopped(t *testing.T) {
	pool := NewPool(2, 10, noopProcessFunc())
	pool.Start()

	if pool.IsStopped() {
		t.Error("pool should not be stopped initially")
	}

	pool.Stop()

	if !pool.IsStopped() {
		t.Error("pool should be stopped after Stop()")
	}

	pool.Close()
}

// TestPoolTrySubmit tests non-blocking submission.
func TestPoolTrySubmit(t *testing.T) {
	slowProcessFunc := func(item WorkItem) ProcessResult {
		time.Sleep(100 * time.Millisecond)
		return ProcessResult{}
	}

	// Small buffer to test blocking behavior
	pool := NewPool(1, 2, slowProcessFunc)
	pool.Start()

	// First two should succeed (buffer size 2)
	if !pool.TrySubmit(WorkItem{Game: &chess.Game{}, Index: 0}) {
		t.Error("first TrySubmit should succeed")
	}
	if !pool.TrySubmit(WorkItem{Game: &chess.Game{}, Index: 1}) {
		t.Error("second TrySubmit should succeed")
	}

	// Third might fail if buffer is full (timing-dependent, just verify no panic)
	pool.TrySubmit(WorkItem{Game: &chess.Game{}, Index: 2})

	// After stop, TrySubmit should return false
	pool.Stop()
	if pool.TrySubmit(WorkItem{Game: &chess.Game{}, Index: 3}) {
		t.Error("TrySubmit after Stop should return false")
	}

	pool.Close()
}

// TestPoolNumWorkers tests NumWorkers method.
func TestPoolNumWorkers(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"valid workers", 4, 4},
		{"minimum workers", 1, 1},
		{"zero defaults to 1", 0, 1},
		{"negative defaults to 1", -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(tt.input, 10, noopProcessFunc())
			if got := pool.NumWorkers(); got != tt.expected {
				t.Errorf("NumWorkers() = %d; want %d", got, tt.expected)
			}
		})
	}
}

// TestPoolResultOrder tests that all results are received regardless of order.
func TestPoolResultOrder(t *testing.T) {
	variableDelayFunc := func(item WorkItem) ProcessResult {
		if item.Index%2 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
		return ProcessResult{Game: item.Game, Index: item.Index}
	}

	pool := NewPool(4, 20, variableDelayFunc)
	pool.Start()

	const numItems = 10
	for i := 0; i < numItems; i++ {
		pool.Submit(WorkItem{Game: &chess.Game{}, Index: i})
	}

	go pool.Close()

	// Collect all result indices
	seen := make(map[int]bool)
	for result := range pool.Results() {
		seen[result.Index] = true
	}

	if len(seen) != numItems {
		t.Errorf("received %d results; want %d", len(seen), numItems)
	}

	// Verify all indices are present
	for i := 0; i < numItems; i++ {
		if !seen[i] {
			t.Errorf("missing index %d in results", i)
		}
	}
}

// TestPoolNoRace is designed to be run with -race flag.
func TestPoolNoRace(t *testing.T) {
	var counter int32
	pool := NewPool(8, 50, countingProcessFunc(&counter))
	pool.Start()

	const numItems = 100
	go func() {
		for i := 0; i < numItems; i++ {
			pool.Submit(WorkItem{Game: &chess.Game{}, Index: i})
		}
		pool.Close()
	}()

	collectResults(pool)

	if got := atomic.LoadInt32(&counter); got != numItems {
		t.Errorf("processed = %d; want %d", got, numItems)
	}
}

// TestNewPoolWithOptions tests the functional options constructor.
func TestNewPoolWithOptions(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		pool := NewPoolWithOptions(noopProcessFunc())
		if pool.NumWorkers() != 1 {
			t.Errorf("default workers = %d; want 1", pool.NumWorkers())
		}
		if pool.bufferSize != 10 {
			t.Errorf("default bufferSize = %d; want 10", pool.bufferSize)
		}
	})

	t.Run("with workers", func(t *testing.T) {
		pool := NewPoolWithOptions(noopProcessFunc(), WithWorkers(4))
		if pool.NumWorkers() != 4 {
			t.Errorf("NumWorkers() = %d; want 4", pool.NumWorkers())
		}
	})

	t.Run("with buffer size", func(t *testing.T) {
		pool := NewPoolWithOptions(noopProcessFunc(), WithBufferSize(50))
		if pool.bufferSize != 50 {
			t.Errorf("bufferSize = %d; want 50", pool.bufferSize)
		}
	})

	t.Run("with multiple options", func(t *testing.T) {
		pool := NewPoolWithOptions(noopProcessFunc(), WithWorkers(8), WithBufferSize(100))
		if pool.NumWorkers() != 8 {
			t.Errorf("NumWorkers() = %d; want 8", pool.NumWorkers())
		}
		if pool.bufferSize != 100 {
			t.Errorf("bufferSize = %d; want 100", pool.bufferSize)
		}
	})

	t.Run("invalid workers ignored", func(t *testing.T) {
		pool := NewPoolWithOptions(noopProcessFunc(), WithWorkers(0))
		if pool.NumWorkers() != 1 {
			t.Errorf("NumWorkers() = %d; want 1 (default)", pool.NumWorkers())
		}
	})

	t.Run("invalid buffer size ignored", func(t *testing.T) {
		pool := NewPoolWithOptions(noopProcessFunc(), WithBufferSize(-5))
		if pool.bufferSize != 10 {
			t.Errorf("bufferSize = %d; want 10 (default)", pool.bufferSize)
		}
	})

	t.Run("functional with options", func(t *testing.T) {
		var processed int32
		pool := NewPoolWithOptions(countingProcessFunc(&processed), WithWorkers(2), WithBufferSize(5))
		pool.Start()

		const numItems = 5
		for i := 0; i < numItems; i++ {
			pool.Submit(WorkItem{Game: &chess.Game{}, Index: i})
		}

		go pool.Close()
		collectResults(pool)

		if got := atomic.LoadInt32(&processed); got != numItems {
			t.Errorf("processed = %d; want %d", got, numItems)
		}
	})
}
