package hashing

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

func TestZobristHashConsistency(t *testing.T) {
	// Create two identical boards and verify they produce the same hash
	board1 := chess.NewBoard()
	board1.SetupInitialPosition()

	board2 := chess.NewBoard()
	board2.SetupInitialPosition()

	hash1 := GenerateZobristHash(board1)
	hash2 := GenerateZobristHash(board2)

	if hash1 != hash2 {
		t.Errorf("Identical boards produced different hashes: %x != %x", hash1, hash2)
	}
}

func TestZobristHashDifferentPositions(t *testing.T) {
	// Initial position
	board1 := chess.NewBoard()
	board1.SetupInitialPosition()

	// Modified position (move a pawn)
	board2 := chess.NewBoard()
	board2.SetupInitialPosition()
	// Manually move e2 to e4
	board2.Set('e', '2', chess.Empty)
	board2.Set('e', '4', chess.W(chess.Pawn))

	hash1 := GenerateZobristHash(board1)
	hash2 := GenerateZobristHash(board2)

	if hash1 == hash2 {
		t.Error("Different positions produced the same hash")
	}
}

func TestWeakHashConsistency(t *testing.T) {
	board1 := chess.NewBoard()
	board1.SetupInitialPosition()

	board2 := chess.NewBoard()
	board2.SetupInitialPosition()

	hash1 := WeakHash(board1)
	hash2 := WeakHash(board2)

	if hash1 != hash2 {
		t.Errorf("Identical boards produced different weak hashes: %x != %x", hash1, hash2)
	}
}

func TestDuplicateDetector(t *testing.T) {
	detector := NewDuplicateDetector(false)

	board := chess.NewBoard()
	board.SetupInitialPosition()

	game := &chess.Game{
		Tags:  make(map[string]string),
		Moves: nil,
	}

	// First game should not be a duplicate
	if detector.CheckAndAdd(game, board) {
		t.Error("First game was marked as duplicate")
	}

	// Same game should be a duplicate
	if !detector.CheckAndAdd(game, board) {
		t.Error("Duplicate game was not detected")
	}

	if detector.DuplicateCount() != 1 {
		t.Errorf("Expected 1 duplicate, got %d", detector.DuplicateCount())
	}
}

func TestDuplicateDetectorDifferentGames(t *testing.T) {
	detector := NewDuplicateDetector(false)

	// Game 1 - initial position
	board1 := chess.NewBoard()
	board1.SetupInitialPosition()
	game1 := &chess.Game{
		Tags:  make(map[string]string),
		Moves: nil,
	}

	// Game 2 - different position
	board2 := chess.NewBoard()
	board2.SetupInitialPosition()
	board2.Set('e', '2', chess.Empty)
	board2.Set('e', '4', chess.W(chess.Pawn))
	game2 := &chess.Game{
		Tags:  make(map[string]string),
		Moves: nil,
	}

	// Neither should be duplicates
	if detector.CheckAndAdd(game1, board1) {
		t.Error("Game 1 was incorrectly marked as duplicate")
	}

	if detector.CheckAndAdd(game2, board2) {
		t.Error("Game 2 was incorrectly marked as duplicate")
	}

	if detector.DuplicateCount() != 0 {
		t.Errorf("Expected 0 duplicates, got %d", detector.DuplicateCount())
	}

	if detector.UniqueCount() != 2 {
		t.Errorf("Expected 2 unique games, got %d", detector.UniqueCount())
	}
}

func TestDuplicateDetectorReset(t *testing.T) {
	detector := NewDuplicateDetector(false)

	board := chess.NewBoard()
	board.SetupInitialPosition()
	game := &chess.Game{
		Tags:  make(map[string]string),
		Moves: nil,
	}

	detector.CheckAndAdd(game, board)
	detector.CheckAndAdd(game, board)

	if detector.DuplicateCount() != 1 {
		t.Errorf("Expected 1 duplicate before reset, got %d", detector.DuplicateCount())
	}

	detector.Reset()

	if detector.DuplicateCount() != 0 {
		t.Errorf("Expected 0 duplicates after reset, got %d", detector.DuplicateCount())
	}

	if detector.UniqueCount() != 0 {
		t.Errorf("Expected 0 unique games after reset, got %d", detector.UniqueCount())
	}
}

func TestSideToMoveAffectsHash(t *testing.T) {
	board1 := chess.NewBoard()
	board1.SetupInitialPosition()
	board1.ToMove = chess.White

	board2 := chess.NewBoard()
	board2.SetupInitialPosition()
	board2.ToMove = chess.Black

	hash1 := GenerateZobristHash(board1)
	hash2 := GenerateZobristHash(board2)

	if hash1 == hash2 {
		t.Error("Same position with different side to move should have different hashes")
	}
}
