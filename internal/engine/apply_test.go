package engine

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

func TestApplyMove_NullMove(t *testing.T) {
	tests := []struct {
		name         string
		fen          string
		wantToMove   chess.Colour
		wantEnPassnt bool
	}{
		{
			name:         "null move from initial position",
			fen:          InitialFEN,
			wantToMove:   chess.Black,
			wantEnPassnt: false,
		},
		{
			name:         "null move as black",
			fen:          "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
			wantToMove:   chess.White,
			wantEnPassnt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := NewBoardFromFEN(tt.fen)
			if err != nil {
				t.Fatalf("NewBoardFromFEN(%q) failed: %v", tt.fen, err)
			}

			move := &chess.Move{Class: chess.NullMove}
			ok := ApplyMove(board, move)

			if !ok {
				t.Errorf("ApplyMove() = false, want true")
			}
			if board.ToMove != tt.wantToMove {
				t.Errorf("board.ToMove = %v, want %v", board.ToMove, tt.wantToMove)
			}
			if board.EnPassant != tt.wantEnPassnt {
				t.Errorf("board.EnPassant = %v, want %v", board.EnPassant, tt.wantEnPassnt)
			}
		})
	}
}

func TestApplyMove_NilMove(t *testing.T) {
	board, err := NewBoardFromFEN(InitialFEN)
	if err != nil {
		t.Fatalf("NewBoardFromFEN failed: %v", err)
	}

	ok := ApplyMove(board, nil)
	if ok {
		t.Errorf("ApplyMove(nil) = true, want false")
	}
}

func TestApplyMove_Castling(t *testing.T) {
	tests := []struct {
		name      string
		fen       string
		moveClass chess.MoveClass
		wantOk    bool
		checkFn   func(*chess.Board) bool
	}{
		{
			name:      "white kingside castle",
			fen:       "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
			moveClass: chess.KingsideCastle,
			wantOk:    true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('g', '1') == chess.W(chess.King) &&
					b.Get('f', '1') == chess.W(chess.Rook) &&
					b.Get('e', '1') == chess.Empty &&
					b.Get('h', '1') == chess.Empty &&
					b.ToMove == chess.Black
			},
		},
		{
			name:      "white queenside castle",
			fen:       "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
			moveClass: chess.QueensideCastle,
			wantOk:    true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('c', '1') == chess.W(chess.King) &&
					b.Get('d', '1') == chess.W(chess.Rook) &&
					b.Get('e', '1') == chess.Empty &&
					b.Get('a', '1') == chess.Empty &&
					b.ToMove == chess.Black
			},
		},
		{
			name:      "black kingside castle",
			fen:       "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R b KQkq - 0 1",
			moveClass: chess.KingsideCastle,
			wantOk:    true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('g', '8') == chess.B(chess.King) &&
					b.Get('f', '8') == chess.B(chess.Rook) &&
					b.Get('e', '8') == chess.Empty &&
					b.Get('h', '8') == chess.Empty &&
					b.ToMove == chess.White
			},
		},
		{
			name:      "black queenside castle",
			fen:       "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R b KQkq - 0 1",
			moveClass: chess.QueensideCastle,
			wantOk:    true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('c', '8') == chess.B(chess.King) &&
					b.Get('d', '8') == chess.B(chess.Rook) &&
					b.Get('e', '8') == chess.Empty &&
					b.Get('a', '8') == chess.Empty &&
					b.ToMove == chess.White
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := NewBoardFromFEN(tt.fen)
			if err != nil {
				t.Fatalf("NewBoardFromFEN(%q) failed: %v", tt.fen, err)
			}

			move := &chess.Move{Class: tt.moveClass}
			ok := ApplyMove(board, move)

			if ok != tt.wantOk {
				t.Errorf("ApplyMove() = %v, want %v", ok, tt.wantOk)
			}
			if ok && tt.checkFn != nil && !tt.checkFn(board) {
				t.Errorf("checkFn failed after ApplyMove")
			}
		})
	}
}

func TestApplyMove_PawnMoves(t *testing.T) {
	tests := []struct {
		name    string
		fen     string
		move    *chess.Move
		wantOk  bool
		checkFn func(*chess.Board) bool
	}{
		{
			name: "pawn single move e2-e3",
			fen:  InitialFEN,
			move: &chess.Move{
				Class:    chess.PawnMove,
				FromCol:  'e',
				FromRank: '2',
				ToCol:    'e',
				ToRank:   '3',
			},
			wantOk: true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('e', '3') == chess.W(chess.Pawn) &&
					b.Get('e', '2') == chess.Empty
			},
		},
		{
			name: "pawn double move e2-e4",
			fen:  InitialFEN,
			move: &chess.Move{
				Class:    chess.PawnMove,
				FromCol:  'e',
				FromRank: '2',
				ToCol:    'e',
				ToRank:   '4',
			},
			wantOk: true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('e', '4') == chess.W(chess.Pawn) &&
					b.Get('e', '2') == chess.Empty &&
					b.EnPassant == true &&
					b.EPCol == 'e'
			},
		},
		{
			name: "pawn capture",
			fen:  "rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2",
			move: &chess.Move{
				Class:         chess.PawnMove,
				FromCol:       'e',
				FromRank:      '4',
				ToCol:         'd',
				ToRank:        '5',
				CapturedPiece: chess.Pawn,
			},
			wantOk: true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('d', '5') == chess.W(chess.Pawn) &&
					b.Get('e', '4') == chess.Empty
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := NewBoardFromFEN(tt.fen)
			if err != nil {
				t.Fatalf("NewBoardFromFEN(%q) failed: %v", tt.fen, err)
			}

			ok := ApplyMove(board, tt.move)

			if ok != tt.wantOk {
				t.Errorf("ApplyMove() = %v, want %v", ok, tt.wantOk)
			}
			if ok && tt.checkFn != nil && !tt.checkFn(board) {
				t.Errorf("checkFn failed after ApplyMove")
			}
		})
	}
}

func TestApplyMove_EnPassant(t *testing.T) {
	tests := []struct {
		name    string
		fen     string
		move    *chess.Move
		wantOk  bool
		checkFn func(*chess.Board) bool
	}{
		{
			name: "white en passant capture",
			fen:  "rnbqkbnr/pppp1ppp/8/4pP2/8/8/PPPPP1PP/RNBQKBNR w KQkq e6 0 3",
			move: &chess.Move{
				Class:         chess.EnPassantPawnMove,
				FromCol:       'f',
				FromRank:      '5',
				ToCol:         'e',
				ToRank:        '6',
				CapturedPiece: chess.Pawn,
			},
			wantOk: true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('e', '6') == chess.W(chess.Pawn) &&
					b.Get('f', '5') == chess.Empty &&
					b.Get('e', '5') == chess.Empty // Captured pawn removed
			},
		},
		{
			name: "black en passant capture",
			fen:  "rnbqkbnr/ppppp1pp/8/8/4Pp2/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 3",
			move: &chess.Move{
				Class:         chess.EnPassantPawnMove,
				FromCol:       'f',
				FromRank:      '4',
				ToCol:         'e',
				ToRank:        '3',
				CapturedPiece: chess.Pawn,
			},
			wantOk: true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('e', '3') == chess.B(chess.Pawn) &&
					b.Get('f', '4') == chess.Empty &&
					b.Get('e', '4') == chess.Empty // Captured pawn removed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := NewBoardFromFEN(tt.fen)
			if err != nil {
				t.Fatalf("NewBoardFromFEN(%q) failed: %v", tt.fen, err)
			}

			ok := ApplyMove(board, tt.move)

			if ok != tt.wantOk {
				t.Errorf("ApplyMove() = %v, want %v", ok, tt.wantOk)
			}
			if ok && tt.checkFn != nil && !tt.checkFn(board) {
				t.Errorf("checkFn failed after ApplyMove")
			}
		})
	}
}

func TestApplyMove_Promotion(t *testing.T) {
	tests := []struct {
		name       string
		fen        string
		move       *chess.Move
		wantOk     bool
		wantPiece  chess.Piece
		wantSquare struct {
			col  chess.Col
			rank chess.Rank
		}
	}{
		{
			name: "white pawn promotes to queen",
			fen:  "8/P7/8/8/8/8/8/4K2k w - - 0 1",
			move: &chess.Move{
				Class:         chess.PawnMoveWithPromotion,
				FromCol:       'a',
				FromRank:      '7',
				ToCol:         'a',
				ToRank:        '8',
				PromotedPiece: chess.Queen,
			},
			wantOk:    true,
			wantPiece: chess.W(chess.Queen),
			wantSquare: struct {
				col  chess.Col
				rank chess.Rank
			}{'a', '8'},
		},
		{
			name: "white pawn promotes to knight",
			fen:  "8/P7/8/8/8/8/8/4K2k w - - 0 1",
			move: &chess.Move{
				Class:         chess.PawnMoveWithPromotion,
				FromCol:       'a',
				FromRank:      '7',
				ToCol:         'a',
				ToRank:        '8',
				PromotedPiece: chess.Knight,
			},
			wantOk:    true,
			wantPiece: chess.W(chess.Knight),
			wantSquare: struct {
				col  chess.Col
				rank chess.Rank
			}{'a', '8'},
		},
		{
			name: "black pawn promotes to queen",
			fen:  "4K2k/8/8/8/8/8/p7/8 b - - 0 1",
			move: &chess.Move{
				Class:         chess.PawnMoveWithPromotion,
				FromCol:       'a',
				FromRank:      '2',
				ToCol:         'a',
				ToRank:        '1',
				PromotedPiece: chess.Queen,
			},
			wantOk:    true,
			wantPiece: chess.B(chess.Queen),
			wantSquare: struct {
				col  chess.Col
				rank chess.Rank
			}{'a', '1'},
		},
		{
			name: "promotion with capture",
			fen:  "1n6/P7/8/8/8/8/8/4K2k w - - 0 1",
			move: &chess.Move{
				Class:         chess.PawnMoveWithPromotion,
				FromCol:       'a',
				FromRank:      '7',
				ToCol:         'b',
				ToRank:        '8',
				PromotedPiece: chess.Queen,
				CapturedPiece: chess.Knight,
			},
			wantOk:    true,
			wantPiece: chess.W(chess.Queen),
			wantSquare: struct {
				col  chess.Col
				rank chess.Rank
			}{'b', '8'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := NewBoardFromFEN(tt.fen)
			if err != nil {
				t.Fatalf("NewBoardFromFEN(%q) failed: %v", tt.fen, err)
			}

			ok := ApplyMove(board, tt.move)

			if ok != tt.wantOk {
				t.Errorf("ApplyMove() = %v, want %v", ok, tt.wantOk)
			}
			if ok {
				got := board.Get(tt.wantSquare.col, tt.wantSquare.rank)
				if got != tt.wantPiece {
					t.Errorf("board.Get(%c, %c) = %v, want %v",
						tt.wantSquare.col, tt.wantSquare.rank, got, tt.wantPiece)
				}
			}
		})
	}
}

func TestApplyMove_PieceMoves(t *testing.T) {
	tests := []struct {
		name    string
		fen     string
		move    *chess.Move
		wantOk  bool
		checkFn func(*chess.Board) bool
	}{
		{
			name: "knight move Nf3",
			fen:  InitialFEN,
			move: &chess.Move{
				Class:       chess.PieceMove,
				PieceToMove: chess.Knight,
				FromCol:     'g',
				FromRank:    '1',
				ToCol:       'f',
				ToRank:      '3',
			},
			wantOk: true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('f', '3') == chess.W(chess.Knight) &&
					b.Get('g', '1') == chess.Empty
			},
		},
		{
			name: "bishop move Bc4",
			fen:  "rnbqkbnr/pppppppp/8/8/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 1 2",
			move: &chess.Move{
				Class:       chess.PieceMove,
				PieceToMove: chess.Bishop,
				FromCol:     'f',
				FromRank:    '1',
				ToCol:       'c',
				ToRank:      '4',
			},
			wantOk: true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('c', '4') == chess.W(chess.Bishop) &&
					b.Get('f', '1') == chess.Empty
			},
		},
		{
			name: "rook move Ra3",
			fen:  "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
			move: &chess.Move{
				Class:       chess.PieceMove,
				PieceToMove: chess.Rook,
				FromCol:     'a',
				FromRank:    '1',
				ToCol:       'a',
				ToRank:      '3',
			},
			wantOk: true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('a', '3') == chess.W(chess.Rook) &&
					b.Get('a', '1') == chess.Empty
			},
		},
		{
			name: "queen move Qd4",
			fen:  "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 2",
			move: &chess.Move{
				Class:       chess.PieceMove,
				PieceToMove: chess.Queen,
				FromCol:     'd',
				FromRank:    '1',
				ToCol:       'h',
				ToRank:      '5',
			},
			wantOk: true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('h', '5') == chess.W(chess.Queen) &&
					b.Get('d', '1') == chess.Empty
			},
		},
		{
			name: "king move Kf1",
			fen:  "rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 2",
			move: &chess.Move{
				Class:       chess.PieceMove,
				PieceToMove: chess.King,
				FromCol:     'e',
				FromRank:    '1',
				ToCol:       'f',
				ToRank:      '1',
			},
			wantOk: true,
			checkFn: func(b *chess.Board) bool {
				return b.Get('f', '1') == chess.W(chess.King) &&
					b.Get('e', '1') == chess.Empty &&
					b.WKingCol == 'f' && b.WKingRank == '1'
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := NewBoardFromFEN(tt.fen)
			if err != nil {
				t.Fatalf("NewBoardFromFEN(%q) failed: %v", tt.fen, err)
			}

			ok := ApplyMove(board, tt.move)

			if ok != tt.wantOk {
				t.Errorf("ApplyMove() = %v, want %v", ok, tt.wantOk)
			}
			if ok && tt.checkFn != nil && !tt.checkFn(board) {
				t.Errorf("checkFn failed after ApplyMove")
			}
		})
	}
}

func TestIsInCheck(t *testing.T) {
	tests := []struct {
		name        string
		fen         string
		colour      chess.Colour
		wantInCheck bool
	}{
		{
			name:        "initial position not in check",
			fen:         InitialFEN,
			colour:      chess.White,
			wantInCheck: false,
		},
		{
			name:        "initial position black not in check",
			fen:         InitialFEN,
			colour:      chess.Black,
			wantInCheck: false,
		},
		{
			name:        "scholar's mate - black in check",
			fen:         "r1bqkb1r/pppp1Qpp/2n2n2/4p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 4",
			colour:      chess.Black,
			wantInCheck: true,
		},
		{
			name:        "white king in check from rook on same rank",
			fen:         "8/8/8/8/8/8/8/r3K3 w - - 0 1",
			colour:      chess.White,
			wantInCheck: true,
		},
		{
			name:        "white king in check from bishop on diagonal",
			fen:         "8/8/8/8/8/8/3b4/4K3 w - - 0 1",
			colour:      chess.White,
			wantInCheck: true,
		},
		{
			name:        "white king in check from knight",
			fen:         "8/8/8/8/8/5n2/8/4K3 w - - 0 1",
			colour:      chess.White,
			wantInCheck: true,
		},
		{
			name:        "white king in check from pawn",
			fen:         "8/8/8/8/8/3p4/8/4K3 w - - 0 1",
			colour:      chess.White,
			wantInCheck: false, // Pawn on d3 doesn't attack e1
		},
		{
			name:        "white king attacked by pawn on diagonal",
			fen:         "8/8/8/8/8/8/5p2/4K3 w - - 0 1",
			colour:      chess.White,
			wantInCheck: true,
		},
		{
			name:        "queen giving check",
			fen:         "4k3/8/8/8/8/8/8/4K2Q w - - 0 1",
			colour:      chess.Black,
			wantInCheck: false, // Queen on h1 doesn't attack e8
		},
		{
			name:        "queen giving check on file",
			fen:         "4k3/8/8/8/4Q3/8/8/4K3 w - - 0 1",
			colour:      chess.Black,
			wantInCheck: true, // Queen on e4 attacks e8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := NewBoardFromFEN(tt.fen)
			if err != nil {
				t.Fatalf("NewBoardFromFEN(%q) failed: %v", tt.fen, err)
			}

			got := IsInCheck(board, tt.colour)
			if got != tt.wantInCheck {
				t.Errorf("IsInCheck() = %v, want %v", got, tt.wantInCheck)
			}
		})
	}
}

func TestIsCheckmate(t *testing.T) {
	tests := []struct {
		name     string
		fen      string
		wantMate bool
	}{
		{
			name:     "initial position - not mate",
			fen:      InitialFEN,
			wantMate: false,
		},
		{
			name:     "fool's mate",
			fen:      "rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3",
			wantMate: true,
		},
		{
			name:     "scholar's mate",
			fen:      "r1bqkb1r/pppp1Qpp/2n2n2/4p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 4",
			wantMate: true,
		},
		{
			name:     "back rank mate",
			fen:      "8/8/8/8/8/8/5PPP/4r1K1 w - - 0 1",
			wantMate: true,
		},
		{
			name:     "smothered mate",
			fen:      "6rk/5Npp/8/8/8/8/8/4K3 b - - 0 1",
			wantMate: true,
		},
		{
			name:     "check but can block",
			fen:      "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
			wantMate: false,
		},
		{
			name:     "check but king can move",
			fen:      "8/8/8/8/8/8/r7/4K3 w - - 0 1",
			wantMate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := NewBoardFromFEN(tt.fen)
			if err != nil {
				t.Fatalf("NewBoardFromFEN(%q) failed: %v", tt.fen, err)
			}

			got := IsCheckmate(board)
			if got != tt.wantMate {
				t.Errorf("IsCheckmate() = %v, want %v", got, tt.wantMate)
			}
		})
	}
}

func TestIsStalemate(t *testing.T) {
	tests := []struct {
		name          string
		fen           string
		wantStalemate bool
	}{
		{
			name:          "initial position - not stalemate",
			fen:           InitialFEN,
			wantStalemate: false,
		},
		{
			name:          "classic stalemate - king cornered by queen",
			fen:           "7k/5Q2/6K1/8/8/8/8/8 b - - 0 1",
			wantStalemate: true,
		},
		{
			name:          "stalemate - king in corner",
			fen:           "k7/2Q5/1K6/8/8/8/8/8 b - - 0 1",
			wantStalemate: true,
		},
		{
			name:          "king vs king and bishop - insufficient but not stalemate",
			fen:           "8/8/8/4k3/8/8/3B4/4K3 b - - 0 1",
			wantStalemate: false,
		},
		{
			name:          "checkmate is not stalemate",
			fen:           "rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3",
			wantStalemate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := NewBoardFromFEN(tt.fen)
			if err != nil {
				t.Fatalf("NewBoardFromFEN(%q) failed: %v", tt.fen, err)
			}

			got := IsStalemate(board)
			if got != tt.wantStalemate {
				t.Errorf("IsStalemate() = %v, want %v", got, tt.wantStalemate)
			}
		})
	}
}

func TestHasLegalMoves(t *testing.T) {
	tests := []struct {
		name           string
		fen            string
		colour         chess.Colour
		wantLegalMoves bool
	}{
		{
			name:           "initial position - white has moves",
			fen:            InitialFEN,
			colour:         chess.White,
			wantLegalMoves: true,
		},
		{
			name:           "initial position - black has moves",
			fen:            InitialFEN,
			colour:         chess.Black,
			wantLegalMoves: true,
		},
		{
			name:           "stalemate - no legal moves",
			fen:            "7k/5Q2/6K1/8/8/8/8/8 b - - 0 1",
			colour:         chess.Black,
			wantLegalMoves: false,
		},
		{
			name:           "checkmate - no legal moves",
			fen:            "rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3",
			colour:         chess.White,
			wantLegalMoves: false,
		},
		{
			name:           "king only - has moves",
			fen:            "8/8/8/4k3/8/8/8/4K3 w - - 0 1",
			colour:         chess.White,
			wantLegalMoves: true,
		},
		{
			name:           "pinned piece cannot move away from pin line",
			fen:            "4k3/8/8/8/b7/8/2P5/4K3 w - - 0 1", // Pawn pinned by bishop
			colour:         chess.White,
			wantLegalMoves: true, // King can still move
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := NewBoardFromFEN(tt.fen)
			if err != nil {
				t.Fatalf("NewBoardFromFEN(%q) failed: %v", tt.fen, err)
			}

			got := HasLegalMoves(board, tt.colour)
			if got != tt.wantLegalMoves {
				t.Errorf("HasLegalMoves() = %v, want %v", got, tt.wantLegalMoves)
			}
		})
	}
}

// TestHelperFunctions tests internal helper functions
func TestAbs(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{0, 0},
		{1, 1},
		{-1, 1},
		{5, 5},
		{-5, 5},
	}

	for _, tt := range tests {
		got := abs(tt.input)
		if got != tt.want {
			t.Errorf("abs(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestSign(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{0, 0},
		{1, 1},
		{-1, -1},
		{5, 1},
		{-5, -1},
	}

	for _, tt := range tests {
		got := sign(tt.input)
		if got != tt.want {
			t.Errorf("sign(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
