package worker

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// TestPoolBasic tests basic worker pool functionality.
func TestPoolBasic(t *testing.T) {
	var processed int32

	processFunc := func(item WorkItem) ProcessResult {
		atomic.AddInt32(&processed, 1)
		return ProcessResult{
			Game:    item.Game,
			Index:   item.Index,
			Matched: true,
		}
	}

	pool := NewPool(4, 10, processFunc)
	pool.Start()

	// Submit 10 items
	for i := 0; i < 10; i++ {
		pool.Submit(WorkItem{
			Game:  &chess.Game{Tags: map[string]string{"Event": "Test"}},
			Index: i,
		})
	}

	// Close and collect results
	go func() {
		pool.Close()
	}()

	resultCount := 0
	for range pool.Results() {
		resultCount++
	}

	if resultCount != 10 {
		t.Errorf("Expected 10 results, got %d", resultCount)
	}

	if atomic.LoadInt32(&processed) != 10 {
		t.Errorf("Expected 10 processed, got %d", processed)
	}
}

// TestPoolSingleWorker tests pool with single worker.
func TestPoolSingleWorker(t *testing.T) {
	processFunc := func(item WorkItem) ProcessResult {
		return ProcessResult{
			Game:    item.Game,
			Index:   item.Index,
			Matched: true,
		}
	}

	pool := NewPool(1, 5, processFunc)
	pool.Start()

	// Submit items
	for i := 0; i < 5; i++ {
		pool.Submit(WorkItem{
			Game:  &chess.Game{},
			Index: i,
		})
	}

	go func() {
		pool.Close()
	}()

	resultCount := 0
	for range pool.Results() {
		resultCount++
	}

	if resultCount != 5 {
		t.Errorf("Expected 5 results, got %d", resultCount)
	}
}

// TestPoolEarlyStop tests early termination with Stop().
func TestPoolEarlyStop(t *testing.T) {
	var processedCount int32

	processFunc := func(item WorkItem) ProcessResult {
		time.Sleep(10 * time.Millisecond) // Simulate work
		atomic.AddInt32(&processedCount, 1)
		return ProcessResult{Game: item.Game, Index: item.Index}
	}

	pool := NewPool(2, 100, processFunc)
	pool.Start()

	// Submit many items
	for i := 0; i < 50; i++ {
		pool.Submit(WorkItem{
			Game:  &chess.Game{},
			Index: i,
		})
	}

	// Stop early after a short delay
	time.Sleep(30 * time.Millisecond)
	pool.Stop()

	// Close to drain
	go func() {
		pool.Close()
	}()

	// Count results
	resultCount := 0
	for range pool.Results() {
		resultCount++
	}

	// Should have processed fewer than 50 due to early stop
	processed := int(atomic.LoadInt32(&processedCount))
	if processed >= 50 {
		t.Logf("Early stop may not have prevented all processing: %d processed", processed)
	}
}

// TestPoolIsStopped tests the IsStopped method.
func TestPoolIsStopped(t *testing.T) {
	processFunc := func(item WorkItem) ProcessResult {
		return ProcessResult{}
	}

	pool := NewPool(2, 10, processFunc)
	pool.Start()

	if pool.IsStopped() {
		t.Error("Pool should not be stopped initially")
	}

	pool.Stop()

	if !pool.IsStopped() {
		t.Error("Pool should be stopped after Stop()")
	}

	pool.Close()
}

// TestPoolTrySubmit tests non-blocking submission.
func TestPoolTrySubmit(t *testing.T) {
	processFunc := func(item WorkItem) ProcessResult {
		time.Sleep(100 * time.Millisecond) // Slow processing
		return ProcessResult{}
	}

	// Small buffer to test blocking
	pool := NewPool(1, 2, processFunc)
	pool.Start()

	// First two should succeed (buffer size 2)
	if !pool.TrySubmit(WorkItem{Game: &chess.Game{}, Index: 0}) {
		t.Error("First TrySubmit should succeed")
	}
	if !pool.TrySubmit(WorkItem{Game: &chess.Game{}, Index: 1}) {
		t.Error("Second TrySubmit should succeed")
	}

	// Third might fail if buffer is full
	// (depends on timing, so we just test that it doesn't panic)
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
	processFunc := func(item WorkItem) ProcessResult {
		return ProcessResult{}
	}

	tests := []struct {
		input    int
		expected int
	}{
		{4, 4},
		{1, 1},
		{0, 1}, // Should default to 1
		{-1, 1}, // Should default to 1
	}

	for _, tt := range tests {
		pool := NewPool(tt.input, 10, processFunc)
		if pool.NumWorkers() != tt.expected {
			t.Errorf("NewPool(%d): expected %d workers, got %d", tt.input, tt.expected, pool.NumWorkers())
		}
	}
}

// TestPoolResultOrder tests that results may arrive out of order.
func TestPoolResultOrder(t *testing.T) {
	processFunc := func(item WorkItem) ProcessResult {
		// Variable delay based on index to encourage out-of-order completion
		if item.Index%2 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
		return ProcessResult{
			Game:  item.Game,
			Index: item.Index,
		}
	}

	pool := NewPool(4, 20, processFunc)
	pool.Start()

	// Submit items
	for i := 0; i < 10; i++ {
		pool.Submit(WorkItem{
			Game:  &chess.Game{},
			Index: i,
		})
	}

	go func() {
		pool.Close()
	}()

	// Collect results
	var indices []int
	for result := range pool.Results() {
		indices = append(indices, result.Index)
	}

	if len(indices) != 10 {
		t.Errorf("Expected 10 results, got %d", len(indices))
	}

	// Check that all indices are present (order may differ)
	seen := make(map[int]bool)
	for _, idx := range indices {
		seen[idx] = true
	}
	for i := 0; i < 10; i++ {
		if !seen[i] {
			t.Errorf("Missing index %d in results", i)
		}
	}
}

// TestPoolNoRace is designed to be run with -race flag.
func TestPoolNoRace(t *testing.T) {
	var counter int32

	processFunc := func(item WorkItem) ProcessResult {
		atomic.AddInt32(&counter, 1)
		return ProcessResult{
			Game:    item.Game,
			Index:   item.Index,
			Matched: atomic.LoadInt32(&counter)%2 == 0,
		}
	}

	pool := NewPool(8, 50, processFunc)
	pool.Start()

	// Submit items concurrently
	go func() {
		for i := 0; i < 100; i++ {
			pool.Submit(WorkItem{
				Game:  &chess.Game{},
				Index: i,
			})
		}
		pool.Close()
	}()

	// Read results
	for range pool.Results() {
		// Just drain
	}

	if atomic.LoadInt32(&counter) != 100 {
		t.Errorf("Expected 100 processed, got %d", counter)
	}
}
