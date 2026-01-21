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

var benchGames = map[string]string{
	"RuyLopez": `[Event "Test"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6
8. c3 O-O 9. h3 Nb8 10. d4 Nbd7 *
`,
	"Sicilian": `[Event "Test"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 *
`,
	"French": `[Event "Test"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e6 2. d4 d5 3. Nc3 Bb4 4. e5 c5 5. a3 Bxc3+ 6. bxc3 *
`,
	"NoMatch": `[Event "Test"]
[White "A"]
[Black "B"]
[Result "*"]

1. a3 a6 2. b3 b6 3. c3 c6 4. d3 d6 *
`,
}

func newECOClassifier() *ECOClassifier {
	ec := NewECOClassifier()
	ec.LoadFromReader(strings.NewReader(sampleECOData))
	return ec
}

func parseBenchGame(pgn string) *chess.Game {
	cfg := config.NewConfig()
	cfg.Verbosity = 0
	p := parser.NewParser(strings.NewReader(pgn), cfg)
	game, _ := p.ParseGame()
	return game
}

func BenchmarkECOClassifier_ClassifyGame(b *testing.B) {
	ec := newECOClassifier()

	for name, pgn := range benchGames {
		game := parseBenchGame(pgn)
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ec.ClassifyGame(game)
			}
		})
	}
}

func BenchmarkECOClassifier_LoadFromReader(b *testing.B) {
	b.Run("Standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ec := NewECOClassifier()
			ec.LoadFromReader(strings.NewReader(sampleECOData))
		}
	})

	b.Run("Large", func(b *testing.B) {
		var sb strings.Builder
		for _, code := range []string{"A", "B", "C", "D", "E"} {
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
	})
}

func BenchmarkECOClassifier_AddECOTags(b *testing.B) {
	ec := newECOClassifier()
	game := parseBenchGame(benchGames["RuyLopez"])

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		delete(game.Tags, "ECO")
		delete(game.Tags, "Opening")
		delete(game.Tags, "Variation")
		ec.AddECOTags(game)
	}
}

func BenchmarkECOClassifier_findMatch(b *testing.B) {
	ec := newECOClassifier()
	var posHash uint64 = 0x12345678
	var cumHash uint64 = 0x87654321
	halfMoves := 6

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec.findMatch(posHash, cumHash, halfMoves)
	}
}
