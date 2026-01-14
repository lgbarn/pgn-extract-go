package testutil

import "testing"

// assertTag is a test helper that verifies a game tag matches the expected value.
func assertTag(t *testing.T, game interface{ GetTag(string) string }, tag, want string) {
	t.Helper()
	if want == "" {
		return
	}
	if got := game.GetTag(tag); got != want {
		t.Errorf("game.GetTag(%q) = %q, want %q", tag, got, want)
	}
}

func TestParseTestGame(t *testing.T) {
	tests := []struct {
		name       string
		pgn        string
		wantNil    bool
		wantMoves  int
		wantEvent  string
		wantWhite  string
		wantBlack  string
		wantResult string
	}{
		{
			name: "valid simple game",
			pgn: `[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Nf3 1-0`,
			wantNil:    false,
			wantMoves:  3,
			wantEvent:  "Test",
			wantWhite:  "Player1",
			wantBlack:  "Player2",
			wantResult: "1-0",
		},
		{
			name:    "empty PGN",
			pgn:     "",
			wantNil: true,
		},
		{
			name:    "whitespace only",
			pgn:     "   \n\t  ",
			wantNil: true,
		},
		{
			name: "game with castling",
			pgn: `[Event "Castle Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. O-O *`,
			wantNil:   false,
			wantMoves: 7,
			wantEvent: "Castle Test",
		},
		{
			name: "game with variations",
			pgn: `[Event "Variation Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 (1... c5 2. Nf3) 2. Nf3 *`,
			wantNil:   false,
			wantMoves: 3,
			wantEvent: "Variation Test",
		},
		{
			name: "game with comments",
			pgn: `[Event "Comment Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 {Best by test} e5 2. Nf3 *`,
			wantNil:   false,
			wantMoves: 3,
			wantEvent: "Comment Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := ParseTestGame(tt.pgn)

			if tt.wantNil {
				if game != nil {
					t.Errorf("ParseTestGame() = %v, want nil", game)
				}
				return
			}

			if game == nil {
				t.Fatal("ParseTestGame() = nil, want game")
			}

			if tt.wantMoves > 0 && game.PlyCount() != tt.wantMoves {
				t.Errorf("game.PlyCount() = %d, want %d", game.PlyCount(), tt.wantMoves)
			}

			assertTag(t, game, "Event", tt.wantEvent)
			assertTag(t, game, "White", tt.wantWhite)
			assertTag(t, game, "Black", tt.wantBlack)
			assertTag(t, game, "Result", tt.wantResult)
		})
	}
}

func TestMustParseGame(t *testing.T) {
	pgn := `[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 *`

	game := MustParseGame(t, pgn)
	if game == nil {
		t.Error("MustParseGame() returned nil for valid PGN")
	}
}

func TestParseTestGames(t *testing.T) {
	tests := []struct {
		name      string
		pgn       string
		wantCount int
	}{
		{
			name:      "empty PGN",
			pgn:       "",
			wantCount: 0,
		},
		{
			name: "single game",
			pgn: `[Event "Test1"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 *`,
			wantCount: 1,
		},
		{
			name: "two games",
			pgn: `[Event "Test1"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 *

[Event "Test2"]
[Site "Test"]
[Date "2024.01.01"]
[Round "2"]
[White "C"]
[Black "D"]
[Result "*"]

1. d4 *`,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			games := ParseTestGames(tt.pgn)
			if len(games) != tt.wantCount {
				t.Errorf("ParseTestGames() returned %d games, want %d", len(games), tt.wantCount)
			}
		})
	}
}
