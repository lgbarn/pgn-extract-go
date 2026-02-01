// Package hashing provides duplicate detection for chess games.
package hashing

import (
	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// DuplicateChecker defines the interface for duplicate detection implementations.
// Both DuplicateDetector and ThreadSafeDuplicateDetector implement this interface.
type DuplicateChecker interface {
	// CheckAndAdd checks if a game is a duplicate and adds it to the hash table.
	// Returns true if the game is a duplicate.
	CheckAndAdd(game *chess.Game, board *chess.Board) bool
	// DuplicateCount returns the number of duplicates detected.
	DuplicateCount() int
	// UniqueCount returns the number of unique games.
	UniqueCount() int
}

// DuplicateDetector tracks seen positions for duplicate game detection.
type DuplicateDetector struct {
	hashTable      map[uint64][]GameSignature
	useExactMatch  bool
	duplicateCount int
	maxCapacity    int // 0 = unlimited
}

// GameSignature stores identifying information about a game.
type GameSignature struct {
	Hash      uint64
	MoveCount int
	WeakHash  chess.HashCode
}

// NewDuplicateDetector creates a new duplicate detector.
// maxCapacity of 0 means unlimited capacity.
func NewDuplicateDetector(exactMatch bool, maxCapacity int) *DuplicateDetector {
	return &DuplicateDetector{
		hashTable:     make(map[uint64][]GameSignature),
		useExactMatch: exactMatch,
		maxCapacity:   maxCapacity,
	}
}

// CheckAndAdd checks if a game is a duplicate and adds it to the hash table.
// Returns true if the game is a duplicate.
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
	if existing, ok := d.hashTable[hash]; ok {
		for _, existingSig := range existing {
			if d.signaturesMatch(sig, existingSig) {
				d.duplicateCount++
				return true
			}
		}
	}

	// Add to hash table if not at capacity
	if d.maxCapacity <= 0 || len(d.hashTable) < d.maxCapacity {
		d.hashTable[hash] = append(d.hashTable[hash], sig)
	}
	return false
}

// signaturesMatch checks if two game signatures match.
func (d *DuplicateDetector) signaturesMatch(a, b GameSignature) bool {
	if a.Hash != b.Hash || a.WeakHash != b.WeakHash {
		return false
	}
	return !d.useExactMatch || a.MoveCount == b.MoveCount
}

// DuplicateCount returns the number of duplicates detected.
func (d *DuplicateDetector) DuplicateCount() int {
	return d.duplicateCount
}

// UniqueCount returns the number of unique games.
func (d *DuplicateDetector) UniqueCount() int {
	count := 0
	for _, sigs := range d.hashTable {
		count += len(sigs)
	}
	return count
}

// Reset clears the hash table.
func (d *DuplicateDetector) Reset() {
	d.hashTable = make(map[uint64][]GameSignature)
	d.duplicateCount = 0
}

// IsFull returns true if the detector has reached its capacity limit.
// Always returns false for unlimited capacity (maxCapacity = 0).
func (d *DuplicateDetector) IsFull() bool {
	return d.maxCapacity > 0 && len(d.hashTable) >= d.maxCapacity
}

// countMoves counts the number of half-moves in a game.
func countMoves(game *chess.Game) int {
	count := 0
	for move := game.Moves; move != nil; move = move.Next {
		count++
	}
	return count
}

// HashType specifies what to hash for duplicate detection.
type HashType int

const (
	// HashFinalPosition hashes only the final position
	HashFinalPosition HashType = iota
	// HashAllPositions hashes all positions throughout the game
	HashAllPositions
	// HashMoveSequence hashes the actual move sequence
	HashMoveSequence
)

// GameHasher provides different hashing strategies for games.
type GameHasher struct {
	hashType HashType
}

// NewGameHasher creates a new game hasher with the specified strategy.
func NewGameHasher(ht HashType) *GameHasher {
	return &GameHasher{hashType: ht}
}

// HashGame generates a hash for the game based on the hash type.
func (gh *GameHasher) HashGame(game *chess.Game, board *chess.Board) uint64 {
	switch gh.hashType {
	case HashAllPositions:
		// This would require replaying the game
		// For now, fall back to final position
		return GenerateZobristHash(board)
	case HashMoveSequence:
		return gh.hashMoveSequence(game)
	default:
		return GenerateZobristHash(board)
	}
}

// hashMoveSequence creates a hash from the move texts.
func (gh *GameHasher) hashMoveSequence(game *chess.Game) uint64 {
	var hash uint64
	const multiplier = 31

	for move := game.Moves; move != nil; move = move.Next {
		for _, c := range move.Text {
			hash = hash*multiplier + uint64(c)
		}
	}

	return hash
}

// FuzzyDuplicateDetector detects duplicates based on position at a specific ply depth.
type FuzzyDuplicateDetector struct {
	depth          int
	positionHashes map[uint64][]gameAtDepth
	duplicateCount int
}

// gameAtDepth stores game info along with position at the fuzzy depth.
type gameAtDepth struct {
	hash      uint64
	moveCount int
}

// NewFuzzyDuplicateDetector creates a new fuzzy duplicate detector with the given depth.
func NewFuzzyDuplicateDetector(depth int) *FuzzyDuplicateDetector {
	return &FuzzyDuplicateDetector{
		depth:          depth,
		positionHashes: make(map[uint64][]gameAtDepth),
	}
}

// CheckAndAdd checks if a game is a duplicate at the fuzzy depth and adds it.
// Returns true if this is a duplicate.
func (d *FuzzyDuplicateDetector) CheckAndAdd(game *chess.Game, positions []uint64) bool {
	// Get position hash at the specified depth
	var hashAtDepth uint64
	if d.depth > 0 && d.depth < len(positions) {
		hashAtDepth = positions[d.depth]
	} else if len(positions) > 0 {
		// Use last position if depth exceeds game length
		hashAtDepth = positions[len(positions)-1]
	} else {
		return false
	}

	info := gameAtDepth{
		hash:      hashAtDepth,
		moveCount: len(positions) - 1, // -1 because positions includes initial position
	}

	// Check for duplicates
	if existing, ok := d.positionHashes[hashAtDepth]; ok {
		for _, e := range existing {
			if e.hash == info.hash {
				d.duplicateCount++
				return true
			}
		}
	}

	// Add to hash table
	d.positionHashes[hashAtDepth] = append(d.positionHashes[hashAtDepth], info)
	return false
}

// DuplicateCount returns the number of fuzzy duplicates detected.
func (d *FuzzyDuplicateDetector) DuplicateCount() int {
	return d.duplicateCount
}

// Depth returns the configured fuzzy depth.
func (d *FuzzyDuplicateDetector) Depth() int {
	return d.depth
}

// SetupDuplicateDetector tracks seen starting positions for same-setup deletion.
type SetupDuplicateDetector struct {
	seenSetups     map[string]bool // FEN string -> seen
	duplicateCount int
}

// NewSetupDuplicateDetector creates a new setup duplicate detector.
func NewSetupDuplicateDetector() *SetupDuplicateDetector {
	return &SetupDuplicateDetector{
		seenSetups: make(map[string]bool),
	}
}

// CheckAndAdd checks if a game's starting position is a duplicate and tracks it.
// Returns true if this starting position was already seen.
func (d *SetupDuplicateDetector) CheckAndAdd(game *chess.Game) bool {
	// Get the starting FEN, or use standard position if not specified
	fen := game.FEN()
	if fen == "" {
		fen = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" // Standard starting position
	}

	// Check if we've seen this setup before
	if d.seenSetups[fen] {
		d.duplicateCount++
		return true
	}

	// Mark as seen
	d.seenSetups[fen] = true
	return false
}

// DuplicateCount returns the number of same-setup duplicates detected.
func (d *SetupDuplicateDetector) DuplicateCount() int {
	return d.duplicateCount
}

// Reset clears the setup tracker.
func (d *SetupDuplicateDetector) Reset() {
	d.seenSetups = make(map[string]bool)
	d.duplicateCount = 0
}
