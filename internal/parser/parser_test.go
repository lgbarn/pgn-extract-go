package parser

import (
	"strings"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
)

// parseTestGame is a helper that parses a PGN string and returns the game.
func parseTestGame(t *testing.T, pgn string) *chess.Game {
	t.Helper()
	p := NewParser(strings.NewReader(pgn), config.NewConfig())
	game, err := p.ParseGame()
	if err != nil {
		t.Fatalf("ParseGame error: %v", err)
	}
	if game == nil {
		t.Fatal("Expected game, got nil")
	}
	return game
}

func TestParseSimpleGame(t *testing.T) {
	pgn := `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0
`

	game := parseTestGame(t, pgn)

	// Check tags
	if got := game.GetTag("Event"); got != "Test" {
		t.Errorf("Event = %q, want %q", got, "Test")
	}
	if got := game.GetTag("White"); got != "Player1" {
		t.Errorf("White = %q, want %q", got, "Player1")
	}
	if got := game.GetTag("Black"); got != "Player2" {
		t.Errorf("Black = %q, want %q", got, "Player2")
	}

	// Check moves
	if game.Moves == nil {
		t.Fatal("Expected moves, got nil")
	}
	if count := game.PlyCount(); count != 6 {
		t.Errorf("PlyCount = %d, want 6", count)
	}
	if got := game.Moves.Text; got != "e4" {
		t.Errorf("First move = %q, want %q", got, "e4")
	}

	// Check result
	lastMove := game.LastMove()
	if lastMove == nil {
		t.Fatal("Expected last move, got nil")
	}
	if got := lastMove.TerminatingResult; got != "1-0" {
		t.Errorf("Result = %q, want %q", got, "1-0")
	}
}

func TestParseFoolsMate(t *testing.T) {
	pgn := `1. f3 e5 2. g4 Qh4# 0-1`
	game := parseTestGame(t, pgn)

	if count := game.PlyCount(); count != 4 {
		t.Errorf("PlyCount = %d, want 4", count)
	}
	if got := game.LastMove().TerminatingResult; got != "0-1" {
		t.Errorf("Result = %q, want %q", got, "0-1")
	}
}

func TestParseWithComments(t *testing.T) {
	pgn := `[Event "Test"]
[White "Player1"]
[Black "Player2"]
[Result "*"]

1. e4 {Best by test} e5 2. Nf3 Nc6 *
`

	game := parseTestGame(t, pgn)

	if len(game.Moves.Comments) == 0 {
		t.Fatal("Expected comment on first move")
	}
	if got := game.Moves.Comments[0].Text; got != "Best by test" {
		t.Errorf("Comment = %q, want %q", got, "Best by test")
	}
}

func TestParseWithVariations(t *testing.T) {
	pgn := `[Event "Test"]
[Result "*"]

1. e4 e5 (1... c5 2. Nf3) 2. Nf3 *
`

	game := parseTestGame(t, pgn)

	if game.Moves == nil || game.Moves.Next == nil {
		t.Fatal("Expected at least 2 moves")
	}

	e5 := game.Moves.Next
	if len(e5.Variations) == 0 {
		t.Error("Expected variation on 1...e5")
	}
}

func TestParseCastling(t *testing.T) {
	tests := []struct {
		name     string
		pgn      string
		expected string
	}{
		{"O-O", "1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. O-O *", "O-O"},
		{"O-O-O", "1. d4 d5 2. Nc3 Nc6 3. Bf4 Bf5 4. Qd2 Qd7 5. O-O-O *", "O-O-O"},
		{"0-0", "1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. 0-0 *", "O-O"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := parseTestGame(t, tt.pgn)

			found := false
			for move := game.Moves; move != nil; move = move.Next {
				if move.Text == tt.expected || move.Text == "0-0" || move.Text == "0-0-0" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected castling move %s not found", tt.expected)
			}
		})
	}
}

func TestParseMultipleGames(t *testing.T) {
	pgn := `[Event "Game 1"]
[Result "1-0"]

1. e4 e5 1-0

[Event "Game 2"]
[Result "0-1"]

1. d4 d5 0-1
`

	p := NewParser(strings.NewReader(pgn), config.NewConfig())
	games, err := p.ParseAllGames()
	if err != nil {
		t.Fatalf("ParseAllGames error: %v", err)
	}

	if len(games) != 2 {
		t.Fatalf("len(games) = %d, want 2", len(games))
	}
	if got := games[0].GetTag("Event"); got != "Game 1" {
		t.Errorf("games[0].Event = %q, want %q", got, "Game 1")
	}
	if got := games[1].GetTag("Event"); got != "Game 2" {
		t.Errorf("games[1].Event = %q, want %q", got, "Game 2")
	}
}

func TestParseNAGs(t *testing.T) {
	pgn := `[Result "*"]

1. e4! e5? 2. Nf3!! Nc6?? *
`

	game := parseTestGame(t, pgn)

	if len(game.Moves.NAGs) == 0 {
		t.Error("Expected NAG on first move (e4!)")
	}
}
