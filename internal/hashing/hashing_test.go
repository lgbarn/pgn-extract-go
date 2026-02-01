package hashing

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

func TestZobristHash_IdenticalBoards_SameHash(t *testing.T) {
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

func TestZobristHash_DifferentPositions_DifferentHash(t *testing.T) {
	board1 := chess.NewBoard()
	board1.SetupInitialPosition()

	board2 := chess.NewBoard()
	board2.SetupInitialPosition()
	board2.Set('e', '2', chess.Empty)
	board2.Set('e', '4', chess.W(chess.Pawn))

	hash1 := GenerateZobristHash(board1)
	hash2 := GenerateZobristHash(board2)

	if hash1 == hash2 {
		t.Error("Different positions produced the same hash")
	}
}

func TestWeakHash_IdenticalBoards_SameHash(t *testing.T) {
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

func TestDuplicateDetector_CheckAndAdd(t *testing.T) {
	detector := NewDuplicateDetector(false, 0)

	board := chess.NewBoard()
	board.SetupInitialPosition()

	game := &chess.Game{
		Tags:  make(map[string]string),
		Moves: nil,
	}

	if detector.CheckAndAdd(game, board) {
		t.Error("First game was marked as duplicate")
	}

	if !detector.CheckAndAdd(game, board) {
		t.Error("Duplicate game was not detected")
	}

	if detector.DuplicateCount() != 1 {
		t.Errorf("Expected 1 duplicate, got %d", detector.DuplicateCount())
	}
}

func TestDuplicateDetector_DifferentGames(t *testing.T) {
	detector := NewDuplicateDetector(false, 0)

	board1 := chess.NewBoard()
	board1.SetupInitialPosition()
	game1 := &chess.Game{Tags: make(map[string]string)}

	board2 := chess.NewBoard()
	board2.SetupInitialPosition()
	board2.Set('e', '2', chess.Empty)
	board2.Set('e', '4', chess.W(chess.Pawn))
	game2 := &chess.Game{Tags: make(map[string]string)}

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

func TestDuplicateDetector_Reset(t *testing.T) {
	detector := NewDuplicateDetector(false, 0)

	board := chess.NewBoard()
	board.SetupInitialPosition()
	game := &chess.Game{Tags: make(map[string]string)}

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

func TestZobristHash_DifferentSideToMove_DifferentHash(t *testing.T) {
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

func TestDuplicateDetector_UnlimitedCapacity(t *testing.T) {
	detector := NewDuplicateDetector(false, 0)

	// Add many unique games by creating different piece configurations
	// We'll add games and verify the detector can grow without limit
	const attempts = 100
	uniqueCount := 0

	for i := 0; i < attempts; i++ {
		testBoard := chess.NewBoard()
		testBoard.SetupInitialPosition()

		// Create unique positions by removing different pieces
		testBoard.Set(chess.Col('a'+(i%8)), chess.Rank('1'+(i/8)%8), chess.Empty)
		if i%2 == 0 {
			testBoard.Set(chess.Col('h'-(i%8)), chess.Rank('8'-(i/8)%8), chess.Empty)
		}

		game := &chess.Game{Tags: make(map[string]string)}
		isDupe := detector.CheckAndAdd(game, testBoard)
		if !isDupe {
			uniqueCount++
		}
	}

	// With unlimited capacity, we should be able to add many unique games
	// Even with hash collisions, we should get at least 20 unique positions
	if detector.UniqueCount() < 20 {
		t.Errorf("Expected at least 20 unique games with unlimited capacity, got %d", detector.UniqueCount())
	}

	if detector.IsFull() {
		t.Error("Unlimited capacity detector should never be full")
	}

	// Verify the exact unique count matches
	if detector.UniqueCount() != uniqueCount {
		t.Errorf("UniqueCount() = %d, but we counted %d unique games", detector.UniqueCount(), uniqueCount)
	}
}

func TestDuplicateDetector_BoundedCapacity(t *testing.T) {
	const capacity = 5
	detector := NewDuplicateDetector(false, capacity)

	// Add enough attempts to try to exceed capacity
	// We'll track how many unique games we actually added
	uniqueAdded := 0
	for i := 0; i < 20; i++ {
		testBoard := chess.NewBoard()
		testBoard.SetupInitialPosition()
		// Create different positions
		testBoard.Set(chess.Col('a'+(i%8)), chess.Rank('1'+(i/8)%8), chess.Empty)
		if i%2 == 0 {
			testBoard.Set(chess.Col('h'-(i%8)), chess.Rank('8'-(i/8)%8), chess.Empty)
		}
		game := &chess.Game{Tags: make(map[string]string)}
		isDupe := detector.CheckAndAdd(game, testBoard)
		if !isDupe {
			uniqueAdded++
		}
	}

	// After adding enough games, detector should be full
	// UniqueCount may be >= capacity due to hash collisions
	// but IsFull should return true once we hit the limit
	if uniqueAdded >= capacity && !detector.IsFull() {
		t.Error("Detector should be full after adding enough unique games")
	}
}

func TestDuplicateDetector_DuplicatesDetectedWhenFull(t *testing.T) {
	const capacity = 3
	detector := NewDuplicateDetector(false, capacity)

	boards := make([]*chess.Board, 5)
	games := make([]*chess.Game, 5)

	// Create 5 unique positions
	for i := 0; i < 5; i++ {
		boards[i] = chess.NewBoard()
		boards[i].SetupInitialPosition()
		col := chess.Col('a' + i)
		boards[i].Set(col, '2', chess.Empty)
		games[i] = &chess.Game{Tags: make(map[string]string)}
	}

	// Add first 3 games (fills detector to capacity)
	for i := 0; i < 3; i++ {
		isDupe := detector.CheckAndAdd(games[i], boards[i])
		if isDupe {
			t.Errorf("Game %d should not be a duplicate", i)
		}
	}

	if detector.UniqueCount() != capacity {
		t.Errorf("Expected %d unique games, got %d", capacity, detector.UniqueCount())
	}

	// Add games 4 and 5 (capacity is full, these won't be stored)
	for i := 3; i < 5; i++ {
		isDupe := detector.CheckAndAdd(games[i], boards[i])
		if isDupe {
			t.Errorf("Game %d should not be a duplicate", i)
		}
	}

	// Detector should still have only 3 unique games
	if detector.UniqueCount() != capacity {
		t.Errorf("Expected %d unique games after adding more, got %d", capacity, detector.UniqueCount())
	}

	// Now re-add one of the first 3 games - should detect as duplicate
	isDupe := detector.CheckAndAdd(games[1], boards[1])
	if !isDupe {
		t.Error("Game 1 should be detected as duplicate")
	}

	if detector.DuplicateCount() != 1 {
		t.Errorf("Expected 1 duplicate, got %d", detector.DuplicateCount())
	}

	// Try to add game 4 again - should NOT be detected as duplicate
	// because it wasn't stored (capacity was full)
	isDupe = detector.CheckAndAdd(games[4], boards[4])
	if isDupe {
		t.Error("Game 4 should not be detected as duplicate (was never stored)")
	}

	if detector.DuplicateCount() != 1 {
		t.Errorf("Expected 1 duplicate still, got %d", detector.DuplicateCount())
	}
}

func TestDuplicateDetector_IsFull(t *testing.T) {
	tests := []struct {
		name       string
		capacity   int
		numGames   int
		expectFull bool
	}{
		{
			name:       "unlimited capacity never full",
			capacity:   0,
			numGames:   100,
			expectFull: false,
		},
		{
			name:       "bounded capacity not yet full",
			capacity:   20,
			numGames:   5,
			expectFull: false,
		},
		{
			name:       "bounded capacity over limit",
			capacity:   5,
			numGames:   50,
			expectFull: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDuplicateDetector(false, tt.capacity)

			uniqueAdded := 0
			for i := 0; i < tt.numGames && (tt.capacity == 0 || uniqueAdded < tt.numGames); i++ {
				testBoard := chess.NewBoard()
				testBoard.SetupInitialPosition()
				// Create more variation to reduce hash collisions
				col := chess.Col('a' + i%8)
				rank := chess.Rank('1' + i/8)
				testBoard.Set(col, rank, chess.Empty)
				if i%2 == 0 {
					testBoard.Set(chess.Col('h'-(i%8)), chess.Rank('8'-(i/8)%8), chess.Empty)
				}
				game := &chess.Game{Tags: make(map[string]string)}
				isDupe := detector.CheckAndAdd(game, testBoard)
				if !isDupe {
					uniqueAdded++
				}
			}

			if detector.IsFull() != tt.expectFull {
				t.Errorf("IsFull() = %v, want %v (added %d unique games)", detector.IsFull(), tt.expectFull, uniqueAdded)
			}
		})
	}
}

// TestDuplicateDetector_BehaviorUnchanged_BelowCapacity verifies that
// duplicate detection behavior is unchanged when operating below capacity.
func TestDuplicateDetector_BehaviorUnchanged_BelowCapacity(t *testing.T) {
	const capacity = 1000
	const numGames = 100

	detector := NewDuplicateDetector(false, capacity)

	// Create and add 100 unique games
	uniqueGames := make([]*chess.Game, numGames)
	uniqueBoards := make([]*chess.Board, numGames)

	duplicatesOnFirstAdd := 0
	for i := 0; i < numGames; i++ {
		board := chess.NewBoard()
		board.SetupInitialPosition()

		// Create unique positions by moving pieces around to create truly distinct positions
		// Remove a piece from the first rank based on index
		col := chess.Col('a' + i%8)
		board.Set(col, '1', chess.Empty)

		// Also remove a pawn to create more variation
		col2 := chess.Col('a' + (i/8)%8)
		board.Set(col2, '2', chess.Empty)

		game := &chess.Game{Tags: make(map[string]string)}

		// First add - should not be detected as duplicate
		isDupe := detector.CheckAndAdd(game, board)
		if isDupe {
			duplicatesOnFirstAdd++
		}

		uniqueGames[i] = game
		uniqueBoards[i] = board
	}

	// Some hash collisions are expected due to limited variation
	actualUnique := detector.UniqueCount()
	if actualUnique < 20 {
		t.Errorf("UniqueCount=%d is too low (expected at least 20)", actualUnique)
	}

	if detector.IsFull() {
		t.Errorf("Detector should not be full: UniqueCount=%d, capacity=%d", detector.UniqueCount(), capacity)
	}

	initialDuplicateCount := detector.DuplicateCount()

	// Now add duplicates of each unique game - should all be detected
	duplicatesDetected := 0
	for i := 0; i < numGames; i++ {
		if detector.CheckAndAdd(uniqueGames[i], uniqueBoards[i]) {
			duplicatesDetected++
		}
	}

	// All second adds should be duplicates (even those that collided on first add)
	if duplicatesDetected != numGames {
		t.Errorf("Detected %d duplicates on second add, want %d", duplicatesDetected, numGames)
	}

	// Verify final duplicate count
	expectedDuplicates := initialDuplicateCount + numGames
	if detector.DuplicateCount() != expectedDuplicates {
		t.Errorf("DuplicateCount=%d, want %d", detector.DuplicateCount(), expectedDuplicates)
	}

	// UniqueCount should remain unchanged
	if detector.UniqueCount() != actualUnique {
		t.Errorf("After duplicates: UniqueCount=%d, want %d (unchanged)", detector.UniqueCount(), actualUnique)
	}
}

// TestDuplicateDetector_BehaviorUnchanged_Unlimited verifies that
// duplicate detection with unlimited capacity works correctly.
func TestDuplicateDetector_BehaviorUnchanged_Unlimited(t *testing.T) {
	const numGames = 500

	detector := NewDuplicateDetector(false, 0) // unlimited capacity

	// Create and add 500 unique games
	uniqueGames := make([]*chess.Game, numGames)
	uniqueBoards := make([]*chess.Board, numGames)

	for i := 0; i < numGames; i++ {
		board := chess.NewBoard()
		board.SetupInitialPosition()

		// Create unique positions by removing pieces from different columns
		col := chess.Col('a' + i%8)
		board.Set(col, '1', chess.Empty)

		// Remove pawn for additional variation
		col2 := chess.Col('a' + (i/8)%8)
		board.Set(col2, '2', chess.Empty)

		game := &chess.Game{Tags: make(map[string]string)}

		// First add - may have some hash collisions due to limited position variation
		detector.CheckAndAdd(game, board)

		uniqueGames[i] = game
		uniqueBoards[i] = board
	}

	// Verify detector never becomes full
	if detector.IsFull() {
		t.Error("Unlimited capacity detector should never be full")
	}

	// UniqueCount will be less than numGames due to hash collisions
	actualUnique := detector.UniqueCount()
	if actualUnique < 20 {
		t.Errorf("UniqueCount=%d is suspiciously low for %d games", actualUnique, numGames)
	}

	initialDuplicateCount := detector.DuplicateCount()

	// Add duplicates of all games - all should be detected now
	duplicatesDetected := 0
	for i := 0; i < numGames; i++ {
		if detector.CheckAndAdd(uniqueGames[i], uniqueBoards[i]) {
			duplicatesDetected++
		}
	}

	// All second adds should be detected as duplicates
	if duplicatesDetected != numGames {
		t.Errorf("Detected %d duplicates on second add, want %d", duplicatesDetected, numGames)
	}

	// Verify final duplicate count
	expectedDuplicates := initialDuplicateCount + numGames
	if detector.DuplicateCount() != expectedDuplicates {
		t.Errorf("DuplicateCount=%d, want %d", detector.DuplicateCount(), expectedDuplicates)
	}

	// UniqueCount should remain unchanged after adding duplicates
	if detector.UniqueCount() != actualUnique {
		t.Errorf("After duplicates: UniqueCount=%d, want %d (unchanged)", detector.UniqueCount(), actualUnique)
	}
}
