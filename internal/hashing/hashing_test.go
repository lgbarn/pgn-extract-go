package hashing

import (
	"os"
	"path/filepath"
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
	detector := NewDuplicateDetector(false)

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
	detector := NewDuplicateDetector(false)

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
	detector := NewDuplicateDetector(false)

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

// ============== File I/O Tests ==============

func TestDuplicateDetector_SaveToFile_Success(t *testing.T) {
	detector := NewDuplicateDetector(false)

	board := chess.NewBoard()
	board.SetupInitialPosition()
	game := &chess.Game{Tags: make(map[string]string)}

	detector.CheckAndAdd(game, board)

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_hashes.bin")

	err := detector.SaveToFile(filename)
	if err != nil {
		t.Errorf("SaveToFile() error = %v, want nil", err)
	}

	// Verify file was created
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("SaveToFile() did not create file")
	}
}

func TestDuplicateDetector_SaveToFile_EmptyDetector(t *testing.T) {
	detector := NewDuplicateDetector(false)

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "empty_hashes.bin")

	err := detector.SaveToFile(filename)
	if err != nil {
		t.Errorf("SaveToFile() error = %v, want nil", err)
	}

	// Verify file was created
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("SaveToFile() did not create file for empty detector")
	}
}

func TestDuplicateDetector_LoadFromFile_Success(t *testing.T) {
	// Create and save a detector
	detector1 := NewDuplicateDetector(false)

	board := chess.NewBoard()
	board.SetupInitialPosition()
	game := &chess.Game{Tags: make(map[string]string)}

	detector1.CheckAndAdd(game, board)

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_hashes.bin")

	if err := detector1.SaveToFile(filename); err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	// Load into a new detector
	detector2 := NewDuplicateDetector(false)
	if err := detector2.LoadFromFile(filename); err != nil {
		t.Errorf("LoadFromFile() error = %v, want nil", err)
	}

	// The second detector should detect the game as a duplicate
	if !detector2.CheckAndAdd(game, board) {
		t.Error("LoadFromFile() did not restore hash table - game not detected as duplicate")
	}
}

func TestDuplicateDetector_LoadFromFile_NonExistent(t *testing.T) {
	detector := NewDuplicateDetector(false)

	err := detector.LoadFromFile("/nonexistent/path/file.bin")
	if err != nil {
		t.Errorf("LoadFromFile() should return nil for non-existent file, got error = %v", err)
	}
}

func TestDuplicateDetector_LoadFromFile_Corrupted(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "corrupted.bin")

	// Write some garbage data
	if err := os.WriteFile(filename, []byte("corrupted data"), 0644); err != nil {
		t.Fatalf("failed to create corrupted file: %v", err)
	}

	detector := NewDuplicateDetector(false)
	err := detector.LoadFromFile(filename)
	if err == nil {
		t.Error("LoadFromFile() should return error for corrupted file")
	}
}

func TestDuplicateDetector_SaveLoadRoundTrip(t *testing.T) {
	detector1 := NewDuplicateDetector(true) // exactMatch mode

	// Add multiple different positions
	for i := 0; i < 5; i++ {
		board := chess.NewBoard()
		board.SetupInitialPosition()
		// Make each position unique by moving a piece
		board.Set('a'+chess.Col(i), '2', chess.Empty)
		board.Set('a'+chess.Col(i), '3', chess.W(chess.Pawn))

		game := &chess.Game{Tags: make(map[string]string)}
		for j := 0; j < i+1; j++ {
			move := &chess.Move{Text: "dummy"}
			if game.Moves == nil {
				game.Moves = move
			}
		}
		detector1.CheckAndAdd(game, board)
	}

	if detector1.UniqueCount() != 5 {
		t.Fatalf("expected 5 unique games, got %d", detector1.UniqueCount())
	}

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "roundtrip.bin")

	if err := detector1.SaveToFile(filename); err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	// Load into new detector
	detector2 := NewDuplicateDetector(true)
	if err := detector2.LoadFromFile(filename); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	// All games should be detected as duplicates
	for i := 0; i < 5; i++ {
		board := chess.NewBoard()
		board.SetupInitialPosition()
		board.Set('a'+chess.Col(i), '2', chess.Empty)
		board.Set('a'+chess.Col(i), '3', chess.W(chess.Pawn))

		game := &chess.Game{Tags: make(map[string]string)}
		for j := 0; j < i+1; j++ {
			move := &chess.Move{Text: "dummy"}
			if game.Moves == nil {
				game.Moves = move
			}
		}

		if !detector2.CheckAndAdd(game, board) {
			t.Errorf("game %d was not detected as duplicate after roundtrip", i)
		}
	}
}

func TestDuplicateDetector_SaveLoadRoundTrip_LargeDataset(t *testing.T) {
	detector1 := NewDuplicateDetector(false)

	// Add many positions
	for i := 0; i < 100; i++ {
		board := chess.NewBoard()
		board.SetupInitialPosition()
		// Create unique positions
		col := 'a' + chess.Col(i%8)
		rank := chess.Rank('2')
		board.Set(col, rank, chess.Empty)
		board.Set(col, chess.Rank('3'+i/8), chess.W(chess.Pawn))

		game := &chess.Game{Tags: make(map[string]string)}
		detector1.CheckAndAdd(game, board)
	}

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "large.bin")

	if err := detector1.SaveToFile(filename); err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	detector2 := NewDuplicateDetector(false)
	if err := detector2.LoadFromFile(filename); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	// Verify by checking that unique count is preserved
	originalUnique := detector1.UniqueCount()
	if originalUnique == 0 {
		t.Fatal("original detector should have unique games")
	}

	// Check that we can detect duplicates after load
	board := chess.NewBoard()
	board.SetupInitialPosition()
	board.Set('a', '2', chess.Empty)
	board.Set('a', '3', chess.W(chess.Pawn))
	game := &chess.Game{Tags: make(map[string]string)}

	if !detector2.CheckAndAdd(game, board) {
		t.Error("first position should be detected as duplicate after loading")
	}
}

// ============== FuzzyDuplicateDetector Tests ==============

func TestFuzzyDuplicateDetector_New(t *testing.T) {
	detector := NewFuzzyDuplicateDetector(10)

	if detector == nil {
		t.Fatal("NewFuzzyDuplicateDetector() returned nil")
	}
	if detector.Depth() != 10 {
		t.Errorf("Depth() = %d, want 10", detector.Depth())
	}
	if detector.DuplicateCount() != 0 {
		t.Errorf("DuplicateCount() = %d, want 0", detector.DuplicateCount())
	}
}

func TestFuzzyDuplicateDetector_NewWithDepth(t *testing.T) {
	tests := []struct {
		name  string
		depth int
	}{
		{"depth 0", 0},
		{"depth 5", 5},
		{"depth 20", 20},
		{"depth 100", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			detector := NewFuzzyDuplicateDetector(tt.depth)
			if detector.Depth() != tt.depth {
				t.Errorf("Depth() = %d, want %d", detector.Depth(), tt.depth)
			}
		})
	}
}

func TestFuzzyDuplicateDetector_CheckAndAdd_FirstGame(t *testing.T) {
	detector := NewFuzzyDuplicateDetector(5)

	game := &chess.Game{Tags: make(map[string]string)}
	positions := []uint64{123456, 234567, 345678, 456789, 567890, 678901}

	isDup := detector.CheckAndAdd(game, positions)
	if isDup {
		t.Error("first game should not be a duplicate")
	}
	if detector.DuplicateCount() != 0 {
		t.Errorf("DuplicateCount() = %d, want 0", detector.DuplicateCount())
	}
}

func TestFuzzyDuplicateDetector_CheckAndAdd_Duplicate(t *testing.T) {
	detector := NewFuzzyDuplicateDetector(3)

	game := &chess.Game{Tags: make(map[string]string)}
	positions := []uint64{100, 200, 300, 400, 500}

	// Add first game
	if detector.CheckAndAdd(game, positions) {
		t.Error("first game should not be duplicate")
	}

	// Same positions should be detected as duplicate
	if !detector.CheckAndAdd(game, positions) {
		t.Error("same game should be detected as duplicate")
	}

	if detector.DuplicateCount() != 1 {
		t.Errorf("DuplicateCount() = %d, want 1", detector.DuplicateCount())
	}
}

func TestFuzzyDuplicateDetector_CheckAndAdd_Similar(t *testing.T) {
	detector := NewFuzzyDuplicateDetector(3)

	game := &chess.Game{Tags: make(map[string]string)}

	// Two games with same position at depth 3 but different later positions
	positions1 := []uint64{100, 200, 300, 400, 500, 600}
	positions2 := []uint64{100, 200, 300, 400, 999, 888} // Same at index 3

	detector.CheckAndAdd(game, positions1)

	// Since positions at depth 3 are the same (400), should be duplicate
	if !detector.CheckAndAdd(game, positions2) {
		t.Error("games with same position at fuzzy depth should be duplicates")
	}
}

func TestFuzzyDuplicateDetector_ShortGame(t *testing.T) {
	detector := NewFuzzyDuplicateDetector(10)

	game := &chess.Game{Tags: make(map[string]string)}
	// Game shorter than fuzzy depth - should use last position
	positions := []uint64{100, 200, 300}

	isDup := detector.CheckAndAdd(game, positions)
	if isDup {
		t.Error("first short game should not be duplicate")
	}

	// Same short game
	if !detector.CheckAndAdd(game, positions) {
		t.Error("same short game should be duplicate")
	}
}

func TestFuzzyDuplicateDetector_EmptyGame(t *testing.T) {
	detector := NewFuzzyDuplicateDetector(5)

	game := &chess.Game{Tags: make(map[string]string)}
	positions := []uint64{} // Empty positions

	isDup := detector.CheckAndAdd(game, positions)
	if isDup {
		t.Error("empty game should return false (not duplicate)")
	}
}

// ============== SetupDuplicateDetector Tests ==============

func TestSetupDuplicateDetector_New(t *testing.T) {
	detector := NewSetupDuplicateDetector()

	if detector == nil {
		t.Fatal("NewSetupDuplicateDetector() returned nil")
	}
	if detector.DuplicateCount() != 0 {
		t.Errorf("DuplicateCount() = %d, want 0", detector.DuplicateCount())
	}
}

func TestSetupDuplicateDetector_StandardPosition(t *testing.T) {
	detector := NewSetupDuplicateDetector()

	// Game with no FEN tag uses standard starting position
	game1 := &chess.Game{Tags: make(map[string]string)}
	game2 := &chess.Game{Tags: make(map[string]string)}

	if detector.CheckAndAdd(game1) {
		t.Error("first game should not be duplicate")
	}

	if !detector.CheckAndAdd(game2) {
		t.Error("second game with same standard position should be duplicate")
	}

	if detector.DuplicateCount() != 1 {
		t.Errorf("DuplicateCount() = %d, want 1", detector.DuplicateCount())
	}
}

func TestSetupDuplicateDetector_CustomFEN(t *testing.T) {
	detector := NewSetupDuplicateDetector()

	// Game with custom FEN
	game1 := &chess.Game{
		Tags: map[string]string{
			"FEN": "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		},
	}
	// Game with different custom FEN
	game2 := &chess.Game{
		Tags: map[string]string{
			"FEN": "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2",
		},
	}

	if detector.CheckAndAdd(game1) {
		t.Error("first custom FEN game should not be duplicate")
	}

	if detector.CheckAndAdd(game2) {
		t.Error("second game with different FEN should not be duplicate")
	}

	if detector.DuplicateCount() != 0 {
		t.Errorf("DuplicateCount() = %d, want 0", detector.DuplicateCount())
	}
}

func TestSetupDuplicateDetector_DuplicateSetup(t *testing.T) {
	detector := NewSetupDuplicateDetector()

	customFEN := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"

	game1 := &chess.Game{
		Tags: map[string]string{"FEN": customFEN},
	}
	game2 := &chess.Game{
		Tags: map[string]string{"FEN": customFEN},
	}

	detector.CheckAndAdd(game1)

	if !detector.CheckAndAdd(game2) {
		t.Error("game with same custom FEN should be duplicate")
	}
}

func TestSetupDuplicateDetector_Reset(t *testing.T) {
	detector := NewSetupDuplicateDetector()

	game := &chess.Game{Tags: make(map[string]string)}
	detector.CheckAndAdd(game)
	detector.CheckAndAdd(game)

	if detector.DuplicateCount() != 1 {
		t.Errorf("DuplicateCount() = %d before reset, want 1", detector.DuplicateCount())
	}

	detector.Reset()

	if detector.DuplicateCount() != 0 {
		t.Errorf("DuplicateCount() = %d after reset, want 0", detector.DuplicateCount())
	}

	// After reset, same game should not be duplicate
	if detector.CheckAndAdd(game) {
		t.Error("game after reset should not be duplicate")
	}
}

// ============== GameHasher Tests ==============

func TestGameHasher_New(t *testing.T) {
	tests := []struct {
		name     string
		hashType HashType
	}{
		{"final position", HashFinalPosition},
		{"all positions", HashAllPositions},
		{"move sequence", HashMoveSequence},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewGameHasher(tt.hashType)
			if hasher == nil {
				t.Fatal("NewGameHasher() returned nil")
			}
		})
	}
}

func TestGameHasher_HashGame_FinalPosition(t *testing.T) {
	hasher := NewGameHasher(HashFinalPosition)

	board := chess.NewBoard()
	board.SetupInitialPosition()
	game := &chess.Game{Tags: make(map[string]string)}

	hash := hasher.HashGame(game, board)
	if hash == 0 {
		t.Error("HashGame() should return non-zero hash")
	}

	// Same position should produce same hash
	board2 := chess.NewBoard()
	board2.SetupInitialPosition()
	hash2 := hasher.HashGame(game, board2)

	if hash != hash2 {
		t.Error("same position should produce same hash")
	}
}

func TestGameHasher_HashGame_MoveSequence(t *testing.T) {
	hasher := NewGameHasher(HashMoveSequence)

	board := chess.NewBoard()
	board.SetupInitialPosition()

	// Game with moves
	game1 := &chess.Game{
		Tags:  make(map[string]string),
		Moves: &chess.Move{Text: "e4", Next: &chess.Move{Text: "e5"}},
	}

	hash1 := hasher.HashGame(game1, board)
	if hash1 == 0 {
		t.Error("HashGame() should return non-zero hash for game with moves")
	}

	// Different move sequence
	game2 := &chess.Game{
		Tags:  make(map[string]string),
		Moves: &chess.Move{Text: "d4", Next: &chess.Move{Text: "d5"}},
	}

	hash2 := hasher.HashGame(game2, board)
	if hash1 == hash2 {
		t.Error("different move sequences should produce different hashes")
	}
}

func TestGameHasher_HashGame_EmptyMoveSequence(t *testing.T) {
	hasher := NewGameHasher(HashMoveSequence)

	board := chess.NewBoard()
	board.SetupInitialPosition()
	game := &chess.Game{
		Tags:  make(map[string]string),
		Moves: nil,
	}

	hash := hasher.HashGame(game, board)
	// Empty move sequence should produce zero hash
	if hash != 0 {
		t.Errorf("HashGame() for empty move sequence = %d, want 0", hash)
	}
}

// ============== DuplicateDetector with ExactMatch Tests ==============

func TestDuplicateDetector_ExactMatch_SameMoveCount(t *testing.T) {
	detector := NewDuplicateDetector(true) // exactMatch mode

	board := chess.NewBoard()
	board.SetupInitialPosition()

	// Two games with same position and same move count
	game1 := &chess.Game{
		Tags:  make(map[string]string),
		Moves: &chess.Move{Text: "e4", Next: &chess.Move{Text: "e5"}},
	}
	game2 := &chess.Game{
		Tags:  make(map[string]string),
		Moves: &chess.Move{Text: "d4", Next: &chess.Move{Text: "d5"}},
	}

	detector.CheckAndAdd(game1, board)

	// Same board, same move count - should be duplicate in exact match mode
	if !detector.CheckAndAdd(game2, board) {
		t.Error("same position and move count should be duplicate in exact match mode")
	}
}

func TestDuplicateDetector_ExactMatch_DifferentMoveCount(t *testing.T) {
	detector := NewDuplicateDetector(true) // exactMatch mode

	board := chess.NewBoard()
	board.SetupInitialPosition()

	// Game with 2 moves
	game1 := &chess.Game{
		Tags:  make(map[string]string),
		Moves: &chess.Move{Text: "e4", Next: &chess.Move{Text: "e5"}},
	}
	// Game with 3 moves
	game2 := &chess.Game{
		Tags:  make(map[string]string),
		Moves: &chess.Move{Text: "d4", Next: &chess.Move{Text: "d5", Next: &chess.Move{Text: "Nf3"}}},
	}

	detector.CheckAndAdd(game1, board)

	// Same board but different move count - should NOT be duplicate in exact match mode
	if detector.CheckAndAdd(game2, board) {
		t.Error("same position but different move count should NOT be duplicate in exact match mode")
	}
}

func TestDuplicateDetector_NilBoard(t *testing.T) {
	detector := NewDuplicateDetector(false)
	game := &chess.Game{Tags: make(map[string]string)}

	// Passing nil board should return false
	if detector.CheckAndAdd(game, nil) {
		t.Error("CheckAndAdd with nil board should return false")
	}
}

// ============== countMoves helper Tests ==============

func TestCountMoves(t *testing.T) {
	tests := []struct {
		name      string
		game      *chess.Game
		wantCount int
	}{
		{
			name:      "no moves",
			game:      &chess.Game{Moves: nil},
			wantCount: 0,
		},
		{
			name: "one move",
			game: &chess.Game{
				Moves: &chess.Move{Text: "e4"},
			},
			wantCount: 1,
		},
		{
			name: "three moves",
			game: &chess.Game{
				Moves: &chess.Move{
					Text: "e4",
					Next: &chess.Move{
						Text: "e5",
						Next: &chess.Move{Text: "Nf3"},
					},
				},
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countMoves(tt.game)
			if got != tt.wantCount {
				t.Errorf("countMoves() = %d, want %d", got, tt.wantCount)
			}
		})
	}
}
