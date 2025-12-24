package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/parser"
)

func parseTestGame(pgn string) *chess.Game {
	cfg := config.NewConfig()
	cfg.Verbosity = 0
	p := parser.NewParser(strings.NewReader(pgn), cfg)
	games, _ := p.ParseAllGames()
	if len(games) > 0 {
		return games[0]
	}
	return nil
}

// TestPGNWriter_WriteGame verifies PGN writer outputs correct format
func TestPGNWriter_WriteGame(t *testing.T) {
	game := parseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Fischer"]
[Black "Spassky"]
[Result "1-0"]

1. e4 e5 2. Nf3 1-0
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	var buf bytes.Buffer
	cfg := config.NewConfig()
	cfg.SetOutput(&buf)

	writer := NewPGNWriter(&buf, cfg)
	err := writer.WriteGame(game)
	if err != nil {
		t.Fatalf("WriteGame failed: %v", err)
	}

	output := buf.String()

	// Verify PGN structure
	if !strings.Contains(output, `[Event "Test"]`) {
		t.Error("Missing Event tag")
	}
	if !strings.Contains(output, `[White "Fischer"]`) {
		t.Error("Missing White tag")
	}
	if !strings.Contains(output, "e4") {
		t.Error("Missing moves")
	}
}

// TestJSONWriter_WriteGame verifies JSON writer outputs correct format
func TestJSONWriter_WriteGame(t *testing.T) {
	game := parseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Fischer"]
[Black "Spassky"]
[Result "1-0"]

1. e4 e5 2. Nf3 1-0
`)
	if game == nil {
		t.Fatal("Failed to parse test game")
	}

	var buf bytes.Buffer
	cfg := config.NewConfig()

	writer := NewJSONWriter(&buf, cfg)
	err := writer.WriteGame(game)
	if err != nil {
		t.Fatalf("WriteGame failed: %v", err)
	}

	// Flush to ensure all output is written
	err = writer.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	output := buf.String()

	// Verify JSON structure
	if !strings.Contains(output, `"tags"`) {
		t.Error("Missing tags field in JSON")
	}
	if !strings.Contains(output, `"Fischer"`) {
		t.Error("Missing player name in JSON")
	}
}

// TestGameWriter_Interface verifies that writers implement the interface
func TestGameWriter_Interface(t *testing.T) {
	cfg := config.NewConfig()
	var buf bytes.Buffer

	// Verify PGNWriter implements GameWriter
	var _ GameWriter = NewPGNWriter(&buf, cfg)

	// Verify JSONWriter implements GameWriter
	var _ GameWriter = NewJSONWriter(&buf, cfg)
}

// TestPGNWriter_Close verifies Close doesn't error
func TestPGNWriter_Close(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.NewConfig()

	writer := NewPGNWriter(&buf, cfg)
	err := writer.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestJSONWriter_Close verifies Close flushes pending games
func TestJSONWriter_Close(t *testing.T) {
	game := parseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 *
`)

	var buf bytes.Buffer
	cfg := config.NewConfig()

	writer := NewJSONWriter(&buf, cfg)
	writer.WriteGame(game)
	err := writer.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Output should have content after close
	if buf.Len() == 0 {
		t.Error("Expected output after Close")
	}
}

// TestPGNWriter_Flush verifies Flush works correctly
func TestPGNWriter_Flush(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.NewConfig()

	writer := NewPGNWriter(&buf, cfg)
	err := writer.Flush()
	if err != nil {
		t.Errorf("Flush failed: %v", err)
	}
}
