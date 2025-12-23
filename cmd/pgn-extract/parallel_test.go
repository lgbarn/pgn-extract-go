package main

import (
	"sort"
	"strings"
	"testing"
)

// extractGameEvents extracts Event tag values for comparison (order-independent).
func extractGameEvents(pgn string) []string {
	var events []string
	lines := strings.Split(pgn, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "[Event ") {
			events = append(events, line)
		}
	}
	return events
}

// extractGameResults extracts full game results (Event + Result) for comparison.
func extractGameResults(pgn string) []string {
	var results []string
	lines := strings.Split(pgn, "\n")
	var currentEvent, currentResult string
	for _, line := range lines {
		if strings.HasPrefix(line, "[Event ") {
			currentEvent = line
		}
		if strings.HasPrefix(line, "[Result ") {
			currentResult = line
			if currentEvent != "" {
				results = append(results, currentEvent+"|"+currentResult)
			}
		}
	}
	return results
}

// TestParallelMatchesSequential verifies that parallel processing produces
// the same games as sequential processing (order may differ).
func TestParallelMatchesSequential(t *testing.T) {
	// Run with workers=1 (sequential baseline)
	seqOut, seqErr := runPgnExtract(t, "-s", "--workers", "1", inputFile("fischer.pgn"))
	if strings.Contains(seqErr, "flag provided but not defined") {
		t.Skip("--workers flag not implemented yet")
	}
	seqGames := extractGameResults(seqOut)

	// Run with workers=4 (parallel)
	parOut, _ := runPgnExtract(t, "-s", "--workers", "4", inputFile("fischer.pgn"))
	parGames := extractGameResults(parOut)

	// Sort both for comparison (order may differ)
	sort.Strings(seqGames)
	sort.Strings(parGames)

	if len(seqGames) != len(parGames) {
		t.Errorf("Game count mismatch: sequential=%d, parallel=%d", len(seqGames), len(parGames))
		return
	}

	for i := range seqGames {
		if seqGames[i] != parGames[i] {
			t.Errorf("Game mismatch at %d:\n  seq: %s\n  par: %s", i, seqGames[i], parGames[i])
		}
	}
}

// TestDefaultWorkersProcessesGames verifies the default worker count works.
func TestDefaultWorkersProcessesGames(t *testing.T) {
	// Run without explicit --workers flag (should use NumCPU)
	out, _ := runPgnExtract(t, "-s", inputFile("fischer.pgn"))
	count := countGames(out)
	if count == 0 {
		t.Error("Expected games in output with default workers")
	}
	// Fischer.pgn has 34 games
	if count != 34 {
		t.Errorf("Expected 34 games, got %d", count)
	}
}

// TestParallelWithTagFilter verifies parallel processing works with tag filters.
func TestParallelWithTagFilter(t *testing.T) {
	// Sequential with player filter
	seqOut, seqErr := runPgnExtract(t, "-s", "--workers", "1", "-Tw", "Fischer", inputFile("fischer.pgn"))
	if strings.Contains(seqErr, "flag provided but not defined") {
		t.Skip("--workers flag not implemented yet")
	}
	seqCount := countGames(seqOut)

	// Parallel with same filter
	parOut, _ := runPgnExtract(t, "-s", "--workers", "4", "-Tw", "Fischer", inputFile("fischer.pgn"))
	parCount := countGames(parOut)

	if seqCount != parCount {
		t.Errorf("Tag filter results differ: sequential=%d, parallel=%d", seqCount, parCount)
	}

	// Verify the filter actually worked (should filter to Fischer as White)
	if seqCount == 0 {
		t.Error("Expected at least one game with Fischer as White")
	}
}

// TestParallelDuplicateDetection verifies duplicate detection works with parallel processing.
func TestParallelDuplicateDetection(t *testing.T) {
	// Process same file twice with duplicate detection - sequential
	seqOut, seqErr := runPgnExtract(t, "-s", "-D", "--workers", "1",
		inputFile("fischer.pgn"), inputFile("fischer.pgn"))
	if strings.Contains(seqErr, "flag provided but not defined") {
		t.Skip("--workers flag not implemented yet")
	}
	seqCount := countGames(seqOut)

	// Process same file twice with duplicate detection - parallel
	parOut, _ := runPgnExtract(t, "-s", "-D", "--workers", "4",
		inputFile("fischer.pgn"), inputFile("fischer.pgn"))
	parCount := countGames(parOut)

	if seqCount != parCount {
		t.Errorf("Duplicate detection differs: sequential=%d, parallel=%d", seqCount, parCount)
	}

	// With duplicate suppression, should output only unique games (34, not 68)
	if seqCount > 34 {
		t.Errorf("Expected at most 34 unique games, got %d", seqCount)
	}
}

// TestParallelStopAfter verifies --stopafter works correctly with parallel processing.
func TestParallelStopAfter(t *testing.T) {
	// Stop after 3 games
	out, stderr := runPgnExtract(t, "-s", "--stopafter", "3", "--workers", "4", inputFile("fischer.pgn"))
	if strings.Contains(stderr, "flag provided but not defined") {
		t.Skip("--workers flag not implemented yet")
	}
	count := countGames(out)

	// Should stop at or before 3 (might process slightly more due to parallel nature)
	if count > 3 {
		t.Errorf("stopAfter not respected: expected <= 3, got %d", count)
	}
	if count == 0 {
		t.Error("Expected at least 1 game")
	}
}

// TestWorkersZeroDefaultsToNumCPU verifies that workers=0 uses NumCPU.
func TestWorkersZeroDefaultsToNumCPU(t *testing.T) {
	out, stderr := runPgnExtract(t, "-s", "--workers", "0", inputFile("fischer.pgn"))
	if strings.Contains(stderr, "flag provided but not defined") {
		t.Skip("--workers flag not implemented yet")
	}
	count := countGames(out)
	if count == 0 {
		t.Error("Expected games in output with workers=0")
	}
	if count != 34 {
		t.Errorf("Expected 34 games, got %d", count)
	}
}

// TestSingleWorkerIsDeterministic verifies single worker produces consistent results.
func TestSingleWorkerIsDeterministic(t *testing.T) {
	out1, stderr := runPgnExtract(t, "-s", "--workers", "1", inputFile("fools-mate.pgn"))
	if strings.Contains(stderr, "flag provided but not defined") {
		t.Skip("--workers flag not implemented yet")
	}
	// Run twice to verify deterministic
	out2, _ := runPgnExtract(t, "-s", "--workers", "1", inputFile("fools-mate.pgn"))

	if out1 != out2 {
		t.Error("Single worker should produce deterministic output")
	}
}

// TestParallelWithECO verifies ECO classification works with parallel processing.
func TestParallelWithECO(t *testing.T) {
	// Sequential with ECO
	seqOut, seqErr := runPgnExtract(t, "-s", "--workers", "1", "-e", testEcoFile(), inputFile("fischer.pgn"))
	if strings.Contains(seqErr, "flag provided but not defined") {
		t.Skip("--workers flag not implemented yet")
	}

	// Parallel with ECO
	parOut, _ := runPgnExtract(t, "-s", "--workers", "4", "-e", testEcoFile(), inputFile("fischer.pgn"))

	// Both should have ECO tags added
	seqHasECO := strings.Contains(seqOut, "[ECO ")
	parHasECO := strings.Contains(parOut, "[ECO ")

	if seqHasECO != parHasECO {
		t.Errorf("ECO classification differs: sequential has ECO=%v, parallel has ECO=%v", seqHasECO, parHasECO)
	}

	// Count games should be the same
	if countGames(seqOut) != countGames(parOut) {
		t.Errorf("Game count differs with ECO: seq=%d, par=%d", countGames(seqOut), countGames(parOut))
	}
}

// TestParallelWithNegation verifies negated matching works with parallel processing.
func TestParallelWithNegation(t *testing.T) {
	// Sequential: games NOT matching Fischer as White
	seqOut, seqErr := runPgnExtract(t, "-s", "--workers", "1", "-n", "-Tw", "Fischer", inputFile("fischer.pgn"))
	if strings.Contains(seqErr, "flag provided but not defined") {
		t.Skip("--workers flag not implemented yet")
	}
	seqCount := countGames(seqOut)

	// Parallel: same negated filter
	parOut, _ := runPgnExtract(t, "-s", "--workers", "4", "-n", "-Tw", "Fischer", inputFile("fischer.pgn"))
	parCount := countGames(parOut)

	if seqCount != parCount {
		t.Errorf("Negated filter results differ: sequential=%d, parallel=%d", seqCount, parCount)
	}
}

// TestParallelMultipleFiles verifies parallel processing works with multiple input files.
func TestParallelMultipleFiles(t *testing.T) {
	// Sequential
	seqOut, seqErr := runPgnExtract(t, "-s", "--workers", "1",
		inputFile("fischer.pgn"), inputFile("fools-mate.pgn"))
	if strings.Contains(seqErr, "flag provided but not defined") {
		t.Skip("--workers flag not implemented yet")
	}
	seqCount := countGames(seqOut)

	// Parallel
	parOut, _ := runPgnExtract(t, "-s", "--workers", "4",
		inputFile("fischer.pgn"), inputFile("fools-mate.pgn"))
	parCount := countGames(parOut)

	if seqCount != parCount {
		t.Errorf("Multiple files count differs: sequential=%d, parallel=%d", seqCount, parCount)
	}

	// Fischer has 34 games, fools-mate has 1
	expected := 35
	if seqCount != expected {
		t.Errorf("Expected %d games from multiple files, got %d", expected, seqCount)
	}
}

// TestParallelWithValidation verifies validation mode works with parallel processing.
func TestParallelWithValidation(t *testing.T) {
	// Sequential with validation
	seqOut, seqErr := runPgnExtract(t, "-s", "--workers", "1", "--validate", inputFile("fischer.pgn"))
	if strings.Contains(seqErr, "flag provided but not defined") {
		t.Skip("--workers flag not implemented yet")
	}
	seqCount := countGames(seqOut)

	// Parallel with validation
	parOut, _ := runPgnExtract(t, "-s", "--workers", "4", "--validate", inputFile("fischer.pgn"))
	parCount := countGames(parOut)

	if seqCount != parCount {
		t.Errorf("Validation mode results differ: sequential=%d, parallel=%d", seqCount, parCount)
	}
}
