package matching

import (
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

