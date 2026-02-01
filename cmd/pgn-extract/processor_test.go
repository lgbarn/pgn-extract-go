package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/hashing"
	"github.com/lgbarn/pgn-extract-go/internal/testutil"
	"github.com/lgbarn/pgn-extract-go/internal/worker"
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
			parsedGames := make([]*chess.Game, 0, len(tt.games))
			for _, pgnStr := range tt.games {
				game := testutil.MustParseGame(t, pgnStr)
				parsedGames = append(parsedGames, game)
			}

			// Run sequential detection
			seqDetector := hashing.NewDuplicateDetector(false, 0)
			for _, game := range parsedGames {
				board := replayGame(game)
				seqDetector.CheckAndAdd(game, board)
			}

			// Run parallel detection with 4 goroutines
			tsDetector := hashing.NewThreadSafeDuplicateDetector(false, 0)
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
	baseDetector := hashing.NewDuplicateDetector(false, 0)
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
	parsedNewGames := make([]*chess.Game, 0, len(newGames))
	for _, pgnStr := range newGames {
		game := testutil.MustParseGame(t, pgnStr)
		parsedNewGames = append(parsedNewGames, game)
	}

	// Create thread-safe detector and load from base
	tsDetector := hashing.NewThreadSafeDuplicateDetector(false, 0)
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

// makeMinimalGame creates a minimal game with a given ECO code for testing.
func makeMinimalGame(eco string) *chess.Game {
	game := chess.NewGame()
	game.SetTag("Event", "Test")
	game.SetTag("White", "TestWhite")
	game.SetTag("Black", "TestBlack")
	game.SetTag("ECO", eco)
	game.SetTag("Result", "*")
	return game
}

// TestECOSplitWriter_LRU_EvictsOldestHandle verifies that the LRU cache
// evicts the least recently used file handle when maxHandles is exceeded.
func TestECOSplitWriter_LRU_EvictsOldestHandle(t *testing.T) {
	tmpDir := t.TempDir()
	baseName := filepath.Join(tmpDir, "eco")
	cfg := config.NewConfig()
	cfg.OutputFile = os.Stdout

	writer := NewECOSplitWriter(baseName, 3, cfg, 3) // maxHandles=3
	defer writer.Close()

	// Write games with 4 different ECO codes
	ecoCodes := []string{"A00", "B00", "C00", "D00"}
	for _, eco := range ecoCodes {
		game := makeMinimalGame(eco)
		if err := writer.WriteGame(game); err != nil {
			t.Fatalf("WriteGame(%s) failed: %v", eco, err)
		}
	}

	// Verify: All 4 files should exist
	if writer.FileCount() != 4 {
		t.Errorf("FileCount = %d, want 4", writer.FileCount())
	}

	// Verify: Only 3 file handles should be open (A00 was evicted)
	if writer.OpenHandleCount() != 3 {
		t.Errorf("OpenHandleCount = %d, want 3", writer.OpenHandleCount())
	}

	// Verify all 4 files exist on disk
	for _, eco := range ecoCodes {
		filename := filepath.Join(tmpDir, "eco_"+eco+".pgn")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Errorf("File %s does not exist", filename)
		}
	}
}

// TestECOSplitWriter_LRU_ReopensEvictedFile verifies that evicted files
// are reopened in append mode when accessed again.
func TestECOSplitWriter_LRU_ReopensEvictedFile(t *testing.T) {
	tmpDir := t.TempDir()
	baseName := filepath.Join(tmpDir, "eco")
	cfg := config.NewConfig()
	cfg.OutputFile = os.Stdout

	writer := NewECOSplitWriter(baseName, 3, cfg, 2) // maxHandles=2
	defer writer.Close()

	// Write A00, B00, C00 (A00 gets evicted)
	for _, eco := range []string{"A00", "B00", "C00"} {
		game := makeMinimalGame(eco)
		if err := writer.WriteGame(game); err != nil {
			t.Fatalf("WriteGame(%s) failed: %v", eco, err)
		}
	}

	// Write A00 again (should reopen and append)
	game := makeMinimalGame("A00")
	if err := writer.WriteGame(game); err != nil {
		t.Fatalf("WriteGame(A00) second time failed: %v", err)
	}

	// Verify: OpenHandleCount should still be 2 (maxHandles) after reopen
	if writer.OpenHandleCount() != 2 {
		t.Errorf("After reopening A00: OpenHandleCount = %d, want 2", writer.OpenHandleCount())
	}

	// Close to flush all writes
	writer.Close()

	// Verify A00 file has both games (contains "Event" twice)
	filename := filepath.Join(tmpDir, "eco_A00.pgn")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile(%s) failed: %v", filename, err)
	}

	eventCount := strings.Count(string(content), "[Event")
	if eventCount != 2 {
		t.Errorf("A00 file has %d games, want 2", eventCount)
	}
}

// TestECOSplitWriter_LRU_UnlimitedWhenHigh verifies that when maxHandles
// is high, all handles remain open without eviction.
func TestECOSplitWriter_LRU_UnlimitedWhenHigh(t *testing.T) {
	tmpDir := t.TempDir()
	baseName := filepath.Join(tmpDir, "eco")
	cfg := config.NewConfig()
	cfg.OutputFile = os.Stdout

	writer := NewECOSplitWriter(baseName, 3, cfg, 1000) // maxHandles=1000
	defer writer.Close()

	// Write 10 different ECO codes
	ecoCodes := []string{"A00", "A01", "B00", "B01", "C00", "C01", "D00", "D01", "E00", "E01"}
	for _, eco := range ecoCodes {
		game := makeMinimalGame(eco)
		if err := writer.WriteGame(game); err != nil {
			t.Fatalf("WriteGame(%s) failed: %v", eco, err)
		}
	}

	// Verify: All 10 files created
	if writer.FileCount() != 10 {
		t.Errorf("FileCount = %d, want 10", writer.FileCount())
	}

	// Verify: All 10 handles still open (no eviction)
	if writer.OpenHandleCount() != 10 {
		t.Errorf("OpenHandleCount = %d, want 10", writer.OpenHandleCount())
	}
}

// TestECOSplitWriter_LRU_HandleCountBounded verifies that the LRU cache
// properly bounds the number of open file handles when writing games
// with many distinct ECO codes.
func TestECOSplitWriter_LRU_HandleCountBounded(t *testing.T) {
	tmpDir := t.TempDir()
	baseName := filepath.Join(tmpDir, "eco")
	cfg := config.NewConfig()
	cfg.OutputFile = os.Stdout

	const maxHandles = 5
	const level = 3 // Full ECO code: A00-E99
	writer := NewECOSplitWriter(baseName, level, cfg, maxHandles)
	defer writer.Close()

	// Create games with 20 distinct ECO codes (A00-A19)
	ecoCodes := []string{
		"A00", "A01", "A02", "A03", "A04",
		"A05", "A06", "A07", "A08", "A09",
		"A10", "A11", "A12", "A13", "A14",
		"A15", "A16", "A17", "A18", "A19",
	}

	for i, eco := range ecoCodes {
		game := makeMinimalGame(eco)
		if err := writer.WriteGame(game); err != nil {
			t.Fatalf("WriteGame(%s) failed: %v", eco, err)
		}

		// After the 5th write, OpenHandleCount should never exceed maxHandles
		if i >= maxHandles {
			if writer.OpenHandleCount() > maxHandles {
				t.Errorf("After writing game %d (%s): OpenHandleCount=%d exceeds maxHandles=%d",
					i+1, eco, writer.OpenHandleCount(), maxHandles)
			}
		}
	}

	// Verify all 20 distinct ECO codes created files
	if writer.FileCount() != len(ecoCodes) {
		t.Errorf("FileCount=%d, want %d", writer.FileCount(), len(ecoCodes))
	}

	// Verify handle count is bounded by maxHandles
	if writer.OpenHandleCount() > maxHandles {
		t.Errorf("Final OpenHandleCount=%d exceeds maxHandles=%d",
			writer.OpenHandleCount(), maxHandles)
	}

	// Close writer to flush all data
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Verify all 20 files exist on disk
	for _, eco := range ecoCodes {
		filename := filepath.Join(tmpDir, "eco_"+eco+".pgn")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Errorf("File %s does not exist after Close()", filename)
		}
	}
}

// --- Helper: test PGN data ---

const processorTestPGN = `[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0`

const processorTestPGN2 = `[Event "Test2"]
[Site "Test"]
[Date "2024.01.02"]
[Round "2"]
[White "Player3"]
[Black "Player4"]
[Result "0-1"]

1. d4 d5 2. c4 e6 0-1`

const processorTestPGN3 = `[Event "Test3"]
[Site "Test"]
[Date "2024.01.03"]
[Round "3"]
[White "Player5"]
[Black "Player6"]
[Result "1/2-1/2"]

1. Nf3 Nf6 2. g3 g6 1/2-1/2`

const threeGamePGN = processorTestPGN + "\n\n" + processorTestPGN2 + "\n\n" + processorTestPGN3

// resetGlobalState resets all global state modified by the processing pipeline.
func resetGlobalState(t *testing.T) {
	t.Helper()
	atomic.StoreInt64(&matchedCount, 0)
	atomic.StoreInt64(&gamePositionCounter, 0)
	selectOnlySet = nil
	skipMatchingSet = nil
	parsedPlyRange = [2]int{0, 0}
	parsedMoveRange = [2]int{0, 0}
}

// saveFlagPointers saves and returns a restore function for global flag pointers that tests modify.
func saveFlagPointers(t *testing.T) func() {
	t.Helper()
	origStopAfter := *stopAfter
	origSelectOnly := *selectOnly
	origSkipMatching := *skipMatching
	origWorkers := *workers
	origReportOnly := *reportOnly
	origQuiet := *quiet
	origFixableMode := *fixableMode
	origNegateMatch := *negateMatch
	origCheckmateFilter := *checkmateFilter
	origStalemateFilter := *stalemateFilter
	origMinPly := *minPly
	origMaxPly := *maxPly
	origExactPly := *exactPly
	origPlyRange := *plyRange
	origMoveRange := *moveRange
	origDropPly := *dropPly
	origStartPly := *startPly
	origPlyLimit := *plyLimit
	origDropBefore := *dropBefore
	origStrictMode := *strictMode
	origValidateMode := *validateMode

	return func() {
		*stopAfter = origStopAfter
		*selectOnly = origSelectOnly
		*skipMatching = origSkipMatching
		*workers = origWorkers
		*reportOnly = origReportOnly
		*quiet = origQuiet
		*fixableMode = origFixableMode
		*negateMatch = origNegateMatch
		*checkmateFilter = origCheckmateFilter
		*stalemateFilter = origStalemateFilter
		*minPly = origMinPly
		*maxPly = origMaxPly
		*exactPly = origExactPly
		*plyRange = origPlyRange
		*moveRange = origMoveRange
		*dropPly = origDropPly
		*startPly = origStartPly
		*plyLimit = origPlyLimit
		*dropBefore = origDropBefore
		*strictMode = origStrictMode
		*validateMode = origValidateMode
	}
}

// newTestContext creates a minimal ProcessingContext with config pointing at the given buffer.
func newTestContext(buf *bytes.Buffer) *ProcessingContext {
	cfg := config.NewConfig()
	cfg.OutputFile = buf
	cfg.Verbosity = 0
	return &ProcessingContext{cfg: cfg}
}

// ============================================================
// Task 1: SplitWriter, processInput, withOutputFile, output helpers
// ============================================================

func TestSplitWriterRotation(t *testing.T) {
	tmpDir := t.TempDir()
	baseName := filepath.Join(tmpDir, "split")
	sw := NewSplitWriter(baseName, 2) // 2 games per file
	defer sw.Close()

	// Write 3 "games" (just write bytes + increment)
	for i := 0; i < 3; i++ {
		_, err := fmt.Fprintf(sw, "[Event \"Game %d\"]\n\n1. e4 *\n\n", i+1)
		if err != nil {
			t.Fatalf("Write failed on game %d: %v", i+1, err)
		}
		sw.IncrementGameCount()
	}

	if err := sw.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Expect file 1 (2 games) and file 2 (1 game)
	file1 := fmt.Sprintf("%s_%d.pgn", baseName, 1)
	file2 := fmt.Sprintf("%s_%d.pgn", baseName, 2)

	if _, err := os.Stat(file1); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", file1)
	}
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", file2)
	}

	// File 1 should have 2 events, file 2 should have 1
	content1, _ := os.ReadFile(file1)
	content2, _ := os.ReadFile(file2)
	if count := strings.Count(string(content1), "[Event"); count != 2 {
		t.Errorf("File 1 has %d events, want 2", count)
	}
	if count := strings.Count(string(content2), "[Event"); count != 1 {
		t.Errorf("File 2 has %d events, want 1", count)
	}
}

func TestSplitWriterCustomPattern(t *testing.T) {
	tmpDir := t.TempDir()
	baseName := filepath.Join(tmpDir, "custom")
	sw := NewSplitWriterWithPattern(baseName, 1, "%s-part%d.pgn")
	defer sw.Close()

	_, err := sw.Write([]byte("game data"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	sw.IncrementGameCount()
	sw.Close()

	expected := fmt.Sprintf("%s-part%d.pgn", baseName, 1)
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist with custom pattern", expected)
	}
}

func TestSplitWriterCloseNilFile(t *testing.T) {
	sw := NewSplitWriter("/tmp/unused", 10)
	// currentFile is nil since we never wrote
	if err := sw.Close(); err != nil {
		t.Errorf("Close on nil file returned error: %v", err)
	}
}

func TestProcessInput(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Verbosity = 0

	t.Run("valid PGN", func(t *testing.T) {
		r := strings.NewReader(processorTestPGN)
		games := processInput(r, "test.pgn", cfg)
		if len(games) != 1 {
			t.Fatalf("Expected 1 game, got %d", len(games))
		}
		if games[0].GetTag("Event") != "Test" {
			t.Errorf("Expected Event=Test, got %q", games[0].GetTag("Event"))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		r := strings.NewReader("")
		games := processInput(r, "empty.pgn", cfg)
		if len(games) != 0 {
			t.Errorf("Expected 0 games from empty input, got %d", len(games))
		}
	})

	t.Run("multiple games", func(t *testing.T) {
		r := strings.NewReader(threeGamePGN)
		games := processInput(r, "multi.pgn", cfg)
		if len(games) != 3 {
			t.Fatalf("Expected 3 games, got %d", len(games))
		}
	})
}

func TestWithOutputFile(t *testing.T) {
	cfg := config.NewConfig()
	originalBuf := &bytes.Buffer{}
	tempBuf := &bytes.Buffer{}
	cfg.OutputFile = originalBuf

	withOutputFile(cfg, tempBuf, func() {
		if cfg.OutputFile != tempBuf {
			t.Error("OutputFile not redirected inside fn")
		}
		fmt.Fprint(cfg.OutputFile, "hello")
	})

	if cfg.OutputFile != originalBuf {
		t.Error("OutputFile not restored after withOutputFile")
	}
	if tempBuf.String() != "hello" {
		t.Errorf("Expected 'hello' in temp buffer, got %q", tempBuf.String())
	}
	if originalBuf.Len() != 0 {
		t.Errorf("Original buffer should be empty, got %q", originalBuf.String())
	}
}

func TestOutputNonMatchingGame(t *testing.T) {
	game := testutil.MustParseGame(t, processorTestPGN)

	t.Run("with NonMatchingFile", func(t *testing.T) {
		cfg := config.NewConfig()
		nmBuf := &bytes.Buffer{}
		cfg.NonMatchingFile = nmBuf
		cfg.OutputFile = &bytes.Buffer{} // prevent writing to stdout

		outputNonMatchingGame(game, cfg)

		if nmBuf.Len() == 0 {
			t.Error("Expected game written to NonMatchingFile")
		}
		if !strings.Contains(nmBuf.String(), "[Event") {
			t.Error("NonMatchingFile output should contain game tags")
		}
	})

	t.Run("with nil NonMatchingFile", func(t *testing.T) {
		cfg := config.NewConfig()
		cfg.NonMatchingFile = nil
		cfg.OutputFile = &bytes.Buffer{}
		// Should not panic
		outputNonMatchingGame(game, cfg)
	})
}

func TestOutputDuplicateGame(t *testing.T) {
	game := testutil.MustParseGame(t, processorTestPGN)

	t.Run("with DuplicateFile", func(t *testing.T) {
		cfg := config.NewConfig()
		dupBuf := &bytes.Buffer{}
		cfg.Duplicate.DuplicateFile = dupBuf
		cfg.OutputFile = &bytes.Buffer{}

		outputDuplicateGame(game, cfg)

		if dupBuf.Len() == 0 {
			t.Error("Expected game written to DuplicateFile")
		}
		if !strings.Contains(dupBuf.String(), "[Event") {
			t.Error("DuplicateFile output should contain game tags")
		}
	})

	t.Run("with nil DuplicateFile", func(t *testing.T) {
		cfg := config.NewConfig()
		cfg.Duplicate.DuplicateFile = nil
		cfg.OutputFile = &bytes.Buffer{}
		// Should not panic
		outputDuplicateGame(game, cfg)
	})

	t.Run("with JSON format", func(t *testing.T) {
		cfg := config.NewConfig()
		dupBuf := &bytes.Buffer{}
		cfg.Duplicate.DuplicateFile = dupBuf
		cfg.Output.JSONFormat = true
		cfg.OutputFile = &bytes.Buffer{}

		outputDuplicateGame(game, cfg)

		if dupBuf.Len() == 0 {
			t.Error("Expected JSON output to DuplicateFile")
		}
	})
}

func TestShouldOutputUnique(t *testing.T) {
	tests := []struct {
		suppress          bool
		suppressOriginals bool
		expected          bool
	}{
		{false, false, true},
		{true, false, true},
		{false, true, true},
		{true, true, false}, // only case where unique games are suppressed
	}

	for _, tt := range tests {
		name := fmt.Sprintf("suppress=%v,suppressOriginals=%v", tt.suppress, tt.suppressOriginals)
		t.Run(name, func(t *testing.T) {
			cfg := config.NewConfig()
			cfg.Duplicate.Suppress = tt.suppress
			cfg.Duplicate.SuppressOriginals = tt.suppressOriginals
			got := shouldOutputUnique(cfg)
			if got != tt.expected {
				t.Errorf("shouldOutputUnique() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ============================================================
// Task 2: applyFilters pipeline and handleGameOutput
// ============================================================

func TestApplyFiltersMinimal(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()

	game := testutil.MustParseGame(t, processorTestPGN)
	buf := &bytes.Buffer{}
	ctx := newTestContext(buf)

	result := applyFilters(game, ctx)
	if !result.Matched {
		t.Error("Expected game to match with no filters")
	}
	if result.SkipOutput {
		t.Error("Expected SkipOutput=false with no filters")
	}
}

func TestApplyFiltersFixable(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()
	*fixableMode = true

	// Create game missing some tags
	pgnMissing := `[Event "Test"]
[Result "1-0"]

1. e4 *`
	game := testutil.MustParseGame(t, pgnMissing)
	buf := &bytes.Buffer{}
	ctx := newTestContext(buf)

	result := applyFilters(game, ctx)
	if !result.Matched {
		t.Error("Expected fixable game to still match")
	}

	// fixGame should have added missing tags
	if game.GetTag("White") == "" {
		t.Error("Expected fixGame to add missing White tag")
	}
}

func TestApplyFiltersNegate(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()
	*negateMatch = true

	game := testutil.MustParseGame(t, processorTestPGN)
	buf := &bytes.Buffer{}
	ctx := newTestContext(buf)

	result := applyFilters(game, ctx)
	if result.Matched {
		t.Error("Expected negated match to be false for a normally matching game")
	}
}

func TestApplyFiltersPlyBounds(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()

	// processorTestPGN has 6 plies: 1. e4 e5 2. Nf3 Nc6 3. Bb5 a6
	game := testutil.MustParseGame(t, processorTestPGN)
	buf := &bytes.Buffer{}
	ctx := newTestContext(buf)

	t.Run("minPly too high", func(t *testing.T) {
		resetGlobalState(t)
		*minPly = 20
		result := applyFilters(game, ctx)
		if result.Matched {
			t.Error("Expected game with 6 plies to fail minPly=20")
		}
		*minPly = 0
	})

	t.Run("minPly within range", func(t *testing.T) {
		resetGlobalState(t)
		*minPly = 4
		result := applyFilters(game, ctx)
		if !result.Matched {
			t.Error("Expected game with 6 plies to pass minPly=4")
		}
		*minPly = 0
	})
}

func TestApplyFiltersCheckmate(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()
	*checkmateFilter = true

	// Simple game that does NOT end in checkmate
	game := testutil.MustParseGame(t, processorTestPGN)
	buf := &bytes.Buffer{}
	ctx := newTestContext(buf)

	result := applyFilters(game, ctx)
	if result.Matched {
		t.Error("Expected non-checkmate game to fail checkmateFilter")
	}

	// Scholar's mate (checkmate)
	checkmatePGN := `[Event "Checkmate"]
[Site "?"]
[Date "????.??.??"]
[Round "?"]
[White "W"]
[Black "B"]
[Result "1-0"]

1. e4 e5 2. Bc4 Nc6 3. Qh5 Nf6 4. Qxf7# 1-0`
	checkmateGame := testutil.MustParseGame(t, checkmatePGN)
	result2 := applyFilters(checkmateGame, ctx)
	if !result2.Matched {
		t.Error("Expected checkmate game to pass checkmateFilter")
	}
}

func TestHandleGameOutput(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()

	t.Run("no detector", func(t *testing.T) {
		resetGlobalState(t)
		game := testutil.MustParseGame(t, processorTestPGN)
		buf := &bytes.Buffer{}
		ctx := newTestContext(buf)
		var jsonGames []*chess.Game

		out, dup := handleGameOutput(game, nil, nil, ctx, &jsonGames)
		if out != 1 || dup != 0 {
			t.Errorf("Expected (1,0), got (%d,%d)", out, dup)
		}
		if buf.Len() == 0 {
			t.Error("Expected game written to output")
		}
	})

	t.Run("detector unique game", func(t *testing.T) {
		resetGlobalState(t)
		game := testutil.MustParseGame(t, processorTestPGN)
		buf := &bytes.Buffer{}
		ctx := newTestContext(buf)
		ctx.detector = hashing.NewDuplicateDetector(false, 0)
		var jsonGames []*chess.Game

		out, dup := handleGameOutput(game, nil, nil, ctx, &jsonGames)
		if out != 1 || dup != 0 {
			t.Errorf("Expected (1,0), got (%d,%d)", out, dup)
		}
	})

	t.Run("detector duplicate game", func(t *testing.T) {
		resetGlobalState(t)
		game1 := testutil.MustParseGame(t, processorTestPGN)
		game2 := testutil.MustParseGame(t, processorTestPGN) // same moves

		buf := &bytes.Buffer{}
		ctx := newTestContext(buf)
		ctx.detector = hashing.NewDuplicateDetector(false, 0)
		var jsonGames []*chess.Game

		// First game is unique
		handleGameOutput(game1, nil, nil, ctx, &jsonGames)
		resetGlobalState(t) // reset matchedCount for clarity

		// Second game is duplicate
		out, dup := handleGameOutput(game2, nil, nil, ctx, &jsonGames)
		if out != 0 || dup != 1 {
			t.Errorf("Expected (0,1) for duplicate, got (%d,%d)", out, dup)
		}
	})

	t.Run("detector duplicate with SuppressOriginals", func(t *testing.T) {
		resetGlobalState(t)
		game1 := testutil.MustParseGame(t, processorTestPGN)
		game2 := testutil.MustParseGame(t, processorTestPGN)

		buf := &bytes.Buffer{}
		ctx := newTestContext(buf)
		ctx.cfg.Duplicate.SuppressOriginals = true
		ctx.detector = hashing.NewDuplicateDetector(false, 0)
		var jsonGames []*chess.Game

		handleGameOutput(game1, nil, nil, ctx, &jsonGames)
		resetGlobalState(t)

		out, dup := handleGameOutput(game2, nil, nil, ctx, &jsonGames)
		if out != 1 || dup != 1 {
			t.Errorf("Expected (1,1) for duplicate+SuppressOriginals, got (%d,%d)", out, dup)
		}
	})
}

func TestOutputGameWithECOSplit(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()

	t.Run("JSON format collects game", func(t *testing.T) {
		cfg := config.NewConfig()
		cfg.Output.JSONFormat = true
		cfg.OutputFile = &bytes.Buffer{}
		var jsonGames []*chess.Game
		game := testutil.MustParseGame(t, processorTestPGN)

		outputGameWithECOSplit(game, cfg, nil, &jsonGames, nil)

		if len(jsonGames) != 1 {
			t.Errorf("Expected 1 game in jsonGames, got %d", len(jsonGames))
		}
	})

	t.Run("no JSON no ECO writes to output", func(t *testing.T) {
		cfg := config.NewConfig()
		buf := &bytes.Buffer{}
		cfg.OutputFile = buf
		var jsonGames []*chess.Game
		game := testutil.MustParseGame(t, processorTestPGN)

		outputGameWithECOSplit(game, cfg, nil, &jsonGames, nil)

		if buf.Len() == 0 {
			t.Error("Expected game written to output buffer")
		}
		if !strings.Contains(buf.String(), "[Event") {
			t.Error("Output should contain game tags")
		}
	})
}

// ============================================================
// Task 3: Sequential and parallel processing pipelines
// ============================================================

func TestOutputGamesSequential(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()
	*quiet = true

	games := testutil.MustParseGames(t, threeGamePGN)
	buf := &bytes.Buffer{}
	ctx := newTestContext(buf)

	out, dup := outputGamesSequential(games, ctx)

	if out != 3 {
		t.Errorf("Expected 3 games output, got %d", out)
	}
	if dup != 0 {
		t.Errorf("Expected 0 duplicates, got %d", dup)
	}
	// Verify output has all game events
	for _, event := range []string{"Test", "Test2", "Test3"} {
		if !strings.Contains(buf.String(), event) {
			t.Errorf("Output missing event %q", event)
		}
	}
}

func TestOutputGamesSequentialStopAfter(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()
	*stopAfter = 1
	*quiet = true

	games := testutil.MustParseGames(t, threeGamePGN)
	buf := &bytes.Buffer{}
	ctx := newTestContext(buf)

	out, _ := outputGamesSequential(games, ctx)

	if out != 1 {
		t.Errorf("Expected 1 game output with stopAfter=1, got %d", out)
	}
}

func TestOutputGamesSequentialSelectOnly(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()
	*quiet = true

	// selectOnly=2 means only output the 2nd game
	selectOnlySet = map[int]bool{2: true}

	games := testutil.MustParseGames(t, threeGamePGN)
	buf := &bytes.Buffer{}
	ctx := newTestContext(buf)

	out, _ := outputGamesSequential(games, ctx)

	if out != 1 {
		t.Errorf("Expected 1 game output with selectOnly=2, got %d", out)
	}
	if !strings.Contains(buf.String(), "Test2") {
		t.Error("Expected output to contain second game (Test2)")
	}
}

func TestOutputGamesSequentialReportOnly(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()
	*reportOnly = true
	*quiet = true

	games := testutil.MustParseGames(t, threeGamePGN)
	buf := &bytes.Buffer{}
	ctx := newTestContext(buf)

	out, _ := outputGamesSequential(games, ctx)

	if out != 3 {
		t.Errorf("Expected 3 games counted in reportOnly, got %d", out)
	}
	if buf.Len() != 0 {
		t.Errorf("Expected no output in reportOnly mode, got %d bytes", buf.Len())
	}
}

func TestProcessGameWorker(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()

	game := testutil.MustParseGame(t, processorTestPGN)
	buf := &bytes.Buffer{}
	ctx := newTestContext(buf)

	item := worker.WorkItem{Game: game, Index: 0}
	result := processGameWorker(item, ctx)

	if !result.Matched {
		t.Error("Expected game to match with no filters")
	}
	if !result.ShouldOutput {
		t.Error("Expected ShouldOutput=true with no filters")
	}
	if result.Game != game {
		t.Error("Expected result.Game to be the same game")
	}
}

func TestOutputGamesWithProcessingRouting(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()
	*quiet = true

	games := testutil.MustParseGames(t, threeGamePGN)

	t.Run("workers=1 routes to sequential", func(t *testing.T) {
		resetGlobalState(t)
		*workers = 1
		buf := &bytes.Buffer{}
		ctx := newTestContext(buf)

		out, dup := outputGamesWithProcessing(games, ctx)
		if out != 3 {
			t.Errorf("Expected 3 games output with workers=1, got %d", out)
		}
		if dup != 0 {
			t.Errorf("Expected 0 duplicates, got %d", dup)
		}
	})

	t.Run("workers>1 with enough games", func(t *testing.T) {
		resetGlobalState(t)
		*workers = 2
		buf := &bytes.Buffer{}
		ctx := newTestContext(buf)

		out, dup := outputGamesWithProcessing(games, ctx)
		if out != 3 {
			t.Errorf("Expected 3 games output with workers=2, got %d", out)
		}
		if dup != 0 {
			t.Errorf("Expected 0 duplicates, got %d", dup)
		}
	})
}

func TestOutputGamesParallel(t *testing.T) {
	resetGlobalState(t)
	restore := saveFlagPointers(t)
	defer restore()
	*quiet = true

	// Create 5+ games for parallel processing
	fiveGamePGN := threeGamePGN + "\n\n" + `[Event "Test4"]
[Site "Test"]
[Date "2024.01.04"]
[Round "4"]
[White "Player7"]
[Black "Player8"]
[Result "1-0"]

1. c4 c5 2. Nc3 Nc6 1-0` + "\n\n" + `[Event "Test5"]
[Site "Test"]
[Date "2024.01.05"]
[Round "5"]
[White "Player9"]
[Black "Player10"]
[Result "0-1"]

1. f4 e5 2. fxe5 d6 0-1`

	games := testutil.MustParseGames(t, fiveGamePGN)
	if len(games) < 5 {
		t.Fatalf("Expected at least 5 games, got %d", len(games))
	}

	buf := &bytes.Buffer{}
	ctx := newTestContext(buf)

	out, dup := outputGamesParallel(games, ctx, 2)

	if out != len(games) {
		t.Errorf("Expected %d games output, got %d", len(games), out)
	}
	if dup != 0 {
		t.Errorf("Expected 0 duplicates, got %d", dup)
	}
	if buf.Len() == 0 {
		t.Error("Expected output to be non-empty")
	}
}
