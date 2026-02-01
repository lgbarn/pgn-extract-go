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
	file, err := os.Open(filename) //nolint:gosec // G304: CLI tool opens user-specified files
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
				if err := gf.PositionMatcher.AddFEN(rest, ""); err != nil {
				continue // skip invalid FEN lines
			}
			}
		} else {
			if err := gf.TagMatcher.ParseCriterion(line); err != nil {
			continue // skip unparseable criterion lines
		}
		}
	}

	return scanner.Err()
}

// AddTagCriterion adds a tag criterion directly.
func (gf *GameFilter) AddTagCriterion(tagName, value string, op TagOperator) {
	_ = gf.TagMatcher.AddCriterion(tagName, value, op) // caller controls operator; non-regex ops cannot fail
}

// AddPlayerFilter adds a filter for player name (matches White or Black).
func (gf *GameFilter) AddPlayerFilter(name string) {
	gf.TagMatcher.AddPlayerCriterion(name)
}

// AddWhiteFilter adds a filter for White player.
func (gf *GameFilter) AddWhiteFilter(name string) {
	_ = gf.TagMatcher.AddCriterion("White", name, OpContains) // OpContains cannot fail
}

// AddBlackFilter adds a filter for Black player.
func (gf *GameFilter) AddBlackFilter(name string) {
	_ = gf.TagMatcher.AddCriterion("Black", name, OpContains) // OpContains cannot fail
}

// AddECOFilter adds a filter for ECO code prefix.
func (gf *GameFilter) AddECOFilter(eco string) {
	_ = gf.TagMatcher.AddCriterion("ECO", eco, OpContains) // OpContains cannot fail
}

// AddResultFilter adds a filter for game result.
func (gf *GameFilter) AddResultFilter(result string) {
	_ = gf.TagMatcher.AddCriterion("Result", result, OpEqual) // OpEqual cannot fail
}

// AddDateFilter adds a date filter with operator.
func (gf *GameFilter) AddDateFilter(date string, op TagOperator) {
	_ = gf.TagMatcher.AddCriterion("Date", date, op) // caller controls operator
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

	tagMatches := !hasTagCriteria || gf.TagMatcher.MatchGame(game)
	positionMatches := !hasPositionCriteria || gf.PositionMatcher.MatchGame(game) != nil

	// Both criteria types must match when present (AND logic)
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
