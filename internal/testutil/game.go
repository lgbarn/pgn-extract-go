// Package testutil provides shared test utilities for the pgn-extract-go project.
// These utilities reduce code duplication across test files and provide
// consistent test setup helpers.
package testutil

import (
	"strings"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/parser"
)

// ParseTestGame parses a PGN string and returns the first game, or nil if
// parsing fails or no games are found. Use this for tests where parse failure
// is an acceptable outcome.
func ParseTestGame(pgn string) *chess.Game {
	if games := ParseTestGames(pgn); len(games) > 0 {
		return games[0]
	}
	return nil
}

// ParseTestGames parses a PGN string and returns all games found.
// Returns an empty slice if parsing fails or no games are found.
func ParseTestGames(pgn string) []*chess.Game {
	cfg := config.NewConfig()
	cfg.Verbosity = 0
	p := parser.NewParser(strings.NewReader(pgn), cfg)
	games, err := p.ParseAllGames()
	if err != nil || len(games) == 0 {
		return nil
	}
	return games
}

// MustParseGame parses a PGN string and returns the first game.
// It calls t.Fatal if parsing fails or no games are found.
// Use this in test setup where parse failure should abort the test.
func MustParseGame(t *testing.T, pgn string) *chess.Game {
	t.Helper()
	game := ParseTestGame(pgn)
	if game == nil {
		t.Fatalf("failed to parse test game:\n%s", pgn)
	}
	return game
}

// MustParseGames parses a PGN string and returns all games found.
// It calls t.Fatal if parsing fails or no games are found.
func MustParseGames(t *testing.T, pgn string) []*chess.Game {
	t.Helper()
	games := ParseTestGames(pgn)
	if len(games) == 0 {
		t.Fatalf("failed to parse any games from PGN:\n%s", pgn)
	}
	return games
}
