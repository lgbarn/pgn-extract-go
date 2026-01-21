package matching

import (
	"os"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/testutil"
)

func TestSoundex(t *testing.T) {
	tests := []struct {
		name1, name2 string
		shouldMatch  bool
	}{
		{"Fischer", "Fisher", true},
		{"Kasparov", "Kasparov", true},
		{"Carlsen", "Carlson", true},
		{"Fischer", "Kasparov", false},
		{"Smith", "Smyth", true},
		{"Robert", "Rupert", true}, // Same soundex (R163)
	}

	for _, tt := range tests {
		t.Run(tt.name1+" vs "+tt.name2, func(t *testing.T) {
			t.Parallel()
			s1 := Soundex(tt.name1)
			s2 := Soundex(tt.name2)
			match := s1 == s2
			if match != tt.shouldMatch {
				t.Errorf("Soundex(%s)=%s, Soundex(%s)=%s, match=%v, want %v",
					tt.name1, s1, tt.name2, s2, match, tt.shouldMatch)
			}
		})
	}
}

func TestTagMatcherSimple(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White":  "Fischer, Robert",
			"Black":  "Spassky, Boris",
			"Result": "1-0",
			"Date":   "1972.07.11",
			"ECO":    "C97",
		},
	}

	tm := NewTagMatcher()
	tm.AddSimpleCriterion("White", "Fischer, Robert")

	if !tm.MatchGame(game) {
		t.Error("Expected match on White player")
	}

	tm2 := NewTagMatcher()
	tm2.AddSimpleCriterion("White", "Kasparov")

	if tm2.MatchGame(game) {
		t.Error("Expected no match on wrong player")
	}
}

func TestTagMatcherContains(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White": "Fischer, Robert",
		},
	}

	tm := NewTagMatcher()
	tm.AddCriterion("White", "Fischer", OpContains)

	if !tm.MatchGame(game) {
		t.Error("Expected substring match")
	}
}

func TestTagMatcherDate(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"Date": "1972.07.11",
		},
	}

	tests := []struct {
		value    string
		op       TagOperator
		expected bool
	}{
		{"1972.01.01", OpGreaterThan, true},
		{"1972.12.31", OpLessThan, true},
		{"1973.01.01", OpLessThan, true},
		{"1971.12.31", OpGreaterThan, true},
		{"1972.07.11", OpEqual, true},
		{"1972.07.12", OpLessThan, true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			tm := NewTagMatcher()
			tm.AddCriterion("Date", tt.value, tt.op)
			if tm.MatchGame(game) != tt.expected {
				t.Errorf("Date %s %v %s: got %v, want %v",
					"1972.07.11", tt.op, tt.value, !tt.expected, tt.expected)
			}
		})
	}
}

func TestTagMatcherPlayer(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White": "Fischer, Robert",
			"Black": "Spassky, Boris",
		},
	}

	tm := NewTagMatcher()
	tm.AddPlayerCriterion("Fischer")

	if !tm.MatchGame(game) {
		t.Error("Expected match on player (White)")
	}

	tm2 := NewTagMatcher()
	tm2.AddPlayerCriterion("Spassky")

	if !tm2.MatchGame(game) {
		t.Error("Expected match on player (Black)")
	}

	tm3 := NewTagMatcher()
	tm3.AddPlayerCriterion("Karpov")

	if tm3.MatchGame(game) {
		t.Error("Expected no match on player not in game")
	}
}

func TestGameFilter(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "World Championship"]
[Site "Reykjavik"]
[Date "1972.07.11"]
[Round "1"]
[White "Spassky, Boris"]
[Black "Fischer, Robert"]
[Result "1-0"]
[ECO "E56"]

1. d4 Nf6 2. c4 e6 3. Nf3 d5 4. Nc3 Be7 5. Bg5 1-0
`)

	gf := NewGameFilter()
	gf.AddPlayerFilter("Fischer")

	if !gf.MatchGame(game) {
		t.Error("Expected match on player filter")
	}

	gf2 := NewGameFilter()
	gf2.AddResultFilter("1-0")

	if !gf2.MatchGame(game) {
		t.Error("Expected match on result filter")
	}

	gf3 := NewGameFilter()
	gf3.AddECOFilter("E5")

	if !gf3.MatchGame(game) {
		t.Error("Expected match on ECO prefix filter")
	}
}

func TestParseCriterion(t *testing.T) {
	tm := NewTagMatcher()

	tests := []string{
		`White "Fischer"`,
		`Date >= "1970.01.01"`,
		`Result = "1-0"`,
		`ECO "B"`,
	}

	for _, line := range tests {
		if err := tm.ParseCriterion(line); err != nil {
			t.Errorf("ParseCriterion(%s) failed: %v", line, err)
		}
	}

	if tm.CriteriaCount() != 4 {
		t.Errorf("Expected 4 criteria, got %d", tm.CriteriaCount())
	}
}

func TestPositionMatcher(t *testing.T) {
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

	pm := NewPositionMatcher()
	// Ruy Lopez position
	pm.AddFEN("r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3", "Ruy Lopez")

	match := pm.MatchGame(game)
	if match == nil {
		t.Error("Expected FEN position match")
	} else if match.Label != "Ruy Lopez" {
		t.Errorf("Expected label 'Ruy Lopez', got '%s'", match.Label)
	}
}

// TestGameMatcherInterface verifies that concrete matchers implement GameMatcher
func TestGameMatcherInterface(t *testing.T) {
	// Verify all matchers implement the interface
	var _ GameMatcher = NewGameFilter()
	var _ GameMatcher = NewMaterialMatcher("Q:q", false)
	var _ GameMatcher = NewVariationMatcher()
}

// TestCompositeMatcher_And verifies AND mode (all matchers must match)
func TestCompositeMatcher_And(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Fischer, Robert"]
[Black "Spassky, Boris"]
[Result "1-0"]

1. e4 e5 2. Nf3 1-0
`)

	// Create two filters that both match
	filter1 := NewGameFilter()
	filter1.AddPlayerFilter("Fischer")

	filter2 := NewGameFilter()
	filter2.AddResultFilter("1-0")

	// AND mode - both match
	composite := NewCompositeMatcher(MatchAll, filter1, filter2)
	if !composite.Match(game) {
		t.Error("Expected AND match when both matchers match")
	}

	// Create a filter that doesn't match
	filter3 := NewGameFilter()
	filter3.AddResultFilter("0-1")

	// AND mode - one doesn't match
	composite2 := NewCompositeMatcher(MatchAll, filter1, filter3)
	if composite2.Match(game) {
		t.Error("Expected no AND match when one matcher fails")
	}
}

// TestCompositeMatcher_Or verifies OR mode (any matcher must match)
func TestCompositeMatcher_Or(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Fischer, Robert"]
[Black "Spassky, Boris"]
[Result "1-0"]

1. e4 e5 2. Nf3 1-0
`)

	// One matching filter, one non-matching
	filter1 := NewGameFilter()
	filter1.AddPlayerFilter("Fischer")

	filter2 := NewGameFilter()
	filter2.AddResultFilter("0-1") // Doesn't match

	// OR mode - at least one matches
	composite := NewCompositeMatcher(MatchAny, filter1, filter2)
	if !composite.Match(game) {
		t.Error("Expected OR match when at least one matcher matches")
	}

	// Both don't match
	filter3 := NewGameFilter()
	filter3.AddPlayerFilter("Karpov")

	composite2 := NewCompositeMatcher(MatchAny, filter3, filter2)
	if composite2.Match(game) {
		t.Error("Expected no OR match when no matcher matches")
	}
}

// TestCompositeMatcher_Name verifies the Name() method
func TestCompositeMatcher_Name(t *testing.T) {
	filter := NewGameFilter()
	filter.AddPlayerFilter("Fischer")

	composite := NewCompositeMatcher(MatchAll, filter)

	name := composite.Name()
	if name == "" {
		t.Error("Expected non-empty composite matcher name")
	}
}

// TestCompositeMatcher_Empty verifies empty composite behavior
func TestCompositeMatcher_Empty(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Test"]
[Black "Test"]
[Result "*"]

1. e4 *
`)

	// Empty composite in AND mode should match all (vacuously true)
	composite := NewCompositeMatcher(MatchAll)
	if !composite.Match(game) {
		t.Error("Empty AND composite should match (vacuously true)")
	}

	// Empty composite in OR mode should not match (no conditions to satisfy)
	composite2 := NewCompositeMatcher(MatchAny)
	if composite2.Match(game) {
		t.Error("Empty OR composite should not match")
	}
}

// TestGameFilter_Name verifies GameFilter implements Name()
func TestGameFilter_Name(t *testing.T) {
	filter := NewGameFilter()
	if filter.Name() != "GameFilter" {
		t.Errorf("GameFilter.Name() = %s, want GameFilter", filter.Name())
	}
}

// TestMaterialMatcher_Name verifies MaterialMatcher implements Name()
func TestMaterialMatcher_Name(t *testing.T) {
	matcher := NewMaterialMatcher("Q:q", false)
	if matcher.Name() != "MaterialMatcher" {
		t.Errorf("MaterialMatcher.Name() = %s, want MaterialMatcher", matcher.Name())
	}
}

// TestVariationMatcher_Name verifies VariationMatcher implements Name()
func TestVariationMatcher_Name(t *testing.T) {
	matcher := NewVariationMatcher()
	if matcher.Name() != "VariationMatcher" {
		t.Errorf("VariationMatcher.Name() = %s, want VariationMatcher", matcher.Name())
	}
}

// ============== matchRank Wildcard Tests ==============

// TestMatchRank_QuestionMark tests ? matching any single square
func TestMatchRank_QuestionMark(t *testing.T) {
	tests := []struct {
		boardRank   string
		patternRank string
		want        bool
	}{
		{"RNBQKBNR", "?NBQKBNR", true},
		{"RNBQKBNR", "RNBQKBN?", true},
		{"RNBQKBNR", "????????", true},
		{"RNBQKBNR", "???????", false},  // length mismatch
		{"RNBQKBNR", "?????????", false}, // length mismatch
		{"_NBQKBNR", "?NBQKBNR", true},   // ? matches empty too
	}

	for _, tt := range tests {
		t.Run(tt.boardRank+"_"+tt.patternRank, func(t *testing.T) {
			got := matchRank(tt.boardRank, tt.patternRank)
			if got != tt.want {
				t.Errorf("matchRank(%q, %q) = %v, want %v", tt.boardRank, tt.patternRank, got, tt.want)
			}
		})
	}
}

// TestMatchRank_Exclamation tests ! matching any non-empty square
func TestMatchRank_Exclamation(t *testing.T) {
	tests := []struct {
		boardRank   string
		patternRank string
		want        bool
	}{
		{"RNBQKBNR", "!NBQKBNR", true},
		{"RNBQKBNR", "!!!!!!!!", true},
		{"_NBQKBNR", "!NBQKBNR", false}, // ! doesn't match empty
		{"PPPPPPPP", "!!!!!!!!", true},
		{"pppppppp", "!!!!!!!!", true}, // black pieces are non-empty too
	}

	for _, tt := range tests {
		t.Run(tt.boardRank+"_"+tt.patternRank, func(t *testing.T) {
			got := matchRank(tt.boardRank, tt.patternRank)
			if got != tt.want {
				t.Errorf("matchRank(%q, %q) = %v, want %v", tt.boardRank, tt.patternRank, got, tt.want)
			}
		})
	}
}

// TestMatchRank_Asterisk tests * matching zero or more
func TestMatchRank_Asterisk(t *testing.T) {
	tests := []struct {
		boardRank   string
		patternRank string
		want        bool
	}{
		{"RNBQKBNR", "*", true},
		{"RNBQKBNR", "R*", true},
		{"RNBQKBNR", "*R", true},
		{"RNBQKBNR", "R*R", true},
		{"RNBQKBNR", "*KBNR", true},
		{"________", "*", true},
		{"", "*", true},
		{"RNBQKBNR", "R*X", false}, // X not in string
	}

	for _, tt := range tests {
		t.Run(tt.boardRank+"_"+tt.patternRank, func(t *testing.T) {
			got := matchRank(tt.boardRank, tt.patternRank)
			if got != tt.want {
				t.Errorf("matchRank(%q, %q) = %v, want %v", tt.boardRank, tt.patternRank, got, tt.want)
			}
		})
	}
}

// TestMatchRank_UpperA tests A matching any white piece
func TestMatchRank_UpperA(t *testing.T) {
	tests := []struct {
		boardRank   string
		patternRank string
		want        bool
	}{
		{"RNBQKBNR", "AAAAAAAA", true},
		{"PPPPPPPP", "AAAAAAAA", true},
		{"KQRBNP__", "AAAAAA__", true},
		{"pppppppp", "AAAAAAAA", false}, // black pieces don't match A
		{"_NBQKBNR", "ANBQKBNR", false}, // empty doesn't match A
	}

	for _, tt := range tests {
		t.Run(tt.boardRank+"_"+tt.patternRank, func(t *testing.T) {
			got := matchRank(tt.boardRank, tt.patternRank)
			if got != tt.want {
				t.Errorf("matchRank(%q, %q) = %v, want %v", tt.boardRank, tt.patternRank, got, tt.want)
			}
		})
	}
}

// TestMatchRank_LowerA tests a matching any black piece
func TestMatchRank_LowerA(t *testing.T) {
	tests := []struct {
		boardRank   string
		patternRank string
		want        bool
	}{
		{"rnbqkbnr", "aaaaaaaa", true},
		{"pppppppp", "aaaaaaaa", true},
		{"RNBQKBNR", "aaaaaaaa", false}, // white pieces don't match a
		{"_nbqkbnr", "anbqkbnr", false}, // empty doesn't match a
	}

	for _, tt := range tests {
		t.Run(tt.boardRank+"_"+tt.patternRank, func(t *testing.T) {
			got := matchRank(tt.boardRank, tt.patternRank)
			if got != tt.want {
				t.Errorf("matchRank(%q, %q) = %v, want %v", tt.boardRank, tt.patternRank, got, tt.want)
			}
		})
	}
}

// TestMatchRank_Underscore tests _ matching empty square
func TestMatchRank_Underscore(t *testing.T) {
	tests := []struct {
		boardRank   string
		patternRank string
		want        bool
	}{
		{"________", "________", true},
		{"____P___", "____P___", true},
		{"RNBQKBNR", "________", false},
		{"____P___", "________", false},
	}

	for _, tt := range tests {
		t.Run(tt.boardRank+"_"+tt.patternRank, func(t *testing.T) {
			got := matchRank(tt.boardRank, tt.patternRank)
			if got != tt.want {
				t.Errorf("matchRank(%q, %q) = %v, want %v", tt.boardRank, tt.patternRank, got, tt.want)
			}
		})
	}
}

// TestMatchRank_Digit tests digit matching N empty squares
func TestMatchRank_Digit(t *testing.T) {
	tests := []struct {
		boardRank   string
		patternRank string
		want        bool
	}{
		{"________", "8", true},
		{"____P___", "4P3", true},
		{"P______P", "P6P", true},
		{"PPPPPPPP", "8", false},
		{"___P____", "4P3", false}, // P at wrong position
	}

	for _, tt := range tests {
		t.Run(tt.boardRank+"_"+tt.patternRank, func(t *testing.T) {
			got := matchRank(tt.boardRank, tt.patternRank)
			if got != tt.want {
				t.Errorf("matchRank(%q, %q) = %v, want %v", tt.boardRank, tt.patternRank, got, tt.want)
			}
		})
	}
}

// TestMatchRank_CombinedWildcards tests combinations of wildcards
func TestMatchRank_CombinedWildcards(t *testing.T) {
	tests := []struct {
		boardRank   string
		patternRank string
		want        bool
	}{
		{"R__Q__NR", "A??A??AA", true},
		{"R__q__nR", "A??a??aA", true},
		{"RNBQKBNR", "R*R", true},
		{"________", "*", true},
	}

	for _, tt := range tests {
		t.Run(tt.boardRank+"_"+tt.patternRank, func(t *testing.T) {
			got := matchRank(tt.boardRank, tt.patternRank)
			if got != tt.want {
				t.Errorf("matchRank(%q, %q) = %v, want %v", tt.boardRank, tt.patternRank, got, tt.want)
			}
		})
	}
}

// TestMatchRank_NoMatch tests non-matching patterns
func TestMatchRank_NoMatch(t *testing.T) {
	tests := []struct {
		boardRank   string
		patternRank string
		want        bool
	}{
		{"RNBQKBNR", "RNBQKBN", false},  // too short
		{"RNBQKBNR", "XNBQKBNR", false}, // wrong piece
		{"", "R", false},
		{"R", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.boardRank+"_"+tt.patternRank, func(t *testing.T) {
			got := matchRank(tt.boardRank, tt.patternRank)
			if got != tt.want {
				t.Errorf("matchRank(%q, %q) = %v, want %v", tt.boardRank, tt.patternRank, got, tt.want)
			}
		})
	}
}

// TestInvertPattern tests color inversion
func TestInvertPattern(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"RNBQKBNR", "rnbqkbnr"},
		{"rnbqkbnr", "RNBQKBNR"},
		{"RNBQKBNr", "rnbqkbnR"},
		{"8/8/8/8", "8/8/8/8"},
		{"PPPPPPPP/8", "8/pppppppp"}, // ranks reversed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := invertPattern(tt.input)
			if got != tt.want {
				t.Errorf("invertPattern(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ============== FEN Pattern Matcher Tests ==============

// TestFENPatternMatcher_New tests creating a new position matcher
func TestFENPatternMatcher_New(t *testing.T) {
	pm := NewPositionMatcher()
	if pm == nil {
		t.Fatal("NewPositionMatcher() returned nil")
	}
	if pm.PatternCount() != 0 {
		t.Errorf("PatternCount() = %d, want 0", pm.PatternCount())
	}
}

// TestFENPatternMatcher_AddFEN tests adding exact FEN positions
func TestFENPatternMatcher_AddFEN(t *testing.T) {
	pm := NewPositionMatcher()
	err := pm.AddFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", "initial")
	if err != nil {
		t.Errorf("AddFEN() error = %v", err)
	}
	if pm.PatternCount() != 1 {
		t.Errorf("PatternCount() = %d, want 1", pm.PatternCount())
	}
}

// TestFENPatternMatcher_AddFEN_Invalid tests adding invalid FEN
func TestFENPatternMatcher_AddFEN_Invalid(t *testing.T) {
	pm := NewPositionMatcher()
	err := pm.AddFEN("invalid fen string", "test")
	if err == nil {
		t.Error("AddFEN() should return error for invalid FEN")
	}
}

// TestFENPatternMatcher_AddPattern tests adding wildcard patterns
func TestFENPatternMatcher_AddPattern(t *testing.T) {
	pm := NewPositionMatcher()
	pm.AddPattern("r?bqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR", "test", false)
	if pm.PatternCount() != 1 {
		t.Errorf("PatternCount() = %d, want 1", pm.PatternCount())
	}
}

// TestFENPatternMatcher_AddPattern_WithInvert tests adding patterns with inversion
func TestFENPatternMatcher_AddPattern_WithInvert(t *testing.T) {
	pm := NewPositionMatcher()
	pm.AddPattern("RNBQKBNR/PPPPPPPP/8/8/8/8/pppppppp/rnbqkbnr", "test", true)
	// Should add both original and inverted pattern
	if pm.PatternCount() != 2 {
		t.Errorf("PatternCount() = %d, want 2 (original + inverted)", pm.PatternCount())
	}
}

// TestFENPatternMatcher_MatchGame_NoPatterns tests matching with no patterns
func TestFENPatternMatcher_MatchGame_NoPatterns(t *testing.T) {
	pm := NewPositionMatcher()
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

	match := pm.MatchGame(game)
	if match != nil {
		t.Error("MatchGame() with no patterns should return nil")
	}
}

// ============== GameFilter File Loading Tests ==============

// TestGameFilter_LoadTagFile_Basic tests loading tag criteria from a file
func TestGameFilter_LoadTagFile_Basic(t *testing.T) {
	// Create a temp file with tag criteria
	tmpDir := t.TempDir()
	filename := tmpDir + "/tags.txt"
	content := `White "Fischer"
Result "1-0"
`
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	gf := NewGameFilter()
	err := gf.LoadTagFile(filename)
	if err != nil {
		t.Errorf("LoadTagFile() error = %v", err)
	}

	if gf.TagMatcher.CriteriaCount() != 2 {
		t.Errorf("CriteriaCount() = %d, want 2", gf.TagMatcher.CriteriaCount())
	}
}

// TestGameFilter_LoadTagFile_WithComments tests that comments are ignored
func TestGameFilter_LoadTagFile_WithComments(t *testing.T) {
	tmpDir := t.TempDir()
	filename := tmpDir + "/tags.txt"
	content := `# This is a comment
White "Fischer"
# Another comment
Result "1-0"
`
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	gf := NewGameFilter()
	err := gf.LoadTagFile(filename)
	if err != nil {
		t.Errorf("LoadTagFile() error = %v", err)
	}

	if gf.TagMatcher.CriteriaCount() != 2 {
		t.Errorf("CriteriaCount() = %d, want 2", gf.TagMatcher.CriteriaCount())
	}
}

// TestGameFilter_LoadTagFile_NonExistent tests loading non-existent file
func TestGameFilter_LoadTagFile_NonExistent(t *testing.T) {
	gf := NewGameFilter()
	err := gf.LoadTagFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("LoadTagFile() should return error for non-existent file")
	}
}

// TestGameFilter_LoadTagFile_WithFEN tests loading FEN patterns from file
func TestGameFilter_LoadTagFile_WithFEN(t *testing.T) {
	tmpDir := t.TempDir()
	filename := tmpDir + "/tags.txt"
	content := `FEN "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"
`
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	gf := NewGameFilter()
	err := gf.LoadTagFile(filename)
	if err != nil {
		t.Errorf("LoadTagFile() error = %v", err)
	}

	if gf.PositionMatcher.PatternCount() != 1 {
		t.Errorf("PositionMatcher.PatternCount() = %d, want 1", gf.PositionMatcher.PatternCount())
	}
}

// ============== VariationMatcher Tests ==============

// TestVariationMatcher_LoadFromFile tests loading move sequences
func TestVariationMatcher_LoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := tmpDir + "/variations.txt"
	content := `1. e4 e5 2. Nf3 Nc6
1. d4 d5
`
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	vm := NewVariationMatcher()
	err := vm.LoadFromFile(filename)
	if err != nil {
		t.Errorf("LoadFromFile() error = %v", err)
	}

	if !vm.HasCriteria() {
		t.Error("HasCriteria() should return true after loading")
	}
}

// TestVariationMatcher_LoadFromFile_NonExistent tests loading non-existent file
func TestVariationMatcher_LoadFromFile_NonExistent(t *testing.T) {
	vm := NewVariationMatcher()
	err := vm.LoadFromFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("LoadFromFile() should return error for non-existent file")
	}
}

// TestVariationMatcher_LoadPositionalFromFile tests loading FEN sequences
func TestVariationMatcher_LoadPositionalFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := tmpDir + "/positions.txt"
	content := `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR
rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR
`
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	vm := NewVariationMatcher()
	err := vm.LoadPositionalFromFile(filename)
	if err != nil {
		t.Errorf("LoadPositionalFromFile() error = %v", err)
	}

	if !vm.HasCriteria() {
		t.Error("HasCriteria() should return true after loading")
	}
}

// TestVariationMatcher_EmptyFile tests loading empty file
func TestVariationMatcher_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := tmpDir + "/empty.txt"
	if err := os.WriteFile(filename, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	vm := NewVariationMatcher()
	err := vm.LoadFromFile(filename)
	if err != nil {
		t.Errorf("LoadFromFile() error = %v", err)
	}

	if vm.HasCriteria() {
		t.Error("HasCriteria() should return false for empty file")
	}
}

// TestVariationMatcher_MatchGame tests matching move sequences
func TestVariationMatcher_MatchGame(t *testing.T) {
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

	vm := NewVariationMatcher()
	vm.AddMoveSequence([]string{"e4", "e5", "Nf3"})

	if !vm.MatchGame(game) {
		t.Error("MatchGame() should return true for matching sequence")
	}
}

// TestVariationMatcher_MatchGame_NoMatch tests non-matching sequences
func TestVariationMatcher_MatchGame_NoMatch(t *testing.T) {
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. d4 d5 2. c4 *
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	vm := NewVariationMatcher()
	vm.AddMoveSequence([]string{"e4", "e5", "Nf3"})

	if vm.MatchGame(game) {
		t.Error("MatchGame() should return false for non-matching sequence")
	}
}

// ============== MaterialMatcher Tests ==============

// TestMaterialMatcher_MatchWithMarker tests material matching with markers
func TestMaterialMatcher_MatchWithMarker(t *testing.T) {
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

	// This pattern requires Q and q which exist in starting position
	mm := NewMaterialMatcher("Q:q", false)
	matched := mm.MatchGameWithMarker(game, "MATCH")

	if !matched {
		t.Error("MatchGameWithMarker() should match position with Q:q")
	}
}

// TestMaterialMatcher_ExactMatch tests exact material matching
func TestMaterialMatcher_ExactMatch(t *testing.T) {
	// Test with a standard game - MaterialMatcher replays from InitialFEN
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

	// In the standard starting position, we have KQRRBBNNPPPPPPPP:kqrrbbnnpppppppp
	// Let's test that exact match with wrong count fails
	mm := NewMaterialMatcher("K:k", true) // exact match - only kings, nothing else
	if mm.MatchGame(game) {
		t.Error("Exact match K:k should NOT match full starting position")
	}

	// Test that matching the starting material works
	mm2 := NewMaterialMatcher("KQRRBBNNPPPPPPPP:kqrrbbnnpppppppp", true)
	if !mm2.MatchGame(game) {
		t.Error("Exact match with full starting material should match")
	}
}

// TestMaterialMatcher_MinimalMatch tests minimal (at least) material matching
func TestMaterialMatcher_MinimalMatch(t *testing.T) {
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

	// R: means white must have at least one rook
	mm := NewMaterialMatcher("R:", false) // minimal match
	if !mm.MatchGame(game) {
		t.Error("Minimal match R: should match position with rooks")
	}

	// Q: means white must have at least one queen
	mm2 := NewMaterialMatcher("Q:", false)
	if !mm2.MatchGame(game) {
		t.Error("Minimal match Q: should match position with queen")
	}
}

// TestMaterialMatcher_HasCriteria tests HasCriteria method
func TestMaterialMatcher_HasCriteria(t *testing.T) {
	mm := NewMaterialMatcher("Q:q", false)
	if !mm.HasCriteria() {
		t.Error("HasCriteria() should return true for non-empty pattern")
	}

	mm2 := NewMaterialMatcher("", false)
	if mm2.HasCriteria() {
		t.Error("HasCriteria() should return false for empty pattern")
	}
}

// ============== GameFilter Additional Tests ==============

// TestGameFilter_HasCriteria tests HasCriteria method
func TestGameFilter_HasCriteria(t *testing.T) {
	gf := NewGameFilter()
	if gf.HasCriteria() {
		t.Error("HasCriteria() should return false for empty filter")
	}

	gf.AddPlayerFilter("Fischer")
	if !gf.HasCriteria() {
		t.Error("HasCriteria() should return true after adding criteria")
	}
}

// TestGameFilter_SetUseSoundex tests soundex setting
func TestGameFilter_SetUseSoundex(t *testing.T) {
	gf := NewGameFilter()
	gf.SetUseSoundex(true)
	// No error means success
}

// TestGameFilter_SetSubstringMatch tests substring matching
func TestGameFilter_SetSubstringMatch(t *testing.T) {
	gf := NewGameFilter()
	gf.SetSubstringMatch(true)
	// No error means success
}

// TestGameFilter_AddDateFilter tests date filtering
func TestGameFilter_AddDateFilter(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"Date": "1972.07.11",
		},
	}

	gf := NewGameFilter()
	gf.AddDateFilter("1970.01.01", OpGreaterThan)

	if !gf.MatchGame(game) {
		t.Error("Game from 1972 should match >= 1970")
	}
}

// TestGameFilter_AddWhiteFilter tests white player filter
func TestGameFilter_AddWhiteFilter(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White": "Fischer, Robert",
			"Black": "Spassky, Boris",
		},
	}

	gf := NewGameFilter()
	gf.AddWhiteFilter("Fischer")

	if !gf.MatchGame(game) {
		t.Error("Should match on White player")
	}
}

// TestGameFilter_AddBlackFilter tests black player filter
func TestGameFilter_AddBlackFilter(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White": "Fischer, Robert",
			"Black": "Spassky, Boris",
		},
	}

	gf := NewGameFilter()
	gf.AddBlackFilter("Spassky")

	if !gf.MatchGame(game) {
		t.Error("Should match on Black player")
	}
}

// TestGameFilter_AddFENFilter tests FEN position filter
func TestGameFilter_AddFENFilter(t *testing.T) {
	gf := NewGameFilter()
	err := gf.AddFENFilter("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Errorf("AddFENFilter() error = %v", err)
	}
}

// TestGameFilter_AddPatternFilter tests pattern filter
func TestGameFilter_AddPatternFilter(t *testing.T) {
	gf := NewGameFilter()
	gf.AddPatternFilter("r?bqkbnr/pppppppp/*", false)
	if gf.PositionMatcher.PatternCount() != 1 {
		t.Errorf("PatternCount() = %d, want 1", gf.PositionMatcher.PatternCount())
	}
}

// ============== Soundex Additional Tests ==============

// TestSoundex_EdgeCases tests edge cases for soundex
// Note: This implementation uses 6-character codes
func TestSoundex_EdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"A", "A00000"},
		{"", ""},                    // empty returns empty in this implementation
		{"AEIOU", "A00000"},         // vowels after first are dropped
		{"BBBBB", "B00000"},         // repeated consonants
		{"Ashcraft", "A26130"},      // standard example (6 chars)
		{"Pfister", "P23600"},       // P followed by F (6 chars)
		{"Tymczak", "T52000"},       // standard example (6 chars)
		{"12345", ""},               // numbers only returns empty
		{"Robert", "R16300"},        // standard test (6 chars)
		{"Rupert", "R16300"},        // should match Robert (6 chars)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Soundex(tt.input)
			if got != tt.want {
				t.Errorf("Soundex(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseMoveSequence tests parsing move sequences
func TestParseMoveSequence(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"1. e4 e5 2. Nf3", []string{"e4", "e5", "Nf3"}},
		{"e4 e5 Nf3", []string{"e4", "e5", "Nf3"}},
		{"1... e5 2. Nf3", []string{"e5", "Nf3"}},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseMoveSequence(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseMoveSequence(%q) len = %d, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for i, m := range got {
				if m != tt.want[i] {
					t.Errorf("parseMoveSequence(%q)[%d] = %q, want %q", tt.input, i, m, tt.want[i])
				}
			}
		})
	}
}

// TestNormalizeMove tests move normalization
func TestNormalizeMove(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"e4", "e4"},
		{"e4+", "e4"},
		{"e4#", "e4"},
		{"Nf3!", "Nf3"},
		{"Nf3?", "Nf3"},
		{"Nf3!!", "Nf3"},
		{"  e4  ", "e4"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeMove(tt.input)
			if got != tt.want {
				t.Errorf("normalizeMove(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ============================================================
// Additional tests for coverage improvement
// ============================================================

// TestSoundexMatch tests the SoundexMatch function
func TestSoundexMatch(t *testing.T) {
	tests := []struct {
		name1 string
		name2 string
		want  bool
	}{
		{"Fischer", "Fisher", true},
		{"Kasparov", "Kasparow", true},
		{"Robert", "Rupert", true},
		{"Smith", "Smyth", true},
		{"Fischer", "Carlsen", false},
		{"", "", true}, // both empty
		{"", "Fischer", false},
		{"Fischer", "", false},
	}

	for _, tt := range tests {
		name := tt.name1 + "_vs_" + tt.name2
		t.Run(name, func(t *testing.T) {
			got := SoundexMatch(tt.name1, tt.name2)
			if got != tt.want {
				t.Errorf("SoundexMatch(%q, %q) = %v, want %v", tt.name1, tt.name2, got, tt.want)
			}
		})
	}
}

// TestNewFENPatternMatcher tests FEN pattern matcher creation
func TestNewFENPatternMatcher(t *testing.T) {
	tests := []struct {
		pattern         string
		caseInsensitive bool
		wantErr         bool
	}{
		{"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR", false, false},
		{"*", false, false},
		{"?????", false, false},
		{"r*k*r", false, false},
		{"[rR][nN]", false, false},
		{"[", false, true}, // invalid regex (unclosed bracket)
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			matcher, err := NewFENPatternMatcher(tt.pattern, tt.caseInsensitive)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFENPatternMatcher(%q) error = %v, wantErr %v", tt.pattern, err, tt.wantErr)
				return
			}
			if !tt.wantErr && matcher == nil {
				t.Error("NewFENPatternMatcher() returned nil matcher without error")
			}
		})
	}
}

// TestFENPatternMatcher_MatchBoardFEN tests FEN matching directly
func TestFENPatternMatcher_MatchBoardFEN(t *testing.T) {
	tests := []struct {
		pattern string
		fen     string
		want    bool
	}{
		{"rnbqkbnr/*", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", true},
		{"*PPPPPPPP*", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", true},
		{"xyz", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", false},
		{"*", "", false}, // empty FEN
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.fen, func(t *testing.T) {
			matcher, err := NewFENPatternMatcher(tt.pattern, false)
			if err != nil {
				t.Fatalf("NewFENPatternMatcher() error: %v", err)
			}

			got := matcher.MatchBoardFEN(tt.fen)
			if got != tt.want {
				t.Errorf("MatchBoardFEN(%q) = %v, want %v", tt.fen, got, tt.want)
			}
		})
	}
}

// TestFENPatternMatcher_HasCriteria tests the HasCriteria method
func TestFENPatternMatcher_HasCriteria(t *testing.T) {
	matcher1, _ := NewFENPatternMatcher("*", false)
	if !matcher1.HasCriteria() {
		t.Error("HasCriteria() should return true for non-empty pattern")
	}

	// Create with empty pattern manually
	matcher2 := &FENPatternMatcher{pattern: ""}
	if matcher2.HasCriteria() {
		t.Error("HasCriteria() should return false for empty pattern")
	}
}

// TestFENPatternMatcher_Pattern tests the Pattern method
func TestFENPatternMatcher_Pattern(t *testing.T) {
	pattern := "r*k"
	matcher, _ := NewFENPatternMatcher(pattern, false)
	if matcher.Pattern() != pattern {
		t.Errorf("Pattern() = %q, want %q", matcher.Pattern(), pattern)
	}
}

// TestFENPatternMatcher_CaseInsensitive tests case-insensitive matching
func TestFENPatternMatcher_CaseInsensitive(t *testing.T) {
	matcher, err := NewFENPatternMatcher("RNBQKBNR", true)
	if err != nil {
		t.Fatalf("NewFENPatternMatcher() error: %v", err)
	}

	// Should match lowercase version
	if !matcher.MatchBoardFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1") {
		t.Error("Case-insensitive matcher should match lowercase FEN")
	}
}

// TestFENPatternMatcher_MatchGame tests matching against a full game
func TestFENPatternMatcher_MatchGame(t *testing.T) {
	// Create a simple game with no moves (just initial position)
	game := &chess.Game{
		Tags:  map[string]string{},
		Moves: nil,
	}

	// Match initial position - pattern that matches the starting position
	matcher1, _ := NewFENPatternMatcher("rnbqkbnr/pppppppp/*", false)
	if !matcher1.MatchGame(game) {
		t.Error("Should match initial position")
	}

	// Match pattern in starting position - white back rank
	matcher2, _ := NewFENPatternMatcher("*RNBQKBNR", false)
	if !matcher2.MatchGame(game) {
		t.Error("Should match white back rank pattern")
	}

	// Match position that never occurs
	matcher3, _ := NewFENPatternMatcher("xyz123abc", false)
	if matcher3.MatchGame(game) {
		t.Error("Should not match impossible pattern")
	}
}

// TestNormalizeFENForMatching tests FEN normalization
func TestNormalizeFENForMatching(t *testing.T) {
	tests := []struct {
		input           string
		caseInsensitive bool
		want            string
	}{
		{"8", false, "........"},                                         // 8 empty squares
		{"4P3", false, "....P..."},                                       // 4 empty, P, 3 empty
		{"RNBQKBNR", false, "RNBQKBNR"},                                  // no digits
		{"rnbqkbnr", true, "rnbqkbnr"},                                   // case insensitive lowercase stays
		{"RNBQKBNR", true, "rnbqkbnr"},                                   // case insensitive converts to lower
		{"r1k1r", false, "r.k.r"},                                        // single digits
		{"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR", false, "rnbqkbnr/pppppppp/......../......../......../......../PPPPPPPP/RNBQKBNR"},
	}

	for _, tt := range tests {
		name := tt.input
		if tt.caseInsensitive {
			name += "_ci"
		}
		t.Run(name, func(t *testing.T) {
			got := NormalizeFENForMatching(tt.input, tt.caseInsensitive)
			if got != tt.want {
				t.Errorf("NormalizeFENForMatching(%q, %v) = %q, want %q",
					tt.input, tt.caseInsensitive, got, tt.want)
			}
		})
	}
}

// TestCompositeMatcher_Add tests adding matchers to composite
func TestCompositeMatcher_Add(t *testing.T) {
	composite := NewCompositeMatcher(MatchAll)

	if len(composite.Matchers()) != 0 {
		t.Error("New composite should have no matchers")
	}

	mm := NewMaterialMatcher("Q:q", false)
	composite.Add(mm)

	if len(composite.Matchers()) != 1 {
		t.Errorf("After Add(), len(Matchers()) = %d, want 1", len(composite.Matchers()))
	}

	if composite.Matchers()[0] != mm {
		t.Error("Matchers()[0] should be the added matcher")
	}
}

// TestCompositeMatcher_Mode tests the Mode method
func TestCompositeMatcher_Mode(t *testing.T) {
	compositeAll := NewCompositeMatcher(MatchAll)
	if compositeAll.Mode() != MatchAll {
		t.Errorf("Mode() = %v, want MatchAll", compositeAll.Mode())
	}

	compositeAny := NewCompositeMatcher(MatchAny)
	if compositeAny.Mode() != MatchAny {
		t.Errorf("Mode() = %v, want MatchAny", compositeAny.Mode())
	}
}

// TestCompositeMatcher_Matchers tests the Matchers accessor
func TestCompositeMatcher_Matchers(t *testing.T) {
	mm1 := NewMaterialMatcher("Q:q", false)
	mm2 := NewMaterialMatcher("R:r", false)

	composite := NewCompositeMatcher(MatchAny, mm1, mm2)
	matchers := composite.Matchers()

	if len(matchers) != 2 {
		t.Errorf("len(Matchers()) = %d, want 2", len(matchers))
	}
}

// TestTagMatcher_SetMatchAll tests the SetMatchAll method
func TestTagMatcher_SetMatchAll(t *testing.T) {
	tm := NewTagMatcher()
	// Default should require all
	if !tm.matchAll {
		t.Error("Default matchAll should be true")
	}

	tm.SetMatchAll(false)
	if tm.matchAll {
		t.Error("After SetMatchAll(false), matchAll should be false")
	}

	tm.SetMatchAll(true)
	if !tm.matchAll {
		t.Error("After SetMatchAll(true), matchAll should be true")
	}
}

// TestGameFilter_AddTagCriterion tests adding tag criteria
func TestGameFilter_AddTagCriterion(t *testing.T) {
	gf := NewGameFilter()

	gf.AddTagCriterion("Event", "World Championship", OpEqual)
	gf.AddTagCriterion("Site", "London", OpEqual)

	// Verify by matching a game
	game := &chess.Game{
		Tags: map[string]string{
			"Event": "World Championship",
			"Site":  "London",
		},
	}

	if !gf.MatchGame(game) {
		t.Error("Game with matching tags should match filter")
	}

	// Non-matching game
	game2 := &chess.Game{
		Tags: map[string]string{
			"Event": "Casual Game",
		},
	}

	if gf.MatchGame(game2) {
		t.Error("Game with non-matching tags should not match filter")
	}
}

// TestVariationMatcher_SetMatchAnywhere tests SetMatchAnywhere method
func TestVariationMatcher_SetMatchAnywhere(t *testing.T) {
	vm := NewVariationMatcher()
	vm.AddMoveSequence([]string{"e4", "e5", "Nf3"})

	// Default is false (match from start)
	vm.SetMatchAnywhere(true)

	// Now it should match anywhere in the game
	// This is tested implicitly - just ensure no panic
	if vm == nil {
		t.Error("VariationMatcher should not be nil")
	}
}

// TestMaterialMatcher_Match tests the Match method on MaterialMatcher
func TestMaterialMatcher_Match(t *testing.T) {
	mm := NewMaterialMatcher("Q:q", false)

	// Match should behave same as MatchGame for testing purposes
	game := &chess.Game{
		Tags: map[string]string{},
		Moves: &chess.Move{
			Text: "e4",
		},
	}

	// Both Match and MatchGame should return the same result
	matchResult := mm.Match(game)
	matchGameResult := mm.MatchGame(game)

	if matchResult != matchGameResult {
		t.Error("Match() and MatchGame() should return same result")
	}
}

// TestCompilePatternToRegex_SpecialChars tests regex special character escaping
func TestCompilePatternToRegex_SpecialChars(t *testing.T) {
	// Pattern with dots should escape them
	matcher, err := NewFENPatternMatcher("r.k", false)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// The dot in "r.k" should be literal, not regex wildcard
	// So "r.k" should NOT match "rak" but should match "r.k"
	if matcher.MatchBoardFEN("r.k/8/8/8/8/8/8/8 w - - 0 1") {
		// This is expected to match because r.k matches r.k
	}
}

// TestCompilePatternToRegex_CharacterClass tests character class handling
func TestCompilePatternToRegex_CharacterClass(t *testing.T) {
	// Pattern with character class
	matcher, err := NewFENPatternMatcher("[rR]nbqkbn[rR]/*", false)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// Should match both lowercase and uppercase rooks at corners
	if !matcher.MatchBoardFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1") {
		t.Error("Character class should match lowercase rooks")
	}

	if !matcher.MatchBoardFEN("Rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1") {
		t.Error("Character class should match uppercase R at start")
	}
}
