package parser

import (
	"strings"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/config"
)

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

	cfg := config.NewConfig()
	p := NewParser(strings.NewReader(pgn), cfg)

	game, err := p.ParseGame()
	if err != nil {
		t.Fatalf("ParseGame error: %v", err)
	}
	if game == nil {
		t.Fatal("Expected game, got nil")
	}

	// Check tags
	if game.GetTag("Event") != "Test" {
		t.Errorf("Expected Event='Test', got '%s'", game.GetTag("Event"))
	}
	if game.GetTag("White") != "Player1" {
		t.Errorf("Expected White='Player1', got '%s'", game.GetTag("White"))
	}
	if game.GetTag("Black") != "Player2" {
		t.Errorf("Expected Black='Player2', got '%s'", game.GetTag("Black"))
	}

	// Check moves
	if game.Moves == nil {
		t.Fatal("Expected moves, got nil")
	}

	// Count moves
	count := game.PlyCount()
	if count != 6 {
		t.Errorf("Expected 6 plies, got %d", count)
	}

	// Check first move
	if game.Moves.Text != "e4" {
		t.Errorf("Expected first move 'e4', got '%s'", game.Moves.Text)
	}

	// Check result
	lastMove := game.LastMove()
	if lastMove == nil {
		t.Fatal("Expected last move, got nil")
	}
	if lastMove.TerminatingResult != "1-0" {
		t.Errorf("Expected result '1-0', got '%s'", lastMove.TerminatingResult)
	}
}

func TestParseFoolsMate(t *testing.T) {
	// Fool's mate - shortest possible checkmate
	pgn := `1. f3 e5 2. g4 Qh4# 0-1`

	cfg := config.NewConfig()
	p := NewParser(strings.NewReader(pgn), cfg)

	game, err := p.ParseGame()
	if err != nil {
		t.Fatalf("ParseGame error: %v", err)
	}
	if game == nil {
		t.Fatal("Expected game, got nil")
	}

	count := game.PlyCount()
	if count != 4 {
		t.Errorf("Expected 4 plies, got %d", count)
	}

	lastMove := game.LastMove()
	if lastMove.TerminatingResult != "0-1" {
		t.Errorf("Expected result '0-1', got '%s'", lastMove.TerminatingResult)
	}
}

func TestParseWithComments(t *testing.T) {
	pgn := `[Event "Test"]
[White "Player1"]
[Black "Player2"]
[Result "*"]

1. e4 {Best by test} e5 2. Nf3 Nc6 *
`

	cfg := config.NewConfig()
	p := NewParser(strings.NewReader(pgn), cfg)

	game, err := p.ParseGame()
	if err != nil {
		t.Fatalf("ParseGame error: %v", err)
	}
	if game == nil {
		t.Fatal("Expected game, got nil")
	}

	// Check first move has comment
	if len(game.Moves.Comments) == 0 {
		t.Error("Expected comment on first move")
	} else if game.Moves.Comments[0].Text != "Best by test" {
		t.Errorf("Expected comment 'Best by test', got '%s'", game.Moves.Comments[0].Text)
	}
}

func TestParseWithVariations(t *testing.T) {
	pgn := `[Event "Test"]
[Result "*"]

1. e4 e5 (1... c5 2. Nf3) 2. Nf3 *
`

	cfg := config.NewConfig()
	p := NewParser(strings.NewReader(pgn), cfg)

	game, err := p.ParseGame()
	if err != nil {
		t.Fatalf("ParseGame error: %v", err)
	}
	if game == nil {
		t.Fatal("Expected game, got nil")
	}

	// Check second ply (e5) has a variation
	if game.Moves == nil || game.Moves.Next == nil {
		t.Fatal("Expected at least 2 moves")
	}

	e5 := game.Moves.Next // 1...e5
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
			cfg := config.NewConfig()
			p := NewParser(strings.NewReader(tt.pgn), cfg)

			game, err := p.ParseGame()
			if err != nil {
				t.Fatalf("ParseGame error: %v", err)
			}
			if game == nil {
				t.Fatal("Expected game, got nil")
			}

			// Find the castling move
			found := false
			for move := game.Moves; move != nil; move = move.Next {
				if move.Text == tt.expected || move.Text == "0-0" || move.Text == "0-0-0" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected to find castling move %s", tt.expected)
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

	cfg := config.NewConfig()
	p := NewParser(strings.NewReader(pgn), cfg)

	games, err := p.ParseAllGames()
	if err != nil {
		t.Fatalf("ParseAllGames error: %v", err)
	}

	if len(games) != 2 {
		t.Fatalf("Expected 2 games, got %d", len(games))
	}

	if games[0].GetTag("Event") != "Game 1" {
		t.Errorf("Expected first game Event='Game 1', got '%s'", games[0].GetTag("Event"))
	}
	if games[1].GetTag("Event") != "Game 2" {
		t.Errorf("Expected second game Event='Game 2', got '%s'", games[1].GetTag("Event"))
	}
}

func TestParseNAGs(t *testing.T) {
	pgn := `[Result "*"]

1. e4! e5? 2. Nf3!! Nc6?? *
`

	cfg := config.NewConfig()
	p := NewParser(strings.NewReader(pgn), cfg)

	game, err := p.ParseGame()
	if err != nil {
		t.Fatalf("ParseGame error: %v", err)
	}
	if game == nil {
		t.Fatal("Expected game, got nil")
	}

	// First move should have NAG
	if len(game.Moves.NAGs) == 0 {
		t.Error("Expected NAG on first move (e4!)")
	}
}
