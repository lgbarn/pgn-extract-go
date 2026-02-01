package hashing

import (
	"sync"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

func TestThreadSafeDuplicateDetector_Concurrent(t *testing.T) {
	detector := NewThreadSafeDuplicateDetector(false)
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
	detector := NewThreadSafeDuplicateDetector(false)

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
	detector := NewThreadSafeDuplicateDetector(false)
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

	threadSafe := NewThreadSafeDuplicateDetector(false)
	threadSafe.LoadFromDetector(regular)

	isDupe := threadSafe.CheckAndAdd(game, board)
	if !isDupe {
		t.Error("Expected duplicate after loading from regular detector")
	}

	if threadSafe.DuplicateCount() != 1 {
		t.Errorf("Expected 1 duplicate, got %d", threadSafe.DuplicateCount())
	}
}
