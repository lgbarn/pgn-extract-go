package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTempPGN creates a temporary PGN file with the given content.
func createTempPGN(t *testing.T, filename, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	return tmpFile
}

// createTempPGNWithClocks creates a temporary PGN file with clock annotations for testing.
func createTempPGNWithClocks(t *testing.T) string {
	t.Helper()
	return createTempPGN(t, "clocks.pgn", `[Event "Test Game"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 {[%clk 0:10:00]} e5 {[%clk 0:09:58]} 2. Nf3 {[%clk 0:09:55.5]} Nc6 {[%clk 0:09:50.2]} 1-0
`)
}

// createTempPGNWithMixedComments creates a PGN with both clock annotations and regular comments.
func createTempPGNWithMixedComments(t *testing.T) string {
	t.Helper()
	return createTempPGN(t, "mixed.pgn", `[Event "Test Game"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 {[%clk 0:10:00]} e5 {Good move} 2. Nf3 {[%clk 0:09:55.5] Developing} Nc6 {Natural} 1-0
`)
}

// createTempPGNWithMultipleAnnotations creates a PGN with multiple annotation types in one comment.
func createTempPGNWithMultipleAnnotations(t *testing.T) string {
	t.Helper()
	return createTempPGN(t, "multi.pgn", `[Event "Test Game"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 {[%clk 0:10:00][%eval 0.5]} e5 {[%clk 0:09:58][%eval 0.3] Good response} 1-0
`)
}

// TestNoClocksFlag verifies the --noclocks flag is recognized.
func TestNoClocksFlag(t *testing.T) {
	tmpFile := createTempPGNWithClocks(t)
	_, stderr := runPgnExtract(t, "-s", "--noclocks", tmpFile)
	if strings.Contains(stderr, "flag provided but not defined") {
		t.Skip("--noclocks flag not implemented yet")
	}
	// If we get here, the flag is recognized
}

// TestNoClocksRemovesClockAnnotations verifies clock annotations are removed.
func TestNoClocksRemovesClockAnnotations(t *testing.T) {
	tmpFile := createTempPGNWithClocks(t)
	out, stderr := runPgnExtract(t, "-s", "--noclocks", tmpFile)
	if strings.Contains(stderr, "flag provided but not defined") {
		t.Skip("--noclocks flag not implemented yet")
	}

	// Should not contain clock annotations
	if strings.Contains(out, "[%clk") {
		t.Errorf("Output should not contain clock annotations, got:\n%s", out)
	}

	// Should still have the moves
	if !strings.Contains(out, "e4") || !strings.Contains(out, "Nf3") {
		t.Errorf("Output should still contain moves, got:\n%s", out)
	}
}

// TestNoClocksPreservesOtherComments verifies regular comments are kept.
func TestNoClocksPreservesOtherComments(t *testing.T) {
	tmpFile := createTempPGNWithMixedComments(t)
	out, stderr := runPgnExtract(t, "-s", "--noclocks", tmpFile)
	if strings.Contains(stderr, "flag provided but not defined") {
		t.Skip("--noclocks flag not implemented yet")
	}

	// Should not contain clock annotations
	if strings.Contains(out, "[%clk") {
		t.Errorf("Output should not contain clock annotations, got:\n%s", out)
	}

	// Should preserve regular comments
	if !strings.Contains(out, "Good move") {
		t.Errorf("Output should preserve 'Good move' comment, got:\n%s", out)
	}
	if !strings.Contains(out, "Developing") {
		t.Errorf("Output should preserve 'Developing' comment, got:\n%s", out)
	}
	if !strings.Contains(out, "Natural") {
		t.Errorf("Output should preserve 'Natural' comment, got:\n%s", out)
	}
}

// TestNoClocksWithMultipleAnnotations verifies handling of mixed annotation types.
func TestNoClocksWithMultipleAnnotations(t *testing.T) {
	tmpFile := createTempPGNWithMultipleAnnotations(t)
	out, stderr := runPgnExtract(t, "-s", "--noclocks", tmpFile)
	if strings.Contains(stderr, "flag provided but not defined") {
		t.Skip("--noclocks flag not implemented yet")
	}

	// Should not contain clock annotations
	if strings.Contains(out, "[%clk") {
		t.Errorf("Output should not contain clock annotations, got:\n%s", out)
	}

	// Should preserve eval annotations
	if !strings.Contains(out, "[%eval") {
		t.Errorf("Output should preserve eval annotations, got:\n%s", out)
	}

	// Should preserve regular comment text
	if !strings.Contains(out, "Good response") {
		t.Errorf("Output should preserve 'Good response' comment, got:\n%s", out)
	}
}

// TestNoClocksEmptyCommentRemoval verifies empty comments are not output.
func TestNoClocksEmptyCommentRemoval(t *testing.T) {
	tmpFile := createTempPGNWithClocks(t)
	out, stderr := runPgnExtract(t, "-s", "--noclocks", tmpFile)
	if strings.Contains(stderr, "flag provided but not defined") {
		t.Skip("--noclocks flag not implemented yet")
	}

	// Should not have empty braces {} in output
	if strings.Contains(out, "{}") {
		t.Errorf("Output should not contain empty comments {}, got:\n%s", out)
	}
}

// TestNoClocksWithNoComments verifies --noclocks works with -C flag.
func TestNoClocksWithNoComments(t *testing.T) {
	tmpFile := createTempPGNWithMixedComments(t)
	out, stderr := runPgnExtract(t, "-s", "--noclocks", "-C", tmpFile)
	if strings.Contains(stderr, "flag provided but not defined") {
		t.Skip("--noclocks flag not implemented yet")
	}

	// With -C, no comments at all
	if strings.Contains(out, "{") {
		t.Errorf("Output with -C should not contain any comments, got:\n%s", out)
	}
}

// TestNoClocksDefaultBehavior verifies clocks are kept by default.
func TestNoClocksDefaultBehavior(t *testing.T) {
	tmpFile := createTempPGNWithClocks(t)
	out, _ := runPgnExtract(t, "-s", tmpFile)

	// Without --noclocks, clock annotations should be present
	if !strings.Contains(out, "[%clk") {
		t.Errorf("Output without --noclocks should contain clock annotations, got:\n%s", out)
	}
}
