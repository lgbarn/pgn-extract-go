package matching

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/testutil"
)

// ---------------------------------------------------------------------------
// Helper: write a temp file with given content, return its path
// ---------------------------------------------------------------------------

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file %s: %v", path, err)
	}
	return path
}

// ---------------------------------------------------------------------------
// Standard PGN snippets used across tests
// ---------------------------------------------------------------------------

const italianGamePGN = `[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 *
`

const sicilianPGN = `[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 *
`

const shortGamePGN = `[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 *
`

// ===========================================================================
// Task 1: LoadFromFile and LoadPositionalFromFile tests
// ===========================================================================

func TestLoadFromFile_BasicMoveSequences(t *testing.T) {
	dir := t.TempDir()
	content := "1. e4 e5 2. Nf3 Nc6\n1. d4 d5 2. c4\n"
	path := writeTempFile(t, dir, "moves.txt", content)

	vm := NewVariationMatcher()
	if err := vm.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if len(vm.moveSequences) != 2 {
		t.Fatalf("expected 2 move sequences, got %d", len(vm.moveSequences))
	}

	// Move numbers should be stripped
	want0 := []string{"e4", "e5", "Nf3", "Nc6"}
	want1 := []string{"d4", "d5", "c4"}
	for i, w := range want0 {
		if vm.moveSequences[0][i] != w {
			t.Errorf("seq[0][%d] = %q, want %q", i, vm.moveSequences[0][i], w)
		}
	}
	for i, w := range want1 {
		if vm.moveSequences[1][i] != w {
			t.Errorf("seq[1][%d] = %q, want %q", i, vm.moveSequences[1][i], w)
		}
	}
}

func TestLoadFromFile_CommentsAndEmptyLines(t *testing.T) {
	dir := t.TempDir()
	content := "# This is a comment\n\n1. e4 e5\n# Another comment\n\n1. d4 d5\n"
	path := writeTempFile(t, dir, "moves.txt", content)

	vm := NewVariationMatcher()
	if err := vm.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if len(vm.moveSequences) != 2 {
		t.Errorf("expected 2 move sequences (comments and blanks skipped), got %d", len(vm.moveSequences))
	}
}

func TestLoadFromFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "empty.txt", "")

	vm := NewVariationMatcher()
	if err := vm.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile on empty file: %v", err)
	}

	if len(vm.moveSequences) != 0 {
		t.Errorf("expected 0 move sequences from empty file, got %d", len(vm.moveSequences))
	}
}

func TestLoadFromFile_OnlyComments(t *testing.T) {
	dir := t.TempDir()
	content := "# comment 1\n# comment 2\n"
	path := writeTempFile(t, dir, "comments.txt", content)

	vm := NewVariationMatcher()
	if err := vm.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if len(vm.moveSequences) != 0 {
		t.Errorf("expected 0 move sequences from comment-only file, got %d", len(vm.moveSequences))
	}
}

func TestLoadFromFile_NonExistentFile(t *testing.T) {
	vm := NewVariationMatcher()
	err := vm.LoadFromFile("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestLoadPositionalFromFile_BasicSequences(t *testing.T) {
	dir := t.TempDir()
	// Two sequences separated by a blank line
	content := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR\nrnbqkbnr/pppp1ppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR\n\nrnbqkbnr/pppppppp/8/8/3P4/8/PPP1PPPP/RNBQKBNR\n"
	path := writeTempFile(t, dir, "positions.txt", content)

	vm := NewVariationMatcher()
	if err := vm.LoadPositionalFromFile(path); err != nil {
		t.Fatalf("LoadPositionalFromFile: %v", err)
	}

	if len(vm.positionSequences) != 2 {
		t.Fatalf("expected 2 position sequences, got %d", len(vm.positionSequences))
	}

	if len(vm.positionSequences[0]) != 2 {
		t.Errorf("first sequence should have 2 positions, got %d", len(vm.positionSequences[0]))
	}
	if len(vm.positionSequences[1]) != 1 {
		t.Errorf("second sequence should have 1 position, got %d", len(vm.positionSequences[1]))
	}
}

func TestLoadPositionalFromFile_CommentsAndBlanks(t *testing.T) {
	dir := t.TempDir()
	content := "# A positional sequence\nrnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR\n\n# Another\nrnbqkbnr/pppppppp/8/8/3P4/8/PPP1PPPP/RNBQKBNR\n"
	path := writeTempFile(t, dir, "positions.txt", content)

	vm := NewVariationMatcher()
	if err := vm.LoadPositionalFromFile(path); err != nil {
		t.Fatalf("LoadPositionalFromFile: %v", err)
	}

	if len(vm.positionSequences) != 2 {
		t.Errorf("expected 2 position sequences, got %d", len(vm.positionSequences))
	}
}

func TestLoadPositionalFromFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "empty.txt", "")

	vm := NewVariationMatcher()
	if err := vm.LoadPositionalFromFile(path); err != nil {
		t.Fatalf("LoadPositionalFromFile: %v", err)
	}

	if len(vm.positionSequences) != 0 {
		t.Errorf("expected 0 position sequences, got %d", len(vm.positionSequences))
	}
}

func TestLoadPositionalFromFile_NonExistentFile(t *testing.T) {
	vm := NewVariationMatcher()
	err := vm.LoadPositionalFromFile("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestLoadPositionalFromFile_TrailingSequenceNoBlankLine(t *testing.T) {
	dir := t.TempDir()
	// File ends without a trailing blank line -- the last sequence must still be captured
	content := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR"
	path := writeTempFile(t, dir, "positions.txt", content)

	vm := NewVariationMatcher()
	if err := vm.LoadPositionalFromFile(path); err != nil {
		t.Fatalf("LoadPositionalFromFile: %v", err)
	}

	if len(vm.positionSequences) != 1 {
		t.Errorf("expected 1 position sequence (trailing), got %d", len(vm.positionSequences))
	}
}

func TestLoadPositionalFromFile_MultipleBlankLinesBetweenSequences(t *testing.T) {
	dir := t.TempDir()
	content := "pos1\n\n\n\npos2\n"
	path := writeTempFile(t, dir, "positions.txt", content)

	vm := NewVariationMatcher()
	if err := vm.LoadPositionalFromFile(path); err != nil {
		t.Fatalf("LoadPositionalFromFile: %v", err)
	}

	// Multiple blank lines should not create empty sequences
	if len(vm.positionSequences) != 2 {
		t.Errorf("expected 2 position sequences, got %d", len(vm.positionSequences))
	}
}

// ===========================================================================
// Task 2: Move sequence matching tests
// ===========================================================================

func TestParseMoveSequence(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "standard notation with move numbers",
			input: "1. e4 e5 2. Nf3 Nc6",
			want:  []string{"e4", "e5", "Nf3", "Nc6"},
		},
		{
			name:  "no move numbers",
			input: "e4 e5 Nf3 Nc6",
			want:  []string{"e4", "e5", "Nf3", "Nc6"},
		},
		{
			name:  "black continuation with ellipsis",
			input: "1... e5 2. Nf3",
			want:  []string{"e5", "Nf3"},
		},
		{
			name:  "single move",
			input: "1. e4",
			want:  []string{"e4"},
		},
		{
			name:  "empty line",
			input: "",
			want:  nil,
		},
		{
			name:  "only move numbers",
			input: "1. 2. 3.",
			want:  nil,
		},
		{
			name:  "moves with annotations",
			input: "1. e4! e5? 2. Nf3+",
			want:  []string{"e4!", "e5?", "Nf3+"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseMoveSequence(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseMoveSequence(%q) returned %d moves, want %d: %v",
					tt.input, len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseMoveSequence(%q)[%d] = %q, want %q",
						tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNormalizeMove(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"e4", "e4"},
		{"Nf3+", "Nf3"},
		{"Qh5#", "Qh5"},
		{"e4!", "e4"},
		{"e4!!", "e4"},
		{"e4?", "e4"},
		{"e4??", "e4"},
		{"Nf3+!", "Nf3"},
		{"  e4  ", "e4"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeMove(tt.input)
			if got != tt.want {
				t.Errorf("normalizeMove(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchMoveSequence_MatchAtStart(t *testing.T) {
	game := testutil.MustParseGame(t, italianGamePGN)

	vm := NewVariationMatcher()

	// Sequence that matches from the start: e4 e5 Nf3
	if !vm.matchMoveSequence(game, []string{"e4", "e5", "Nf3"}) {
		t.Error("expected match for opening sequence e4 e5 Nf3")
	}
}

func TestMatchMoveSequence_MatchAnywhere(t *testing.T) {
	game := testutil.MustParseGame(t, italianGamePGN)

	vm := NewVariationMatcher()

	// Sequence later in the game: Nf3 Nc6 Bc4
	if !vm.matchMoveSequence(game, []string{"Nf3", "Nc6", "Bc4"}) {
		t.Error("expected match for mid-game sequence Nf3 Nc6 Bc4")
	}
}

func TestMatchMoveSequence_NoMatch(t *testing.T) {
	game := testutil.MustParseGame(t, italianGamePGN)

	vm := NewVariationMatcher()

	// Sequence not in the game
	if vm.matchMoveSequence(game, []string{"d4", "d5", "c4"}) {
		t.Error("expected no match for d4 d5 c4 in Italian Game")
	}
}

func TestMatchMoveSequence_EmptySequence(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	if !vm.matchMoveSequence(game, []string{}) {
		t.Error("expected match for empty sequence")
	}
}

func TestMatchMoveSequence_FullGameMatch(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	if !vm.matchMoveSequence(game, []string{"e4", "e5"}) {
		t.Error("expected match for complete game moves")
	}
}

func TestMatchMoveSequence_SequenceLongerThanGame(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	if vm.matchMoveSequence(game, []string{"e4", "e5", "Nf3", "Nc6"}) {
		t.Error("expected no match when sequence is longer than game")
	}
}

func TestMatchMoveSequence_WithAnnotations(t *testing.T) {
	game := testutil.MustParseGame(t, italianGamePGN)
	vm := NewVariationMatcher()

	// Annotations should be stripped during comparison
	if !vm.matchMoveSequence(game, []string{"e4+", "e5!", "Nf3#"}) {
		t.Error("expected match despite annotations in search sequence")
	}
}

func TestMatchMoveSequence_ResetOnMismatch(t *testing.T) {
	// Game: e4 e5 Nf3 Nc6 Bc4 Bc5
	// Search: e5 Nf3 Nc6
	// After matching e4 (no), e5 should start matching
	game := testutil.MustParseGame(t, italianGamePGN)
	vm := NewVariationMatcher()

	if !vm.matchMoveSequence(game, []string{"e5", "Nf3", "Nc6"}) {
		t.Error("expected match for sequence starting at move 2")
	}
}

func TestMatchMoveSequence_ResetRestartsFromCurrent(t *testing.T) {
	// Build a game with moves: a3 a6 a3 a6 b3
	// Search for: a3 a6 b3
	// The first a3 a6 matches, then b3 mismatches at a3 -- sequence should
	// restart and eventually NOT find a3 a6 b3 contiguously.
	game := chess.NewGame()
	moves := []string{"a3", "a6", "a3", "a6", "b3"}
	var prev *chess.Move
	for _, m := range moves {
		mv := chess.NewMove()
		mv.Text = m
		if prev != nil {
			prev.Next = mv
			mv.Prev = prev
		} else {
			game.Moves = mv
		}
		prev = mv
	}

	vm := NewVariationMatcher()
	// a3 a6 b3 is NOT contiguous in the game -- the actual sequence is a3 a6 a3 a6 b3
	// After first a3, a6 matches. Then a3 mismatches b3, reset. a3 matches, a6 matches, b3 matches!
	// Actually: positions a3(match a3) -> a6(match a6) -> a3(mismatch b3, reset, check a3==a3 yes, seqIdx=1)
	// -> a6(match a6, seqIdx=2) -> b3(match b3, seqIdx=3, done!) => TRUE
	if !vm.matchMoveSequence(game, []string{"a3", "a6", "b3"}) {
		t.Error("expected match for a3 a6 b3 in game a3 a6 a3 a6 b3")
	}
}

// ===========================================================================
// Task 3: Positional sequence matching and config tests
// ===========================================================================

func TestMatchesFENPosition_FullFEN(t *testing.T) {
	board, err := engine.NewBoardFromFEN(engine.InitialFEN)
	if err != nil {
		t.Fatalf("NewBoardFromFEN: %v", err)
	}

	// Full FEN should match on piece placement
	if !matchesFENPosition(board, engine.InitialFEN) {
		t.Error("expected initial board to match initial FEN")
	}
}

func TestMatchesFENPosition_PartialFEN(t *testing.T) {
	board, err := engine.NewBoardFromFEN(engine.InitialFEN)
	if err != nil {
		t.Fatalf("NewBoardFromFEN: %v", err)
	}

	// Partial FEN (piece placement only) should match
	piecePlacement := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"
	if !matchesFENPosition(board, piecePlacement) {
		t.Error("expected initial board to match partial FEN (piece placement only)")
	}
}

func TestMatchesFENPosition_NoMatch(t *testing.T) {
	board, err := engine.NewBoardFromFEN(engine.InitialFEN)
	if err != nil {
		t.Fatalf("NewBoardFromFEN: %v", err)
	}

	// After e4 the position is different
	afterE4 := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR"
	if matchesFENPosition(board, afterE4) {
		t.Error("expected initial board NOT to match position after e4")
	}
}

func TestMatchesFENPosition_EmptyFEN(t *testing.T) {
	board, err := engine.NewBoardFromFEN(engine.InitialFEN)
	if err != nil {
		t.Fatalf("NewBoardFromFEN: %v", err)
	}

	if matchesFENPosition(board, "") {
		t.Error("expected no match for empty FEN string")
	}
}

func TestMatchPositionSequence_EmptySequence(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	if !vm.matchPositionSequence(game, []string{}) {
		t.Error("expected match for empty position sequence")
	}
}

func TestMatchPositionSequence_InitialPosition(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	// Match just the initial position
	initialPP := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"
	if !vm.matchPositionSequence(game, []string{initialPP}) {
		t.Error("expected match for initial position in any game")
	}
}

func TestMatchPositionSequence_AfterFirstMove(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	// Position after 1. e4
	afterE4 := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR"
	if !vm.matchPositionSequence(game, []string{afterE4}) {
		t.Error("expected match for position after 1. e4")
	}
}

func TestMatchPositionSequence_TwoPositionSequence(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	initialPP := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"
	afterE4 := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR"
	if !vm.matchPositionSequence(game, []string{initialPP, afterE4}) {
		t.Error("expected match for initial -> after e4 position sequence")
	}
}

func TestMatchPositionSequence_NoMatch(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	// Position after d4 should not appear in 1.e4 e5 game
	afterD4 := "rnbqkbnr/pppppppp/8/8/3P4/8/PPP1PPPP/RNBQKBNR"
	if vm.matchPositionSequence(game, []string{afterD4}) {
		t.Error("expected no match for d4 position in e4 e5 game")
	}
}

func TestMatchPositionSequence_SequenceTooLong(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	// Three positions but game only has 2 moves
	initialPP := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"
	afterE4 := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR"
	afterE4E5 := "rnbqkbnr/pppp1ppp/8/8/4Pp2/8/PPPP1PPP/RNBQKBNR" // wrong, just need something
	// Even with correct FEN, if we add a 4th position that doesn't exist, should fail
	if vm.matchPositionSequence(game, []string{initialPP, afterE4, afterE4E5, "fake/position"}) {
		t.Error("expected no match for position sequence longer than game")
	}
}

// ---------------------------------------------------------------------------
// MatchGame integration tests
// ---------------------------------------------------------------------------

func TestMatchGame_NoCriteria(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	// With no criteria, MatchGame returns true
	if !vm.MatchGame(game) {
		t.Error("expected MatchGame to return true when no criteria are set")
	}
}

func TestMatchGame_WithMoveSequence(t *testing.T) {
	game := testutil.MustParseGame(t, italianGamePGN)
	vm := NewVariationMatcher()
	vm.AddMoveSequence([]string{"e4", "e5", "Nf3"})

	if !vm.MatchGame(game) {
		t.Error("expected MatchGame to return true for matching move sequence")
	}
}

func TestMatchGame_WithNonMatchingMoveSequence(t *testing.T) {
	game := testutil.MustParseGame(t, italianGamePGN)
	vm := NewVariationMatcher()
	vm.AddMoveSequence([]string{"d4", "d5", "c4"})

	if vm.MatchGame(game) {
		t.Error("expected MatchGame to return false for non-matching move sequence")
	}
}

func TestMatchGame_MultipleSequencesOneMatches(t *testing.T) {
	game := testutil.MustParseGame(t, italianGamePGN)
	vm := NewVariationMatcher()
	vm.AddMoveSequence([]string{"d4", "d5"})     // does not match
	vm.AddMoveSequence([]string{"e4", "e5"})      // matches
	vm.AddMoveSequence([]string{"c4", "e5"})      // does not match

	if !vm.MatchGame(game) {
		t.Error("expected MatchGame to return true when at least one sequence matches")
	}
}

func TestMatchGame_AllSequencesFail(t *testing.T) {
	game := testutil.MustParseGame(t, italianGamePGN)
	vm := NewVariationMatcher()
	vm.AddMoveSequence([]string{"d4", "d5"})
	vm.AddMoveSequence([]string{"c4", "e5"})

	if vm.MatchGame(game) {
		t.Error("expected MatchGame to return false when no sequences match")
	}
}

func TestMatchGame_WithPositionSequence(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	afterE4 := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR"
	vm.positionSequences = append(vm.positionSequences, []string{afterE4})

	if !vm.MatchGame(game) {
		t.Error("expected MatchGame to return true for matching position sequence")
	}
}

func TestMatchGame_MoveSequenceTakesPriority(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	// Add a matching move sequence
	vm.AddMoveSequence([]string{"e4", "e5"})
	// Add a non-matching position sequence
	vm.positionSequences = append(vm.positionSequences, []string{"fake/position"})

	// Move sequence matches first, so MatchGame should return true
	if !vm.MatchGame(game) {
		t.Error("expected MatchGame to return true when move sequence matches")
	}
}

// ---------------------------------------------------------------------------
// Match interface method test
// ---------------------------------------------------------------------------

func TestMatch_DelegatesToMatchGame(t *testing.T) {
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()
	vm.AddMoveSequence([]string{"e4", "e5"})

	if !vm.Match(game) {
		t.Error("expected Match() to delegate to MatchGame and return true")
	}
}

// ---------------------------------------------------------------------------
// Configuration tests
// ---------------------------------------------------------------------------

func TestSetMatchAnywhere(t *testing.T) {
	vm := NewVariationMatcher()

	if vm.matchAnywhere {
		t.Error("expected matchAnywhere to default to false")
	}

	vm.SetMatchAnywhere(true)
	if !vm.matchAnywhere {
		t.Error("expected matchAnywhere to be true after SetMatchAnywhere(true)")
	}

	vm.SetMatchAnywhere(false)
	if vm.matchAnywhere {
		t.Error("expected matchAnywhere to be false after SetMatchAnywhere(false)")
	}
}

func TestHasCriteria(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*VariationMatcher)
		expected bool
	}{
		{
			name:     "no criteria",
			setup:    func(vm *VariationMatcher) {},
			expected: false,
		},
		{
			name: "with move sequence",
			setup: func(vm *VariationMatcher) {
				vm.AddMoveSequence([]string{"e4", "e5"})
			},
			expected: true,
		},
		{
			name: "with position sequence",
			setup: func(vm *VariationMatcher) {
				vm.positionSequences = append(vm.positionSequences, []string{"some/fen"})
			},
			expected: true,
		},
		{
			name: "with both",
			setup: func(vm *VariationMatcher) {
				vm.AddMoveSequence([]string{"e4"})
				vm.positionSequences = append(vm.positionSequences, []string{"some/fen"})
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewVariationMatcher()
			tt.setup(vm)
			if got := vm.HasCriteria(); got != tt.expected {
				t.Errorf("HasCriteria() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAddMoveSequence(t *testing.T) {
	vm := NewVariationMatcher()

	vm.AddMoveSequence([]string{"e4", "e5"})
	if len(vm.moveSequences) != 1 {
		t.Fatalf("expected 1 move sequence, got %d", len(vm.moveSequences))
	}

	vm.AddMoveSequence([]string{"d4", "d5"})
	if len(vm.moveSequences) != 2 {
		t.Fatalf("expected 2 move sequences, got %d", len(vm.moveSequences))
	}

	if vm.moveSequences[0][0] != "e4" || vm.moveSequences[1][0] != "d4" {
		t.Error("move sequences not stored in order")
	}
}

func TestName(t *testing.T) {
	vm := NewVariationMatcher()
	if vm.Name() != "VariationMatcher" {
		t.Errorf("Name() = %q, want %q", vm.Name(), "VariationMatcher")
	}
}

func TestNewVariationMatcher(t *testing.T) {
	vm := NewVariationMatcher()
	if vm == nil {
		t.Fatal("NewVariationMatcher() returned nil")
	}
	if vm.moveSequences != nil {
		t.Error("expected moveSequences to be nil initially")
	}
	if vm.positionSequences != nil {
		t.Error("expected positionSequences to be nil initially")
	}
	if vm.matchAnywhere {
		t.Error("expected matchAnywhere to be false initially")
	}
}

// ---------------------------------------------------------------------------
// LoadFromFile integration: load then match
// ---------------------------------------------------------------------------

func TestLoadFromFile_ThenMatch(t *testing.T) {
	dir := t.TempDir()
	content := "1. e4 e5 2. Nf3\n"
	path := writeTempFile(t, dir, "moves.txt", content)

	vm := NewVariationMatcher()
	if err := vm.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	game := testutil.MustParseGame(t, italianGamePGN)
	if !vm.MatchGame(game) {
		t.Error("expected match after loading move file and matching Italian Game")
	}

	game2 := testutil.MustParseGame(t, sicilianPGN)
	if vm.MatchGame(game2) {
		t.Error("expected no match for Sicilian with Italian opening sequence")
	}
}

// ---------------------------------------------------------------------------
// Positional file integration: load then match
// ---------------------------------------------------------------------------

func TestLoadPositionalFromFile_ThenMatch(t *testing.T) {
	dir := t.TempDir()
	// Position after 1. e4 (piece placement only)
	afterE4 := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR"
	content := afterE4 + "\n"
	path := writeTempFile(t, dir, "positions.txt", content)

	vm := NewVariationMatcher()
	if err := vm.LoadPositionalFromFile(path); err != nil {
		t.Fatalf("LoadPositionalFromFile: %v", err)
	}

	game := testutil.MustParseGame(t, shortGamePGN)
	if !vm.MatchGame(game) {
		t.Error("expected position match after loading positional file")
	}
}

func TestMatchPositionSequence_SinglePositionMatchesInitial(t *testing.T) {
	// A single-element sequence that matches the initial position should return
	// true immediately (covers the seqIdx >= len(seq) branch after initial check).
	game := testutil.MustParseGame(t, shortGamePGN)
	vm := NewVariationMatcher()

	initialPP := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"
	if !vm.matchPositionSequence(game, []string{initialPP}) {
		t.Error("expected match for single-position sequence matching initial position")
	}
}

func TestMatchGame_NilMoves(t *testing.T) {
	game := chess.NewGame()
	vm := NewVariationMatcher()
	vm.AddMoveSequence([]string{"e4"})

	if vm.MatchGame(game) {
		t.Error("expected no match for game with nil moves")
	}
}
