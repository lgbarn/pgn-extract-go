package matching

import (
	"bufio"
	"os"
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// GameFilter combines tag and position matching.
type GameFilter struct {
	TagMatcher      *TagMatcher
	PositionMatcher *PositionMatcher
	RequireBoth     bool // true = both tag AND position must match
}

// NewGameFilter creates a new game filter.
func NewGameFilter() *GameFilter {
	return &GameFilter{
		TagMatcher:      NewTagMatcher(),
		PositionMatcher: NewPositionMatcher(),
		RequireBoth:     false, // default: either matches
	}
}

// LoadTagFile loads tag criteria from a file.
// File format: one criterion per line
// TagName "value"
// TagName < "value"
// TagName >= "value"
// etc.
func (gf *GameFilter) LoadTagFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for special patterns
		if strings.HasPrefix(line, "FEN ") || strings.HasPrefix(line, "FENPattern ") {
			// FEN or pattern for position matching
			rest := strings.TrimPrefix(line, "FEN ")
			rest = strings.TrimPrefix(rest, "FENPattern ")
			rest = strings.Trim(rest, "\"")

			// Check if it has wildcards
			if strings.ContainsAny(rest, "?!*Aa") {
				gf.PositionMatcher.AddPattern(rest, "", false)
			} else {
				gf.PositionMatcher.AddFEN(rest, "")
			}
		} else {
			gf.TagMatcher.ParseCriterion(line)
		}
	}

	return scanner.Err()
}

// AddTagCriterion adds a tag criterion directly.
func (gf *GameFilter) AddTagCriterion(tagName, value string, op TagOperator) {
	gf.TagMatcher.AddCriterion(tagName, value, op)
}

// AddPlayerFilter adds a filter for player name (matches White or Black).
func (gf *GameFilter) AddPlayerFilter(name string) {
	gf.TagMatcher.AddPlayerCriterion(name)
}

// AddWhiteFilter adds a filter for White player.
func (gf *GameFilter) AddWhiteFilter(name string) {
	gf.TagMatcher.AddCriterion("White", name, OpContains)
}

// AddBlackFilter adds a filter for Black player.
func (gf *GameFilter) AddBlackFilter(name string) {
	gf.TagMatcher.AddCriterion("Black", name, OpContains)
}

// AddECOFilter adds a filter for ECO code prefix.
func (gf *GameFilter) AddECOFilter(eco string) {
	gf.TagMatcher.AddCriterion("ECO", eco, OpContains)
}

// AddResultFilter adds a filter for game result.
func (gf *GameFilter) AddResultFilter(result string) {
	gf.TagMatcher.AddCriterion("Result", result, OpEqual)
}

// AddDateFilter adds a date filter with operator.
func (gf *GameFilter) AddDateFilter(date string, op TagOperator) {
	gf.TagMatcher.AddCriterion("Date", date, op)
}

// AddFENFilter adds an exact FEN position filter.
func (gf *GameFilter) AddFENFilter(fen string) error {
	return gf.PositionMatcher.AddFEN(fen, "")
}

// AddPatternFilter adds a FEN pattern filter.
func (gf *GameFilter) AddPatternFilter(pattern string, includeInvert bool) {
	gf.PositionMatcher.AddPattern(pattern, "", includeInvert)
}

// MatchGame checks if a game matches the filter criteria.
func (gf *GameFilter) MatchGame(game *chess.Game) bool {
	hasTagCriteria := gf.TagMatcher.CriteriaCount() > 0
	hasPositionCriteria := gf.PositionMatcher.PatternCount() > 0

	if !hasTagCriteria && !hasPositionCriteria {
		return true // no criteria = match all
	}

	// Check tag criteria if present
	tagMatches := true
	if hasTagCriteria {
		tagMatches = gf.TagMatcher.MatchGame(game)
	}

	// Check position criteria if present
	positionMatches := true
	if hasPositionCriteria {
		positionMatches = gf.PositionMatcher.MatchGame(game) != nil
	}

	// If both types of criteria are present, both must match (AND)
	// If only one type, that type must match
	if hasTagCriteria && hasPositionCriteria {
		if gf.RequireBoth {
			return tagMatches && positionMatches
		}
		// Default: both must match when both are specified
		return tagMatches && positionMatches
	}

	// Only one type of criteria is present
	return tagMatches && positionMatches
}

// HasCriteria returns true if any filter criteria are set.
func (gf *GameFilter) HasCriteria() bool {
	return gf.TagMatcher.CriteriaCount() > 0 || gf.PositionMatcher.PatternCount() > 0
}

// SetUseSoundex enables soundex matching for player names.
func (gf *GameFilter) SetUseSoundex(use bool) {
	gf.TagMatcher.SetUseSoundex(use)
}

// SetSubstringMatch enables substring matching for tag values.
func (gf *GameFilter) SetSubstringMatch(use bool) {
	gf.TagMatcher.SetSubstringMatch(use)
}

// Match implements GameMatcher interface.
func (gf *GameFilter) Match(game *chess.Game) bool {
	return gf.MatchGame(game)
}

// Name implements GameMatcher interface.
func (gf *GameFilter) Name() string {
	return "GameFilter"
}
