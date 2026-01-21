package hashing

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

// Benchmark positions
var (
	initialFEN   = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	midgameFEN   = "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4"
	endgameFEN   = "8/5k2/8/8/8/8/5K2/4R3 w - - 0 1"
	complexFEN   = "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	enPassantFEN = "rnbqkbnr/pppp1ppp/8/4pP2/8/8/PPPPP1PP/RNBQKBNR w KQkq e6 0 3"
)

func BenchmarkGenerateZobristHash_Initial(b *testing.B) {
	board, _ := engine.NewBoardFromFEN(initialFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateZobristHash(board)
	}
}

func BenchmarkGenerateZobristHash_Midgame(b *testing.B) {
	board, _ := engine.NewBoardFromFEN(midgameFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateZobristHash(board)
	}
}

func BenchmarkGenerateZobristHash_Endgame(b *testing.B) {
	board, _ := engine.NewBoardFromFEN(endgameFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateZobristHash(board)
	}
}

func BenchmarkGenerateZobristHash_Complex(b *testing.B) {
	board, _ := engine.NewBoardFromFEN(complexFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateZobristHash(board)
	}
}

func BenchmarkGenerateZobristHash_EnPassant(b *testing.B) {
	board, _ := engine.NewBoardFromFEN(enPassantFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateZobristHash(board)
	}
}

func BenchmarkWeakHash_Initial(b *testing.B) {
	board, _ := engine.NewBoardFromFEN(initialFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WeakHash(board)
	}
}

func BenchmarkWeakHash_Midgame(b *testing.B) {
	board, _ := engine.NewBoardFromFEN(midgameFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WeakHash(board)
	}
}

func BenchmarkWeakHash_Endgame(b *testing.B) {
	board, _ := engine.NewBoardFromFEN(endgameFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WeakHash(board)
	}
}

func BenchmarkDuplicateDetector_CheckAndAdd_Unique(b *testing.B) {
	dd := NewDuplicateDetector(false)
	games := make([]*chess.Game, 100)
	boards := make([]*chess.Board, 100)
	for i := range boards {
		boards[i], _ = engine.NewBoardFromFEN(initialFEN)
		// Make each board unique by moving a piece
		boards[i].Set(chess.Col('a'+i%8), chess.Rank('2'), chess.Empty)
		games[i] = &chess.Game{Tags: make(map[string]string)}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dd.CheckAndAdd(games[i%100], boards[i%100])
	}
}

func BenchmarkDuplicateDetector_CheckAndAdd_Duplicates(b *testing.B) {
	dd := NewDuplicateDetector(false)
	board, _ := engine.NewBoardFromFEN(initialFEN)
	game := &chess.Game{Tags: make(map[string]string)}

	// Pre-add the game
	dd.CheckAndAdd(game, board)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dd.CheckAndAdd(game, board)
	}
}

func BenchmarkGameHasher_HashGame(b *testing.B) {
	hasher := NewGameHasher(HashFinalPosition)

	// Create a simple test game
	board, _ := engine.NewBoardFromFEN(initialFEN)
	game := &chess.Game{
		Tags:  make(map[string]string),
		Moves: createTestMoves(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasher.HashGame(game, board)
	}
}

// createTestMoves creates a simple sequence of moves for benchmarking
func createTestMoves() *chess.Move {
	// e4 e5 Nf3 Nc6 (Italian Game opening)
	moves := []*chess.Move{
		{Text: "e4", FromCol: 'e', FromRank: '2', ToCol: 'e', ToRank: '4', PieceToMove: chess.Pawn, Class: chess.PawnMove},
		{Text: "e5", FromCol: 'e', FromRank: '7', ToCol: 'e', ToRank: '5', PieceToMove: chess.Pawn, Class: chess.PawnMove},
		{Text: "Nf3", FromCol: 'g', FromRank: '1', ToCol: 'f', ToRank: '3', PieceToMove: chess.Knight, Class: chess.PieceMove},
		{Text: "Nc6", FromCol: 'b', FromRank: '8', ToCol: 'c', ToRank: '6', PieceToMove: chess.Knight, Class: chess.PieceMove},
	}

	// Link the moves
	for i := 0; i < len(moves)-1; i++ {
		moves[i].Next = moves[i+1]
	}

	return moves[0]
}
