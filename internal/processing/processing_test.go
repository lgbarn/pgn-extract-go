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

	if got := analysis.FiftyMoveTriggered(); got != true {
		t.Errorf("FiftyMoveTriggered() = %v, want true", got)
	}
	if got := analysis.RepetitionDetected(); got != false {
		t.Errorf("RepetitionDetected() = %v, want false", got)
	}
	if got := analysis.UnderpromotionFound(); got != true {
		t.Errorf("UnderpromotionFound() = %v, want true", got)
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

	if result.Valid != false {
		t.Errorf("ValidationResult.Valid = %v, want false", result.Valid)
	}
	if result.ErrorPly != 5 {
		t.Errorf("ValidationResult.ErrorPly = %d, want 5", result.ErrorPly)
	}
	if result.ErrorMsg != "Illegal move" {
		t.Errorf("ValidationResult.ErrorMsg = %q, want \"Illegal move\"", result.ErrorMsg)
	}
	if len(result.ParseErrors) != 2 {
		t.Errorf("len(ValidationResult.ParseErrors) = %d, want 2", len(result.ParseErrors))
	}
}

// ============== Extended Draw Rule Tests ==============

// TestAnalyzeGame_75MoveRule tests detection of 75-move rule
func TestAnalyzeGame_75MoveRule_NotTriggered(t *testing.T) {
	// A short game doesn't trigger the rule
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 Nc6 *
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	_, analysis := AnalyzeGame(game)

	if analysis.Has75MoveRule {
		t.Errorf("AnalyzeGame(short game).Has75MoveRule = true, want false")
	}
}

// TestAnalyzeGame_5FoldRepetition_NotTriggered tests that 5-fold is not triggered for 3-fold
func TestAnalyzeGame_5FoldRepetition_NotTriggered(t *testing.T) {
	// A game with only 3-fold repetition
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
		t.Errorf("AnalyzeGame(3-fold repetition game).HasRepetition = false, want true")
	}
	if analysis.Has5FoldRepetition {
		t.Errorf("AnalyzeGame(3-fold repetition game).Has5FoldRepetition = true, want false")
	}
}

// ============== Insufficient Material Tests ==============

// TestAnalyzeGame_InsufficientMaterial tests insufficient material detection via AnalyzeGame
func TestAnalyzeGame_InsufficientMaterial(t *testing.T) {
	tests := []struct {
		name   string
		fen    string
		move   string
		result string
		want   bool // true = insufficient material expected
	}{
		{
			name:   "K vs K",
			fen:    "8/8/8/4k3/8/8/8/4K3 w - - 0 1",
			move:   "1. Ke2 Ke4",
			result: "1/2-1/2",
			want:   true,
		},
		{
			name:   "K+B vs K",
			fen:    "8/8/8/4k3/8/8/8/4K2B w - - 0 1",
			move:   "1. Ke2",
			result: "1/2-1/2",
			want:   true,
		},
		{
			name:   "K+N vs K",
			fen:    "8/8/8/4k3/8/8/8/4K2N w - - 0 1",
			move:   "1. Ke2",
			result: "1/2-1/2",
			want:   true,
		},
		{
			name:   "K+B vs K+B same color bishops",
			fen:    "4k3/8/8/8/8/5b2/8/4K2B w - - 0 1",
			move:   "1. Ke2",
			result: "1/2-1/2",
			want:   true,
		},
		{
			name:   "K+R vs K (sufficient)",
			fen:    "8/8/8/4k3/8/8/8/4K2R w - - 0 1",
			move:   "1. Ke2",
			result: "*",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgn := `[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "` + tt.result + `"]
[FEN "` + tt.fen + `"]
[SetUp "1"]

` + tt.move + ` ` + tt.result + `
`
			game := testutil.ParseTestGame(pgn)
			if game == nil {
				t.Fatalf("Failed to parse test game")
			}

			_, analysis := AnalyzeGame(game)

			if analysis.HasInsufficientMaterial != tt.want {
				t.Errorf("HasInsufficientMaterial = %v, want %v", analysis.HasInsufficientMaterial, tt.want)
			}
		})
	}
}

// ============== SplitVariations Tests ==============

// TestSplitVariations_NoVariations tests splitting a game with no variations
func TestSplitVariations_NoVariations(t *testing.T) {
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

	games := SplitVariations(game)

	if len(games) != 1 {
		t.Errorf("SplitVariations() returned %d games, want 1", len(games))
	}

	// Verify main line preserved
	count := CountPlies(games[0])
	if count != 3 {
		t.Errorf("Main line has %d plies, want 3", count)
	}
}

// TestSplitVariations_SingleVariation tests splitting a game with one variation
func TestSplitVariations_SingleVariation(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 (1... c5) 2. Nf3 *
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	games := SplitVariations(game)

	if len(games) != 2 {
		t.Errorf("SplitVariations() returned %d games, want 2", len(games))
	}
}

// TestSplitVariations_PreservesHeaders tests that headers are preserved
func TestSplitVariations_PreservesHeaders(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test Event"]
[Site "Test Site"]
[Date "2024.01.01"]
[Round "5"]
[White "Player A"]
[Black "Player B"]
[Result "1-0"]

1. e4 e5 (1... c5) 2. Nf3 1-0
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	games := SplitVariations(game)

	for i, g := range games {
		if g.GetTag("Event") != "Test Event" {
			t.Errorf("Game %d missing Event tag", i)
		}
		if g.GetTag("White") != "Player A" {
			t.Errorf("Game %d missing White tag", i)
		}
		if g.GetTag("Black") != "Player B" {
			t.Errorf("Game %d missing Black tag", i)
		}
	}
}

// TestSplitVariations_EmptyGame tests splitting an empty game
func TestSplitVariations_EmptyGame(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

*
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	games := SplitVariations(game)

	if len(games) != 1 {
		t.Errorf("SplitVariations() for empty game returned %d games, want 1", len(games))
	}
}

// ============== HasComments Tests ==============

// TestHasComments_True tests detection of comments
func TestHasComments_True(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 {A good move} e5 *
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	if !HasComments(game) {
		t.Error("HasComments should return true for game with comments")
	}
}

// TestHasComments_False tests detection when no comments present
func TestHasComments_False(t *testing.T) {
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

	if HasComments(game) {
		t.Error("HasComments should return false for game without comments")
	}
}

// TestHasComments_EmptyGame tests game with no moves
func TestHasComments_EmptyGame(t *testing.T) {
	game := &chess.Game{
		Tags:  make(map[string]string),
		Moves: nil,
	}

	if HasComments(game) {
		t.Error("HasComments should return false for empty game")
	}
}

// ============== Material Odds Tests ==============

// TestAnalyzeGame_MaterialOdds tests material odds detection
func TestAnalyzeGame_MaterialOdds(t *testing.T) {
	// Game starting without white's queen
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]
[FEN "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNB1KBNR w KQkq - 0 1"]
[SetUp "1"]

1. e4 *
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	_, analysis := AnalyzeGame(game)

	if !analysis.HasMaterialOdds {
		t.Error("Game starting without queen should have material odds")
	}
}

// TestAnalyzeGame_NoMaterialOdds tests standard position has no material odds
func TestAnalyzeGame_NoMaterialOdds(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 *
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	_, analysis := AnalyzeGame(game)

	if analysis.HasMaterialOdds {
		t.Error("Standard game should not have material odds")
	}
}

// ============== ValidateGame Additional Tests ==============

// TestValidateGame_MissingTags tests validation with missing required tags
func TestValidateGame_MissingTags(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"Event": "Test",
			// Missing other required tags
		},
	}

	result := ValidateGame(game)

	if len(result.ParseErrors) == 0 {
		t.Error("ValidateGame should report missing required tags")
	}
}

// TestValidateGame_InvalidResult tests validation with invalid result
func TestValidateGame_InvalidResult(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"Event":  "Test",
			"Site":   "Test",
			"Date":   "2024.01.01",
			"Round":  "1",
			"White":  "A",
			"Black":  "B",
			"Result": "invalid",
		},
	}

	result := ValidateGame(game)

	hasResultError := false
	for _, err := range result.ParseErrors {
		if err == "invalid result: invalid" {
			hasResultError = true
			break
		}
	}

	if !hasResultError {
		t.Error("ValidateGame should report invalid result")
	}
}

// TestValidateGame_InvalidFEN tests validation with invalid FEN
func TestValidateGame_InvalidFEN(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"Event":  "Test",
			"Site":   "Test",
			"Date":   "2024.01.01",
			"Round":  "1",
			"White":  "A",
			"Black":  "B",
			"Result": "*",
			"FEN":    "invalid fen string",
		},
		Moves: &chess.Move{Text: "e4"},
	}

	result := ValidateGame(game)

	if result.Valid {
		t.Error("ValidateGame should return invalid for invalid FEN")
	}
	if result.ErrorMsg == "" {
		t.Error("ValidateGame should provide error message for invalid FEN")
	}
}

// TestValidateGame_NoMoves tests validation of game with no moves
func TestValidateGame_NoMoves(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"Event":  "Test",
			"Site":   "Test",
			"Date":   "2024.01.01",
			"Round":  "1",
			"White":  "A",
			"Black":  "B",
			"Result": "*",
		},
		Moves: nil,
	}

	result := ValidateGame(game)

	if !result.Valid {
		t.Error("ValidateGame should return valid for game with no moves but valid tags")
	}
}

// TestIsValidResult tests the result validation
func TestIsValidResult(t *testing.T) {
	tests := []struct {
		result string
		want   bool
	}{
		{"1-0", true},
		{"0-1", true},
		{"1/2-1/2", true},
		{"*", true},
		{"", false},
		{"invalid", false},
		{"1-1", false},
		{"1/2", false},
	}

	for _, tt := range tests {
		t.Run(tt.result, func(t *testing.T) {
			got := isValidResult(tt.result)
			if got != tt.want {
				t.Errorf("isValidResult(%q) = %v, want %v", tt.result, got, tt.want)
			}
		})
	}
}

// TestCountPlies_Empty tests counting plies in empty game
func TestCountPlies_Empty(t *testing.T) {
	game := &chess.Game{Moves: nil}
	if CountPlies(game) != 0 {
		t.Error("CountPlies of empty game should be 0")
	}
}

// TestReplayGame_EmptyGame tests replaying empty game
func TestReplayGame_EmptyGame(t *testing.T) {
	game := &chess.Game{
		Tags:  make(map[string]string),
		Moves: nil,
	}

	board := ReplayGame(game)
	if board == nil {
		t.Fatal("ReplayGame should return non-nil board")
	}

	// Should be initial position
	if board.Get('e', '1') != chess.W(chess.King) {
		t.Error("ReplayGame empty game should return initial position")
	}
}

// TestReplayGame_WithFEN tests replaying game with custom FEN
func TestReplayGame_WithFEN(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]
[FEN "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"]
[SetUp "1"]

1... e5 *
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	board := ReplayGame(game)
	if board == nil {
		t.Fatal("ReplayGame should return non-nil board")
	}

	// After 1...e5, the e5 square should have a black pawn
	if board.Get('e', '5') != chess.B(chess.Pawn) {
		t.Error("Expected black pawn on e5 after 1...e5")
	}
}
