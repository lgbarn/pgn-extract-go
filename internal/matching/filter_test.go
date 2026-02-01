package matching

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/testutil"
)

func TestNewGameFilter(t *testing.T) {
	gf := NewGameFilter()
	if gf == nil {
		t.Fatal("NewGameFilter returned nil")
	}
	if gf.TagMatcher == nil {
		t.Error("TagMatcher should not be nil")
	}
	if gf.PositionMatcher == nil {
		t.Error("PositionMatcher should not be nil")
	}
	if gf.RequireBoth {
		t.Error("RequireBoth should default to false")
	}
}

func TestGameFilter_HasCriteria(t *testing.T) {
	gf := NewGameFilter()
	if gf.HasCriteria() {
		t.Error("New filter should have no criteria")
	}

	gf.AddWhiteFilter("Fischer")
	if !gf.HasCriteria() {
		t.Error("Filter should have criteria after AddWhiteFilter")
	}

	gf2 := NewGameFilter()
	gf2.AddPatternFilter("???????*/????????/8/8/8/8/????????/*??????", false)
	if !gf2.HasCriteria() {
		t.Error("Filter should have criteria after AddPatternFilter")
	}
}

func TestGameFilter_AddWhiteFilter(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White":  "Kasparov, Garry",
			"Black":  "Karpov, Anatoly",
			"Result": "1-0",
		},
	}

	gf := NewGameFilter()
	gf.AddWhiteFilter("Kasparov")

	if !gf.MatchGame(game) {
		t.Error("Should match White player substring")
	}

	gf2 := NewGameFilter()
	gf2.AddWhiteFilter("Karpov")

	if gf2.MatchGame(game) {
		t.Error("Should not match when Karpov is Black, not White")
	}
}

func TestGameFilter_AddBlackFilter(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White":  "Kasparov, Garry",
			"Black":  "Karpov, Anatoly",
			"Result": "1-0",
		},
	}

	gf := NewGameFilter()
	gf.AddBlackFilter("Karpov")

	if !gf.MatchGame(game) {
		t.Error("Should match Black player substring")
	}

	gf2 := NewGameFilter()
	gf2.AddBlackFilter("Kasparov")

	if gf2.MatchGame(game) {
		t.Error("Should not match when Kasparov is White, not Black")
	}
}

func TestGameFilter_AddDateFilter(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"Date": "1985.11.09",
		},
	}

	tests := []struct {
		name     string
		date     string
		op       TagOperator
		expected bool
	}{
		{"after 1980", "1980.01.01", OpGreaterThan, true},
		{"before 1990", "1990.01.01", OpLessThan, true},
		{"not before 1980", "1980.01.01", OpLessThan, false},
		{"on or after exact", "1985.11.09", OpGreaterOrEqual, true},
		{"on or before exact", "1985.11.09", OpLessOrEqual, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gf := NewGameFilter()
			gf.AddDateFilter(tt.date, tt.op)
			if gf.MatchGame(game) != tt.expected {
				t.Errorf("AddDateFilter(%s, %v): got %v, want %v", tt.date, tt.op, !tt.expected, tt.expected)
			}
		})
	}
}

func TestGameFilter_AddTagCriterion(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"Event":  "Candidates",
			"Site":   "London",
			"Result": "1/2-1/2",
		},
	}

	gf := NewGameFilter()
	gf.AddTagCriterion("Event", "Candidates", OpEqual)

	if !gf.MatchGame(game) {
		t.Error("Should match Event = Candidates")
	}

	gf2 := NewGameFilter()
	gf2.AddTagCriterion("Site", "Moscow", OpEqual)

	if gf2.MatchGame(game) {
		t.Error("Should not match Site = Moscow")
	}
}

func TestGameFilter_AddFENFilter(t *testing.T) {
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

	gf := NewGameFilter()
	// Ruy Lopez position after 3. Bb5
	err := gf.AddFENFilter("r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3")
	if err != nil {
		t.Fatalf("AddFENFilter failed: %v", err)
	}

	if !gf.HasCriteria() {
		t.Error("Should have criteria after AddFENFilter")
	}

	if !gf.MatchGame(game) {
		t.Error("Should match Ruy Lopez position")
	}
}

func TestGameFilter_AddFENFilter_InvalidFEN(t *testing.T) {
	gf := NewGameFilter()
	err := gf.AddFENFilter("not a valid fen string")
	if err == nil {
		t.Error("AddFENFilter should return error for invalid FEN")
	}
}

func TestGameFilter_AddPatternFilter(t *testing.T) {
	gf := NewGameFilter()
	gf.AddPatternFilter("???????*/????????/8/8/8/8/????????/*??????", false)

	if gf.PositionMatcher.PatternCount() != 1 {
		t.Errorf("Expected 1 pattern, got %d", gf.PositionMatcher.PatternCount())
	}

	// With invert, should add 2 patterns
	gf2 := NewGameFilter()
	gf2.AddPatternFilter("???????*/????????/8/8/8/8/????????/*??????", true)

	if gf2.PositionMatcher.PatternCount() != 2 {
		t.Errorf("Expected 2 patterns (original + inverted), got %d", gf2.PositionMatcher.PatternCount())
	}
}

func TestGameFilter_MatchGame_NoCriteria(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White":  "Test",
			"Black":  "Test",
			"Result": "*",
		},
	}

	gf := NewGameFilter()
	if !gf.MatchGame(game) {
		t.Error("Filter with no criteria should match all games")
	}
}

func TestGameFilter_MatchGame_CombinedTagAndPosition(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Fischer, Robert"]
[Black "Spassky, Boris"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 1-0
`)

	// Both tag and position criteria - both must match
	gf := NewGameFilter()
	gf.AddWhiteFilter("Fischer")
	err := gf.AddFENFilter("r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3")
	if err != nil {
		t.Fatalf("AddFENFilter failed: %v", err)
	}

	if !gf.MatchGame(game) {
		t.Error("Should match when both tag and position match")
	}

	// Tag matches but position does not
	gf2 := NewGameFilter()
	gf2.AddWhiteFilter("Fischer")
	err = gf2.AddFENFilter("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKB1R w KQkq - 0 1") // won't appear
	if err != nil {
		t.Fatalf("AddFENFilter failed: %v", err)
	}

	if gf2.MatchGame(game) {
		t.Error("Should not match when position criteria does not match")
	}

	// Position matches but tag does not
	gf3 := NewGameFilter()
	gf3.AddWhiteFilter("Karpov")
	err = gf3.AddFENFilter("r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3")
	if err != nil {
		t.Fatalf("AddFENFilter failed: %v", err)
	}

	if gf3.MatchGame(game) {
		t.Error("Should not match when tag criteria does not match")
	}
}

func TestGameFilter_Match_DelegatesToMatchGame(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White":  "Fischer",
			"Black":  "Spassky",
			"Result": "1-0",
		},
	}

	gf := NewGameFilter()
	gf.AddResultFilter("1-0")

	// Match should behave identically to MatchGame
	if gf.Match(game) != gf.MatchGame(game) {
		t.Error("Match() and MatchGame() should return the same result")
	}
}

func TestGameFilter_SetUseSoundex(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White": "Fischer",
			"Black": "Spassky",
		},
	}

	gf := NewGameFilter()
	gf.SetUseSoundex(true)
	gf.AddPlayerFilter("Fisher") // soundex match for Fischer

	if !gf.MatchGame(game) {
		t.Error("Should match via soundex (Fisher ~ Fischer)")
	}

	// Verify soundex flag is propagated
	if !gf.TagMatcher.useSoundex {
		t.Error("SetUseSoundex should propagate to TagMatcher")
	}
}

func TestGameFilter_SetSubstringMatch(t *testing.T) {
	gf := NewGameFilter()
	gf.SetSubstringMatch(true)
	// This sets the flag on the TagMatcher; verify it's accessible
	if !gf.TagMatcher.substringMatch {
		t.Error("SetSubstringMatch should set the flag on TagMatcher")
	}
}

func TestGameFilter_LoadTagFile(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name         string
		content      string
		wantTags     int
		wantPatterns int
	}{
		{
			name: "tag criteria only",
			content: `White "Fischer"
Result = "1-0"
Date >= "1970.01.01"`,
			wantTags:     3,
			wantPatterns: 0,
		},
		{
			name:         "FEN exact",
			content:      `FEN "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"`,
			wantTags:     0,
			wantPatterns: 1,
		},
		{
			name:         "FEN pattern with wildcards",
			content:      `FEN "???????*/????????/8/8/8/8/????????/*???????"`,
			wantTags:     0,
			wantPatterns: 1,
		},
		{
			name:         "FENPattern keyword",
			content:      `FENPattern "???????*/????????/8/8/8/8/????????/*???????"`,
			wantTags:     0,
			wantPatterns: 1,
		},
		{
			name: "mixed criteria",
			content: `# Comment line
White "Kasparov"

FEN "???????*/????????/8/8/8/8/????????/*???????"
Result = "1-0"`,
			wantTags:     2,
			wantPatterns: 1,
		},
		{
			name:         "empty and comments only",
			content:      "# Just a comment\n\n# Another comment\n",
			wantTags:     0,
			wantPatterns: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(dir, tt.name+".txt")
			if err := os.WriteFile(filename, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}

			gf := NewGameFilter()
			if err := gf.LoadTagFile(filename); err != nil {
				t.Fatalf("LoadTagFile failed: %v", err)
			}

			if gf.TagMatcher.CriteriaCount() != tt.wantTags {
				t.Errorf("Tag criteria count: got %d, want %d", gf.TagMatcher.CriteriaCount(), tt.wantTags)
			}
			if gf.PositionMatcher.PatternCount() != tt.wantPatterns {
				t.Errorf("Pattern count: got %d, want %d", gf.PositionMatcher.PatternCount(), tt.wantPatterns)
			}
		})
	}
}

func TestGameFilter_LoadTagFile_NonExistent(t *testing.T) {
	gf := NewGameFilter()
	err := gf.LoadTagFile("/nonexistent/path/to/file.txt")
	if err == nil {
		t.Error("LoadTagFile should return error for nonexistent file")
	}
}

func TestGameFilter_LoadTagFile_Integration(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "criteria.txt")
	content := `White = "Fischer, Robert"
Result = "1-0"
`
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	gf := NewGameFilter()
	if err := gf.LoadTagFile(filename); err != nil {
		t.Fatalf("LoadTagFile failed: %v", err)
	}

	// Game that matches both criteria
	matchGame := &chess.Game{
		Tags: map[string]string{
			"White":  "Fischer, Robert",
			"Black":  "Spassky, Boris",
			"Result": "1-0",
		},
	}
	if !gf.MatchGame(matchGame) {
		t.Error("Should match game with Fischer as White and result 1-0")
	}

	// Game that matches only one criterion (AND mode)
	partialGame := &chess.Game{
		Tags: map[string]string{
			"White":  "Fischer, Robert",
			"Black":  "Spassky, Boris",
			"Result": "0-1",
		},
	}
	if gf.MatchGame(partialGame) {
		t.Error("Should not match when Result does not match (AND mode)")
	}
}
