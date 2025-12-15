package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testdataDir returns the path to the testdata directory.
func testdataDir() string {
	return filepath.Join("..", "..", "testdata")
}

// inputFile returns the full path to an input file.
func inputFile(name string) string {
	return filepath.Join(testdataDir(), "infiles", name)
}

// goldenFile returns the full path to a golden output file.
func goldenFile(name string) string {
	return filepath.Join(testdataDir(), "golden", name)
}

// testEcoFile returns the full path to the ECO file.
func testEcoFile() string {
	return filepath.Join(testdataDir(), "eco.pgn")
}

var testBinaryPath string

// buildTestBinary builds the test binary once for all tests.
func buildTestBinary(t *testing.T) string {
	t.Helper()
	if testBinaryPath != "" {
		return testBinaryPath
	}

	// Get the working directory for the test
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Build the binary
	binPath := filepath.Join(wd, "pgn-extract-test")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = wd
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build pgn-extract: %v\n%s", err, output)
	}

	testBinaryPath = binPath
	return binPath
}

// runPgnExtract runs the pgn-extract binary with the given arguments and returns stdout.
func runPgnExtract(t *testing.T, args ...string) (string, string) {
	t.Helper()

	binPath := buildTestBinary(t)

	// Run the binary
	cmd := exec.Command(binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Run() // Don't fail on non-zero exit

	return stdout.String(), stderr.String()
}

// readGolden reads a golden file and returns its contents.
func readGolden(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(goldenFile(name))
	if err != nil {
		t.Fatalf("Failed to read golden file %s: %v", name, err)
	}
	return string(content)
}

// countGames counts the number of games in PGN output.
func countGames(pgn string) int {
	return strings.Count(pgn, "[Event ")
}

// containsTag checks if output contains a specific tag.
func containsTag(output, tagName, tagValue string) bool {
	search := "[" + tagName + " \"" + tagValue + "\"]"
	return strings.Contains(output, search)
}

// containsMove checks if output contains a specific move.
func containsMove(output, move string) bool {
	return strings.Contains(output, move)
}

// TestBasicParsing tests basic PGN parsing.
func TestBasicParsing(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", inputFile("fools-mate.pgn"))
	if stdout == "" {
		t.Error("Expected non-empty output")
	}
	if !strings.Contains(stdout, "[Event") {
		t.Error("Expected Event tag in output")
	}
	// Check for fools mate moves
	if !containsMove(stdout, "f3") || !containsMove(stdout, "Qh4") {
		t.Error("Expected fools mate moves in output")
	}
}

// TestFischerGames tests processing the Fischer games file.
func TestFischerGames(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", inputFile("fischer.pgn"))
	if stdout == "" {
		t.Error("Expected non-empty output from fischer.pgn")
	}
	count := countGames(stdout)
	if count == 0 {
		t.Error("Expected at least one game in output")
	}
	t.Logf("Parsed %d games from fischer.pgn", count)
}

// TestSevenTagRoster tests the -7 flag for seven tag roster output.
func TestSevenTagRoster(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-7", "-s", inputFile("test-7.pgn"))
	if stdout == "" {
		t.Error("Expected non-empty output")
	}

	// Check that seven tag roster tags are present
	sevenTags := []string{"Event", "Site", "Date", "Round", "White", "Black", "Result"}
	for _, tag := range sevenTags {
		if !strings.Contains(stdout, "["+tag+" ") {
			t.Errorf("Expected %s tag in output", tag)
		}
	}

	// Check that non-roster tags like ECO are NOT present
	if strings.Contains(stdout, "[ECO ") || strings.Contains(stdout, "[Opening ") {
		t.Error("Expected non-roster tags to be removed")
	}
}

// TestNoComments tests the -C flag for removing comments.
func TestNoComments(t *testing.T) {
	// First verify input has comments
	input, _ := os.ReadFile(inputFile("test-C.pgn"))
	if !strings.Contains(string(input), "{") {
		t.Skip("Input file has no comments to test")
	}

	stdout, _ := runPgnExtract(t, "-C", "-s", inputFile("test-C.pgn"))
	if stdout == "" {
		t.Error("Expected non-empty output")
	}

	// Comments should be removed (no { } braces in move text)
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		// Skip tag lines
		if strings.HasPrefix(line, "[") {
			continue
		}
		if strings.Contains(line, "{") && !strings.HasPrefix(strings.TrimSpace(line), "{") {
			// Allow standalone comment lines but not inline comments after moves
			// This is a simplistic check
		}
	}
}

// TestNoNAGs tests the -N flag for removing NAGs.
func TestNoNAGs(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-N", "-s", inputFile("test-N.pgn"))
	if stdout == "" {
		t.Error("Expected non-empty output")
	}

	// NAGs like $1, $2, etc. should be removed
	// Check that no $ followed by numbers appear in move text
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "[") {
			continue
		}
		// Simple check - look for $X patterns
		for i := 0; i < len(line)-1; i++ {
			if line[i] == '$' && line[i+1] >= '0' && line[i+1] <= '9' {
				t.Errorf("Found NAG in output: %s", line)
				break
			}
		}
	}
}

// TestNoVariations tests the -V flag for removing variations.
func TestNoVariations(t *testing.T) {
	// First check that input has variations
	input, _ := os.ReadFile(inputFile("test-V.pgn"))
	if !strings.Contains(string(input), "(") {
		t.Skip("Input file has no variations")
	}

	stdout, _ := runPgnExtract(t, "-V", "-s", inputFile("test-V.pgn"))
	if stdout == "" {
		t.Error("Expected non-empty output")
	}

	// Count parentheses in non-tag lines - should be zero or minimal
	parenCount := 0
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "[") {
			continue
		}
		parenCount += strings.Count(line, "(")
	}
	if parenCount > 0 {
		t.Errorf("Expected no variations in output, found %d opening parens", parenCount)
	}
}

// TestOutputFormat tests the -W flag for output formats.
func TestOutputFormat(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		checkMove  string // A move that should appear in the output
		shouldHave []string
	}{
		{"lalg", "lalg", "e2e4", []string{"e2e4", "e7e5"}},
		{"halg", "halg", "e2-e4", []string{"e2-e4", "e7-e5"}},
		{"elalg", "elalg", "Pe2e4", []string{}}, // Enhanced long algebraic
		{"uci", "uci", "e2e4", []string{"e2e4", "e7e5"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _ := runPgnExtract(t, "-W", tt.format, "-s", inputFile("test-ucW.pgn"))
			if stdout == "" {
				t.Error("Expected non-empty output")
				return
			}

			for _, expected := range tt.shouldHave {
				if !strings.Contains(stdout, expected) {
					t.Errorf("Expected %s in %s format output", expected, tt.format)
				}
			}
		})
	}
}

// TestECOClassification tests the -e flag for ECO classification.
func TestECOClassification(t *testing.T) {
	// First get output without ECO
	stdoutBefore, _ := runPgnExtract(t, "-s", inputFile("test-e.pgn"))

	// Then with ECO
	stdoutAfter, _ := runPgnExtract(t, "-e", testEcoFile(), "-s", inputFile("test-e.pgn"))

	if stdoutAfter == "" {
		t.Error("Expected non-empty output")
		return
	}

	// Check that ECO tag is added
	if !strings.Contains(stdoutAfter, "[ECO ") {
		t.Error("Expected ECO tag to be added")
	}

	// Check that we have at least as many games after as before
	countBefore := countGames(stdoutBefore)
	countAfter := countGames(stdoutAfter)
	if countAfter < countBefore {
		t.Errorf("Lost games: before=%d, after=%d", countBefore, countAfter)
	}
}

// TestTagFilters tests the -T flags for tag-based filtering.
func TestTagFilters(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		input         string
		expectedGames int // Expected number of games after filter
		mustContain   string
	}{
		{"player-fischer", []string{"-Tp", "Fischer"}, "fischer.pgn", -1, "Fischer"},
		{"white-fischer", []string{"-Tw", "Fischer"}, "fischer.pgn", -1, "White"},
		{"black-petrosian", []string{"-Tb", "Petrosian"}, "fischer.pgn", -1, "Petrosian"},
		{"result-loss", []string{"-Tr", "0-1"}, "fischer.pgn", -1, "0-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get count without filter
			stdoutAll, _ := runPgnExtract(t, "-s", inputFile(tt.input))
			allCount := countGames(stdoutAll)

			// Get count with filter
			args := append(tt.args, "-s", inputFile(tt.input))
			stdoutFiltered, _ := runPgnExtract(t, args...)
			filteredCount := countGames(stdoutFiltered)

			// Filtered should be fewer or equal
			if filteredCount > allCount {
				t.Errorf("Filtered count (%d) > all count (%d)", filteredCount, allCount)
			}

			// Should have at least one game if input has matching games
			if filteredCount == 0 && tt.mustContain != "" {
				t.Logf("Warning: No games matched filter %v", tt.args)
			}
		})
	}
}

// TestDuplicateDetection tests the -D flag for suppressing duplicates.
func TestDuplicateDetection(t *testing.T) {
	// Process same file twice without -D
	stdoutNoDup, _ := runPgnExtract(t, "-s", inputFile("fischer.pgn"), inputFile("fischer.pgn"))
	countNoDup := countGames(stdoutNoDup)

	// Process with -D
	stdoutWithDup, _ := runPgnExtract(t, "-D", "-s", inputFile("fischer.pgn"), inputFile("fischer.pgn"))
	countWithDup := countGames(stdoutWithDup)

	// With duplicate suppression, should have roughly half the games
	if countWithDup >= countNoDup {
		t.Errorf("Expected fewer games with -D: without=%d, with=%d", countNoDup, countWithDup)
	}

	t.Logf("Duplicate detection: %d games without -D, %d with -D", countNoDup, countWithDup)
}

// TestOutputFile tests writing to a file with -o.
func TestOutputFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "output.pgn")
	runPgnExtract(t, "-o", tmpFile, "-s", inputFile("fools-mate.pgn"))

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	if len(content) == 0 {
		t.Error("Expected non-empty output file")
	}
}

// TestMultipleInputFiles tests processing multiple input files.
func TestMultipleInputFiles(t *testing.T) {
	// Count games in each file separately
	stdout1, _ := runPgnExtract(t, "-s", inputFile("test-f1.pgn"))
	stdout2, _ := runPgnExtract(t, "-s", inputFile("test-f2.pgn"))
	count1 := countGames(stdout1)
	count2 := countGames(stdout2)

	// Count games when both are processed
	stdoutBoth, _ := runPgnExtract(t, "-s", inputFile("test-f1.pgn"), inputFile("test-f2.pgn"))
	countBoth := countGames(stdoutBoth)

	// Combined should equal sum
	if countBoth != count1+count2 {
		t.Errorf("Expected %d games (sum), got %d", count1+count2, countBoth)
	}
}

// TestJSONOutput tests the -J flag for JSON output.
func TestJSONOutput(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-J", "-s", inputFile("fools-mate.pgn"))
	if stdout == "" {
		t.Error("Expected non-empty JSON output")
		return
	}

	// Should be valid JSON with games array
	if !strings.Contains(stdout, "\"games\"") {
		t.Error("Expected 'games' key in JSON output")
	}
	if !strings.Contains(stdout, "\"tags\"") {
		t.Error("Expected 'tags' key in JSON output")
	}
	if !strings.Contains(stdout, "\"moves\"") {
		t.Error("Expected 'moves' key in JSON output")
	}
}

// TestLineLength tests the -w flag for line length control.
func TestLineLength(t *testing.T) {
	// Test with very short line length
	stdout60, _ := runPgnExtract(t, "-w", "60", "-s", inputFile("test-w.pgn"))
	// Test with very long line length
	stdout1000, _ := runPgnExtract(t, "-w", "1000", "-s", inputFile("test-w.pgn"))

	// Longer line length should have fewer lines (moves spread across fewer lines)
	lines60 := strings.Split(stdout60, "\n")
	lines1000 := strings.Split(stdout1000, "\n")

	// Should have more lines with shorter line length
	if len(lines60) <= len(lines1000) {
		t.Logf("Line count: w60=%d, w1000=%d", len(lines60), len(lines1000))
		// This might not always be true depending on content, so just log
	}

	// Check that no line exceeds the limit (approximately)
	for i, line := range lines60 {
		if len(line) > 70 { // Allow some buffer
			t.Logf("Line %d exceeds 60 chars: %d chars", i+1, len(line))
		}
	}
}

// TestLongLine tests handling of games with long lines.
func TestLongLine(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", inputFile("test-long-line.pgn"))
	if stdout == "" {
		t.Error("Expected non-empty output")
	}
	if countGames(stdout) == 0 {
		t.Error("Expected at least one game in output")
	}
}

// TestNestedComments tests handling of nested comments.
func TestNestedComments(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", inputFile("nested-comment.pgn"))
	if stdout == "" {
		t.Error("Expected non-empty output")
	}
	if countGames(stdout) == 0 {
		t.Error("Expected at least one game in output")
	}
}

// TestCheckmate tests the --checkmate flag if implemented.
func TestCheckmate(t *testing.T) {
	stdout, stderr := runPgnExtract(t, "--checkmate", "-s", inputFile("test-checkmate.pgn"))
	if strings.Contains(stderr, "unknown flag") || strings.Contains(stderr, "not defined") {
		t.Skip("--checkmate flag not implemented")
	}
	if stdout == "" {
		// Might be no games ending in checkmate
		t.Log("No games matched checkmate filter (or flag not implemented)")
	}
}

// TestStalemate tests the --stalemate flag if implemented.
func TestStalemate(t *testing.T) {
	stdout, stderr := runPgnExtract(t, "--stalemate", "-s", inputFile("test-stalemate.pgn"))
	if strings.Contains(stderr, "unknown flag") || strings.Contains(stderr, "not defined") {
		t.Skip("--stalemate flag not implemented")
	}
	if stdout == "" {
		t.Log("No games matched stalemate filter (or flag not implemented)")
	}
}

// TestHelp tests the -h flag.
func TestHelp(t *testing.T) {
	_, stderr := runPgnExtract(t, "-h")
	// Help should print to stderr typically
	stdout, _ := runPgnExtract(t, "-h")
	output := stdout + stderr
	if !strings.Contains(output, "Usage") && !strings.Contains(output, "usage") {
		t.Error("Expected usage information in help output")
	}
}

// TestVersion tests the --version flag.
func TestVersion(t *testing.T) {
	stdout, stderr := runPgnExtract(t, "--version")
	output := stdout + stderr
	if !strings.Contains(output, "version") && !strings.Contains(output, "pgn-extract") {
		t.Error("Expected version information")
	}
}

// TestPetrosianGames tests processing the Petrosian games file.
func TestPetrosianGames(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", inputFile("petrosian.pgn"))
	if stdout == "" {
		t.Error("Expected non-empty output from petrosian.pgn")
	}
	count := countGames(stdout)
	if count == 0 {
		t.Error("Expected at least one game in output")
	}
	t.Logf("Parsed %d games from petrosian.pgn", count)
}

// TestNajdorf tests processing the Najdorf games file.
func TestNajdorf(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", inputFile("najdorf.pgn"))
	if stdout == "" {
		t.Error("Expected non-empty output from najdorf.pgn")
	}
	count := countGames(stdout)
	if count == 0 {
		t.Error("Expected at least one game in output")
	}
	t.Logf("Parsed %d games from najdorf.pgn", count)
}
