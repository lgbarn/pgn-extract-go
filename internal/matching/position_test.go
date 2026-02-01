package matching

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/testutil"
)

// --- pieceToChar tests ---

func TestPieceToChar(t *testing.T) {
	tests := []struct {
		name  string
		piece chess.Piece
		want  byte
	}{
		{"empty", chess.Empty, '_'},
		{"white pawn", chess.W(chess.Pawn), 'P'},
		{"white knight", chess.W(chess.Knight), 'N'},
		{"white bishop", chess.W(chess.Bishop), 'B'},
		{"white rook", chess.W(chess.Rook), 'R'},
		{"white queen", chess.W(chess.Queen), 'Q'},
		{"white king", chess.W(chess.King), 'K'},
		{"black pawn", chess.B(chess.Pawn), 'p'},
		{"black knight", chess.B(chess.Knight), 'n'},
		{"black bishop", chess.B(chess.Bishop), 'b'},
		{"black rook", chess.B(chess.Rook), 'r'},
		{"black queen", chess.B(chess.Queen), 'q'},
		{"black king", chess.B(chess.King), 'k'},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pieceToChar(tt.piece)
			if got != tt.want {
				t.Errorf("pieceToChar(%v) = %c, want %c", tt.piece, got, tt.want)
			}
		})
	}
}

func TestPieceToChar_OffBoard(t *testing.T) {
	// chess.Off should return '_'
	got := pieceToChar(chess.Off)
	// Off has piece type 0 which hits default case
	if got != '_' {
		t.Errorf("pieceToChar(Off) = %c, want '_'", got)
	}
}

// --- boardToRanks tests ---

func TestBoardToRanks_InitialPosition(t *testing.T) {
	board, err := engine.NewBoardFromFEN(engine.InitialFEN)
	if err != nil {
		t.Fatal(err)
	}
	ranks := boardToRanks(board)

	// Rank 0 = rank 1 (white pieces)
	if ranks[0] != "RNBQKBNR" {
		t.Errorf("rank 1 = %q, want RNBQKBNR", ranks[0])
	}
	// Rank 1 = rank 2 (white pawns)
	if ranks[1] != "PPPPPPPP" {
		t.Errorf("rank 2 = %q, want PPPPPPPP", ranks[1])
	}
	// Ranks 2-5 = empty
	for i := 2; i <= 5; i++ {
		if ranks[i] != "________" {
			t.Errorf("rank %d = %q, want ________", i+1, ranks[i])
		}
	}
	// Rank 6 = rank 7 (black pawns)
	if ranks[6] != "pppppppp" {
		t.Errorf("rank 7 = %q, want pppppppp", ranks[6])
	}
	// Rank 7 = rank 8 (black pieces)
	if ranks[7] != "rnbqkbnr" {
		t.Errorf("rank 8 = %q, want rnbqkbnr", ranks[7])
	}
}

func TestBoardToRanks_CustomFEN(t *testing.T) {
	board, err := engine.NewBoardFromFEN("4k3/8/8/8/8/8/8/4K3 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	ranks := boardToRanks(board)

	if ranks[0] != "____K___" {
		t.Errorf("rank 1 = %q, want ____K___", ranks[0])
	}
	if ranks[7] != "____k___" {
		t.Errorf("rank 8 = %q, want ____k___", ranks[7])
	}
	for i := 1; i <= 6; i++ {
		if ranks[i] != "________" {
			t.Errorf("rank %d = %q, want ________", i+1, ranks[i])
		}
	}
}

// --- matchRank tests ---

func TestMatchRank(t *testing.T) {
	tests := []struct {
		name        string
		boardRank   string
		patternRank string
		want        bool
	}{
		// Exact matches
		{"exact match", "RNBQKBNR", "RNBQKBNR", true},
		{"exact mismatch", "RNBQKBNR", "RNBQKBN_", false},

		// ? wildcard (matches any single square)
		{"question mark any piece", "RNBQKBNR", "?NBQKBNR", true},
		{"question mark empty", "________", "????????", true},
		{"question mark wrong length", "RNBQKBNR", "???????", false},

		// ! wildcard (matches non-empty)
		{"bang matches piece", "RNBQKBNR", "!NBQKBNR", true},
		{"bang fails on empty", "________", "!_______", false},
		{"all bang on full rank", "RNBQKBNR", "!!!!!!!!", true},

		// A wildcard (matches any white/uppercase piece)
		{"A matches white", "RNBQKBNR", "ANBQKBNR", true},
		{"A fails on black", "rnbqkbnr", "Anbqkbnr", false},
		{"A fails on empty", "________", "A_______", false},

		// a wildcard (matches any black/lowercase piece)
		{"a matches black", "rnbqkbnr", "anbqkbnr", true},
		{"a fails on white", "RNBQKBNR", "aNBQKBNR", false},
		{"a fails on empty", "________", "a_______", false},

		// _ wildcard (matches empty square)
		{"underscore matches empty", "________", "________", true},
		{"underscore fails on piece", "RNBQKBNR", "_NBQKBNR", false},

		// Digit patterns (N empty squares)
		{"8 empty squares", "________", "8", true},
		{"4 empty squares prefix", "________", "4____", true},
		{"digit mismatch", "R_______", "8", false},
		{"1 empty", "________", "11111111", true},

		// * wildcard (matches zero or more)
		{"star matches all", "RNBQKBNR", "*", true},
		{"star matches nothing at end", "RNBQKBNR", "RNBQKBNR*", true},
		{"star in middle", "RNBQKBNR", "R*R", true},
		{"star zero chars", "RNBQKBNR", "R*NBQKBNR", true},
		{"star no match", "RNBQKBNR", "R*X", false},
		{"star at start", "RNBQKBNR", "*KBNR", true},
		{"star multiple", "RNBQKBNR", "*Q*", true},
		{"star empty board", "________", "*", true},

		// Mixed patterns
		{"mixed wildcards", "RNBQKBNR", "?N*NR", true},
		{"digit then piece", "____KBNR", "4KBNR", true},
		{"piece then digit", "RNBQ____", "RNBQ4", true},

		// Edge cases
		{"empty pattern empty board", "", "", true},
		{"pattern longer than board", "RN", "RNBQ", false},
		{"board longer than pattern", "RNBQ", "RN", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchRank(tt.boardRank, tt.patternRank)
			if got != tt.want {
				t.Errorf("matchRank(%q, %q) = %v, want %v", tt.boardRank, tt.patternRank, got, tt.want)
			}
		})
	}
}

func TestMatchRank_BangAtEndOfBoard(t *testing.T) {
	// ! when board index is already at end
	got := matchRank("", "!")
	if got != false {
		t.Error("matchRank('', '!') should be false")
	}
}

func TestMatchRank_AAtEndOfBoard(t *testing.T) {
	got := matchRank("", "A")
	if got != false {
		t.Error("matchRank('', 'A') should be false")
	}
}

func TestMatchRank_LowercaseAAtEndOfBoard(t *testing.T) {
	got := matchRank("", "a")
	if got != false {
		t.Error("matchRank('', 'a') should be false")
	}
}

func TestMatchRank_DigitExceedsBoard(t *testing.T) {
	// Digit requesting more empty squares than available
	got := matchRank("___", "8")
	if got != false {
		t.Error("matchRank('___', '8') should be false (only 3 empty squares)")
	}
}

// --- invertPattern tests ---

func TestInvertPattern(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
	}{
		{
			name:  "simple rank swap",
			input: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR",
			want:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR",
		},
		{
			name:  "single rank uppercase to lowercase",
			input: "RNBQKBNR",
			want:  "rnbqkbnr",
		},
		{
			name:  "single rank lowercase to uppercase",
			input: "rnbqkbnr",
			want:  "RNBQKBNR",
		},
		{
			name:  "mixed with wildcards preserved",
			input: "?*!_/PPPPPPPP",
			want:  "pppppppp/?*!_",
		},
		{
			name:  "digits and special chars unchanged",
			input: "4K3/8",
			want:  "8/4k3",
		},
		{
			name:  "two ranks reversed",
			input: "KBNR/kbnr",
			want:  "KBNR/kbnr",
		},
		{
			name:  "three ranks reversed",
			input: "AAA/BBB/ccc",
			want:  "CCC/bbb/aaa",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := invertPattern(tt.input)
			if got != tt.want {
				t.Errorf("invertPattern(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- PositionMatcher tests ---

func TestNewPositionMatcher(t *testing.T) {
	pm := NewPositionMatcher()
	if pm == nil {
		t.Fatal("NewPositionMatcher returned nil")
	}
	if pm.PatternCount() != 0 {
		t.Errorf("PatternCount() = %d, want 0", pm.PatternCount())
	}
}

func TestPositionMatcher_AddFEN(t *testing.T) {
	pm := NewPositionMatcher()
	err := pm.AddFEN(engine.InitialFEN, "initial")
	if err != nil {
		t.Fatalf("AddFEN failed: %v", err)
	}
	if pm.PatternCount() != 1 {
		t.Errorf("PatternCount() = %d, want 1", pm.PatternCount())
	}
}

func TestPositionMatcher_AddFEN_Invalid(t *testing.T) {
	pm := NewPositionMatcher()
	err := pm.AddFEN("invalid fen string", "bad")
	if err == nil {
		t.Error("expected error for invalid FEN")
	}
	if pm.PatternCount() != 0 {
		t.Errorf("PatternCount() = %d, want 0 after invalid FEN", pm.PatternCount())
	}
}

func TestPositionMatcher_AddPattern(t *testing.T) {
	pm := NewPositionMatcher()
	pm.AddPattern("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR", "initial", false)
	if pm.PatternCount() != 1 {
		t.Errorf("PatternCount() = %d, want 1", pm.PatternCount())
	}
}

func TestPositionMatcher_AddPattern_WithInvert(t *testing.T) {
	pm := NewPositionMatcher()
	pm.AddPattern("PPPPPPPP/*/*/*/*/*/*/*", "white pawns on rank 8", true)
	// Should add the original + the inverted pattern
	if pm.PatternCount() != 2 {
		t.Errorf("PatternCount() = %d, want 2 (original + inverted)", pm.PatternCount())
	}
}

func TestPositionMatcher_MatchGame_FEN(t *testing.T) {
	game := testutil.MustParseGame(t, `
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 *
`)

	pm := NewPositionMatcher()
	err := pm.AddFEN("r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3", "Ruy Lopez")
	if err != nil {
		t.Fatal(err)
	}

	match := pm.MatchGame(game)
	if match == nil {
		t.Fatal("expected FEN match for Ruy Lopez")
	}
	if match.Label != "Ruy Lopez" {
		t.Errorf("label = %q, want %q", match.Label, "Ruy Lopez")
	}
}

func TestPositionMatcher_MatchGame_NoMatch(t *testing.T) {
	game := testutil.MustParseGame(t, `
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. d4 d5 *
`)

	pm := NewPositionMatcher()
	// Sicilian Defense FEN - won't appear in a Queen's Pawn game
	err := pm.AddFEN("rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w KQkq c6 0 2", "Sicilian")
	if err != nil {
		t.Fatal(err)
	}

	match := pm.MatchGame(game)
	if match != nil {
		t.Error("expected no match")
	}
}

func TestPositionMatcher_MatchGame_EmptyPatterns(t *testing.T) {
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

	pm := NewPositionMatcher()
	match := pm.MatchGame(game)
	if match != nil {
		t.Error("expected nil match with no patterns")
	}
}

func TestPositionMatcher_MatchGame_PatternWithWildcards(t *testing.T) {
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

	pm := NewPositionMatcher()
	// Pattern that matches initial position: rank 8 = rnbqkbnr, rank 1 = RNBQKBNR
	// Using wildcards for middle ranks
	pm.AddPattern("rnbqkbnr/pppppppp/*/*/*/*/*/RNBQKBNR", "initial-like", false)

	match := pm.MatchGame(game)
	if match == nil {
		t.Fatal("expected wildcard pattern match")
	}
	if match.Label != "initial-like" {
		t.Errorf("label = %q, want %q", match.Label, "initial-like")
	}
}

func TestPositionMatcher_MatchPattern_EmptyRanks(t *testing.T) {
	pm := NewPositionMatcher()
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN)

	pattern := &FENPattern{
		Pattern: "",
		IsExact: false,
		ranks:   []string{},
	}

	if pm.matchPattern(board, pattern) {
		t.Error("empty ranks should not match")
	}
}

func TestPositionMatcher_GetStartingBoard_WithFENTag(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"FEN": "4k3/8/8/8/8/8/8/4K3 w - - 0 1",
		},
	}

	pm := NewPositionMatcher()
	// Pattern matching king-only position
	pm.AddPattern("____k___/8/8/8/8/8/8/____K___", "kings only", false)

	match := pm.MatchGame(game)
	if match == nil {
		t.Fatal("expected match with FEN tag starting position")
	}
}

func TestPositionMatcher_GetStartingBoard_InvalidFENTag(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"FEN": "invalid",
		},
	}

	pm := NewPositionMatcher()
	// Should fall back to initial position
	pm.AddPattern("rnbqkbnr/pppppppp/*/*/*/*/*/RNBQKBNR", "initial", false)

	match := pm.MatchGame(game)
	if match == nil {
		t.Error("expected match using fallback initial position")
	}
}

func TestPositionMatcher_MatchPattern_Wildcards(t *testing.T) {
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN)
	pm := NewPositionMatcher()

	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{
			name:    "exact initial position pattern",
			pattern: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR",
			want:    true,
		},
		{
			name:    "wildcards for empty ranks",
			pattern: "rnbqkbnr/pppppppp/????????/????????/????????/????????/PPPPPPPP/RNBQKBNR",
			want:    true,
		},
		{
			name:    "star for middle ranks",
			pattern: "rnbqkbnr/pppppppp/*/*/*/*/PPPPPPPP/RNBQKBNR",
			want:    true,
		},
		{
			name:    "wrong piece on rank 8",
			pattern: "Rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR",
			want:    false,
		},
		{
			name:    "A wildcard for white piece",
			pattern: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/ANBQKBNR",
			want:    true,
		},
		{
			name:    "a wildcard for black piece",
			pattern: "anbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR",
			want:    true,
		},
		{
			name:    "! on occupied square",
			pattern: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/!NBQKBNR",
			want:    true,
		},
		{
			name:    "! fails on empty",
			pattern: "rnbqkbnr/pppppppp/!7/8/8/8/PPPPPPPP/RNBQKBNR",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &FENPattern{
				Pattern: tt.pattern,
				IsExact: false,
				ranks:   splitRanks(tt.pattern),
			}
			got := pm.matchPattern(board, p)
			if got != tt.want {
				t.Errorf("matchPattern(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

// splitRanks is a test helper to split pattern into ranks
func splitRanks(pattern string) []string {
	if pattern == "" {
		return nil
	}
	result := []string{}
	start := 0
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '/' {
			result = append(result, pattern[start:i])
			start = i + 1
		}
	}
	result = append(result, pattern[start:])
	return result
}

func TestPositionMatcher_AddPattern_InvertMatchesFlipped(t *testing.T) {
	// Create a pattern for white pawns on rank 2 and verify the inverted
	// version matches black pawns on rank 7 (which becomes rank 2 after inversion)
	pm := NewPositionMatcher()

	// This pattern: rank8=specific, rest wildcard
	// After inversion: case swap + rank reversal
	pm.AddPattern("????k???/*/*/*/*/*/*/*", "black king on rank 8", true)

	// The inverted pattern should have white king on rank 1
	if pm.PatternCount() != 2 {
		t.Fatalf("PatternCount() = %d, want 2", pm.PatternCount())
	}

	// Verify the inverted pattern string
	invertedPat := invertPattern("????k???/*/*/*/*/*/*/*")
	// After inversion: ranks are reversed and case swapped
	// Original: "????k???/*/*/*/*/*/*/*"
	// Case swap: "????K???/*/*/*/*/*/*/*"
	// Rank reversal: "*/*/*/*/*/*/????K???"  (but only 8 ranks split by /)
	// Actually with 8 parts: ????k???/*/*/*/*/*/*/*, case swap gives ????K???/*/*/*/*/*/*/* and reverse gives */*/*/*/*/*/????K???
	want := "*/*/*/*/*/*/*/????K???"
	if invertedPat != want {
		t.Errorf("invertPattern() = %q, want %q", invertedPat, want)
	}
}

func TestPositionMatcher_MatchGame_WithMoves(t *testing.T) {
	// Test matching after specific moves are played
	game := testutil.MustParseGame(t, `
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 *
`)

	pm := NewPositionMatcher()
	// After 1. e4 e5, pawns have moved
	// Rank 4 should have white pawn on e4, rank 5 should have black pawn on e5
	pm.AddPattern("rnbqkbnr/pppp_ppp/8/4p3/4P3/8/PPPP_PPP/RNBQKBNR", "e4 e5", false)

	match := pm.MatchGame(game)
	if match == nil {
		t.Fatal("expected pattern match after 1. e4 e5")
	}
	if match.Label != "e4 e5" {
		t.Errorf("label = %q, want %q", match.Label, "e4 e5")
	}
}

func TestPositionMatcher_MatchGame_MatchAtInitial(t *testing.T) {
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

	pm := NewPositionMatcher()
	// Match initial position exactly
	err := pm.AddFEN(engine.InitialFEN, "start")
	if err != nil {
		t.Fatal(err)
	}

	match := pm.MatchGame(game)
	if match == nil {
		t.Fatal("expected match at initial position")
	}
	if match.Label != "start" {
		t.Errorf("label = %q, want %q", match.Label, "start")
	}
}

func TestPositionMatcher_PatternMoreThan8Ranks(t *testing.T) {
	// Pattern with more than 8 ranks - extras should be ignored
	pm := NewPositionMatcher()
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN)

	p := &FENPattern{
		Pattern: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR/extra",
		IsExact: false,
		ranks:   splitRanks("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR/extra"),
	}

	// Should still match (extra ranks beyond 8 are ignored via the break)
	got := pm.matchPattern(board, p)
	if !got {
		t.Error("pattern with >8 ranks should still match (extras ignored)")
	}
}

func TestMatchRank_DefaultCase_SpecificPiece(t *testing.T) {
	// Test matching specific piece characters (default switch case)
	tests := []struct {
		board   string
		pattern string
		want    bool
	}{
		{"R_______", "R_______", true},
		{"R_______", "N_______", false},
		{"p_______", "p_______", true},
		{"p_______", "P_______", false},
	}
	for _, tt := range tests {
		got := matchRank(tt.board, tt.pattern)
		if got != tt.want {
			t.Errorf("matchRank(%q, %q) = %v, want %v", tt.board, tt.pattern, got, tt.want)
		}
	}
}

func TestMatchRank_StarAtEndNoMore(t *testing.T) {
	// Star at end of pattern with nothing after - should match rest
	got := matchRank("RNBQKBNR", "RNB*")
	if !got {
		t.Error("expected true for 'RNB*' matching 'RNBQKBNR'")
	}
}

func TestMatchRank_StarRecursive(t *testing.T) {
	// Star with specific char after - must find the char
	got := matchRank("RNBQKBNR", "*N*R")
	if !got {
		t.Error("expected true for '*N*R' matching 'RNBQKBNR'")
	}

	got2 := matchRank("RNBQKBNR", "*Z*")
	if got2 {
		t.Error("expected false for '*Z*' matching 'RNBQKBNR'")
	}
}

func TestPositionMatcher_FENPattern_IsExact(t *testing.T) {
	pm := NewPositionMatcher()
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN)

	// Add both an exact FEN and a pattern
	err := pm.AddFEN(engine.InitialFEN, "exact")
	if err != nil {
		t.Fatal(err)
	}
	pm.AddPattern("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR", "pattern", false)

	// matchPosition should find exact hash match first
	match := pm.matchPosition(board)
	if match == nil {
		t.Fatal("expected match")
	}
	if match.Label != "exact" {
		t.Errorf("label = %q, want %q (hash match should take priority)", match.Label, "exact")
	}
}

func TestMatchRank_UnderscoreFailsOnPiece(t *testing.T) {
	got := matchRank("R_______", "________")
	if got {
		t.Error("underscore should not match piece")
	}
}

func TestMatchRank_DigitPartialMatch(t *testing.T) {
	// "2" means 2 empty squares
	got := matchRank("__NBQKBNR", "2NBQKBNR")
	if !got {
		t.Error("expected true for digit matching empty squares")
	}

	got2 := matchRank("R_NBQKBNR", "2NBQKBNR")
	if got2 {
		t.Error("expected false - first square is not empty")
	}
}
