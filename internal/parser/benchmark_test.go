package parser

import (
	"strings"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/config"
)

// Sample PGN data for benchmarks
const (
	simplePGN = `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "?"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. c3 Nf6 5. d4 exd4 6. cxd4 Bb4+ 7. Nc3 Nxe4
8. O-O Nxc3 9. bxc3 Bxc3 10. Qb3 Bxa1 11. Bxf7+ Kf8 12. Bg5 Ne7 13. Ne5 Bxd4
14. Bg6 d5 15. Qf3+ Bf5 16. Bxf5 Bxe5 17. Be6+ Bf6 18. Bxf6 gxf6 19. Qxf6+ Ke8
20. Qf7# 1-0
`

	shortPGN = `[Event "Test"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 Nc6 *
`

	annotatedPGN = `[Event "Annotated Game"]
[Site "Test"]
[Date "2024.01.01"]
[White "Fischer"]
[Black "Spassky"]
[Result "1-0"]

1. e4 {Best by test} e5 2. Nf3 Nc6 3. Bb5 {The Ruy Lopez} a6 4. Ba4 Nf6
5. O-O Be7 6. Re1 b5 7. Bb3 d6 8. c3 O-O 9. h3 Nb8!? {A prophylactic retreat}
10. d4 Nbd7 11. Nbd2 Bb7 12. Bc2 Re8 13. Nf1 Bf8 14. Ng3 g6 15. Bg5 h6
16. Bd2 Bg7 17. a4 c5 18. d5 c4 19. b4 Nh7 20. Be3 h5 1-0
`

	variationsPGN = `[Event "With Variations"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 (1. d4 d5 2. c4 {Queen's Gambit}) 1... e5 (1... c5 {Sicilian}) 2. Nf3
(2. Nc3 {Vienna Game}) 2... Nc6 3. Bb5 {Ruy Lopez} *
`

	multiplePGN = `[Event "Game 1"]
[White "A"]
[Black "B"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 1-0

[Event "Game 2"]
[White "C"]
[Black "D"]
[Result "0-1"]

1. d4 d5 2. c4 e6 0-1

[Event "Game 3"]
[White "E"]
[Black "F"]
[Result "1/2-1/2"]

1. c4 c5 2. Nf3 Nc6 1/2-1/2
`
)

func newSilentConfig() *config.Config {
	cfg := config.NewConfig()
	cfg.Verbosity = 0
	return cfg
}

func BenchmarkParser_ParseGame(b *testing.B) {
	cases := map[string]string{
		"Simple":         simplePGN,
		"Short":          shortPGN,
		"Annotated":      annotatedPGN,
		"WithVariations": variationsPGN,
	}

	cfg := newSilentConfig()
	for name, pgn := range cases {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				p := NewParser(strings.NewReader(pgn), cfg)
				p.ParseGame()
			}
		})
	}
}

func BenchmarkParser_ParseAllGames(b *testing.B) {
	cfg := newSilentConfig()
	for i := 0; i < b.N; i++ {
		p := NewParser(strings.NewReader(multiplePGN), cfg)
		p.ParseAllGames()
	}
}

func BenchmarkLexer_NextToken(b *testing.B) {
	cases := map[string]string{
		"Simple":    simplePGN,
		"Annotated": annotatedPGN,
	}

	cfg := newSilentConfig()
	for name, pgn := range cases {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				lexer := NewLexer(strings.NewReader(pgn), cfg)
				b.StartTimer()

				for {
					tok := lexer.NextToken()
					if tok.Type == EOFToken {
						break
					}
				}
			}
		})
	}
}

func BenchmarkDecodeMove(b *testing.B) {
	moves := map[string]string{
		"Pawn":           "e4",
		"Piece":          "Nf3",
		"Capture":        "Bxf7",
		"Promotion":      "e8=Q",
		"Castle":         "O-O",
		"FullyQualified": "Qd1d4",
		"WithCheck":      "Qf7+",
		"WithMate":       "Qf7#",
	}

	for name, move := range moves {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				DecodeMove(move)
			}
		})
	}
}

func BenchmarkParser_LargeInput(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString(simplePGN)
		sb.WriteString("\n")
	}
	largePGN := sb.String()
	cfg := newSilentConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewParser(strings.NewReader(largePGN), cfg)
		p.ParseAllGames()
	}
}
