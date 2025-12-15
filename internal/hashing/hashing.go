// Package hashing provides duplicate detection for chess games.
package hashing

import (
	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// DuplicateDetector tracks seen positions for duplicate game detection.
type DuplicateDetector struct {
	// hashTable stores seen hash codes
	hashTable map[uint64][]GameSignature
	// useExactMatch uses full Zobrist hash (slower but more accurate)
	useExactMatch bool
	// duplicateCount tracks number of duplicates found
	duplicateCount int
}

// GameSignature stores identifying information about a game.
type GameSignature struct {
	// Hash is the Zobrist hash of the final position
	Hash uint64
	// MoveCount is the number of half-moves in the game
	MoveCount int
	// WeakHash is a fast hash for quick comparison
	WeakHash chess.HashCode
}

// NewDuplicateDetector creates a new duplicate detector.
func NewDuplicateDetector(exactMatch bool) *DuplicateDetector {
	return &DuplicateDetector{
		hashTable:     make(map[uint64][]GameSignature),
		useExactMatch: exactMatch,
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

	// Add to hash table
	d.hashTable[hash] = append(d.hashTable[hash], sig)
	return false
}

// signaturesMatch checks if two game signatures match.
func (d *DuplicateDetector) signaturesMatch(a, b GameSignature) bool {
	// Primary check: Zobrist hash must match (already implied by hash table key)
	if a.Hash != b.Hash {
		return false
	}

	// Secondary check: weak hash for additional confidence
	if a.WeakHash != b.WeakHash {
		return false
	}

	// Optional: check move count
	if d.useExactMatch && a.MoveCount != b.MoveCount {
		return false
	}

	return true
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
	var hash uint64 = 0
	multiplier := uint64(31)

	for move := game.Moves; move != nil; move = move.Next {
		// Simple string hash for move text
		for _, c := range move.Text {
			hash = hash*multiplier + uint64(c)
		}
	}

	return hash
}
