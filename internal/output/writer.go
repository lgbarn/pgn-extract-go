// Package output provides game output formatting in various notations.
package output

import (
	"encoding/json"
	"io"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
)

// GameWriter is the interface for writing games to output.
// Different implementations handle different output formats (PGN, JSON, etc.).
type GameWriter interface {
	// WriteGame writes a single game to the output.
	WriteGame(game *chess.Game) error

	// Flush flushes any buffered data to the underlying writer.
	Flush() error

	// Close closes the writer and releases any resources.
	// For batch writers (like JSON), this also writes any pending output.
	Close() error
}

// PGNWriter writes games in PGN format.
type PGNWriter struct {
	w   io.Writer
	cfg *config.Config
}

// NewPGNWriter creates a new PGN writer.
func NewPGNWriter(w io.Writer, cfg *config.Config) *PGNWriter {
	return &PGNWriter{
		w:   w,
		cfg: cfg,
	}
}

// WriteGame writes a game in PGN format.
func (pw *PGNWriter) WriteGame(game *chess.Game) error {
	// Use the existing OutputGame function with a temporary config
	originalOutput := pw.cfg.OutputFile
	pw.cfg.OutputFile = pw.w
	OutputGame(game, pw.cfg)
	pw.cfg.OutputFile = originalOutput
	return nil
}

// Flush flushes the PGN writer (no-op for PGN as it writes immediately).
func (pw *PGNWriter) Flush() error {
	return nil
}

// Close closes the PGN writer.
func (pw *PGNWriter) Close() error {
	return nil
}

// JSONWriter writes games in JSON format.
// It buffers games and writes them as a JSON array on Close or Flush.
type JSONWriter struct {
	w      io.Writer
	cfg    *config.Config
	games  []*chess.Game
	single bool // If true, write each game immediately instead of batching
}

// NewJSONWriter creates a new JSON writer.
// By default, it batches games and writes them as an array on Close().
func NewJSONWriter(w io.Writer, cfg *config.Config) *JSONWriter {
	return &JSONWriter{
		w:      w,
		cfg:    cfg,
		games:  make([]*chess.Game, 0),
		single: false,
	}
}

// NewJSONWriterSingle creates a JSON writer that writes each game immediately.
func NewJSONWriterSingle(w io.Writer, cfg *config.Config) *JSONWriter {
	return &JSONWriter{
		w:      w,
		cfg:    cfg,
		single: true,
	}
}

// WriteGame buffers a game for JSON output (or writes immediately in single mode).
func (jw *JSONWriter) WriteGame(game *chess.Game) error {
	if jw.single {
		// Write immediately
		jsonGame := GameToJSON(game, jw.cfg)
		enc := json.NewEncoder(jw.w)
		enc.SetIndent("", "  ")
		return enc.Encode(jsonGame)
	}

	// Buffer for batch output
	jw.games = append(jw.games, game)
	return nil
}

// Flush writes all buffered games as a JSON array.
func (jw *JSONWriter) Flush() error {
	if jw.single || len(jw.games) == 0 {
		return nil
	}

	output := &JSONOutput{
		Games: make([]*JSONGame, 0, len(jw.games)),
	}

	for _, game := range jw.games {
		jsonGame := GameToJSON(game, jw.cfg)
		output.Games = append(output.Games, jsonGame)
	}

	enc := json.NewEncoder(jw.w)
	enc.SetIndent("", "  ")
	err := enc.Encode(output)

	// Clear buffer after writing
	jw.games = jw.games[:0]

	return err
}

// Close flushes and closes the JSON writer.
func (jw *JSONWriter) Close() error {
	return jw.Flush()
}
