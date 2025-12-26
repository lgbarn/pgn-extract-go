package processing

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/testutil"
)

// TestAnalyzeGame verifies game analysis functionality
func TestAnalyzeGame(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	board, analysis := AnalyzeGame(game)

	if board == nil {
		t.Error("Expected non-nil board")
	}
	if analysis == nil {
		t.Error("Expected non-nil analysis")
	}
}

// TestAnalyzeGame_Repetition verifies repetition detection
func TestAnalyzeGame_Repetition(t *testing.T) {
	// A game with threefold repetition
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "1/2-1/2"]

1. Nf3 Nf6 2. Ng1 Ng8 3. Nf3 Nf6 4. Ng1 Ng8 5. Nf3 Nf6 1/2-1/2
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	_, analysis := AnalyzeGame(game)

	if !analysis.HasRepetition {
		t.Error("Expected repetition to be detected")
	}
}

// TestAnalyzeGame_Underpromotion verifies underpromotion detection
func TestAnalyzeGame_Underpromotion(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "1-0"]

1. e4 d5 2. exd5 c6 3. dxc6 Nf6 4. cxb7 Bd7 5. bxa8=N 1-0
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	_, analysis := AnalyzeGame(game)

	if !analysis.HasUnderpromotion {
		t.Error("Expected underpromotion to be detected")
	}
}

// TestValidateGame verifies game validation
func TestValidateGame(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 *
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	result := ValidateGame(game)

	if !result.Valid {
		t.Errorf("Expected valid game, got error: %s", result.ErrorMsg)
	}
}

// TestCountPlies verifies ply counting
func TestCountPlies(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 *
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	count := CountPlies(game)
	if count != 5 {
		t.Errorf("CountPlies = %d, want 5", count)
	}
}

// TestReplayGame verifies game replay returns final position
func TestReplayGame(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 *
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	board := ReplayGame(game)

	if board == nil {
		t.Fatal("Expected non-nil board")
	}

	// After 1. e4, the e4 square should have a white pawn
	piece := board.Get('e', '4')
	if piece != chess.W(chess.Pawn) {
		t.Errorf("Expected white pawn on e4, got %v", piece)
	}
}

// TestGameAnalysis_Interface verifies GameAnalysis implements worker.GameInfo
func TestGameAnalysis_Interface(t *testing.T) {
	analysis := &GameAnalysis{
		HasFiftyMoveRule:  true,
		HasRepetition:     false,
		HasUnderpromotion: true,
	}

	if !analysis.FiftyMoveTriggered() {
		t.Error("FiftyMoveTriggered should return true")
	}
	if analysis.RepetitionDetected() {
		t.Error("RepetitionDetected should return false")
	}
	if !analysis.UnderpromotionFound() {
		t.Error("UnderpromotionFound should return true")
	}
}

// TestValidationResult_Fields verifies ValidationResult structure
func TestValidationResult_Fields(t *testing.T) {
	result := ValidationResult{
		Valid:       false,
		ErrorPly:    5,
		ErrorMsg:    "Illegal move",
		ParseErrors: []string{"warning1", "warning2"},
	}

	if result.Valid {
		t.Error("Expected Valid to be false")
	}
	if result.ErrorPly != 5 {
		t.Errorf("ErrorPly = %d, want 5", result.ErrorPly)
	}
	if result.ErrorMsg != "Illegal move" {
		t.Errorf("ErrorMsg = %s, want 'Illegal move'", result.ErrorMsg)
	}
	if len(result.ParseErrors) != 2 {
		t.Errorf("ParseErrors length = %d, want 2", len(result.ParseErrors))
	}
}
