// Package eco provides ECO (Encyclopaedia of Chess Openings) classification.
package eco

import (
	"fmt"
	"io"
	"os"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/hashing"
	"github.com/lgbarn/pgn-extract-go/internal/parser"
)

// ECOHalfMoveLimit is the maximum distance from an ECO line for a match.
const ECOHalfMoveLimit = 6

// ECOTableSize is the size of the ECO hash table.
const ECOTableSize = 4096

// ECOEntry represents a single ECO classification entry.
type ECOEntry struct {
	ECOCode        string // e.g., "B33"
	Opening        string // e.g., "Sicilian"
	Variation      string // e.g., "Sveshnikov"
	SubVariation   string
	RequiredHash   uint64 // Position hash for matching
	CumulativeHash uint64 // Cumulative hash of all moves
	HalfMoves      int    // Number of half-moves to reach this position
	Next           *ECOEntry
}

// ECOClassifier provides ECO classification for chess games.
type ECOClassifier struct {
	table         [ECOTableSize]*ECOEntry
	maxHalfMoves  int
	entriesLoaded int
}

// NewECOClassifier creates a new ECO classifier.
func NewECOClassifier() *ECOClassifier {
	return &ECOClassifier{
		maxHalfMoves: ECOHalfMoveLimit,
	}
}

// LoadFromFile loads ECO data from a PGN file.
func (ec *ECOClassifier) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cannot open ECO file: %w", err)
	}
	defer file.Close()

	return ec.LoadFromReader(file)
}

// LoadFromReader loads ECO data from a reader.
func (ec *ECOClassifier) LoadFromReader(r io.Reader) error {
	cfg := config.NewConfig()
	cfg.Verbosity = 0

	p := parser.NewParser(r, cfg)
	games, err := p.ParseAllGames()
	if err != nil {
		return fmt.Errorf("error parsing ECO file: %w", err)
	}

	for _, game := range games {
		ec.addECOEntry(game)
	}

	return nil
}

// addECOEntry processes a game from the ECO file and adds it to the table.
func (ec *ECOClassifier) addECOEntry(game *chess.Game) {
	ecoCode := game.Tags["ECO"]
	if ecoCode == "" {
		return
	}

	board, _ := engine.NewBoardFromFEN(engine.InitialFEN)
	var cumulativeHash uint64
	halfMoves := 0

	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}
		halfMoves++
		cumulativeHash ^= hashing.GenerateZobristHash(board)
	}

	if halfMoves == 0 {
		return
	}

	entry := &ECOEntry{
		ECOCode:        ecoCode,
		Opening:        game.Tags["Opening"],
		Variation:      game.Tags["Variation"],
		SubVariation:   game.Tags["SubVariation"],
		RequiredHash:   hashing.GenerateZobristHash(board),
		CumulativeHash: cumulativeHash,
		HalfMoves:      halfMoves,
	}

	if ec.isDuplicate(entry) {
		return
	}

	ec.insertEntry(entry)
}

// isDuplicate checks if an equivalent entry already exists in the table.
func (ec *ECOClassifier) isDuplicate(entry *ECOEntry) bool {
	ix := entry.RequiredHash % ECOTableSize
	for existing := ec.table[ix]; existing != nil; existing = existing.Next {
		if existing.RequiredHash == entry.RequiredHash &&
			existing.HalfMoves == entry.HalfMoves &&
			existing.CumulativeHash == entry.CumulativeHash {
			return true
		}
	}
	return false
}

// insertEntry adds an entry to the hash table.
func (ec *ECOClassifier) insertEntry(entry *ECOEntry) {
	ix := entry.RequiredHash % ECOTableSize
	entry.Next = ec.table[ix]
	ec.table[ix] = entry
	ec.entriesLoaded++

	if entry.HalfMoves+ECOHalfMoveLimit > ec.maxHalfMoves {
		ec.maxHalfMoves = entry.HalfMoves + ECOHalfMoveLimit
	}
}

// ClassifyGame finds the best ECO match for a game.
// Returns the ECO entry or nil if no match found.
func (ec *ECOClassifier) ClassifyGame(game *chess.Game) *ECOEntry {
	if ec.entriesLoaded == 0 {
		return nil
	}

	board := ec.boardForGame(game)

	var bestMatch *ECOEntry
	var cumulativeHash uint64
	halfMoves := 0

	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}
		halfMoves++

		if halfMoves > ec.maxHalfMoves {
			break
		}

		posHash := hashing.GenerateZobristHash(board)
		cumulativeHash ^= posHash

		if match := ec.findMatch(posHash, cumulativeHash, halfMoves); match != nil {
			bestMatch = match
		}
	}

	return bestMatch
}

// boardForGame returns a board initialized for the game's starting position.
func (ec *ECOClassifier) boardForGame(game *chess.Game) *chess.Board {
	if fen, ok := game.Tags["FEN"]; ok {
		if board, err := engine.NewBoardFromFEN(fen); err == nil {
			return board
		}
	}
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN)
	return board
}

// findMatch looks up a position in the ECO table.
func (ec *ECOClassifier) findMatch(posHash, cumulativeHash uint64, halfMoves int) *ECOEntry {
	ix := posHash % ECOTableSize
	var partialMatch *ECOEntry

	for entry := ec.table[ix]; entry != nil; entry = entry.Next {
		if entry.RequiredHash != posHash {
			continue
		}

		// Exact match takes precedence
		if entry.HalfMoves == halfMoves && entry.CumulativeHash == cumulativeHash {
			return entry
		}

		// Track partial match within limit
		if abs(halfMoves-entry.HalfMoves) <= ECOHalfMoveLimit {
			partialMatch = entry
		}
	}

	return partialMatch
}

// AddECOTags adds ECO, Opening, and Variation tags to a game.
func (ec *ECOClassifier) AddECOTags(game *chess.Game) bool {
	match := ec.ClassifyGame(game)
	if match == nil {
		return false
	}

	setTagIfNotEmpty(game, "ECO", match.ECOCode)
	setTagIfNotEmpty(game, "Opening", match.Opening)
	setTagIfNotEmpty(game, "Variation", match.Variation)
	setTagIfNotEmpty(game, "SubVariation", match.SubVariation)

	return true
}

func setTagIfNotEmpty(game *chess.Game, key, value string) {
	if value != "" {
		game.Tags[key] = value
	}
}

// EntriesLoaded returns the number of ECO entries loaded.
func (ec *ECOClassifier) EntriesLoaded() int {
	return ec.entriesLoaded
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
