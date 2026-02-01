package hashing

import (
	"fmt"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

var benchFENPositions = map[string]string{
	"Initial":   "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
	"Midgame":   "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
	"Endgame":   "8/5k2/8/8/8/8/5K2/4R3 w - - 0 1",
	"Complex":   "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
	"EnPassant": "rnbqkbnr/pppp1ppp/8/4pP2/8/8/PPPPP1PP/RNBQKBNR w KQkq e6 0 3",
}

func BenchmarkGenerateZobristHash(b *testing.B) {
	for name, fen := range benchFENPositions {
		b.Run(name, func(b *testing.B) {
			board, _ := engine.NewBoardFromFEN(fen)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				GenerateZobristHash(board)
			}
		})
	}
}

func BenchmarkWeakHash(b *testing.B) {
	positions := []string{"Initial", "Midgame", "Endgame"}
	for _, name := range positions {
		b.Run(name, func(b *testing.B) {
			board, _ := engine.NewBoardFromFEN(benchFENPositions[name])
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				WeakHash(board)
			}
		})
	}
}

func BenchmarkDuplicateDetector_CheckAndAdd(b *testing.B) {
	initialFEN := benchFENPositions["Initial"]

	b.Run("Unique", func(b *testing.B) {
		dd := NewDuplicateDetector(false, 0)
		games := make([]*chess.Game, 100)
		boards := make([]*chess.Board, 100)
		for i := range boards {
			boards[i], _ = engine.NewBoardFromFEN(initialFEN)
			boards[i].Set(chess.Col('a'+i%8), chess.Rank('2'), chess.Empty)
			games[i] = &chess.Game{Tags: make(map[string]string)}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dd.CheckAndAdd(games[i%100], boards[i%100])
		}
	})

	b.Run("Duplicates", func(b *testing.B) {
		dd := NewDuplicateDetector(false, 0)
		board, _ := engine.NewBoardFromFEN(initialFEN)
		game := &chess.Game{Tags: make(map[string]string)}
		dd.CheckAndAdd(game, board)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dd.CheckAndAdd(game, board)
		}
	})
}

func BenchmarkGameHasher_HashGame(b *testing.B) {
	hasher := NewGameHasher(HashFinalPosition)
	board, _ := engine.NewBoardFromFEN(benchFENPositions["Initial"])
	game := &chess.Game{
		Tags:  make(map[string]string),
		Moves: createLinkedMoves(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasher.HashGame(game, board)
	}
}

func createLinkedMoves() *chess.Move {
	moves := []*chess.Move{
		{Text: "e4", FromCol: 'e', FromRank: '2', ToCol: 'e', ToRank: '4', PieceToMove: chess.Pawn, Class: chess.PawnMove},
		{Text: "e5", FromCol: 'e', FromRank: '7', ToCol: 'e', ToRank: '5', PieceToMove: chess.Pawn, Class: chess.PawnMove},
		{Text: "Nf3", FromCol: 'g', FromRank: '1', ToCol: 'f', ToRank: '3', PieceToMove: chess.Knight, Class: chess.PieceMove},
		{Text: "Nc6", FromCol: 'b', FromRank: '8', ToCol: 'c', ToRank: '6', PieceToMove: chess.Knight, Class: chess.PieceMove},
	}
	for i := 0; i < len(moves)-1; i++ {
		moves[i].Next = moves[i+1]
	}
	return moves[0]
}

// createUniqueGame creates a unique game by modifying board positions.
// Using index to create different positions by clearing and moving pieces.
func createUniqueGame(index int, initialFEN string) (*chess.Game, *chess.Board) {
	board, _ := engine.NewBoardFromFEN(initialFEN)
	game := &chess.Game{Tags: make(map[string]string)}

	// Create unique positions by modifying multiple squares
	// Use different patterns based on index to maximize variation

	// Pattern 1: Clear pieces from different columns
	col1 := chess.Col('a' + index%8)
	board.Set(col1, '2', chess.Empty)

	// Pattern 2: Clear pieces from different ranks
	rank1 := chess.Rank('1' + (index/8)%8)
	col2 := chess.Col('a' + (index/64)%8)
	board.Set(col2, rank1, chess.Empty)

	// Pattern 3: Move pieces to create more variation
	if index%3 == 0 {
		col3 := chess.Col('a' + (index/512)%8)
		board.Set(col3, '7', chess.Empty)
	}

	// Pattern 4: Additional variation for high indices
	if index >= 64 {
		col4 := chess.Col('a' + (index/4096)%8)
		rank2 := chess.Rank('8' - (index/32768)%8)
		board.Set(col4, rank2, chess.Empty)
	}

	return game, board
}

func BenchmarkDuplicateDetector_BoundedMemory(b *testing.B) {
	initialFEN := benchFENPositions["Initial"]

	b.Run("Bounded", func(b *testing.B) {
		const capacity = 1000
		dd := NewDuplicateDetector(false, capacity)

		// Pre-create 100K unique games for testing
		const numGames = 100000
		games := make([]*chess.Game, numGames)
		boards := make([]*chess.Board, numGames)

		for i := 0; i < numGames; i++ {
			games[i], boards[i] = createUniqueGame(i, initialFEN)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Process all games
			for j := 0; j < numGames; j++ {
				dd.CheckAndAdd(games[j], boards[j])
			}

			// Verify IsFull() returns true - we added enough games to exceed capacity
			if !dd.IsFull() {
				b.Errorf("Detector should be full after processing %d games with capacity %d", numGames, capacity)
			}

			// Note: UniqueCount() can exceed capacity due to hash collisions.
			// The capacity limits the number of hash buckets (len(hashTable)), not
			// the total number of signatures stored. Colliding games are still added
			// to existing buckets.

			// Reset for next iteration
			dd.Reset()
		}
	})

	b.Run("Unlimited", func(b *testing.B) {
		dd := NewDuplicateDetector(false, 0)

		// Use smaller dataset for unlimited to keep benchmark fast
		const numGames = 1000
		games := make([]*chess.Game, numGames)
		boards := make([]*chess.Board, numGames)

		for i := 0; i < numGames; i++ {
			games[i], boards[i] = createUniqueGame(i, initialFEN)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := 0; j < numGames; j++ {
				dd.CheckAndAdd(games[j], boards[j])
			}

			// Verify unlimited behavior
			if dd.IsFull() {
				b.Error("Unlimited detector should never be full")
			}

			dd.Reset()
		}
	})
}

func BenchmarkDuplicateDetector_BoundedVsUnlimited(b *testing.B) {
	initialFEN := benchFENPositions["Initial"]
	capacities := []int{0, 100, 1000, 5000}

	const numGames = 10000
	games := make([]*chess.Game, numGames)
	boards := make([]*chess.Board, numGames)

	// Pre-create games
	for i := 0; i < numGames; i++ {
		games[i], boards[i] = createUniqueGame(i, initialFEN)
	}

	for _, capacity := range capacities {
		name := "Unlimited"
		if capacity > 0 {
			name = fmt.Sprintf("Capacity%d", capacity)
		}

		b.Run(name, func(b *testing.B) {
			dd := NewDuplicateDetector(false, capacity)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for j := 0; j < numGames; j++ {
					dd.CheckAndAdd(games[j], boards[j])
				}

				// Report metrics
				uniqueGames := dd.UniqueCount()
				b.ReportMetric(float64(uniqueGames), "unique_games")

				dd.Reset()
			}
		})
	}
}
