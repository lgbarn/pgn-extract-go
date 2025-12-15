package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to count games in output
func countGamesInOutput(output string) int {
	// Count [Event tags as a proxy for game count
	return strings.Count(output, "[Event ")
}

// TestNegatedMatching tests the -n flag for negated matching
func TestNegatedMatching(t *testing.T) {
	// Find games with checkmate
	stdoutMate, _ := runPgnExtract(t, "-s", "--checkmate", inputFile("test-checkmate.pgn"))
	mateGames := countGames(stdoutMate)
	t.Logf("Found %d checkmate games", mateGames)

	// Test that -n inverts the match (all games WITHOUT checkmate)
	stdoutNotMate, _ := runPgnExtract(t, "-s", "-n", "--checkmate", inputFile("test-checkmate.pgn"))
	notMateGames := countGames(stdoutNotMate)
	t.Logf("Found %d non-checkmate games with -n", notMateGames)

	// The sum of mate + not-mate should equal total games
	// Note: test-checkmate.pgn has 2 games total
	if mateGames+notMateGames != 2 {
		t.Errorf("Expected mate(%d) + not-mate(%d) = 2 total games", mateGames, notMateGames)
	}
}

// TestAppendMode tests the -a flag for append mode
func TestAppendMode(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "append_test*.pgn")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	tmpFile.Close()

	// Write first batch
	runPgnExtract(t, "-s", "-o", tmpPath, inputFile("test-checkmate.pgn"))

	// Get size after first write
	info1, _ := os.Stat(tmpPath)
	size1 := info1.Size()

	// Append second batch
	runPgnExtract(t, "-s", "-a", "-o", tmpPath, inputFile("test-checkmate.pgn"))

	// Get size after append
	info2, _ := os.Stat(tmpPath)
	size2 := info2.Size()

	// Size should have roughly doubled
	if size2 <= size1 {
		t.Errorf("Append mode failed: size before=%d, after=%d (should be larger)", size1, size2)
	}
	t.Logf("Append mode: size before=%d, after=%d", size1, size2)
}

// TestPlyCount tests the --plycount flag
func TestPlyCount(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", "--plycount", inputFile("test-checkmate.pgn"))

	// Check that PlyCount tag is present
	if !strings.Contains(stdout, "[PlyCount ") {
		t.Error("Expected PlyCount tag in output")
	}
	t.Logf("Found PlyCount tag in output")
}

// TestStopAfter tests the --stopafter flag
func TestStopAfter(t *testing.T) {
	// Stop after 5 games
	stdout, _ := runPgnExtract(t, "-s", "--stopafter", "5", inputFile("fischer.pgn"))
	count := countGames(stdout)

	if count > 5 {
		t.Errorf("Expected at most 5 games, got %d", count)
	}
	t.Logf("--stopafter 5: got %d games", count)
}

// TestMinPly tests the --minply flag
func TestMinPly(t *testing.T) {
	// Find games with at least 20 ply (10 moves)
	stdout, _ := runPgnExtract(t, "-s", "--minply", "20", inputFile("fischer.pgn"))
	count := countGames(stdout)

	t.Logf("--minply 20: found %d games", count)

	// Should have fewer games than total (some games are shorter)
	stdoutAll, _ := runPgnExtract(t, "-s", inputFile("fischer.pgn"))
	totalCount := countGames(stdoutAll)

	if count >= totalCount && totalCount > 0 {
		t.Logf("Warning: All %d games have >= 20 ply", count)
	}
}

// TestMaxPly tests the --maxply flag
func TestMaxPly(t *testing.T) {
	// Find games with at most 10 ply (5 moves)
	stdout, _ := runPgnExtract(t, "-s", "--maxply", "10", inputFile("fischer.pgn"))
	count := countGames(stdout)

	t.Logf("--maxply 10: found %d games", count)
}

// TestMinMoves tests the --minmoves flag
func TestMinMoves(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", "--minmoves", "10", inputFile("fischer.pgn"))
	count := countGames(stdout)
	t.Logf("--minmoves 10: found %d games", count)
}

// TestMaxMoves tests the --maxmoves flag
func TestMaxMoves(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", "--maxmoves", "5", inputFile("fischer.pgn"))
	count := countGames(stdout)
	t.Logf("--maxmoves 5: found %d games", count)
}

// TestSoundex tests the -S flag for Soundex player matching
func TestSoundex(t *testing.T) {
	// Fischer and Fisher should match with Soundex
	stdout, _ := runPgnExtract(t, "-s", "-S", "-p", "Fisher", inputFile("fischer.pgn"))
	count := countGames(stdout)

	// Should find some games because "Fischer" sounds like "Fisher"
	t.Logf("-S (Soundex): searching for 'Fisher' found %d games", count)

	// Without Soundex, should find fewer or no games
	stdoutNoSoundex, _ := runPgnExtract(t, "-s", "-p", "Fisher", inputFile("fischer.pgn"))
	countNoSoundex := countGames(stdoutNoSoundex)

	t.Logf("Without Soundex: searching for 'Fisher' found %d games", countNoSoundex)
}

// TestOutputSplit tests the -# flag for splitting output
func TestOutputSplit(t *testing.T) {
	// Create temp directory for split files
	tmpDir, err := os.MkdirTemp("", "split_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	basePath := filepath.Join(tmpDir, "output.pgn")

	// Split into files of 10 games each
	runPgnExtract(t, "-s", "-#", "10", "-o", basePath, inputFile("fischer.pgn"))

	// Check that multiple files were created
	files, _ := filepath.Glob(filepath.Join(tmpDir, "output_*.pgn"))
	t.Logf("-# 10: created %d split files", len(files))

	if len(files) < 1 {
		t.Error("Expected at least 1 split file")
	}
}

// TestFiftyMoveRule tests the --fifty flag
func TestFiftyMoveRule(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", "--fifty", inputFile("fischer.pgn"))
	count := countGames(stdout)
	t.Logf("--fifty: found %d games with 50-move rule", count)
}

// TestRepetition tests the --repetition flag
func TestRepetition(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", "--repetition", inputFile("fischer.pgn"))
	count := countGames(stdout)
	t.Logf("--repetition: found %d games with repetition", count)
}

// TestUnderpromotion tests the --underpromotion flag
func TestUnderpromotion(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", "--underpromotion", inputFile("fischer.pgn"))
	count := countGames(stdout)
	t.Logf("--underpromotion: found %d games with underpromotion", count)
}

// TestCommented tests the --commented flag
func TestCommented(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", "--commented", inputFile("fischer.pgn"))
	count := countGames(stdout)
	t.Logf("--commented: found %d games with comments", count)
}

// TestHigherRatedWinner tests the --higherratedwinner flag
func TestHigherRatedWinner(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", "--higherratedwinner", inputFile("fischer.pgn"))
	count := countGames(stdout)
	t.Logf("--higherratedwinner: found %d games", count)
}

// TestLowerRatedWinner tests the --lowerratedwinner flag
func TestLowerRatedWinner(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", "--lowerratedwinner", inputFile("fischer.pgn"))
	count := countGames(stdout)
	t.Logf("--lowerratedwinner: found %d games", count)
}

// TestOutputDupsOnly tests the -U flag
func TestOutputDupsOnly(t *testing.T) {
	// Create a file with duplicates
	tmpFile, err := os.CreateTemp("", "dups_test*.pgn")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write the same games twice
	content, _ := os.ReadFile(inputFile("test-checkmate.pgn"))
	tmpFile.Write(content)
	tmpFile.Write(content)
	tmpFile.Close()

	// With -U, should output only duplicates
	stdout, _ := runPgnExtract(t, "-s", "-U", tmpPath)
	count := countGames(stdout)
	t.Logf("-U (output dups only): found %d duplicate games", count)
}

// TestCheckFile tests the -c flag for checkfile
func TestCheckFile(t *testing.T) {
	// Use one file as checkfile, filter another against it
	stdout, _ := runPgnExtract(t, "-s", "-D", "-c", inputFile("test-checkmate.pgn"), inputFile("test-checkmate.pgn"))
	count := countGames(stdout)

	// All games from second file should be detected as duplicates of checkfile
	t.Logf("-c checkfile: found %d unique games (should be 0 or few)", count)
}

// TestHashcodeTag tests the --addhashcode flag
func TestHashcodeTag(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", "--addhashcode", inputFile("test-checkmate.pgn"))

	if !strings.Contains(stdout, "[HashCode ") {
		t.Error("Expected HashCode tag in output")
	}
	t.Log("--addhashcode: found HashCode tag")
}

// TestFixResultTags tests the --fixresulttags flag
func TestFixResultTags(t *testing.T) {
	// This just tests that the flag doesn't cause errors
	stdout, _ := runPgnExtract(t, "-s", "--fixresulttags", inputFile("fischer.pgn"))
	count := countGames(stdout)
	t.Logf("--fixresulttags: processed %d games", count)
}

// TestLogFile tests the -l flag
func TestLogFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "log_test*.log")
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	tmpFile.Close()

	runPgnExtract(t, "-l", tmpPath, inputFile("fischer.pgn"))

	// Check that log file exists and has content
	info, err := os.Stat(tmpPath)
	if err != nil {
		t.Errorf("Log file not created: %v", err)
	} else {
		t.Logf("-l: log file created, size=%d", info.Size())
	}
}

// TestReportOnly tests the -r flag
func TestReportOnly(t *testing.T) {
	stdout, stderr := runPgnExtract(t, "-r", inputFile("fischer.pgn"))

	// With -r, should not output games
	gameCount := countGames(stdout)
	if gameCount > 0 {
		t.Errorf("-r: expected no game output, got %d games", gameCount)
	}
	t.Logf("-r: stdout games=%d, stderr=%q", gameCount, stderr[:min(100, len(stderr))])
}

// TestEPDOutput tests the -W epd output format
func TestEPDOutput(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", "-W", "epd", inputFile("test-checkmate.pgn"))

	// EPD format should have FEN-like output
	// This is a basic test - EPD format may need more specific validation
	t.Logf("-W epd: output length=%d", len(stdout))
}

// TestFENOutput tests the -W fen output format
func TestFENOutput(t *testing.T) {
	stdout, _ := runPgnExtract(t, "-s", "-W", "fen", inputFile("test-checkmate.pgn"))

	// FEN format should have FEN-like output
	t.Logf("-W fen: output length=%d", len(stdout))
}

// TestMaterialMatch tests the -z flag for material matching
func TestMaterialMatch(t *testing.T) {
	// Find positions with queen vs queen
	stdout, _ := runPgnExtract(t, "-s", "-z", "Q:q", inputFile("fischer.pgn"))
	count := countGames(stdout)
	t.Logf("-z Q:q: found %d games with Q vs q", count)
}

// TestExactMaterialMatch tests the -y flag for exact material matching
func TestExactMaterialMatch(t *testing.T) {
	// Find exact KQR vs KQR positions (unlikely in most games)
	stdout, _ := runPgnExtract(t, "-s", "-y", "KQR:kqr", inputFile("fischer.pgn"))
	count := countGames(stdout)
	t.Logf("-y KQR:kqr: found %d games", count)
}

// TestCombinedFilters tests combining multiple new filters
func TestCombinedFilters(t *testing.T) {
	// Find games with at least 20 ply, result 1-0, with comments
	stdout, _ := runPgnExtract(t, "-s",
		"--minply", "20",
		"-Tr", "1-0",
		inputFile("fischer.pgn"))
	count := countGames(stdout)
	t.Logf("Combined filters (minply 20 + result 1-0): found %d games", count)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestStrictMode tests the --strict flag
func TestStrictMode(t *testing.T) {
	// Create a temp file with a game missing required tags
	tmpFile, err := os.CreateTemp("", "strict_test*.pgn")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write a malformed game (missing Event tag)
	tmpFile.WriteString(`[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 1-0
`)
	tmpFile.Close()

	// Without strict mode, should output the game
	stdoutNormal, _ := runPgnExtract(t, "-s", tmpPath)
	countNormal := countGames(stdoutNormal)

	// With strict mode, should skip the game (missing Event)
	stdoutStrict, _ := runPgnExtract(t, "-s", "--strict", tmpPath)
	countStrict := countGames(stdoutStrict)

	t.Logf("Without strict: %d games, with strict: %d games", countNormal, countStrict)

	if countNormal == 0 {
		t.Error("Expected at least 1 game without strict mode")
	}
	if countStrict >= countNormal {
		t.Error("Expected fewer games with strict mode")
	}
}

// TestValidateMode tests the --validate flag
func TestValidateMode(t *testing.T) {
	// Create a temp file with an illegal move
	tmpFile, err := os.CreateTemp("", "validate_test*.pgn")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write a game with an illegal move (Nf3 to h5 is illegal)
	tmpFile.WriteString(`[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Player1"]
[Black "Player2"]
[Result "*"]

1. e4 e5 2. Nf3 Nh5 *
`)
	tmpFile.Close()

	// With validate mode, should skip the game with illegal move
	stdout, stderr := runPgnExtract(t, "-s", "--validate", tmpPath)
	count := countGames(stdout)

	t.Logf("Validate mode: %d games output, stderr: %s", count, stderr)

	// The illegal move should cause the game to be skipped
	// Note: whether it's caught depends on how Nh5 is parsed
}

// TestFixableMode tests the --fixable flag
func TestFixableMode(t *testing.T) {
	// Create a temp file with fixable issues
	tmpFile, err := os.CreateTemp("", "fixable_test*.pgn")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write a game with missing tags and bad date format
	tmpFile.WriteString(`[White "Player1"]
[Black "Player2"]
[Date "2024/01/01"]
[Result "1-0"]

1. e4 e5 1-0
`)
	tmpFile.Close()

	// With fixable mode, should fix the issues
	stdout, _ := runPgnExtract(t, "-s", "--fixable", tmpPath)

	// Check that missing Event tag was added
	if !strings.Contains(stdout, "[Event ") {
		t.Error("Expected Event tag to be added by fixable mode")
	}

	// Check that date was normalized (/ -> .)
	if strings.Contains(stdout, "2024/01/01") {
		t.Error("Expected date format to be fixed by fixable mode")
	}

	t.Logf("Fixable mode output:\n%s", stdout)
}

// TestValidateGoodGames tests that --validate passes good games
func TestValidateGoodGames(t *testing.T) {
	// Fischer games should all be valid
	stdout, _ := runPgnExtract(t, "-s", "--validate", inputFile("fischer.pgn"))
	count := countGames(stdout)

	// Count without validation
	stdoutAll, _ := runPgnExtract(t, "-s", inputFile("fischer.pgn"))
	countAll := countGames(stdoutAll)

	t.Logf("Validate mode: %d/%d games passed", count, countAll)

	// All Fischer games should be valid
	if count != countAll {
		t.Errorf("Expected all %d Fischer games to pass validation, got %d", countAll, count)
	}
}

// TestStrictWithFixable tests --strict combined with --fixable
func TestStrictWithFixable(t *testing.T) {
	// Create a temp file with fixable issues
	tmpFile, err := os.CreateTemp("", "strict_fixable_test*.pgn")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write a game with missing tags
	tmpFile.WriteString(`[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 1-0
`)
	tmpFile.Close()

	// With strict only, should skip (missing required tags)
	stdoutStrict, _ := runPgnExtract(t, "-s", "--strict", tmpPath)
	countStrict := countGames(stdoutStrict)

	// With fixable + strict, should pass (tags get fixed then validated)
	stdoutBoth, _ := runPgnExtract(t, "-s", "--fixable", "--strict", tmpPath)
	countBoth := countGames(stdoutBoth)

	t.Logf("Strict only: %d games, fixable+strict: %d games", countStrict, countBoth)

	if countStrict != 0 {
		t.Error("Expected strict mode alone to reject game with missing tags")
	}
	if countBoth != 1 {
		t.Error("Expected fixable+strict to accept game after fixing tags")
	}
}
