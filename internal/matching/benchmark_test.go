package matching

import (
	"strings"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/parser"
)

const benchPGN = `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[White "Fischer, Bobby"]
[Black "Spassky, Boris"]
[Result "1-0"]
[ECO "C92"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6
8. c3 O-O 9. h3 Nb8 10. d4 Nbd7 1-0
`

func getBenchGame() *chess.Game {
	cfg := config.NewConfig()
	cfg.Verbosity = 0
	p := parser.NewParser(strings.NewReader(benchPGN), cfg)
	game, _ := p.ParseGame()
	return game
}

func BenchmarkPositionMatcher_MatchGame(b *testing.B) {
	game := getBenchGame()

	b.Run("NoPatterns", func(b *testing.B) {
		pm := NewPositionMatcher()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pm.MatchGame(game)
		}
	})

	b.Run("SingleFEN", func(b *testing.B) {
		pm := NewPositionMatcher()
		pm.AddFEN("r1bqkbnr/1ppp1ppp/p1n5/4p3/B3P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 1 4", "Ruy Lopez")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pm.MatchGame(game)
		}
	})

	b.Run("MultiplePatterns", func(b *testing.B) {
		pm := NewPositionMatcher()
		pm.AddFEN("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1", "1.e4")
		pm.AddFEN("rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 2", "1...e5")
		pm.AddFEN("rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2", "2.Nf3")
		pm.AddFEN("r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3", "2...Nc6")
		pm.AddFEN("r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3", "Ruy Lopez")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pm.MatchGame(game)
		}
	})

	b.Run("WildcardPattern", func(b *testing.B) {
		pm := NewPositionMatcher()
		pm.AddPattern("r?bqkbnr/pppp?ppp/??n?????/????p???/????P???/?????N??/PPPP?PPP/RNBQKB?R", "Open Game", false)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pm.MatchGame(game)
		}
	})
}

func BenchmarkTagMatcher_MatchGame(b *testing.B) {
	game := getBenchGame()

	b.Run("NoTags", func(b *testing.B) {
		tm := NewTagMatcher()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tm.MatchGame(game)
		}
	})

	b.Run("SingleTag", func(b *testing.B) {
		tm := NewTagMatcher()
		tm.AddCriterion("White", "Fischer", OpContains)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tm.MatchGame(game)
		}
	})

	b.Run("MultipleTags", func(b *testing.B) {
		tm := NewTagMatcher()
		tm.AddCriterion("White", "Fischer", OpContains)
		tm.AddCriterion("Black", "Spassky", OpContains)
		tm.AddCriterion("Result", "1-0", OpEqual)
		tm.AddCriterion("ECO", "C", OpContains)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tm.MatchGame(game)
		}
	})

	b.Run("DateComparison", func(b *testing.B) {
		tm := NewTagMatcher()
		tm.AddCriterion("Date", "2023.01.01", OpGreaterThan)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tm.MatchGame(game)
		}
	})

	b.Run("PlayerCriterion", func(b *testing.B) {
		tm := NewTagMatcher()
		tm.AddPlayerCriterion("Fischer")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tm.MatchGame(game)
		}
	})
}

func BenchmarkMaterialMatcher_MatchGame(b *testing.B) {
	game := getBenchGame()

	b.Run("ExactMatch", func(b *testing.B) {
		mm := NewMaterialMatcher("KQRRBBNN/kqrrbbnn", true)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mm.MatchGame(game)
		}
	})

	b.Run("MinimalMatch", func(b *testing.B) {
		mm := NewMaterialMatcher("KR/kr", false)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mm.MatchGame(game)
		}
	})
}

func BenchmarkGameFilter_Match(b *testing.B) {
	gf := NewGameFilter()
	gf.AddWhiteFilter("Fischer")
	gf.AddECOFilter("C")
	gf.PositionMatcher.AddFEN("r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3", "Ruy Lopez")
	game := getBenchGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gf.Match(game)
	}
}

func BenchmarkVariationMatcher_MatchGame(b *testing.B) {
	game := getBenchGame()

	b.Run("Simple", func(b *testing.B) {
		vm := NewVariationMatcher()
		vm.AddMoveSequence([]string{"e4", "e5", "Nf3", "Nc6"})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vm.MatchGame(game)
		}
	})

	b.Run("LongSequence", func(b *testing.B) {
		vm := NewVariationMatcher()
		vm.AddMoveSequence([]string{"e4", "e5", "Nf3", "Nc6", "Bb5", "a6", "Ba4", "Nf6", "O-O", "Be7"})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vm.MatchGame(game)
		}
	})
}
