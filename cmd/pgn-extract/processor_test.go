package main

import (
	"sync"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/hashing"
	"github.com/lgbarn/pgn-extract-go/internal/testutil"
)

// TestParallelDuplicateDetection_MatchesSequential verifies that parallel duplicate
// detection produces identical results to sequential processing.
func TestParallelDuplicateDetection_MatchesSequential(t *testing.T) {
	tests := []struct {
		name  string
		games []string
	}{
		{
			name: "mixed unique and duplicate games",
			games: []string{
				// Game 1: Initial position
				`[Event "Game 1"]
[White "Player A"]
[Black "Player B"]

1. e4 *`,
				// Game 2: Same position as Game 1 (duplicate)
				`[Event "Game 2"]
[White "Player C"]
[Black "Player D"]

1. e4 *`,
				// Game 3: Different opening
				`[Event "Game 3"]
[White "Player E"]
[Black "Player F"]

1. d4 *`,
				// Game 4: Another different opening
				`[Event "Game 4"]
[White "Player G"]
[Black "Player H"]

1. c4 *`,
				// Game 5: Same as Game 3 (duplicate)
				`[Event "Game 5"]
[White "Player I"]
[Black "Player J"]

1. d4 *`,
				// Game 6: Unique game with multiple moves
				`[Event "Game 6"]
[White "Player K"]
[Black "Player L"]

1. e4 e5 2. Nf3 Nc6 *`,
				// Game 7: Different move sequence
				`[Event "Game 7"]
[White "Player M"]
[Black "Player N"]

1. Nf3 Nf6 *`,
				// Game 8: Same as Game 6 (duplicate)
				`[Event "Game 8"]
[White "Player O"]
[Black "Player P"]

1. e4 e5 2. Nf3 Nc6 *`,
				// Game 9: Another unique game
				`[Event "Game 9"]
[White "Player Q"]
[Black "Player R"]

1. e4 c5 *`,
				// Game 10: Unique position
				`[Event "Game 10"]
[White "Player S"]
[Black "Player T"]

1. e4 e5 2. Nf3 Nf6 *`,
				// Games 11-20: More duplicates
				`[Event "Game 11"]
[White "A"]
[Black "B"]

1. e4 *`,
				`[Event "Game 12"]
[White "C"]
[Black "D"]

1. d4 *`,
				`[Event "Game 13"]
[White "E"]
[Black "F"]

1. c4 *`,
				`[Event "Game 14"]
[White "G"]
[Black "H"]

1. e4 e5 2. Nf3 Nc6 *`,
				`[Event "Game 15"]
[White "I"]
[Black "J"]

1. Nf3 Nf6 *`,
				`[Event "Game 16"]
[White "K"]
[Black "L"]

1. e4 c5 *`,
				`[Event "Game 17"]
[White "M"]
[Black "N"]

1. e4 e5 2. Nf3 Nf6 *`,
				`[Event "Game 18"]
[White "O"]
[Black "P"]

1. d4 d5 *`,
				`[Event "Game 19"]
[White "Q"]
[Black "R"]

1. e4 d5 *`,
				`[Event "Game 20"]
[White "S"]
[Black "T"]

1. c4 e5 *`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse all test games
			var parsedGames []*chess.Game
			for _, pgnStr := range tt.games {
				game := testutil.MustParseGame(t, pgnStr)
				parsedGames = append(parsedGames, game)
			}

			// Run sequential detection
			seqDetector := hashing.NewDuplicateDetector(false)
			for _, game := range parsedGames {
				board := replayGame(game)
				seqDetector.CheckAndAdd(game, board)
			}

			// Run parallel detection with 4 goroutines
			tsDetector := hashing.NewThreadSafeDuplicateDetector(false)
			var wg sync.WaitGroup
			const numWorkers = 4
			gamesPerWorker := len(parsedGames) / numWorkers
			remainder := len(parsedGames) % numWorkers

			startIdx := 0
			for i := 0; i < numWorkers; i++ {
				count := gamesPerWorker
				if i < remainder {
					count++
				}
				endIdx := startIdx + count

				wg.Add(1)
				go func(start, end int) {
					defer wg.Done()
					for j := start; j < end; j++ {
						board := replayGame(parsedGames[j])
						tsDetector.CheckAndAdd(parsedGames[j], board)
					}
				}(startIdx, endIdx)

				startIdx = endIdx
			}
			wg.Wait()

			// Compare results
			if seqDetector.DuplicateCount() != tsDetector.DuplicateCount() {
				t.Errorf("DuplicateCount mismatch: sequential=%d, parallel=%d",
					seqDetector.DuplicateCount(), tsDetector.DuplicateCount())
			}

			if seqDetector.UniqueCount() != tsDetector.UniqueCount() {
				t.Errorf("UniqueCount mismatch: sequential=%d, parallel=%d",
					seqDetector.UniqueCount(), tsDetector.UniqueCount())
			}
		})
	}
}

// TestParallelDuplicateDetection_WithCheckFile verifies that duplicate detection
// works correctly when pre-loading games from a checkfile.
func TestParallelDuplicateDetection_WithCheckFile(t *testing.T) {
	// Create checkfile games (games that were already processed)
	checkfileGames := []string{
		`[Event "Checkfile Game 1"]
[White "Player A"]
[Black "Player B"]

1. e4 *`,
		`[Event "Checkfile Game 2"]
[White "Player C"]
[Black "Player D"]

1. d4 *`,
		`[Event "Checkfile Game 3"]
[White "Player E"]
[Black "Player F"]

1. c4 *`,
	}

	// Create new games to process (some duplicates, some unique)
	newGames := []string{
		// Duplicate of checkfile game 1
		`[Event "New Game 1"]
[White "New A"]
[Black "New B"]

1. e4 *`,
		// Unique game
		`[Event "New Game 2"]
[White "New C"]
[Black "New D"]

1. Nf3 *`,
		// Duplicate of checkfile game 2
		`[Event "New Game 3"]
[White "New E"]
[Black "New F"]

1. d4 *`,
		// Another unique game
		`[Event "New Game 4"]
[White "New G"]
[Black "New H"]

1. e4 e5 *`,
		// Duplicate of checkfile game 3
		`[Event "New Game 5"]
[White "New I"]
[Black "New J"]

1. c4 *`,
		// Unique game
		`[Event "New Game 6"]
[White "New K"]
[Black "New L"]

1. d4 d5 *`,
	}

	// Parse checkfile games and load into base detector
	baseDetector := hashing.NewDuplicateDetector(false)
	for _, pgnStr := range checkfileGames {
		game := testutil.MustParseGame(t, pgnStr)
		board := replayGame(game)
		baseDetector.CheckAndAdd(game, board)
	}

	// Verify base detector state
	if baseDetector.UniqueCount() != len(checkfileGames) {
		t.Fatalf("checkfile setup failed: expected %d unique, got %d",
			len(checkfileGames), baseDetector.UniqueCount())
	}

	// Parse new games
	var parsedNewGames []*chess.Game
	for _, pgnStr := range newGames {
		game := testutil.MustParseGame(t, pgnStr)
		parsedNewGames = append(parsedNewGames, game)
	}

	// Create thread-safe detector and load from base
	tsDetector := hashing.NewThreadSafeDuplicateDetector(false)
	tsDetector.LoadFromDetector(baseDetector)

	// Verify initial state after loading
	if tsDetector.UniqueCount() != baseDetector.UniqueCount() {
		t.Fatalf("LoadFromDetector failed: expected %d unique, got %d",
			baseDetector.UniqueCount(), tsDetector.UniqueCount())
	}

	// Process new games concurrently
	var wg sync.WaitGroup
	const numWorkers = 3
	gamesPerWorker := len(parsedNewGames) / numWorkers
	remainder := len(parsedNewGames) % numWorkers

	startIdx := 0
	for i := 0; i < numWorkers; i++ {
		count := gamesPerWorker
		if i < remainder {
			count++
		}
		endIdx := startIdx + count

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for j := start; j < end; j++ {
				board := replayGame(parsedNewGames[j])
				tsDetector.CheckAndAdd(parsedNewGames[j], board)
			}
		}(startIdx, endIdx)

		startIdx = endIdx
	}
	wg.Wait()

	// Expected: 3 checkfile games + 3 unique new games = 6 unique total
	// Expected duplicates: 3 (New Game 1, 3, 5 match checkfile games)
	expectedUnique := 6
	expectedDuplicates := 3

	if tsDetector.UniqueCount() != expectedUnique {
		t.Errorf("UniqueCount mismatch: expected %d, got %d",
			expectedUnique, tsDetector.UniqueCount())
	}

	if tsDetector.DuplicateCount() != expectedDuplicates {
		t.Errorf("DuplicateCount mismatch: expected %d, got %d",
			expectedDuplicates, tsDetector.DuplicateCount())
	}
}

