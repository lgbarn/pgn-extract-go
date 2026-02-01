// Package hashing provides duplicate detection for chess games.
package hashing

import (
	"sync"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// ThreadSafeDuplicateDetector wraps DuplicateDetector with mutex protection for concurrent access.
type ThreadSafeDuplicateDetector struct {
	detector *DuplicateDetector
	mu       sync.RWMutex
}

// NewThreadSafeDuplicateDetector creates a new thread-safe detector.
// maxCapacity of 0 means unlimited capacity.
func NewThreadSafeDuplicateDetector(exactMatch bool, maxCapacity int) *ThreadSafeDuplicateDetector {
	return &ThreadSafeDuplicateDetector{
		detector: NewDuplicateDetector(exactMatch, maxCapacity),
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

// IsFull returns true if the detector has reached its capacity limit.
// Always returns false for unlimited capacity (maxCapacity = 0).
func (d *ThreadSafeDuplicateDetector) IsFull() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.detector.IsFull()
}
