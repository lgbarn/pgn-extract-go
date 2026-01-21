package matching

import (
	"strings"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/parser"
)

// Test PGN data
const testPGN = `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[White "Fischer, Bobby"]
[Black "Spassky, Boris"]
[Result "1-0"]
[ECO "C92"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6
8. c3 O-O 9. h3 Nb8 10. d4 Nbd7 1-0
`

func getTestGame() *chess.Game {
	cfg := config.NewConfig()
	cfg.Verbosity = 0
	p := parser.NewParser(strings.NewReader(testPGN), cfg)
	game, _ := p.ParseGame()
	return game
}

// Position Matcher Benchmarks
func BenchmarkPositionMatcher_MatchGame_NoPatterns(b *testing.B) {
	pm := NewPositionMatcher()
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.MatchGame(game)
	}
}

func BenchmarkPositionMatcher_MatchGame_SingleFEN(b *testing.B) {
	pm := NewPositionMatcher()
	// Ruy Lopez position after 3...a6 4.Ba4
	pm.AddFEN("r1bqkbnr/1ppp1ppp/p1n5/4p3/B3P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 1 4", "Ruy Lopez")
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.MatchGame(game)
	}
}

func BenchmarkPositionMatcher_MatchGame_MultiplePatterns(b *testing.B) {
	pm := NewPositionMatcher()
	// Add several patterns
	pm.AddFEN("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1", "1.e4")
	pm.AddFEN("rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 2", "1...e5")
	pm.AddFEN("rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2", "2.Nf3")
	pm.AddFEN("r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3", "2...Nc6")
	pm.AddFEN("r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3", "Ruy Lopez")
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.MatchGame(game)
	}
}

func BenchmarkPositionMatcher_MatchGame_WildcardPattern(b *testing.B) {
	pm := NewPositionMatcher()
	// Pattern with wildcards
	pm.AddPattern("r?bqkbnr/pppp?ppp/??n?????/????p???/????P???/?????N??/PPPP?PPP/RNBQKB?R", "Open Game", false)
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.MatchGame(game)
	}
}

// Tag Matcher Benchmarks
func BenchmarkTagMatcher_MatchGame_NoTags(b *testing.B) {
	tm := NewTagMatcher()
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.MatchGame(game)
	}
}

func BenchmarkTagMatcher_MatchGame_SingleTag(b *testing.B) {
	tm := NewTagMatcher()
	tm.AddCriterion("White", "Fischer", OpContains)
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.MatchGame(game)
	}
}

func BenchmarkTagMatcher_MatchGame_MultipleTags(b *testing.B) {
	tm := NewTagMatcher()
	tm.AddCriterion("White", "Fischer", OpContains)
	tm.AddCriterion("Black", "Spassky", OpContains)
	tm.AddCriterion("Result", "1-0", OpEqual)
	tm.AddCriterion("ECO", "C", OpContains)
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.MatchGame(game)
	}
}

func BenchmarkTagMatcher_MatchGame_DateComparison(b *testing.B) {
	tm := NewTagMatcher()
	tm.AddCriterion("Date", "2023.01.01", OpGreaterThan)
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.MatchGame(game)
	}
}

func BenchmarkTagMatcher_MatchGame_PlayerCriterion(b *testing.B) {
	tm := NewTagMatcher()
	tm.AddPlayerCriterion("Fischer")
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.MatchGame(game)
	}
}

// Material Matcher Benchmarks
func BenchmarkMaterialMatcher_MatchGame_ExactMatch(b *testing.B) {
	mm := NewMaterialMatcher("KQRRBBNN/kqrrbbnn", true) // Starting position material
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mm.MatchGame(game)
	}
}

func BenchmarkMaterialMatcher_MatchGame_MinimalMatch(b *testing.B) {
	mm := NewMaterialMatcher("KR/kr", false) // At least one rook each
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mm.MatchGame(game)
	}
}

// Game Filter (combined) Benchmarks
func BenchmarkGameFilter_Match_AllFilters(b *testing.B) {
	gf := NewGameFilter()
	gf.AddWhiteFilter("Fischer")
	gf.AddECOFilter("C")
	gf.PositionMatcher.AddFEN("r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3", "Ruy Lopez")
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gf.Match(game)
	}
}

// Variation Matcher Benchmarks
func BenchmarkVariationMatcher_MatchGame_Simple(b *testing.B) {
	vm := NewVariationMatcher()
	vm.AddMoveSequence([]string{"e4", "e5", "Nf3", "Nc6"})
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.MatchGame(game)
	}
}

func BenchmarkVariationMatcher_MatchGame_LongSequence(b *testing.B) {
	vm := NewVariationMatcher()
	vm.AddMoveSequence([]string{"e4", "e5", "Nf3", "Nc6", "Bb5", "a6", "Ba4", "Nf6", "O-O", "Be7"})
	game := getTestGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.MatchGame(game)
	}
}
