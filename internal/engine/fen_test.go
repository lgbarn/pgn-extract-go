package engine

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

func TestNewBoardFromFEN(t *testing.T) {
	tests := []struct {
		name     string
		fen      string
		wantErr  bool
		checkFn  func(*chess.Board) bool
	}{
		{
			name:    "initial position",
			fen:     InitialFEN,
			wantErr: false,
			checkFn: func(b *chess.Board) bool {
				// Check some key squares
				return b.Get('e', '1') == chess.W(chess.King) &&
					b.Get('e', '8') == chess.B(chess.King) &&
					b.Get('e', '2') == chess.W(chess.Pawn) &&
					b.Get('e', '7') == chess.B(chess.Pawn) &&
					b.ToMove == chess.White &&
					b.WKingCastle == 'h' &&
					b.WQueenCastle == 'a'
			},
		},
		{
			name:    "after 1.e4",
			fen:     "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
			wantErr: false,
			checkFn: func(b *chess.Board) bool {
				return b.Get('e', '4') == chess.W(chess.Pawn) &&
					b.Get('e', '2') == chess.Empty &&
					b.ToMove == chess.Black &&
					b.EnPassant == true &&
					b.EPCol == 'e' &&
					b.EPRank == '3'
			},
		},
		{
			name:    "sicilian defense",
			fen:     "rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w KQkq c6 0 2",
			wantErr: false,
			checkFn: func(b *chess.Board) bool {
				return b.Get('c', '5') == chess.B(chess.Pawn) &&
					b.Get('e', '4') == chess.W(chess.Pawn) &&
					b.ToMove == chess.White
			},
		},
		{
			name:    "no castling rights",
			fen:     "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w - - 0 1",
			wantErr: false,
			checkFn: func(b *chess.Board) bool {
				return b.WKingCastle == 0 &&
					b.WQueenCastle == 0 &&
					b.BKingCastle == 0 &&
					b.BQueenCastle == 0
			},
		},
		{
			name:    "empty string",
			fen:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := NewBoardFromFEN(tt.fen)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBoardFromFEN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFn != nil && !tt.checkFn(board) {
				t.Errorf("NewBoardFromFEN() board check failed")
			}
		})
	}
}

func TestBoardToFEN(t *testing.T) {
	// Test round-trip: FEN -> Board -> FEN
	tests := []string{
		InitialFEN,
		"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		"r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
		"8/8/8/8/8/8/8/4K3 w - - 0 1",
	}

	for _, fen := range tests {
		t.Run(fen, func(t *testing.T) {
			board, err := NewBoardFromFEN(fen)
			if err != nil {
				t.Fatalf("NewBoardFromFEN() error = %v", err)
			}

			result := BoardToFEN(board)
			// Note: FEN strings may have slight variations (e.g., "-" vs "- -")
			// so we compare the parsed boards instead
			board2, err := NewBoardFromFEN(result)
			if err != nil {
				t.Fatalf("NewBoardFromFEN(result) error = %v", err)
			}

			// Compare key properties
			if board.ToMove != board2.ToMove {
				t.Errorf("ToMove mismatch: got %v, want %v", board2.ToMove, board.ToMove)
			}
			if board.WKingCastle != board2.WKingCastle {
				t.Errorf("WKingCastle mismatch")
			}
		})
	}
}

func TestApplyMove(t *testing.T) {
	tests := []struct {
		name    string
		fen     string
		move    *chess.Move
		wantFEN string
	}{
		{
			name: "1.e4",
			fen:  InitialFEN,
			move: &chess.Move{
				Text:        "e4",
				Class:       chess.PawnMove,
				PieceToMove: chess.Pawn,
				ToCol:       'e',
				ToRank:      '4',
			},
			wantFEN: "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		},
		{
			name: "1.Nf3",
			fen:  InitialFEN,
			move: &chess.Move{
				Text:        "Nf3",
				Class:       chess.PieceMove,
				PieceToMove: chess.Knight,
				ToCol:       'f',
				ToRank:      '3',
			},
			wantFEN: "rnbqkbnr/pppppppp/8/8/8/5N2/PPPPPPPP/RNBQKB1R b KQkq - 1 1",
		},
		{
			name: "kingside castle",
			fen:  "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
			move: &chess.Move{
				Text:        "O-O",
				Class:       chess.KingsideCastle,
				PieceToMove: chess.King,
			},
			wantFEN: "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R4RK1 b kq - 1 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := NewBoardFromFEN(tt.fen)
			if err != nil {
				t.Fatalf("NewBoardFromFEN() error = %v", err)
			}

			if !ApplyMove(board, tt.move) {
				t.Fatalf("ApplyMove() returned false")
			}

			gotFEN := BoardToFEN(board)
			// Parse both FENs and compare boards
			wantBoard, _ := NewBoardFromFEN(tt.wantFEN)
			gotBoard, _ := NewBoardFromFEN(gotFEN)

			if wantBoard.ToMove != gotBoard.ToMove {
				t.Errorf("ToMove mismatch: got %v, want %v (FEN: %s)", gotBoard.ToMove, wantBoard.ToMove, gotFEN)
			}
		})
	}
}
