package eco

import (
	"strings"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/testutil"
)

const testECOData = `
[ECO "B90"]
[Opening "Sicilian"]
[Variation "Najdorf"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 *

[ECO "C50"]
[Opening "Giuoco Piano"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 *

[ECO "D35"]
[Opening "QGD"]
[Variation "exchange variation"]

1. d4 d5 2. c4 e6 3. Nc3 Nf6 4. cxd5 exd5 *
`

func TestECOClassifierLoad(t *testing.T) {
	ec := NewECOClassifier()
	err := ec.LoadFromReader(strings.NewReader(testECOData))
	if err != nil {
		t.Fatalf("Failed to load ECO data: %v", err)
	}

	if ec.EntriesLoaded() != 3 {
		t.Errorf("Expected 3 entries, got %d", ec.EntriesLoaded())
	}
}

func TestECOClassifySicilian(t *testing.T) {
	ec := NewECOClassifier()
	ec.LoadFromReader(strings.NewReader(testECOData))

	// Create a game with Sicilian Najdorf moves
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 *
`)

	match := ec.ClassifyGame(game)
	if match == nil {
		t.Fatal("Expected ECO match, got nil")
	}

	if match.ECOCode != "B90" {
		t.Errorf("Expected ECO B90, got %s", match.ECOCode)
	}
	if match.Opening != "Sicilian" {
		t.Errorf("Expected Opening 'Sicilian', got '%s'", match.Opening)
	}
	if match.Variation != "Najdorf" {
		t.Errorf("Expected Variation 'Najdorf', got '%s'", match.Variation)
	}
}

func TestECOClassifyItalian(t *testing.T) {
	ec := NewECOClassifier()
	ec.LoadFromReader(strings.NewReader(testECOData))

	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 *
`)

	match := ec.ClassifyGame(game)
	if match == nil {
		t.Fatal("Expected ECO match, got nil")
	}

	if match.ECOCode != "C50" {
		t.Errorf("Expected ECO C50, got %s", match.ECOCode)
	}
	if match.Opening != "Giuoco Piano" {
		t.Errorf("Expected Opening 'Giuoco Piano', got '%s'", match.Opening)
	}
}

func TestECOAddTags(t *testing.T) {
	ec := NewECOClassifier()
	ec.LoadFromReader(strings.NewReader(testECOData))

	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 *
`)

	// Verify no ECO tag initially
	if _, ok := game.Tags["ECO"]; ok {
		t.Error("Game should not have ECO tag initially")
	}

	// Add ECO tags
	if !ec.AddECOTags(game) {
		t.Error("AddECOTags should return true")
	}

	// Verify tags were added
	if game.Tags["ECO"] != "B90" {
		t.Errorf("Expected ECO tag 'B90', got '%s'", game.Tags["ECO"])
	}
	if game.Tags["Opening"] != "Sicilian" {
		t.Errorf("Expected Opening tag 'Sicilian', got '%s'", game.Tags["Opening"])
	}
	if game.Tags["Variation"] != "Najdorf" {
		t.Errorf("Expected Variation tag 'Najdorf', got '%s'", game.Tags["Variation"])
	}
}

func TestECONoMatch(t *testing.T) {
	ec := NewECOClassifier()
	ec.LoadFromReader(strings.NewReader(testECOData))

	// A game that doesn't match any ECO entry
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. a3 *
`)

	match := ec.ClassifyGame(game)
	if match != nil {
		t.Errorf("Expected no match, got %s", match.ECOCode)
	}
}

func TestECOPartialMatch(t *testing.T) {
	ec := NewECOClassifier()
	ec.LoadFromReader(strings.NewReader(testECOData))

	// A game that goes beyond the ECO position
	game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 6. Be2 e5 7. Nb3 *
`)

	match := ec.ClassifyGame(game)
	if match == nil {
		t.Fatal("Expected ECO match for extended game, got nil")
	}

	if match.ECOCode != "B90" {
		t.Errorf("Expected ECO B90, got %s", match.ECOCode)
	}
}

// Verify board setup works correctly
func TestBoardSetup(t *testing.T) {
	board, err := engine.NewBoardFromFEN(engine.InitialFEN)
	if err != nil {
		t.Fatalf("Failed to create board: %v", err)
	}

	if board.Get('e', '1') != chess.W(chess.King) {
		t.Error("White king not on e1")
	}
	if board.Get('e', '8') != chess.B(chess.King) {
		t.Error("Black king not on e8")
	}
}
