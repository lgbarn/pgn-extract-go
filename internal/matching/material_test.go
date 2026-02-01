package matching

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/testutil"
)

func TestNewMaterialMatcher(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		exact       bool
		wantWhite   map[chess.Piece]int
		wantBlack   map[chess.Piece]int
		wantPattern string
	}{
		{
			name:    "queen vs queen",
			pattern: "Q:q",
			exact:   false,
			wantWhite: map[chess.Piece]int{
				chess.Queen: 1,
			},
			wantBlack: map[chess.Piece]int{
				chess.Queen: 1,
			},
		},
		{
			name:    "queen and rook vs queen and two rooks",
			pattern: "QR:qrr",
			exact:   false,
			wantWhite: map[chess.Piece]int{
				chess.Queen: 1,
				chess.Rook:  1,
			},
			wantBlack: map[chess.Piece]int{
				chess.Queen: 1,
				chess.Rook:  2,
			},
		},
		{
			name:    "king only vs king only",
			pattern: "K:k",
			exact:   true,
			wantWhite: map[chess.Piece]int{
				chess.King: 1,
			},
			wantBlack: map[chess.Piece]int{
				chess.King: 1,
			},
		},
		{
			name:    "full set",
			pattern: "KQRRBBNNPPPPPPPP:kqrrbbnnpppppppp",
			exact:   true,
			wantWhite: map[chess.Piece]int{
				chess.King:   1,
				chess.Queen:  1,
				chess.Rook:   2,
				chess.Bishop: 2,
				chess.Knight: 2,
				chess.Pawn:   8,
			},
			wantBlack: map[chess.Piece]int{
				chess.King:   1,
				chess.Queen:  1,
				chess.Rook:   2,
				chess.Bishop: 2,
				chess.Knight: 2,
				chess.Pawn:   8,
			},
		},
		{
			name:    "white only pattern",
			pattern: "KQ",
			exact:   false,
			wantWhite: map[chess.Piece]int{
				chess.King:  1,
				chess.Queen: 1,
			},
			wantBlack: map[chess.Piece]int{},
		},
		{
			name:      "empty pattern",
			pattern:   "",
			exact:     false,
			wantWhite: map[chess.Piece]int{},
			wantBlack: map[chess.Piece]int{},
		},
		{
			name:    "pawns only",
			pattern: "PPP:ppp",
			exact:   false,
			wantWhite: map[chess.Piece]int{
				chess.Pawn: 3,
			},
			wantBlack: map[chess.Piece]int{
				chess.Pawn: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mm := NewMaterialMatcher(tt.pattern, tt.exact)
			if mm.pattern != tt.pattern {
				t.Errorf("pattern = %q, want %q", mm.pattern, tt.pattern)
			}
			if mm.exactMatch != tt.exact {
				t.Errorf("exactMatch = %v, want %v", mm.exactMatch, tt.exact)
			}
			for piece, count := range tt.wantWhite {
				if mm.whitePieces[piece] != count {
					t.Errorf("whitePieces[%v] = %d, want %d", piece, mm.whitePieces[piece], count)
				}
			}
			for piece, count := range tt.wantBlack {
				if mm.blackPieces[piece] != count {
					t.Errorf("blackPieces[%v] = %d, want %d", piece, mm.blackPieces[piece], count)
				}
			}
		})
	}
}

func TestMaterialMatcher_HasCriteria(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{"non-empty pattern", "Q:q", true},
		{"empty pattern", "", false},
		{"white only", "K", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mm := NewMaterialMatcher(tt.pattern, false)
			if got := mm.HasCriteria(); got != tt.want {
				t.Errorf("HasCriteria() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaterialMatcher_ParsePieces(t *testing.T) {
	// Test that uppercase and lowercase in pattern both map to piece types
	mm := NewMaterialMatcher("", false)

	// Parse uppercase letters for white
	mm.parsePieces("KQRBNP", chess.White)
	expectedWhite := map[chess.Piece]int{
		chess.King:   1,
		chess.Queen:  1,
		chess.Rook:   1,
		chess.Bishop: 1,
		chess.Knight: 1,
		chess.Pawn:   1,
	}
	for piece, count := range expectedWhite {
		if mm.whitePieces[piece] != count {
			t.Errorf("whitePieces[%v] = %d, want %d", piece, mm.whitePieces[piece], count)
		}
	}

	// Parse lowercase letters for black (they get uppercased internally)
	mm.parsePieces("kqrbnp", chess.Black)
	expectedBlack := map[chess.Piece]int{
		chess.King:   1,
		chess.Queen:  1,
		chess.Rook:   1,
		chess.Bishop: 1,
		chess.Knight: 1,
		chess.Pawn:   1,
	}
	for piece, count := range expectedBlack {
		if mm.blackPieces[piece] != count {
			t.Errorf("blackPieces[%v] = %d, want %d", piece, mm.blackPieces[piece], count)
		}
	}
}

func TestMaterialMatcher_ParsePieces_UnknownChars(t *testing.T) {
	// Unknown characters should be ignored
	mm := NewMaterialMatcher("", false)
	mm.parsePieces("XYZ123", chess.White)
	for _, piece := range []chess.Piece{chess.King, chess.Queen, chess.Rook, chess.Bishop, chess.Knight, chess.Pawn} {
		if mm.whitePieces[piece] != 0 {
			t.Errorf("whitePieces[%v] = %d, want 0 for unknown chars", piece, mm.whitePieces[piece])
		}
	}
}

func TestMaterialMatcher_MatchPosition_InitialPosition(t *testing.T) {
	// Initial position has full material for both sides
	board, err := engine.NewBoardFromFEN(engine.InitialFEN)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		pattern string
		exact   bool
		want    bool
	}{
		{
			name:    "exact full material matches initial",
			pattern: "KQRRBBNNPPPPPPPP:kqrrbbnnpppppppp",
			exact:   true,
			want:    true,
		},
		{
			name:    "minimal queen matches initial",
			pattern: "Q:q",
			exact:   false,
			want:    true,
		},
		{
			name:    "exact king only does not match initial",
			pattern: "K:k",
			exact:   true,
			want:    false,
		},
		{
			name:    "minimal king matches initial",
			pattern: "K:k",
			exact:   false,
			want:    true,
		},
		{
			name:    "exact wrong rook count",
			pattern: "KQRBBNNPPPPPPPP:kqrrbbnnpppppppp",
			exact:   true,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mm := NewMaterialMatcher(tt.pattern, tt.exact)
			got := mm.matchPosition(board)
			if got != tt.want {
				t.Errorf("matchPosition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaterialMatcher_ExactMatch_ExtraPieces(t *testing.T) {
	// Exact match should fail if the board has pieces not in the pattern
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN)

	// Pattern specifies only king for white, but board has all pieces
	mm := NewMaterialMatcher("K:kqrrbbnnpppppppp", true)
	if mm.matchPosition(board) {
		t.Error("exact match should fail when board has extra white pieces")
	}
}

func TestMaterialMatcher_MinimalMatch_SubsetOK(t *testing.T) {
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN)

	// Minimal match: at least 1 queen for each side
	mm := NewMaterialMatcher("Q:q", false)
	if !mm.matchPosition(board) {
		t.Error("minimal match should succeed when board has at least the specified pieces")
	}
}

func TestMaterialMatcher_MinimalMatch_InsufficientPieces(t *testing.T) {
	// Position with only kings
	board, _ := engine.NewBoardFromFEN("4k3/8/8/8/8/8/8/4K3 w - - 0 1")

	// Require at least 1 queen for white
	mm := NewMaterialMatcher("Q:k", false)
	if mm.matchPosition(board) {
		t.Error("minimal match should fail when board lacks required pieces")
	}
}

func TestMaterialMatcher_MatchGame_InitialPosition(t *testing.T) {
	// A game where the initial position itself matches
	game := testutil.MustParseGame(t, `
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 *
`)

	// Full material is present at the start
	mm := NewMaterialMatcher("KQRRBBNNPPPPPPPP:kqrrbbnnpppppppp", true)
	if !mm.MatchGame(game) {
		t.Error("expected match at initial position")
	}
}

func TestMaterialMatcher_MatchGame_AfterCaptures(t *testing.T) {
	// Scandinavian Defense: 1. e4 d5 2. exd5 - white captures a pawn
	game := testutil.MustParseGame(t, `
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 d5 2. exd5 *
`)

	// After exd5, black has 7 pawns. Minimal match for 8 black pawns should fail throughout.
	mm := NewMaterialMatcher("K:kpppppppp", false)
	// The game starts with 8 pawns (matches) but after exd5, black has 7
	// Since we check initial position first, this should match at move 0
	if !mm.MatchGame(game) {
		t.Error("expected match at initial position (8 black pawns)")
	}

	// Exact match for 7 black pawns should match after the capture
	mm2 := NewMaterialMatcher("KQRRBBNNPPPPPPPP:kqrrbbnnppppppp", true)
	if !mm2.MatchGame(game) {
		t.Error("expected exact match after pawn capture")
	}
}

func TestMaterialMatcher_MatchGame_NoMatch(t *testing.T) {
	game := testutil.MustParseGame(t, `
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 *
`)

	// King-only exact match should never occur in this short game
	mm := NewMaterialMatcher("K:k", true)
	if mm.MatchGame(game) {
		t.Error("expected no match for king-only in a full game")
	}
}

func TestMaterialMatcher_Match(t *testing.T) {
	// Match() delegates to MatchGame()
	game := testutil.MustParseGame(t, `
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 *
`)

	mm := NewMaterialMatcher("Q:q", false)
	if !mm.Match(game) {
		t.Error("Match() should delegate to MatchGame()")
	}
}

func TestMaterialMatcher_Name(t *testing.T) {
	mm := NewMaterialMatcher("Q:q", false)
	if mm.Name() != "MaterialMatcher" {
		t.Errorf("Name() = %q, want %q", mm.Name(), "MaterialMatcher")
	}
}

func TestMaterialMatcher_ExactMatch_EmptyPattern(t *testing.T) {
	// Empty pattern in exact mode: only an empty board would match
	// (no pieces specified means all piece counts must be 0)
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN)
	mm := NewMaterialMatcher(":", true)
	if mm.matchPosition(board) {
		t.Error("empty exact pattern should not match initial position (has pieces)")
	}
}

func TestMaterialMatcher_ExactMatch_EmptyBoard(t *testing.T) {
	// A position with only kings (bare minimum for legal chess)
	board, _ := engine.NewBoardFromFEN("4k3/8/8/8/8/8/8/4K3 w - - 0 1")

	mm := NewMaterialMatcher("K:k", true)
	if !mm.matchPosition(board) {
		t.Error("K:k exact should match king-only position")
	}
}

func TestMaterialMatcher_ParsePattern_ColonOnly(t *testing.T) {
	mm := NewMaterialMatcher(":", false)
	// Both sides should have empty piece maps
	for _, piece := range []chess.Piece{chess.King, chess.Queen, chess.Rook, chess.Bishop, chess.Knight, chess.Pawn} {
		if mm.whitePieces[piece] != 0 {
			t.Errorf("whitePieces[%v] = %d, want 0", piece, mm.whitePieces[piece])
		}
		if mm.blackPieces[piece] != 0 {
			t.Errorf("blackPieces[%v] = %d, want 0", piece, mm.blackPieces[piece])
		}
	}
}

func TestMaterialMatcher_ParsePattern_MultipleColons(t *testing.T) {
	// Extra colons should be ignored (only first two parts used)
	mm := NewMaterialMatcher("K:k:extra", false)
	if mm.whitePieces[chess.King] != 1 {
		t.Error("expected 1 white king")
	}
	if mm.blackPieces[chess.King] != 1 {
		t.Error("expected 1 black king")
	}
}
