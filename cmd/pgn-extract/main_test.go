package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/hashing"
)

// --- Task 1: Pure parsing function tests ---

func TestSplitArgsLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{"simple args", "a b c", []string{"a", "b", "c"}},
		{"double quoted string", `"hello world" foo`, []string{"hello world", "foo"}},
		{"single quoted string", `'hello world' foo`, []string{"hello world", "foo"}},
		{"mixed quotes", `"hello world" 'foo bar' baz`, []string{"hello world", "foo bar", "baz"}},
		{"empty string", "", nil},
		{"tabs as separators", "a\tb\tc", []string{"a", "b", "c"}},
		{"single arg", "hello", []string{"hello"}},
		{"multiple spaces", "a   b   c", []string{"a", "b", "c"}},
		{"leading and trailing spaces", "  a b  ", []string{"a", "b"}},
		{"quoted with spaces inside", `--player "Bobby Fischer"`, []string{"--player", "Bobby Fischer"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitArgsLine(tt.line)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitArgsLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestLoadArgsFile(t *testing.T) {
	t.Run("valid file with args comments and empty lines", func(t *testing.T) {
		dir := t.TempDir()
		argsFile := filepath.Join(dir, "args.txt")
		content := `# This is a comment
-o output.pgn
-p "Bobby Fischer"

# Another comment
-D
`
		if err := os.WriteFile(argsFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := loadArgsFile(argsFile)
		if err != nil {
			t.Fatalf("loadArgsFile() error = %v", err)
		}
		want := []string{"-o", "output.pgn", "-p", "Bobby Fischer", "-D"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("loadArgsFile() = %v, want %v", got, want)
		}
	})

	t.Run("non-existent file returns error", func(t *testing.T) {
		_, err := loadArgsFile("/nonexistent/path/args.txt")
		if err == nil {
			t.Error("loadArgsFile() expected error for non-existent file, got nil")
		}
	})

	t.Run("empty file returns nil", func(t *testing.T) {
		dir := t.TempDir()
		argsFile := filepath.Join(dir, "empty.txt")
		if err := os.WriteFile(argsFile, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := loadArgsFile(argsFile)
		if err != nil {
			t.Fatalf("loadArgsFile() error = %v", err)
		}
		if got != nil {
			t.Errorf("loadArgsFile() = %v, want nil", got)
		}
	})
}

func TestLoadFileList(t *testing.T) {
	t.Run("valid file list", func(t *testing.T) {
		dir := t.TempDir()
		listFile := filepath.Join(dir, "files.txt")
		content := `# comment
game1.pgn
game2.pgn

# another comment
game3.pgn
`
		if err := os.WriteFile(listFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := loadFileList(listFile)
		if err != nil {
			t.Fatalf("loadFileList() error = %v", err)
		}
		want := []string{"game1.pgn", "game2.pgn", "game3.pgn"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("loadFileList() = %v, want %v", got, want)
		}
	})

	t.Run("non-existent file returns error", func(t *testing.T) {
		_, err := loadFileList("/nonexistent/path/files.txt")
		if err == nil {
			t.Error("loadFileList() expected error for non-existent file, got nil")
		}
	})

	t.Run("empty file returns nil", func(t *testing.T) {
		dir := t.TempDir()
		listFile := filepath.Join(dir, "empty.txt")
		if err := os.WriteFile(listFile, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := loadFileList(listFile)
		if err != nil {
			t.Fatalf("loadFileList() error = %v", err)
		}
		if got != nil {
			t.Errorf("loadFileList() = %v, want nil", got)
		}
	})
}

func TestReportStatistics(t *testing.T) {
	t.Run("with detector", func(t *testing.T) {
		detector := hashing.NewDuplicateDetector(false, 0)
		// Just verify no panic
		reportStatistics(detector, 10, 2, 15)
	})

	t.Run("without detector", func(t *testing.T) {
		// Just verify no panic
		reportStatistics(nil, 10, 0, 15)
	})
}

// --- Task 2: Setup helper function tests ---

// saveAndRestoreFilterFlags saves current flag values and returns a cleanup function.
func saveAndRestoreFilterFlags(t *testing.T) {
	t.Helper()
	oldPlayerFilter := *playerFilter
	oldWhiteFilter := *whiteFilter
	oldBlackFilter := *blackFilter
	oldEcoFilter := *ecoFilter
	oldResultFilter := *resultFilter
	oldFenFilter := *fenFilter
	oldTagFile := *tagFile
	oldUseSoundex := *useSoundex
	oldTagSubstring := *tagSubstring

	t.Cleanup(func() {
		*playerFilter = oldPlayerFilter
		*whiteFilter = oldWhiteFilter
		*blackFilter = oldBlackFilter
		*ecoFilter = oldEcoFilter
		*resultFilter = oldResultFilter
		*fenFilter = oldFenFilter
		*tagFile = oldTagFile
		*useSoundex = oldUseSoundex
		*tagSubstring = oldTagSubstring
	})
}

func TestSetupGameFilter(t *testing.T) {
	t.Run("no flags set returns empty filter", func(t *testing.T) {
		saveAndRestoreFilterFlags(t)
		*playerFilter = ""
		*whiteFilter = ""
		*blackFilter = ""
		*ecoFilter = ""
		*resultFilter = ""
		*fenFilter = ""
		*tagFile = ""
		*useSoundex = false
		*tagSubstring = false

		filter := setupGameFilter()
		if filter == nil {
			t.Fatal("setupGameFilter() returned nil, want non-nil filter")
		}
	})

	t.Run("with playerFilter", func(t *testing.T) {
		saveAndRestoreFilterFlags(t)
		*playerFilter = "Fischer"
		*whiteFilter = ""
		*blackFilter = ""
		*ecoFilter = ""
		*resultFilter = ""
		*fenFilter = ""
		*tagFile = ""
		*useSoundex = false
		*tagSubstring = false

		filter := setupGameFilter()
		if filter == nil {
			t.Fatal("setupGameFilter() returned nil")
		}
		if !filter.HasCriteria() {
			t.Error("setupGameFilter() filter should have criteria with playerFilter set")
		}
	})

	t.Run("with whiteFilter", func(t *testing.T) {
		saveAndRestoreFilterFlags(t)
		*playerFilter = ""
		*whiteFilter = "Kasparov"
		*blackFilter = ""
		*ecoFilter = ""
		*resultFilter = ""
		*fenFilter = ""
		*tagFile = ""
		*useSoundex = false
		*tagSubstring = false

		filter := setupGameFilter()
		if filter == nil {
			t.Fatal("setupGameFilter() returned nil")
		}
		if !filter.HasCriteria() {
			t.Error("setupGameFilter() filter should have criteria with whiteFilter set")
		}
	})

	t.Run("with blackFilter", func(t *testing.T) {
		saveAndRestoreFilterFlags(t)
		*playerFilter = ""
		*whiteFilter = ""
		*blackFilter = "Karpov"
		*ecoFilter = ""
		*resultFilter = ""
		*fenFilter = ""
		*tagFile = ""
		*useSoundex = false
		*tagSubstring = false

		filter := setupGameFilter()
		if filter == nil {
			t.Fatal("setupGameFilter() returned nil")
		}
		if !filter.HasCriteria() {
			t.Error("setupGameFilter() filter should have criteria with blackFilter set")
		}
	})

	t.Run("with ecoFilter", func(t *testing.T) {
		saveAndRestoreFilterFlags(t)
		*playerFilter = ""
		*whiteFilter = ""
		*blackFilter = ""
		*ecoFilter = "B90"
		*resultFilter = ""
		*fenFilter = ""
		*tagFile = ""
		*useSoundex = false
		*tagSubstring = false

		filter := setupGameFilter()
		if filter == nil {
			t.Fatal("setupGameFilter() returned nil")
		}
		if !filter.HasCriteria() {
			t.Error("setupGameFilter() filter should have criteria with ecoFilter set")
		}
	})

	t.Run("with resultFilter", func(t *testing.T) {
		saveAndRestoreFilterFlags(t)
		*playerFilter = ""
		*whiteFilter = ""
		*blackFilter = ""
		*ecoFilter = ""
		*resultFilter = "1-0"
		*fenFilter = ""
		*tagFile = ""
		*useSoundex = false
		*tagSubstring = false

		filter := setupGameFilter()
		if filter == nil {
			t.Fatal("setupGameFilter() returned nil")
		}
		if !filter.HasCriteria() {
			t.Error("setupGameFilter() filter should have criteria with resultFilter set")
		}
	})
}

func TestLoadMaterialMatcher(t *testing.T) {
	oldMaterialMatch := *materialMatch
	oldMaterialMatchExact := *materialMatchExact
	t.Cleanup(func() {
		*materialMatch = oldMaterialMatch
		*materialMatchExact = oldMaterialMatchExact
	})

	t.Run("empty materialMatch returns nil", func(t *testing.T) {
		*materialMatch = ""
		*materialMatchExact = ""
		got := loadMaterialMatcher()
		if got != nil {
			t.Error("loadMaterialMatcher() expected nil with empty flags")
		}
	})

	t.Run("with materialMatch", func(t *testing.T) {
		*materialMatch = "Q:q"
		*materialMatchExact = ""
		got := loadMaterialMatcher()
		if got == nil {
			t.Error("loadMaterialMatcher() expected non-nil with materialMatch set")
		}
	})

	t.Run("with materialMatchExact", func(t *testing.T) {
		*materialMatch = ""
		*materialMatchExact = "KQR:kqr"
		got := loadMaterialMatcher()
		if got == nil {
			t.Error("loadMaterialMatcher() expected non-nil with materialMatchExact set")
		}
	})

	t.Run("materialMatchExact takes precedence", func(t *testing.T) {
		*materialMatch = "Q:q"
		*materialMatchExact = "KQR:kqr"
		got := loadMaterialMatcher()
		if got == nil {
			t.Error("loadMaterialMatcher() expected non-nil")
		}
	})
}

func TestLoadVariationMatcher(t *testing.T) {
	oldVariationFile := *variationFile
	oldPositionFile := *positionFile
	oldVarAnywhere := *varAnywhere
	t.Cleanup(func() {
		*variationFile = oldVariationFile
		*positionFile = oldPositionFile
		*varAnywhere = oldVarAnywhere
	})

	t.Run("both files empty returns nil", func(t *testing.T) {
		*variationFile = ""
		*positionFile = ""
		*varAnywhere = false
		got := loadVariationMatcher()
		if got != nil {
			t.Error("loadVariationMatcher() expected nil with empty flags")
		}
	})
}

func TestParseCQLQuery(t *testing.T) {
	oldCqlQuery := *cqlQuery
	oldCqlFile := *cqlFile
	t.Cleanup(func() {
		*cqlQuery = oldCqlQuery
		*cqlFile = oldCqlFile
	})

	t.Run("empty query returns nil", func(t *testing.T) {
		*cqlQuery = ""
		*cqlFile = ""
		got := parseCQLQuery()
		if got != nil {
			t.Error("parseCQLQuery() expected nil with empty query")
		}
	})

	t.Run("with cqlQuery mate returns non-nil", func(t *testing.T) {
		*cqlQuery = "mate"
		*cqlFile = ""
		got := parseCQLQuery()
		if got == nil {
			t.Error("parseCQLQuery() expected non-nil with mate query")
		}
	})
}

func TestUsage(t *testing.T) {
	// Just verify no panic
	usage()
}

// --- Task 3: Extended setup tests ---

func TestSetupGameFilterWithTagFile(t *testing.T) {
	saveAndRestoreFilterFlags(t)

	dir := t.TempDir()
	tf := filepath.Join(dir, "tags.txt")
	content := `White "Fischer"
`
	if err := os.WriteFile(tf, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	*playerFilter = ""
	*whiteFilter = ""
	*blackFilter = ""
	*ecoFilter = ""
	*resultFilter = ""
	*fenFilter = ""
	*tagFile = tf
	*useSoundex = false
	*tagSubstring = false

	filter := setupGameFilter()
	if filter == nil {
		t.Fatal("setupGameFilter() returned nil")
	}
	if !filter.HasCriteria() {
		t.Error("setupGameFilter() filter should have criteria from tag file")
	}
}

func TestSetupGameFilterWithFenFilter(t *testing.T) {
	saveAndRestoreFilterFlags(t)

	*playerFilter = ""
	*whiteFilter = ""
	*blackFilter = ""
	*ecoFilter = ""
	*resultFilter = ""
	*fenFilter = "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR"
	*tagFile = ""
	*useSoundex = false
	*tagSubstring = false

	filter := setupGameFilter()
	if filter == nil {
		t.Fatal("setupGameFilter() returned nil")
	}
	if !filter.HasCriteria() {
		t.Error("setupGameFilter() filter should have criteria with fenFilter set")
	}
}

func TestSetupGameFilterWithSoundexAndSubstring(t *testing.T) {
	saveAndRestoreFilterFlags(t)

	*playerFilter = "Fischer"
	*whiteFilter = ""
	*blackFilter = ""
	*ecoFilter = ""
	*resultFilter = ""
	*fenFilter = ""
	*tagFile = ""
	*useSoundex = true
	*tagSubstring = true

	filter := setupGameFilter()
	if filter == nil {
		t.Fatal("setupGameFilter() returned nil")
	}
	if !filter.HasCriteria() {
		t.Error("setupGameFilter() filter should have criteria")
	}
}

func TestLoadArgsFromFileIfSpecified(t *testing.T) {
	t.Run("no -A flag returns nil", func(t *testing.T) {
		oldArgs := os.Args
		t.Cleanup(func() { os.Args = oldArgs })

		os.Args = []string{"pgn-extract", "-o", "out.pgn"}
		got := loadArgsFromFileIfSpecified()
		if got != nil {
			t.Errorf("loadArgsFromFileIfSpecified() = %v, want nil", got)
		}
	})

	t.Run("with -A flag loads args from file", func(t *testing.T) {
		oldArgs := os.Args
		t.Cleanup(func() { os.Args = oldArgs })

		dir := t.TempDir()
		argsFile := filepath.Join(dir, "args.txt")
		content := "-D\n-o output.pgn\n"
		if err := os.WriteFile(argsFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		os.Args = []string{"pgn-extract", "-A", argsFile}
		got := loadArgsFromFileIfSpecified()
		want := []string{"-D", "-o", "output.pgn"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("loadArgsFromFileIfSpecified() = %v, want %v", got, want)
		}
	})

	t.Run("with -A= syntax loads args from file", func(t *testing.T) {
		oldArgs := os.Args
		t.Cleanup(func() { os.Args = oldArgs })

		dir := t.TempDir()
		argsFile := filepath.Join(dir, "args.txt")
		content := "-s\n"
		if err := os.WriteFile(argsFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		os.Args = []string{"pgn-extract", "-A=" + argsFile}
		got := loadArgsFromFileIfSpecified()
		want := []string{"-s"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("loadArgsFromFileIfSpecified() = %v, want %v", got, want)
		}
	})
}
