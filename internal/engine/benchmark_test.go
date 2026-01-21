package engine

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

var benchFENs = map[string]string{
	"Initial":   "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
	"Midgame":   "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
	"Endgame":   "8/5k2/8/8/8/8/5K2/4R3 w - - 0 1",
	"Complex":   "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
	"EnPassant": "rnbqkbnr/pppp1ppp/8/4pP2/8/8/PPPPP1PP/RNBQKBNR w KQkq e6 0 3",
	"Castling":  "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
}

func BenchmarkNewBoardFromFEN(b *testing.B) {
	for name, fen := range benchFENs {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				NewBoardFromFEN(fen)
			}
		})
	}
}

func BenchmarkBoardToFEN(b *testing.B) {
	for name, fen := range benchFENs {
		b.Run(name, func(b *testing.B) {
			board, _ := NewBoardFromFEN(fen)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				BoardToFEN(board)
			}
		})
	}
}

func BenchmarkFEN_RoundTrip(b *testing.B) {
	fen := benchFENs["Midgame"]
	for i := 0; i < b.N; i++ {
		board, _ := NewBoardFromFEN(fen)
		BoardToFEN(board)
	}
}

func BenchmarkApplyMove(b *testing.B) {
	cases := []struct {
		name string
		fen  string
		move *chess.Move
	}{
		{
			name: "PawnMove",
			fen:  benchFENs["Initial"],
			move: &chess.Move{Text: "e4", FromCol: 'e', FromRank: '2', ToCol: 'e', ToRank: '4', PieceToMove: chess.Pawn, Class: chess.PawnMove},
		},
		{
			name: "PieceMove",
			fen:  benchFENs["Initial"],
			move: &chess.Move{Text: "Nf3", FromCol: 'g', FromRank: '1', ToCol: 'f', ToRank: '3', PieceToMove: chess.Knight, Class: chess.PieceMove},
		},
		{
			name: "KingsideCastle",
			fen:  benchFENs["Castling"],
			move: &chess.Move{Text: "O-O", Class: chess.KingsideCastle},
		},
		{
			name: "QueensideCastle",
			fen:  benchFENs["Castling"],
			move: &chess.Move{Text: "O-O-O", Class: chess.QueensideCastle},
		},
		{
			name: "EnPassant",
			fen:  benchFENs["EnPassant"],
			move: &chess.Move{Text: "fxe6", FromCol: 'f', FromRank: '5', ToCol: 'e', ToRank: '6', PieceToMove: chess.Pawn, Class: chess.EnPassantPawnMove},
		},
		{
			name: "Promotion",
			fen:  "8/P7/8/8/8/8/8/4K2k w - - 0 1",
			move: &chess.Move{Text: "a8=Q", FromCol: 'a', FromRank: '7', ToCol: 'a', ToRank: '8', PieceToMove: chess.Pawn, PromotedPiece: chess.Queen, Class: chess.PawnMoveWithPromotion},
		},
	}

	for _, tt := range cases {
		b.Run(tt.name, func(b *testing.B) {
			board, _ := NewBoardFromFEN(tt.fen)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				boardCopy := board.Copy()
				ApplyMove(boardCopy, tt.move)
			}
		})
	}
}

func BenchmarkGameReplay_ItalianOpening(b *testing.B) {
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
		board, _ := NewBoardFromFEN(benchFENs["Initial"])
		for _, move := range moves {
			ApplyMove(board, move)
		}
	}
}

func BenchmarkIsInCheck(b *testing.B) {
	checkFEN := "rnb1kbnr/pppp1ppp/8/4p3/7q/5P2/PPPPP1PP/RNBQKBNR w KQkq - 1 3"

	b.Run("NoCheck", func(b *testing.B) {
		board, _ := NewBoardFromFEN(benchFENs["Initial"])
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			IsInCheck(board, chess.White)
		}
	})

	b.Run("InCheck", func(b *testing.B) {
		board, _ := NewBoardFromFEN(checkFEN)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			IsInCheck(board, chess.White)
		}
	})
}

func BenchmarkHasLegalMoves(b *testing.B) {
	positions := []string{"Initial", "Midgame", "Endgame"}
	for _, name := range positions {
		b.Run(name, func(b *testing.B) {
			board, _ := NewBoardFromFEN(benchFENs[name])
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				HasLegalMoves(board, chess.White)
			}
		})
	}
}

func BenchmarkBoardCopy(b *testing.B) {
	board, _ := NewBoardFromFEN(benchFENs["Midgame"])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.Copy()
	}
}
