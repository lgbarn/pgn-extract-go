package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestSentinelErrors verifies that sentinel errors are properly defined
// and can be checked with errors.Is()
func TestSentinelErrors_Are(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		sentinel error
	}{
		{"ErrInvalidFEN", ErrInvalidFEN, ErrInvalidFEN},
		{"ErrIllegalMove", ErrIllegalMove, ErrIllegalMove},
		{"ErrParseFailure", ErrParseFailure, ErrParseFailure},
		{"ErrCQLSyntax", ErrCQLSyntax, ErrCQLSyntax},
		{"ErrInvalidConfig", ErrInvalidConfig, ErrInvalidConfig},
		{"ErrDuplicateGame", ErrDuplicateGame, ErrDuplicateGame},
		{"ErrMissingTag", ErrMissingTag, ErrMissingTag},
		{"ErrMaterialMismatch", ErrMaterialMismatch, ErrMaterialMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.sentinel) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.err, tt.sentinel)
			}
		})
	}
}

// TestSentinelErrors_Wrapping verifies wrapped sentinel errors can still be detected
func TestSentinelErrors_Wrapping(t *testing.T) {
	wrapped := fmt.Errorf("failed to parse position: %w", ErrInvalidFEN)

	if !errors.Is(wrapped, ErrInvalidFEN) {
		t.Errorf("errors.Is(wrapped, ErrInvalidFEN) = false, want true")
	}
}

// TestGameError_Error verifies the error message format
func TestGameError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *GameError
		contains []string
	}{
		{
			name: "full context",
			err: &GameError{
				Err:      ErrIllegalMove,
				GameNum:  5,
				PlyNum:   12,
				MoveText: "Nxe5",
				File:     "games.pgn",
				Line:     42,
			},
			contains: []string{"game 5", "ply 12", "Nxe5", "games.pgn", "42", "illegal move"},
		},
		{
			name: "minimal context",
			err: &GameError{
				Err:     ErrParseFailure,
				GameNum: 1,
			},
			contains: []string{"game 1", "parse failure"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, s := range tt.contains {
				if !containsIgnoreCase(msg, s) {
					t.Errorf("GameError.Error() = %q, should contain %q", msg, s)
				}
			}
		})
	}
}

// TestGameError_Unwrap verifies that GameError properly implements Unwrap
func TestGameError_Unwrap(t *testing.T) {
	gameErr := &GameError{
		Err:     ErrInvalidFEN,
		GameNum: 1,
		File:    "test.pgn",
	}

	// Unwrap should return the underlying error
	unwrapped := errors.Unwrap(gameErr)
	if !errors.Is(unwrapped, ErrInvalidFEN) {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, ErrInvalidFEN)
	}

	// errors.Is should work through the wrapper
	if !errors.Is(gameErr, ErrInvalidFEN) {
		t.Error("errors.Is(gameErr, ErrInvalidFEN) = false, want true")
	}
}

// TestGameError_As verifies that errors.As works with GameError
func TestGameError_As(t *testing.T) {
	gameErr := &GameError{
		Err:      ErrIllegalMove,
		GameNum:  3,
		PlyNum:   24,
		MoveText: "O-O-O",
	}

	// Wrap it further
	wrapped := fmt.Errorf("processing failed: %w", gameErr)

	// Should be able to extract GameError with errors.As
	var extractedErr *GameError
	if !errors.As(wrapped, &extractedErr) {
		t.Fatal("errors.As() could not extract GameError")
	}

	if extractedErr.GameNum != 3 {
		t.Errorf("extractedErr.GameNum = %d, want 3", extractedErr.GameNum)
	}
	if extractedErr.MoveText != "O-O-O" {
		t.Errorf("extractedErr.MoveText = %q, want %q", extractedErr.MoveText, "O-O-O")
	}
}

// TestParseError_Error verifies ParseError formatting
func TestParseError_Error(t *testing.T) {
	err := &ParseError{
		Err:      ErrParseFailure,
		File:     "tournament.pgn",
		Line:     100,
		Column:   15,
		Expected: "move number",
		Got:      "comment",
	}

	msg := err.Error()

	// Should contain file location
	if !containsIgnoreCase(msg, "tournament.pgn") {
		t.Errorf("ParseError.Error() should contain filename, got %q", msg)
	}
	if !containsIgnoreCase(msg, "100") {
		t.Errorf("ParseError.Error() should contain line number, got %q", msg)
	}
}

// TestParseError_Unwrap verifies ParseError implements Unwrap
func TestParseError_Unwrap(t *testing.T) {
	parseErr := &ParseError{
		Err:  ErrCQLSyntax,
		File: "query.cql",
		Line: 1,
	}

	if !errors.Is(parseErr, ErrCQLSyntax) {
		t.Error("errors.Is(parseErr, ErrCQLSyntax) = false, want true")
	}
}

// TestWrap verifies the Wrap helper function
func TestWrap(t *testing.T) {
	original := ErrInvalidFEN
	wrapped := Wrap(original, "parsing FEN string")

	if !errors.Is(wrapped, ErrInvalidFEN) {
		t.Error("Wrap should preserve the underlying error")
	}

	msg := wrapped.Error()
	if !containsIgnoreCase(msg, "parsing FEN string") {
		t.Errorf("Wrap should include context, got %q", msg)
	}
}

// TestWrapf verifies the Wrapf helper function
func TestWrapf(t *testing.T) {
	original := ErrIllegalMove
	wrapped := Wrapf(original, "move %d in game %d", 15, 3)

	if !errors.Is(wrapped, ErrIllegalMove) {
		t.Error("Wrapf should preserve the underlying error")
	}

	msg := wrapped.Error()
	if !containsIgnoreCase(msg, "move 15") {
		t.Errorf("Wrapf should include formatted context, got %q", msg)
	}
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
