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

const basePGNTags = `[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

`

const sicilianNajdorfPGN = basePGNTags + `1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 *`

const giuocoPianoPGN = basePGNTags + `1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 *`

const noMatchPGN = basePGNTags + `1. a3 *`

const extendedSicilianPGN = basePGNTags + `1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 6. Be2 e5 7. Nb3 *`

func newTestClassifier(t *testing.T) *ECOClassifier {
	t.Helper()
	ec := NewECOClassifier()
	if err := ec.LoadFromReader(strings.NewReader(testECOData)); err != nil {
		t.Fatalf("failed to load ECO data: %v", err)
	}
	return ec
}

func TestECOClassifierLoad(t *testing.T) {
	ec := newTestClassifier(t)

	if got := ec.EntriesLoaded(); got != 3 {
		t.Errorf("EntriesLoaded() = %d; want 3", got)
	}
}

func TestECOClassifySicilian(t *testing.T) {
	ec := newTestClassifier(t)
	game := testutil.MustParseGame(t, sicilianNajdorfPGN)

	match := ec.ClassifyGame(game)
	if match == nil {
		t.Fatal("ClassifyGame() returned nil; want match")
	}

	if match.ECOCode != "B90" {
		t.Errorf("ECOCode = %q; want B90", match.ECOCode)
	}
	if match.Opening != "Sicilian" {
		t.Errorf("Opening = %q; want Sicilian", match.Opening)
	}
	if match.Variation != "Najdorf" {
		t.Errorf("Variation = %q; want Najdorf", match.Variation)
	}
}

func TestECOClassifyItalian(t *testing.T) {
	ec := newTestClassifier(t)
	game := testutil.MustParseGame(t, giuocoPianoPGN)

	match := ec.ClassifyGame(game)
	if match == nil {
		t.Fatal("ClassifyGame() returned nil; want match")
	}

	if match.ECOCode != "C50" {
		t.Errorf("ECOCode = %q; want C50", match.ECOCode)
	}
	if match.Opening != "Giuoco Piano" {
		t.Errorf("Opening = %q; want Giuoco Piano", match.Opening)
	}
}

func TestECOAddTags(t *testing.T) {
	ec := newTestClassifier(t)
	game := testutil.MustParseGame(t, sicilianNajdorfPGN)

	if _, ok := game.Tags["ECO"]; ok {
		t.Error("game should not have ECO tag initially")
	}

	if !ec.AddECOTags(game) {
		t.Error("AddECOTags() = false; want true")
	}

	if got := game.Tags["ECO"]; got != "B90" {
		t.Errorf("Tags[ECO] = %q; want B90", got)
	}
	if got := game.Tags["Opening"]; got != "Sicilian" {
		t.Errorf("Tags[Opening] = %q; want Sicilian", got)
	}
	if got := game.Tags["Variation"]; got != "Najdorf" {
		t.Errorf("Tags[Variation] = %q; want Najdorf", got)
	}
}

func TestECONoMatch(t *testing.T) {
	ec := newTestClassifier(t)
	game := testutil.MustParseGame(t, noMatchPGN)

	match := ec.ClassifyGame(game)
	if match != nil {
		t.Errorf("ClassifyGame() = %q; want nil", match.ECOCode)
	}
}

func TestECOPartialMatch(t *testing.T) {
	ec := newTestClassifier(t)
	game := testutil.MustParseGame(t, extendedSicilianPGN)

	match := ec.ClassifyGame(game)
	if match == nil {
		t.Fatal("ClassifyGame() returned nil; want match for extended game")
	}

	if match.ECOCode != "B90" {
		t.Errorf("ECOCode = %q; want B90", match.ECOCode)
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
