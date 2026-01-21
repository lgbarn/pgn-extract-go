package eco

import (
	"strings"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/parser"
)

// Sample ECO entries in PGN format
const sampleECOData = `[ECO "C00"]
[Opening "French Defense"]

1. e4 e6 *

[ECO "C01"]
[Opening "French Defense"]
[Variation "Exchange Variation"]

1. e4 e6 2. d4 d5 3. exd5 *

[ECO "C10"]
[Opening "French Defense"]
[Variation "Paulsen Variation"]

1. e4 e6 2. d4 d5 3. Nc3 *

[ECO "B20"]
[Opening "Sicilian Defense"]

1. e4 c5 *

[ECO "B21"]
[Opening "Sicilian Defense"]
[Variation "Smith-Morra Gambit"]

1. e4 c5 2. d4 cxd4 3. c3 *

[ECO "C60"]
[Opening "Ruy Lopez"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 *

[ECO "C92"]
[Opening "Ruy Lopez"]
[Variation "Closed"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6 8. c3 O-O 9. h3 *

[ECO "D00"]
[Opening "Queen's Pawn Game"]

1. d4 d5 *

[ECO "D35"]
[Opening "Queen's Gambit Declined"]
[Variation "Exchange Variation"]

1. d4 d5 2. c4 e6 3. Nc3 Nf6 4. cxd5 *

[ECO "A00"]
[Opening "Irregular Opening"]

1. g4 *
`

// Test games
const ruyLopezGame = `[Event "Test"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6
8. c3 O-O 9. h3 Nb8 10. d4 Nbd7 *
`

const sicilianGame = `[Event "Test"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 *
`

const frenchGame = `[Event "Test"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e6 2. d4 d5 3. Nc3 Bb4 4. e5 c5 5. a3 Bxc3+ 6. bxc3 *
`

func getECOClassifier() *ECOClassifier {
	ec := NewECOClassifier()
	ec.LoadFromReader(strings.NewReader(sampleECOData))
	return ec
}

func parseGame(pgn string) *chess.Game {
	cfg := config.NewConfig()
	cfg.Verbosity = 0
	p := parser.NewParser(strings.NewReader(pgn), cfg)
	game, _ := p.ParseGame()
	return game
}

// Benchmark ECO classification
func BenchmarkECOClassifier_ClassifyGame_RuyLopez(b *testing.B) {
	ec := getECOClassifier()
	game := parseGame(ruyLopezGame)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec.ClassifyGame(game)
	}
}

func BenchmarkECOClassifier_ClassifyGame_Sicilian(b *testing.B) {
	ec := getECOClassifier()
	game := parseGame(sicilianGame)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec.ClassifyGame(game)
	}
}

func BenchmarkECOClassifier_ClassifyGame_French(b *testing.B) {
	ec := getECOClassifier()
	game := parseGame(frenchGame)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec.ClassifyGame(game)
	}
}

func BenchmarkECOClassifier_ClassifyGame_NoMatch(b *testing.B) {
	ec := getECOClassifier()
	// A game with an unusual opening that won't match
	unusualGame := parseGame(`[Event "Test"]
[White "A"]
[Black "B"]
[Result "*"]

1. a3 a6 2. b3 b6 3. c3 c6 4. d3 d6 *
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec.ClassifyGame(unusualGame)
	}
}

// Benchmark ECO loading
func BenchmarkECOClassifier_LoadFromReader(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ec := NewECOClassifier()
		ec.LoadFromReader(strings.NewReader(sampleECOData))
	}
}

// Benchmark with larger ECO data
func BenchmarkECOClassifier_LoadFromReader_Large(b *testing.B) {
	// Create larger ECO data by duplicating with different codes
	var sb strings.Builder
	codes := []string{"A", "B", "C", "D", "E"}
	for _, code := range codes {
		for i := 0; i < 20; i++ {
			sb.WriteString("[ECO \"")
			sb.WriteString(code)
			sb.WriteString("\"]\n")
			sb.WriteString("[Opening \"Test Opening\"]\n\n")
			sb.WriteString("1. e4 e5 2. Nf3 *\n\n")
		}
	}
	largeData := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec := NewECOClassifier()
		ec.LoadFromReader(strings.NewReader(largeData))
	}
}

// Benchmark AddECOTags
func BenchmarkECOClassifier_AddECOTags(b *testing.B) {
	ec := getECOClassifier()
	game := parseGame(ruyLopezGame)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset game tags
		delete(game.Tags, "ECO")
		delete(game.Tags, "Opening")
		delete(game.Tags, "Variation")
		ec.AddECOTags(game)
	}
}

// Benchmark hash table lookup
func BenchmarkECOClassifier_findMatch(b *testing.B) {
	ec := getECOClassifier()

	// Use a known hash that exists in the table
	// This is approximate - in real code you'd use actual hash values
	var posHash uint64 = 0x12345678
	var cumHash uint64 = 0x87654321
	halfMoves := 6

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec.findMatch(posHash, cumHash, halfMoves)
	}
}
