package matching

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

func TestTagMatcher_OpRegex(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		tagValue string
		expected bool
	}{
		{"exact match", "^Fischer$", "Fischer", true},
		{"no match", "^Fischer$", "Kasparov", false},
		{"partial match", "Fisch", "Fischer, Robert", true},
		{"wildcard", "K.*ov", "Kasparov", true},
		{"case sensitive match", "fischer", "Fischer", false},
		{"case insensitive flag", "(?i)fischer", "Fischer", true},
		{"alternation", "Fischer|Kasparov", "Kasparov", true},
		{"digit pattern", `\d{4}`, "Game1234", true},
		{"no digit match", `^\d+$`, "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTagMatcher()
			if err := tm.AddCriterion("White", tt.pattern, OpRegex); err != nil {
				t.Fatalf("AddCriterion failed: %v", err)
			}

			game := &chess.Game{
				Tags: map[string]string{"White": tt.tagValue},
			}

			if tm.MatchGame(game) != tt.expected {
				t.Errorf("Regex %q vs %q: got %v, want %v", tt.pattern, tt.tagValue, !tt.expected, tt.expected)
			}
		})
	}
}

func TestTagMatcher_OpRegex_CompilationError(t *testing.T) {
	tm := NewTagMatcher()
	err := tm.AddCriterion("White", "[invalid", OpRegex)
	if err == nil {
		t.Error("Expected error for invalid regex pattern")
	}
}

func TestTagMatcher_OpRegex_NilRegex(t *testing.T) {
	// Construct a criterion with OpRegex but nil Regex to test the guard
	tm := NewTagMatcher()
	tm.criteria = append(tm.criteria, &TagCriterion{
		TagName:  "White",
		Value:    "test",
		Operator: OpRegex,
		Regex:    nil,
	})

	game := &chess.Game{
		Tags: map[string]string{"White": "test"},
	}

	if tm.MatchGame(game) {
		t.Error("Should not match when Regex is nil")
	}
}

func TestTagMatcher_OpSoundex(t *testing.T) {
	tests := []struct {
		name     string
		criterion string
		tagValue  string
		expected  bool
	}{
		{"exact phonetic", "Fischer", "Fischer", true},
		{"similar phonetic", "Fischer", "Fisher", true},
		{"different phonetic", "Fischer", "Kasparov", false},
		{"Carlsen vs Carlson", "Carlsen", "Carlson", true},
		{"Smith vs Smyth", "Smith", "Smyth", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTagMatcher()
			tm.AddCriterion("White", tt.criterion, OpSoundex)

			game := &chess.Game{
				Tags: map[string]string{"White": tt.tagValue},
			}

			if tm.MatchGame(game) != tt.expected {
				t.Errorf("Soundex %q vs %q: got %v, want %v", tt.criterion, tt.tagValue, !tt.expected, tt.expected)
			}
		})
	}
}

func TestTagMatcher_SetUseSoundex_PlayerCriterion(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White": "Fischer",
			"Black": "Spassky",
		},
	}

	// Without soundex, "Fisher" should match via contains
	tm1 := NewTagMatcher()
	tm1.AddPlayerCriterion("Fisher")
	if tm1.MatchGame(game) {
		t.Error("Without soundex, 'Fisher' should NOT substring-match 'Fischer'")
	}

	// With soundex enabled before adding criterion
	tm2 := NewTagMatcher()
	tm2.SetUseSoundex(true)
	tm2.AddPlayerCriterion("Fisher")
	if !tm2.MatchGame(game) {
		t.Error("With soundex, 'Fisher' should match 'Fischer'")
	}

	// Verify that without soundex, exact substring works
	tm3 := NewTagMatcher()
	tm3.AddPlayerCriterion("Fischer")
	if !tm3.MatchGame(game) {
		t.Error("Without soundex, exact contains should still match")
	}
}

func TestTagMatcher_SetMatchAll_ORMode(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White":  "Fischer, Robert",
			"Black":  "Spassky, Boris",
			"Result": "1-0",
		},
	}

	// OR mode: any criterion matching is enough
	tm := NewTagMatcher()
	tm.SetMatchAll(false)
	tm.AddCriterion("White", "Fischer, Robert", OpEqual)
	tm.AddCriterion("Result", "0-1", OpEqual) // does not match

	if !tm.MatchGame(game) {
		t.Error("OR mode: should match when at least one criterion matches")
	}

	// OR mode: none match
	tm2 := NewTagMatcher()
	tm2.SetMatchAll(false)
	tm2.AddCriterion("White", "Karpov", OpEqual)
	tm2.AddCriterion("Result", "0-1", OpEqual)

	if tm2.MatchGame(game) {
		t.Error("OR mode: should not match when no criteria match")
	}
}

func TestTagMatcher_SetMatchAll_ANDMode(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"White":  "Fischer, Robert",
			"Result": "1-0",
		},
	}

	// AND mode (default): all must match
	tm := NewTagMatcher()
	tm.AddCriterion("White", "Fischer, Robert", OpEqual)
	tm.AddCriterion("Result", "1-0", OpEqual)

	if !tm.MatchGame(game) {
		t.Error("AND mode: should match when all criteria match")
	}

	// AND mode: one fails
	tm2 := NewTagMatcher()
	tm2.AddCriterion("White", "Fischer, Robert", OpEqual)
	tm2.AddCriterion("Result", "0-1", OpEqual)

	if tm2.MatchGame(game) {
		t.Error("AND mode: should not match when one criterion fails")
	}
}

func TestTagMatcher_OpNotEqual(t *testing.T) {
	tests := []struct {
		name     string
		tags     map[string]string
		tagName  string
		value    string
		expected bool
	}{
		{
			name:     "value differs",
			tags:     map[string]string{"Result": "1-0"},
			tagName:  "Result",
			value:    "0-1",
			expected: true,
		},
		{
			name:     "value same",
			tags:     map[string]string{"Result": "1-0"},
			tagName:  "Result",
			value:    "1-0",
			expected: false,
		},
		{
			name:     "tag missing",
			tags:     map[string]string{"White": "Test"},
			tagName:  "Result",
			value:    "1-0",
			expected: true, // missing tag treated as not equal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTagMatcher()
			tm.AddCriterion(tt.tagName, tt.value, OpNotEqual)

			game := &chess.Game{Tags: tt.tags}
			if tm.MatchGame(game) != tt.expected {
				t.Errorf("OpNotEqual %s != %s: got %v, want %v", tt.tagName, tt.value, !tt.expected, tt.expected)
			}
		})
	}
}

func TestTagMatcher_OpContains_CaseInsensitive(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{"White": "Fischer, Robert"},
	}

	tm := NewTagMatcher()
	tm.AddCriterion("White", "FISCHER", OpContains)

	if !tm.MatchGame(game) {
		t.Error("OpContains should be case-insensitive")
	}
}

func TestTagMatcher_SubstringMatch(t *testing.T) {
	// SetSubstringMatch sets a flag; verify it is stored
	tm := NewTagMatcher()
	tm.SetSubstringMatch(true)
	if !tm.substringMatch {
		t.Error("SetSubstringMatch(true) should set substringMatch to true")
	}
	tm.SetSubstringMatch(false)
	if tm.substringMatch {
		t.Error("SetSubstringMatch(false) should set substringMatch to false")
	}
}

func TestTagMatcher_NumericComparison(t *testing.T) {
	tests := []struct {
		name      string
		tagValue  string
		critValue string
		op        TagOperator
		expected  bool
	}{
		{"int less than", "100", "200", OpLessThan, true},
		{"int not less than", "200", "100", OpLessThan, false},
		{"int greater than", "200", "100", OpGreaterThan, true},
		{"int less or equal same", "100", "100", OpLessOrEqual, true},
		{"int greater or equal same", "100", "100", OpGreaterOrEqual, true},
		{"float less than", "1.5", "2.5", OpLessThan, true},
		{"float greater than", "3.14", "2.71", OpGreaterThan, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTagMatcher()
			tm.AddCriterion("Elo", tt.critValue, tt.op)

			game := &chess.Game{
				Tags: map[string]string{"Elo": tt.tagValue},
			}

			if tm.MatchGame(game) != tt.expected {
				t.Errorf("%s %v %s: got %v, want %v", tt.tagValue, tt.op, tt.critValue, !tt.expected, tt.expected)
			}
		})
	}
}

func TestTagMatcher_StringComparison(t *testing.T) {
	// When values are neither dates nor numbers, falls back to string comparison
	tests := []struct {
		name      string
		tagValue  string
		critValue string
		op        TagOperator
		expected  bool
	}{
		{"alpha less", "abc", "def", OpLessThan, true},
		{"alpha not less", "def", "abc", OpLessThan, false},
		{"alpha greater", "xyz", "abc", OpGreaterThan, true},
		{"alpha equal boundary", "abc", "abc", OpLessOrEqual, true},
		{"alpha equal boundary gte", "abc", "abc", OpGreaterOrEqual, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTagMatcher()
			tm.AddCriterion("Site", tt.critValue, tt.op)

			game := &chess.Game{
				Tags: map[string]string{"Site": tt.tagValue},
			}

			if tm.MatchGame(game) != tt.expected {
				t.Errorf("%s %v %s: got %v, want %v", tt.tagValue, tt.op, tt.critValue, !tt.expected, tt.expected)
			}
		})
	}
}

func TestTagMatcher_ParseCriterion_AllOperators(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantOp   TagOperator
		wantTag  string
		wantVal  string
	}{
		{"equal explicit", `Result = "1-0"`, OpEqual, "Result", "1-0"},
		{"less than", `Date < "2000.01.01"`, OpLessThan, "Date", "2000.01.01"},
		{"greater than", `Date > "1990.01.01"`, OpGreaterThan, "Date", "1990.01.01"},
		{"less or equal", `Date <= "2000.01.01"`, OpLessOrEqual, "Date", "2000.01.01"},
		{"greater or equal", `Date >= "1990.01.01"`, OpGreaterOrEqual, "Date", "1990.01.01"},
		{"not equal angle", `Result <> "1/2-1/2"`, OpNotEqual, "Result", "1/2-1/2"},
		{"not equal bang", `Result != "1/2-1/2"`, OpNotEqual, "Result", "1/2-1/2"},
		{"regex", `White ~ "^Fischer"`, OpRegex, "White", "^Fischer"},
		{"implicit equal", `White "Fischer"`, OpEqual, "White", "Fischer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTagMatcher()
			err := tm.ParseCriterion(tt.line)
			if err != nil {
				t.Fatalf("ParseCriterion(%q) error: %v", tt.line, err)
			}

			if tm.CriteriaCount() != 1 {
				t.Fatalf("Expected 1 criterion, got %d", tm.CriteriaCount())
			}

			c := tm.criteria[0]
			if c.TagName != tt.wantTag {
				t.Errorf("TagName: got %q, want %q", c.TagName, tt.wantTag)
			}
			if c.Value != tt.wantVal {
				t.Errorf("Value: got %q, want %q", c.Value, tt.wantVal)
			}
			if c.Operator != tt.wantOp {
				t.Errorf("Operator: got %v, want %v", c.Operator, tt.wantOp)
			}
		})
	}
}

func TestTagMatcher_ParseCriterion_SkipsEmptyAndComments(t *testing.T) {
	tm := NewTagMatcher()

	if err := tm.ParseCriterion(""); err != nil {
		t.Errorf("Empty line should not error: %v", err)
	}
	if err := tm.ParseCriterion("# comment"); err != nil {
		t.Errorf("Comment should not error: %v", err)
	}
	if err := tm.ParseCriterion("   "); err != nil {
		t.Errorf("Whitespace should not error: %v", err)
	}

	if tm.CriteriaCount() != 0 {
		t.Errorf("No criteria should be added for empty/comment lines, got %d", tm.CriteriaCount())
	}
}

func TestTagMatcher_ParseCriterion_NoOperator(t *testing.T) {
	tm := NewTagMatcher()
	// A line with no recognizable operator/value separator
	err := tm.ParseCriterion("JustATag")
	if err != nil {
		t.Errorf("Should not error, just skip: %v", err)
	}
	if tm.CriteriaCount() != 0 {
		t.Errorf("Should not add criterion for unparseable line, got %d", tm.CriteriaCount())
	}
}

func TestTagMatcher_ParseCriterion_RegexError(t *testing.T) {
	tm := NewTagMatcher()
	err := tm.ParseCriterion(`White ~ "[invalid"`)
	if err == nil {
		t.Error("Expected error for invalid regex in ParseCriterion")
	}
}

func TestTagMatcher_MatchGame_NoCriteria(t *testing.T) {
	tm := NewTagMatcher()
	game := &chess.Game{Tags: map[string]string{"White": "Test"}}

	if !tm.MatchGame(game) {
		t.Error("Matcher with no criteria should match all games")
	}
}

func TestTagMatcher_CriteriaCount(t *testing.T) {
	tm := NewTagMatcher()
	if tm.CriteriaCount() != 0 {
		t.Error("New matcher should have 0 criteria")
	}

	tm.AddCriterion("White", "Fischer", OpEqual)
	tm.AddCriterion("Result", "1-0", OpEqual)

	if tm.CriteriaCount() != 2 {
		t.Errorf("Expected 2 criteria, got %d", tm.CriteriaCount())
	}
}

func TestTagMatcher_AddSimpleCriterion(t *testing.T) {
	tm := NewTagMatcher()
	tm.AddSimpleCriterion("Event", "Candidates")

	game := &chess.Game{Tags: map[string]string{"Event": "Candidates"}}
	if !tm.MatchGame(game) {
		t.Error("AddSimpleCriterion should add OpEqual criterion")
	}

	game2 := &chess.Game{Tags: map[string]string{"Event": "candidates"}} // case differs
	if !tm.MatchGame(game2) {
		t.Error("OpEqual should be case-insensitive (EqualFold)")
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"1972.07.11", 19720711},
		{"2024.01.01", 20240101},
		{"1985.11", 19851101},   // missing day defaults to 1
		{"2000", 20000101},       // year only
		{"abc", 0},               // invalid
		{"50.01.01", 0},          // year < 100
		{"4000.01.01", 0},        // year > 3000
		{"1972.13.01", 19720101}, // invalid month defaults to 1
		{"1972.07.32", 19720701}, // invalid day defaults to 1
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDate(tt.input)
			if result != tt.expected {
				t.Errorf("parseDate(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTagMatcher_OpNone(t *testing.T) {
	// OpNone should behave like OpEqual
	tm := NewTagMatcher()
	tm.criteria = append(tm.criteria, &TagCriterion{
		TagName:  "White",
		Value:    "Fischer",
		Operator: OpNone,
	})

	game := &chess.Game{Tags: map[string]string{"White": "Fischer"}}
	if !tm.MatchGame(game) {
		t.Error("OpNone should match like OpEqual")
	}

	game2 := &chess.Game{Tags: map[string]string{"White": "Kasparov"}}
	if tm.MatchGame(game2) {
		t.Error("OpNone should not match different value")
	}
}

func TestTagMatcher_PlayerCriterion_MatchesBothColors(t *testing.T) {
	// Verify _Player special tag checks both White and Black
	tm := NewTagMatcher()
	tm.AddPlayerCriterion("Spassky")

	whiteGame := &chess.Game{
		Tags: map[string]string{"White": "Spassky, Boris", "Black": "Fischer"},
	}
	if !tm.MatchGame(whiteGame) {
		t.Error("Player criterion should match when player is White")
	}

	blackGame := &chess.Game{
		Tags: map[string]string{"White": "Fischer", "Black": "Spassky, Boris"},
	}
	if !tm.MatchGame(blackGame) {
		t.Error("Player criterion should match when player is Black")
	}

	neitherGame := &chess.Game{
		Tags: map[string]string{"White": "Karpov", "Black": "Kasparov"},
	}
	if tm.MatchGame(neitherGame) {
		t.Error("Player criterion should not match when player is in neither color")
	}
}

func TestTagMatcher_MissingTag_NonNotEqual(t *testing.T) {
	// For operators other than OpNotEqual, a missing tag should not match
	tm := NewTagMatcher()
	tm.AddCriterion("ECO", "B90", OpEqual)

	game := &chess.Game{Tags: map[string]string{"White": "Test"}} // no ECO tag
	if tm.MatchGame(game) {
		t.Error("OpEqual should not match when tag is missing")
	}
}

func TestTagMatcher_DateComparison_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		tagDate   string
		critDate  string
		op        TagOperator
		expected  bool
	}{
		{"same date lte", "2000.06.15", "2000.06.15", OpLessOrEqual, true},
		{"same date gte", "2000.06.15", "2000.06.15", OpGreaterOrEqual, true},
		{"same date lt", "2000.06.15", "2000.06.15", OpLessThan, false},
		{"same date gt", "2000.06.15", "2000.06.15", OpGreaterThan, false},
		{"year only comparison", "2000", "1999", OpGreaterThan, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTagMatcher()
			tm.AddCriterion("Date", tt.critDate, tt.op)

			game := &chess.Game{Tags: map[string]string{"Date": tt.tagDate}}
			if tm.MatchGame(game) != tt.expected {
				t.Errorf("Date %s %v %s: got %v, want %v", tt.tagDate, tt.op, tt.critDate, !tt.expected, tt.expected)
			}
		})
	}
}
