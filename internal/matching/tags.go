// Package matching provides game filtering by tags and positions.
package matching

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// TagOperator represents comparison operators for tag matching.
type TagOperator int

const (
	OpNone TagOperator = iota
	OpEqual
	OpNotEqual
	OpLessThan
	OpLessOrEqual
	OpGreaterThan
	OpGreaterOrEqual
	OpContains     // substring match
	OpRegex        // regex match
	OpSoundex      // soundex match for names
)

// TagCriterion represents a single tag matching criterion.
type TagCriterion struct {
	TagName    string
	Value      string
	Operator   TagOperator
	Regex      *regexp.Regexp // compiled regex for OpRegex
	Soundex    string         // soundex value for OpSoundex
	LowerValue string         // pre-computed lowercase for OpContains
}

// TagMatcher provides tag-based game filtering.
type TagMatcher struct {
	criteria       []*TagCriterion
	useSoundex     bool
	substringMatch bool
	matchAll       bool // true = AND all criteria, false = OR
}

// NewTagMatcher creates a new tag matcher.
func NewTagMatcher() *TagMatcher {
	return &TagMatcher{
		matchAll: true, // default: all criteria must match
	}
}

// SetMatchAll sets whether all criteria must match (AND) or any (OR).
func (tm *TagMatcher) SetMatchAll(all bool) {
	tm.matchAll = all
}

// SetUseSoundex enables soundex matching for player names.
func (tm *TagMatcher) SetUseSoundex(use bool) {
	tm.useSoundex = use
}

// SetSubstringMatch enables substring matching for all tag values.
func (tm *TagMatcher) SetSubstringMatch(use bool) {
	tm.substringMatch = use
}

// AddCriterion adds a tag matching criterion.
func (tm *TagMatcher) AddCriterion(tagName, value string, op TagOperator) error {
	c := &TagCriterion{
		TagName:  tagName,
		Value:    value,
		Operator: op,
	}

	// Compile regex if needed
	if op == OpRegex {
		re, err := regexp.Compile(value)
		if err != nil {
			return err
		}
		c.Regex = re
	}

	// Calculate soundex if needed
	if op == OpSoundex {
		c.Soundex = Soundex(value)
	}

	// Pre-compute lowercase for contains matching
	if op == OpContains {
		c.LowerValue = strings.ToLower(value)
	}

	tm.criteria = append(tm.criteria, c)
	return nil
}

// AddSimpleCriterion adds a simple equality criterion.
func (tm *TagMatcher) AddSimpleCriterion(tagName, value string) {
	tm.AddCriterion(tagName, value, OpEqual)
}

// AddPlayerCriterion adds a criterion that matches either White or Black.
func (tm *TagMatcher) AddPlayerCriterion(playerName string) {
	// This is handled specially in MatchGame
	op := OpContains
	if tm.useSoundex {
		op = OpSoundex
	}
	tm.AddCriterion("_Player", playerName, op)
}

// ParseCriterion parses a criterion string like "White < \"Fischer\"".
func (tm *TagMatcher) ParseCriterion(line string) error {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil // empty or comment
	}

	// Find tag name
	tagEnd := strings.IndexAny(line, " \t<>=!")
	if tagEnd == -1 {
		return nil
	}

	tagName := strings.TrimSpace(line[:tagEnd])
	rest := strings.TrimSpace(line[tagEnd:])

	// Parse operator
	op := OpEqual
	valueStart := 0

	if strings.HasPrefix(rest, "<=") {
		op = OpLessOrEqual
		valueStart = 2
	} else if strings.HasPrefix(rest, ">=") {
		op = OpGreaterOrEqual
		valueStart = 2
	} else if strings.HasPrefix(rest, "<>") || strings.HasPrefix(rest, "!=") {
		op = OpNotEqual
		valueStart = 2
	} else if strings.HasPrefix(rest, "<") {
		op = OpLessThan
		valueStart = 1
	} else if strings.HasPrefix(rest, ">") {
		op = OpGreaterThan
		valueStart = 1
	} else if strings.HasPrefix(rest, "=") {
		op = OpEqual
		valueStart = 1
	} else if strings.HasPrefix(rest, "~") {
		op = OpRegex
		valueStart = 1
	}

	value := strings.TrimSpace(rest[valueStart:])

	// Remove quotes if present
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}

	return tm.AddCriterion(tagName, value, op)
}

// MatchGame checks if a game matches the criteria.
func (tm *TagMatcher) MatchGame(game *chess.Game) bool {
	if len(tm.criteria) == 0 {
		return true // no criteria = match all
	}

	for _, c := range tm.criteria {
		matches := tm.matchCriterion(game, c)

		if tm.matchAll {
			if !matches {
				return false // AND: any failure = no match
			}
		} else {
			if matches {
				return true // OR: any success = match
			}
		}
	}

	return tm.matchAll // AND: all passed, OR: none passed
}

// matchCriterion checks if a game matches a single criterion.
func (tm *TagMatcher) matchCriterion(game *chess.Game, c *TagCriterion) bool {
	// Special case: _Player matches either White or Black
	if c.TagName == "_Player" {
		white := game.Tags["White"]
		black := game.Tags["Black"]
		return tm.matchValue(white, c) || tm.matchValue(black, c)
	}

	tagValue, ok := game.Tags[c.TagName]
	if !ok {
		// Tag doesn't exist
		return c.Operator == OpNotEqual // only != matches missing tags
	}

	return tm.matchValue(tagValue, c)
}

// matchValue compares a tag value against a criterion.
func (tm *TagMatcher) matchValue(tagValue string, c *TagCriterion) bool {
	switch c.Operator {
	case OpNone, OpEqual:
		return strings.EqualFold(tagValue, c.Value)

	case OpNotEqual:
		return !strings.EqualFold(tagValue, c.Value)

	case OpContains:
		return strings.Contains(strings.ToLower(tagValue), c.LowerValue)

	case OpRegex:
		if c.Regex == nil {
			return false
		}
		return c.Regex.MatchString(tagValue)

	case OpSoundex:
		return Soundex(tagValue) == c.Soundex

	case OpLessThan, OpLessOrEqual, OpGreaterThan, OpGreaterOrEqual:
		return tm.compareValues(tagValue, c.Value, c.Operator)
	}

	return false
}

// compareValues compares values using relational operators.
// Handles dates (YYYY.MM.DD) and numeric values.
func (tm *TagMatcher) compareValues(tagValue, criterionValue string, op TagOperator) bool {
	// Try date comparison first (YYYY.MM.DD format)
	tagDate := parseDate(tagValue)
	criterionDate := parseDate(criterionValue)

	if tagDate > 0 && criterionDate > 0 {
		switch op {
		case OpLessThan:
			return tagDate < criterionDate
		case OpLessOrEqual:
			return tagDate <= criterionDate
		case OpGreaterThan:
			return tagDate > criterionDate
		case OpGreaterOrEqual:
			return tagDate >= criterionDate
		}
	}

	// Try numeric comparison
	tagNum, err1 := strconv.ParseFloat(tagValue, 64)
	criterionNum, err2 := strconv.ParseFloat(criterionValue, 64)

	if err1 == nil && err2 == nil {
		switch op {
		case OpLessThan:
			return tagNum < criterionNum
		case OpLessOrEqual:
			return tagNum <= criterionNum
		case OpGreaterThan:
			return tagNum > criterionNum
		case OpGreaterOrEqual:
			return tagNum >= criterionNum
		}
	}

	// Fall back to string comparison
	cmp := strings.Compare(strings.ToLower(tagValue), strings.ToLower(criterionValue))
	switch op {
	case OpLessThan:
		return cmp < 0
	case OpLessOrEqual:
		return cmp <= 0
	case OpGreaterThan:
		return cmp > 0
	case OpGreaterOrEqual:
		return cmp >= 0
	}

	return false
}

// parseDate parses a date in YYYY.MM.DD format and returns encoded value.
// Returns 0 if parsing fails.
func parseDate(s string) int {
	parts := strings.Split(s, ".")
	if len(parts) < 1 {
		return 0
	}

	year, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || year < 100 || year > 3000 {
		return 0
	}

	month := 1
	day := 1

	if len(parts) >= 2 {
		m, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	if len(parts) >= 3 {
		d, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err == nil && d >= 1 && d <= 31 {
			day = d
		}
	}

	return year*10000 + month*100 + day
}

// CriteriaCount returns the number of criteria.
func (tm *TagMatcher) CriteriaCount() int {
	return len(tm.criteria)
}
