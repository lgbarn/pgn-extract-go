package hashing

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

func TestThreadSafeDuplicateDetector_Concurrent(t *testing.T) {
	detector := NewThreadSafeDuplicateDetector(false, 0)
	board, err := engine.NewBoardFromFEN(engine.InitialFEN)
	if err != nil {
		t.Fatal(err)
	}

	const numGames = 100
	const numWorkers = 10
	gamesPerWorker := numGames / numWorkers

	games := make([]*chess.Game, numGames)
	boards := make([]*chess.Board, numGames)
	for i := range games {
		games[i] = &chess.Game{
			Tags: map[string]string{
				"Event": "Test",
				"White": "Player1",
				"Black": "Player2",
			},
		}
		boards[i] = board.Copy()
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			start := workerID * gamesPerWorker
			end := start + gamesPerWorker
			for j := start; j < end; j++ {
				detector.CheckAndAdd(games[j], boards[j])
			}
		}(i)
	}
	wg.Wait()

	if detector.DuplicateCount() != 99 {
		t.Errorf("Expected 99 duplicates, got %d", detector.DuplicateCount())
	}

	if detector.UniqueCount() != 1 {
		t.Errorf("Expected 1 unique, got %d", detector.UniqueCount())
	}
}

func TestThreadSafeDuplicateDetector_DifferentPositions(t *testing.T) {
	detector := NewThreadSafeDuplicateDetector(false, 0)

	fens := []string{
		engine.InitialFEN,
		"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		"rnbqkbnr/pppppppp/8/8/3P4/8/PPP1PPPP/RNBQKBNR b KQkq d3 0 1",
		"rnbqkbnr/pppppppp/8/8/8/5N2/PPPPPPPP/RNBQKB1R b KQkq - 1 1",
		"rnbqkbnr/pppppppp/8/8/2P5/8/PP1PPPPP/RNBQKBNR b KQkq c3 0 1",
	}

	games := make([]*chess.Game, len(fens))
	boards := make([]*chess.Board, len(fens))

	for i, fen := range fens {
		games[i] = &chess.Game{Tags: map[string]string{"Event": "Test"}}
		board, err := engine.NewBoardFromFEN(fen)
		if err != nil {
			t.Fatalf("Failed to parse FEN %s: %v", fen, err)
		}
		boards[i] = board
	}

	var wg sync.WaitGroup
	for i := range fens {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			detector.CheckAndAdd(games[idx], boards[idx])
		}(i)
	}
	wg.Wait()

	if detector.DuplicateCount() != 0 {
		t.Errorf("Expected 0 duplicates, got %d", detector.DuplicateCount())
	}

	if detector.UniqueCount() != len(fens) {
		t.Errorf("Expected %d unique, got %d", len(fens), detector.UniqueCount())
	}
}

func TestThreadSafeDuplicateDetector_NoRace(t *testing.T) {
	detector := NewThreadSafeDuplicateDetector(false, 0)
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN)
	game := &chess.Game{Tags: map[string]string{"Event": "Test"}}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			detector.CheckAndAdd(game, board)
			_ = detector.DuplicateCount()
			_ = detector.UniqueCount()
		}()
	}
	wg.Wait()
}

func TestThreadSafeDuplicateDetector_LoadFromDetector(t *testing.T) {
	regular := NewDuplicateDetector(false, 0)
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN)
	game := &chess.Game{Tags: map[string]string{"Event": "Test"}}
	regular.CheckAndAdd(game, board)

	if regular.UniqueCount() != 1 {
		t.Errorf("Expected 1 unique in regular detector, got %d", regular.UniqueCount())
	}

	threadSafe := NewThreadSafeDuplicateDetector(false, 0)
	threadSafe.LoadFromDetector(regular)

	isDupe := threadSafe.CheckAndAdd(game, board)
	if !isDupe {
		t.Error("Expected duplicate after loading from regular detector")
	}

	if threadSafe.DuplicateCount() != 1 {
		t.Errorf("Expected 1 duplicate, got %d", threadSafe.DuplicateCount())
	}
}

func TestThreadSafeDuplicateDetector_MaxCapacity(t *testing.T) {
	const capacity = 50
	const numWorkers = 10
	const gamesPerWorker = 100

	detector := NewThreadSafeDuplicateDetector(false, capacity)

	// Track how many unique games we actually add
	uniqueAdded := int32(0)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			localUnique := 0
			for j := 0; j < gamesPerWorker; j++ {
				// Create unique positions
				board := chess.NewBoard()
				board.SetupInitialPosition()
				idx := workerID*gamesPerWorker + j
				col := chess.Col('a' + idx%8)
				rank := chess.Rank('1' + idx/8)
				board.Set(col, rank, chess.Empty)
				if idx%2 == 0 {
					board.Set(chess.Col('h'-(idx%8)), chess.Rank('8'-(idx/8)%8), chess.Empty)
				}
				game := &chess.Game{Tags: map[string]string{"Event": "Test"}}
				isDupe := detector.CheckAndAdd(game, board)
				if !isDupe {
					localUnique++
				}
			}
			atomic.AddInt32(&uniqueAdded, int32(localUnique))
		}(i)
	}
	wg.Wait()

	// With capacity limit, we should stop accepting new unique games after capacity is reached
	// The detector should be full if we added enough unique games
	if uniqueAdded >= int32(capacity) && !detector.IsFull() {
		t.Errorf("Expected detector to be full after adding %d unique games (capacity %d)", uniqueAdded, capacity)
	}

	// Verify that UniqueCount doesn't exceed a reasonable bound
	// (allowing for some hash collisions that share buckets)
	if detector.UniqueCount() > capacity*2 {
		t.Errorf("Expected UniqueCount <= %d (allowing for collisions), got %d", capacity*2, detector.UniqueCount())
	}
}
