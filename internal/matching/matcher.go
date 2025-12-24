// Package matching provides game filtering and matching capabilities.
package matching

import (
	"fmt"
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// GameMatcher is the interface for all game matching implementations.
// Any component that can evaluate whether a game matches certain criteria
// should implement this interface.
type GameMatcher interface {
	// Match returns true if the game matches the matcher's criteria.
	Match(game *chess.Game) bool

	// Name returns a descriptive name for this matcher.
	Name() string
}

// MatchMode specifies how multiple matchers are combined.
type MatchMode int

const (
	// MatchAll requires all matchers to match (AND logic).
	MatchAll MatchMode = iota

	// MatchAny requires at least one matcher to match (OR logic).
	MatchAny
)

// CompositeMatcher combines multiple GameMatchers with AND or OR logic.
type CompositeMatcher struct {
	matchers []GameMatcher
	mode     MatchMode
}

// NewCompositeMatcher creates a new CompositeMatcher with the given mode and matchers.
func NewCompositeMatcher(mode MatchMode, matchers ...GameMatcher) *CompositeMatcher {
	return &CompositeMatcher{
		matchers: matchers,
		mode:     mode,
	}
}

// Match implements GameMatcher.
func (c *CompositeMatcher) Match(game *chess.Game) bool {
	if len(c.matchers) == 0 {
		// Empty composite: AND mode is vacuously true, OR mode has no conditions
		return c.mode == MatchAll
	}

	switch c.mode {
	case MatchAll:
		for _, m := range c.matchers {
			if !m.Match(game) {
				return false
			}
		}
		return true
	case MatchAny:
		for _, m := range c.matchers {
			if m.Match(game) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// Name implements GameMatcher.
func (c *CompositeMatcher) Name() string {
	if len(c.matchers) == 0 {
		return "CompositeMatcher(empty)"
	}

	names := make([]string, len(c.matchers))
	for i, m := range c.matchers {
		names[i] = m.Name()
	}

	modeStr := "AND"
	if c.mode == MatchAny {
		modeStr = "OR"
	}

	return fmt.Sprintf("CompositeMatcher(%s: %s)", modeStr, strings.Join(names, ", "))
}

// Add adds a matcher to the composite.
func (c *CompositeMatcher) Add(m GameMatcher) {
	c.matchers = append(c.matchers, m)
}

// Matchers returns the list of matchers in this composite.
func (c *CompositeMatcher) Matchers() []GameMatcher {
	return c.matchers
}

// Mode returns the match mode (MatchAll or MatchAny).
func (c *CompositeMatcher) Mode() MatchMode {
	return c.mode
}
