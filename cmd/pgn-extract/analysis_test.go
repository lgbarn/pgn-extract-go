package main

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/cql"
	"github.com/lgbarn/pgn-extract-go/internal/testutil"
)

// ---------------------------------------------------------------------------
// fixMissingTags
// ---------------------------------------------------------------------------

func TestFixMissingTags(t *testing.T) {
	tests := []struct {
		name      string
		tags      map[string]string
		wantFixed bool
		checkTag  string
		checkVal  string
	}{
		{
			name: "all seven tags present",
			tags: map[string]string{
				"Event": "Tata Steel", "Site": "Wijk aan Zee",
				"Date": "2024.01.01", "Round": "1",
				"White": "Carlsen", "Black": "Caruana", "Result": "1-0",
			},
			wantFixed: false,
		},
		{
			name: "missing Event",
			tags: map[string]string{
				"Site": "Wijk", "Date": "2024.01.01", "Round": "1",
				"White": "A", "Black": "B", "Result": "*",
			},
			wantFixed: true,
			checkTag:  "Event",
			checkVal:  "?",
		},
		{
			name:      "missing all tags",
			tags:      map[string]string{},
			wantFixed: true,
			checkTag:  "White",
			checkVal:  "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := chess.NewGame()
			for k, v := range tt.tags {
				game.SetTag(k, v)
			}
			got := fixMissingTags(game)
			if got != tt.wantFixed {
				t.Errorf("fixMissingTags() = %v; want %v", got, tt.wantFixed)
			}
			if tt.checkTag != "" {
				val := game.GetTag(tt.checkTag)
				if val != tt.checkVal {
					t.Errorf("tag %q = %q; want %q", tt.checkTag, val, tt.checkVal)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// fixResultTag
// ---------------------------------------------------------------------------

func TestFixResultTag(t *testing.T) {
	tests := []struct {
		name      string
		result    string
		wantFixed bool
		wantVal   string
	}{
		{"valid 1-0", "1-0", false, "1-0"},
		{"valid 0-1", "0-1", false, "0-1"},
		{"valid draw", "1/2-1/2", false, "1/2-1/2"},
		{"valid unknown", "*", false, "*"},
		{"white -> 1-0", "white", true, "1-0"},
		{"White wins -> 1-0", "White wins", true, "1-0"},
		{"black -> 0-1", "black", true, "0-1"},
		{"draw -> 1/2-1/2", "draw", true, "1/2-1/2"},
		{"0.5-0.5 -> 1/2-1/2", "0.5-0.5", true, "1/2-1/2"},
		{"garbage -> *", "???", true, "*"},
		{"empty -> *", "", true, "*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := chess.NewGame()
			game.SetTag("Result", tt.result)
			got := fixResultTag(game)
			if got != tt.wantFixed {
				t.Errorf("fixResultTag() = %v; want %v", got, tt.wantFixed)
			}
			val := game.GetTag("Result")
			if val != tt.wantVal {
				t.Errorf("Result = %q; want %q", val, tt.wantVal)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// fixDateFormat
// ---------------------------------------------------------------------------

func TestFixDateFormat(t *testing.T) {
	tests := []struct {
		name      string
		date      string
		wantFixed bool
		wantVal   string
	}{
		{"normal dot date", "2024.01.15", false, "2024.01.15"},
		{"slash -> dot", "2024/01/15", true, "2024.01.15"},
		{"dash -> dot", "2024-01-15", true, "2024.01.15"},
		{"empty date", "", false, ""},
		{"unknown date", "????.??.??", false, "????.??.??"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := chess.NewGame()
			if tt.date != "" {
				game.SetTag("Date", tt.date)
			}
			got := fixDateFormat(game)
			if got != tt.wantFixed {
				t.Errorf("fixDateFormat() = %v; want %v", got, tt.wantFixed)
			}
			val := game.GetTag("Date")
			if val != tt.wantVal {
				t.Errorf("Date = %q; want %q", val, tt.wantVal)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// cleanAllTags
// ---------------------------------------------------------------------------

func TestCleanAllTags(t *testing.T) {
	t.Run("tags with control chars get cleaned", func(t *testing.T) {
		game := chess.NewGame()
		game.SetTag("Event", "World\x00Cup")
		game.SetTag("White", "Normal")
		got := cleanAllTags(game)
		if !got {
			t.Error("cleanAllTags() = false; want true")
		}
		if game.GetTag("Event") != "WorldCup" {
			t.Errorf("Event = %q; want %q", game.GetTag("Event"), "WorldCup")
		}
	})

	t.Run("normal tags unchanged", func(t *testing.T) {
		game := chess.NewGame()
		game.SetTag("Event", "Normal Event")
		game.SetTag("White", "Player One")
		got := cleanAllTags(game)
		if got {
			t.Error("cleanAllTags() = true; want false")
		}
	})

	t.Run("whitespace trimmed", func(t *testing.T) {
		game := chess.NewGame()
		game.SetTag("Event", "  Leading Space  ")
		got := cleanAllTags(game)
		if !got {
			t.Error("cleanAllTags() = false; want true")
		}
		if game.GetTag("Event") != "Leading Space" {
			t.Errorf("Event = %q; want %q", game.GetTag("Event"), "Leading Space")
		}
	})
}

// ---------------------------------------------------------------------------
// fixGame (calls all sub-fixers)
// ---------------------------------------------------------------------------

func TestFixGame(t *testing.T) {
	t.Run("game needing fixes returns true", func(t *testing.T) {
		game := chess.NewGame()
		// Missing tags, bad result, bad date, control chars
		game.SetTag("Result", "white")
		game.SetTag("Date", "2024/01/01")
		game.SetTag("Site", "Test\x00Site")
		got := fixGame(game)
		if !got {
			t.Error("fixGame() = false; want true")
		}
		// All seven tags should now be present
		if game.GetTag("Event") != "?" {
			t.Errorf("Event = %q; want %q", game.GetTag("Event"), "?")
		}
		if game.GetTag("Result") != "1-0" {
			t.Errorf("Result = %q; want %q", game.GetTag("Result"), "1-0")
		}
		if game.GetTag("Date") != "2024.01.01" {
			t.Errorf("Date = %q; want %q", game.GetTag("Date"), "2024.01.01")
		}
	})

	t.Run("complete game needs no fixing", func(t *testing.T) {
		game := chess.NewGame()
		game.SetTag("Event", "Test")
		game.SetTag("Site", "Here")
		game.SetTag("Date", "2024.01.01")
		game.SetTag("Round", "1")
		game.SetTag("White", "A")
		game.SetTag("Black", "B")
		game.SetTag("Result", "1-0")
		got := fixGame(game)
		if got {
			t.Error("fixGame() = true; want false")
		}
	})
}

// ---------------------------------------------------------------------------
// analyzeGame
// ---------------------------------------------------------------------------

func TestAnalyzeGame(t *testing.T) {
	game := testutil.MustParseGame(t, `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 1-0
`)

	board, analysis := analyzeGame(game)
	if board == nil {
		t.Fatal("analyzeGame returned nil board")
	}
	if analysis == nil {
		t.Fatal("analyzeGame returned nil analysis")
	}
}

// ---------------------------------------------------------------------------
// validateGame
// ---------------------------------------------------------------------------

func TestValidateGame(t *testing.T) {
	game := testutil.MustParseGame(t, `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 1-0
`)

	result := validateGame(game)
	if result == nil {
		t.Fatal("validateGame returned nil")
	}
	if !result.Valid {
		t.Errorf("validateGame().Valid = false; want true; error: %s", result.ErrorMsg)
	}
}

// ---------------------------------------------------------------------------
// replayGame
// ---------------------------------------------------------------------------

func TestReplayGame(t *testing.T) {
	game := testutil.MustParseGame(t, `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 1-0
`)

	board := replayGame(game)
	if board == nil {
		t.Fatal("replayGame returned nil board")
	}
}

// ---------------------------------------------------------------------------
// matchesCQL
// ---------------------------------------------------------------------------

func TestMatchesCQL(t *testing.T) {
	mateNode, err := cql.Parse("mate")
	if err != nil {
		t.Fatalf("cql.Parse(\"mate\") error: %v", err)
	}

	t.Run("checkmate game matches mate", func(t *testing.T) {
		game := testutil.MustParseGame(t, `[Event "Fool's Mate"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "0-1"]

1. f3 e5 2. g4 Qh4# 0-1
`)
		if !matchesCQL(game, mateNode) {
			t.Error("matchesCQL(checkmate game, mate) = false; want true")
		}
	})

	t.Run("non-checkmate game does not match mate", func(t *testing.T) {
		game := testutil.MustParseGame(t, `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 1-0
`)
		if matchesCQL(game, mateNode) {
			t.Error("matchesCQL(non-checkmate game, mate) = true; want false")
		}
	})
}

func TestMatchesCQL_CheckQuery(t *testing.T) {
	checkNode, err := cql.Parse("check")
	if err != nil {
		t.Fatalf("cql.Parse(\"check\") error: %v", err)
	}

	// Fool's mate ends with Qh4# which is both check and mate
	game := testutil.MustParseGame(t, `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "0-1"]

1. f3 e5 2. g4 Qh4# 0-1
`)
	if !matchesCQL(game, checkNode) {
		t.Error("matchesCQL(game with check, check) = false; want true")
	}
}

// ---------------------------------------------------------------------------
// fixResultTag edge cases
// ---------------------------------------------------------------------------

func TestFixResultTag_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		result  string
		wantVal string
	}{
		{"1/2 shorthand", "1/2", "1/2-1/2"},
		{"Black wins", "Black wins", "0-1"},
		{"uppercase WHITE", "WHITE", "1-0"},
		{"mixed case Draw", "Draw", "1/2-1/2"},
		{"trailing space", "1-0 ", "1-0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := chess.NewGame()
			game.SetTag("Result", tt.result)
			fixResultTag(game)
			val := game.GetTag("Result")
			if val != tt.wantVal {
				t.Errorf("Result = %q; want %q", val, tt.wantVal)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// analyzeGame with games containing special features
// ---------------------------------------------------------------------------

func TestAnalyzeGame_WithVariation(t *testing.T) {
	// Game with a variation
	game := testutil.MustParseGame(t, `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "1-0"]

1. e4 e5 2. Nf3 (2. Bc4 Nc6) 2... Nc6 1-0
`)

	board, analysis := analyzeGame(game)
	if board == nil {
		t.Fatal("analyzeGame returned nil board")
	}
	if analysis == nil {
		t.Fatal("analyzeGame returned nil analysis")
	}
}

func TestAnalyzeGame_WithComments(t *testing.T) {
	game := testutil.MustParseGame(t, `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "1-0"]

1. e4 {The King's Pawn opening} 1... e5 2. Nf3 Nc6 1-0
`)

	board, analysis := analyzeGame(game)
	if board == nil {
		t.Fatal("analyzeGame returned nil board")
	}
	if analysis == nil {
		t.Fatal("analyzeGame returned nil analysis")
	}
}
