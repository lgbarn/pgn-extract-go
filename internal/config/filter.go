package config

import (
	"fmt"

	"github.com/lgbarn/pgn-extract-go/internal/errors"
)

// FilterConfig holds settings for game filtering and matching.
type FilterConfig struct {
	// Move bounds
	CheckMoveBounds bool
	LowerMoveBound  uint
	UpperMoveBound  uint
	OutputPlyLimit  int

	// Match conditions
	MatchCheckmate      bool
	MatchStalemate      bool
	MatchUnderpromotion bool
	CheckRepetition     bool
	CheckFiftyMoveRule  bool
	TagMatchAnywhere    bool

	// Game selection
	MaxMatches      uint
	KeepBrokenGames bool

	// Ply manipulation
	DropPlyNumber int
	StartPly      uint

	// Positional search
	PositionalSearchDepth uint
	MatchPermutations     bool
	PositionalVariations  bool
	UseSoundex            bool

	// Quiescence
	QuiescenceThreshold uint
}

// NewFilterConfig creates a FilterConfig with default values.
func NewFilterConfig() *FilterConfig {
	return &FilterConfig{
		// Most filter options disabled by default
		CheckMoveBounds:       false,
		MatchCheckmate:        false,
		MatchStalemate:        false,
		MatchUnderpromotion:   false,
		CheckRepetition:       false,
		CheckFiftyMoveRule:    false,
		KeepBrokenGames:       false,
		PositionalSearchDepth: 0,
	}
}

// Validate checks that the filter configuration is valid.
func (f *FilterConfig) Validate() error {
	if f.CheckMoveBounds && f.LowerMoveBound > f.UpperMoveBound {
		return fmt.Errorf("lower move bound (%d) > upper move bound (%d): %w",
			f.LowerMoveBound, f.UpperMoveBound, errors.ErrInvalidConfig)
	}
	return nil
}
