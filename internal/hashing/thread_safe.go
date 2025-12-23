// Package hashing provides duplicate detection for chess games.
package hashing

import (
	"sync"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// ThreadSafeDuplicateDetector wraps DuplicateDetector with mutex protection
// for concurrent access from multiple goroutines.
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

// CheckAndAdd checks if a game is a duplicate and adds it to the hash table atomically.
// Returns true if the game is a duplicate.
func (d *ThreadSafeDuplicateDetector) CheckAndAdd(game *chess.Game, board *chess.Board) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.detector.CheckAndAdd(game, board)
}

// DuplicateCount returns the current duplicate count.
func (d *ThreadSafeDuplicateDetector) DuplicateCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.detector.DuplicateCount()
}

// UniqueCount returns the current unique game count.
func (d *ThreadSafeDuplicateDetector) UniqueCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.detector.UniqueCount()
}

// LoadFromDetector copies entries from an existing detector (for checkFile support).
// This should be called before concurrent use.
func (d *ThreadSafeDuplicateDetector) LoadFromDetector(other *DuplicateDetector) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// Copy the hash table entries
	for hash, sigs := range other.hashTable {
		d.detector.hashTable[hash] = append(d.detector.hashTable[hash], sigs...)
	}
}
