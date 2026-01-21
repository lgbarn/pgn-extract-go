package engine

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// Benchmark FEN positions
var (
	benchInitialFEN   = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	benchMidgameFEN   = "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4"
	benchEndgameFEN   = "8/5k2/8/8/8/8/5K2/4R3 w - - 0 1"
	benchComplexFEN   = "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	benchEnPassantFEN = "rnbqkbnr/pppp1ppp/8/4pP2/8/8/PPPPP1PP/RNBQKBNR w KQkq e6 0 3"
	benchCastlingFEN  = "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1"
)

// FEN parsing benchmarks
func BenchmarkNewBoardFromFEN_Initial(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewBoardFromFEN(benchInitialFEN)
	}
}

func BenchmarkNewBoardFromFEN_Midgame(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewBoardFromFEN(benchMidgameFEN)
	}
}

func BenchmarkNewBoardFromFEN_Complex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewBoardFromFEN(benchComplexFEN)
	}
}

func BenchmarkBoardToFEN_Initial(b *testing.B) {
	board, _ := NewBoardFromFEN(benchInitialFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BoardToFEN(board)
	}
}

func BenchmarkBoardToFEN_Midgame(b *testing.B) {
	board, _ := NewBoardFromFEN(benchMidgameFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BoardToFEN(board)
	}
}

func BenchmarkFEN_RoundTrip(b *testing.B) {
	for i := 0; i < b.N; i++ {
		board, _ := NewBoardFromFEN(benchMidgameFEN)
		BoardToFEN(board)
	}
}

// Move application benchmarks
func BenchmarkApplyMove_PawnMove(b *testing.B) {
	move := &chess.Move{
		Text:        "e4",
		FromCol:     'e',
		FromRank:    '2',
		ToCol:       'e',
		ToRank:      '4',
		PieceToMove: chess.Pawn,
		Class:       chess.PawnMove,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		board, _ := NewBoardFromFEN(benchInitialFEN)
		b.StartTimer()
		ApplyMove(board, move)
	}
}

func BenchmarkApplyMove_PieceMove(b *testing.B) {
	move := &chess.Move{
		Text:        "Nf3",
		FromCol:     'g',
		FromRank:    '1',
		ToCol:       'f',
		ToRank:      '3',
		PieceToMove: chess.Knight,
		Class:       chess.PieceMove,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		board, _ := NewBoardFromFEN(benchInitialFEN)
		b.StartTimer()
		ApplyMove(board, move)
	}
}

func BenchmarkApplyMove_KingsideCastle(b *testing.B) {
	move := &chess.Move{
		Text:  "O-O",
		Class: chess.KingsideCastle,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		board, _ := NewBoardFromFEN(benchCastlingFEN)
		b.StartTimer()
		ApplyMove(board, move)
	}
}

func BenchmarkApplyMove_QueensideCastle(b *testing.B) {
	move := &chess.Move{
		Text:  "O-O-O",
		Class: chess.QueensideCastle,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		board, _ := NewBoardFromFEN(benchCastlingFEN)
		b.StartTimer()
		ApplyMove(board, move)
	}
}

func BenchmarkApplyMove_EnPassant(b *testing.B) {
	move := &chess.Move{
		Text:        "fxe6",
		FromCol:     'f',
		FromRank:    '5',
		ToCol:       'e',
		ToRank:      '6',
		PieceToMove: chess.Pawn,
		Class:       chess.EnPassantPawnMove,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		board, _ := NewBoardFromFEN(benchEnPassantFEN)
		b.StartTimer()
		ApplyMove(board, move)
	}
}

func BenchmarkApplyMove_Promotion(b *testing.B) {
	// Position with pawn on 7th rank ready to promote
	promotionFEN := "8/P7/8/8/8/8/8/4K2k w - - 0 1"
	move := &chess.Move{
		Text:          "a8=Q",
		FromCol:       'a',
		FromRank:      '7',
		ToCol:         'a',
		ToRank:        '8',
		PieceToMove:   chess.Pawn,
		PromotedPiece: chess.Queen,
		Class:         chess.PawnMoveWithPromotion,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		board, _ := NewBoardFromFEN(promotionFEN)
		b.StartTimer()
		ApplyMove(board, move)
	}
}

// Game replay benchmark - simulates processing a full game
func BenchmarkGameReplay_ItalianOpening(b *testing.B) {
	// Italian Game: 1.e4 e5 2.Nf3 Nc6 3.Bc4 Bc5
	moves := []*chess.Move{
		{Text: "e4", FromCol: 'e', FromRank: '2', ToCol: 'e', ToRank: '4', PieceToMove: chess.Pawn, Class: chess.PawnMove},
		{Text: "e5", FromCol: 'e', FromRank: '7', ToCol: 'e', ToRank: '5', PieceToMove: chess.Pawn, Class: chess.PawnMove},
		{Text: "Nf3", FromCol: 'g', FromRank: '1', ToCol: 'f', ToRank: '3', PieceToMove: chess.Knight, Class: chess.PieceMove},
		{Text: "Nc6", FromCol: 'b', FromRank: '8', ToCol: 'c', ToRank: '6', PieceToMove: chess.Knight, Class: chess.PieceMove},
		{Text: "Bc4", FromCol: 'f', FromRank: '1', ToCol: 'c', ToRank: '4', PieceToMove: chess.Bishop, Class: chess.PieceMove},
		{Text: "Bc5", FromCol: 'f', FromRank: '8', ToCol: 'c', ToRank: '5', PieceToMove: chess.Bishop, Class: chess.PieceMove},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board, _ := NewBoardFromFEN(benchInitialFEN)
		for _, move := range moves {
			ApplyMove(board, move)
		}
	}
}

// Check detection benchmarks
func BenchmarkIsInCheck_NoCheck(b *testing.B) {
	board, _ := NewBoardFromFEN(benchInitialFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsInCheck(board, chess.White)
	}
}

func BenchmarkIsInCheck_InCheck(b *testing.B) {
	// Position where white king is in check
	checkFEN := "rnb1kbnr/pppp1ppp/8/4p3/7q/5P2/PPPPP1PP/RNBQKBNR w KQkq - 1 3"
	board, _ := NewBoardFromFEN(checkFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsInCheck(board, chess.White)
	}
}

// Legal moves benchmarks
func BenchmarkHasLegalMoves_Initial(b *testing.B) {
	board, _ := NewBoardFromFEN(benchInitialFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HasLegalMoves(board, chess.White)
	}
}

func BenchmarkHasLegalMoves_Midgame(b *testing.B) {
	board, _ := NewBoardFromFEN(benchMidgameFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HasLegalMoves(board, chess.White)
	}
}

func BenchmarkHasLegalMoves_Endgame(b *testing.B) {
	board, _ := NewBoardFromFEN(benchEndgameFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HasLegalMoves(board, chess.White)
	}
}

// Board copy benchmark
func BenchmarkBoardCopy(b *testing.B) {
	board, _ := NewBoardFromFEN(benchMidgameFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.Copy()
	}
}
