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
	// Extract ECO tags
	ecoCode := game.Tags["ECO"]
	opening := game.Tags["Opening"]
	variation := game.Tags["Variation"]
	subVariation := game.Tags["SubVariation"]

	if ecoCode == "" {
		return // Skip entries without ECO code
	}

	// Replay the game to get position hashes
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN)
	var cumulativeHash uint64
	halfMoves := 0

	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			// Move failed, stop here
			break
		}
		halfMoves++
		// Accumulate hash (simple XOR for cumulative)
		posHash := hashing.GenerateZobristHash(board)
		cumulativeHash ^= posHash
	}

	if halfMoves == 0 {
		return // No moves in this entry
	}

	// Create ECO entry
	entry := &ECOEntry{
		ECOCode:        ecoCode,
		Opening:        opening,
		Variation:      variation,
		SubVariation:   subVariation,
		RequiredHash:   hashing.GenerateZobristHash(board),
		CumulativeHash: cumulativeHash,
		HalfMoves:      halfMoves,
	}

	// Check for collision
	ix := entry.RequiredHash % ECOTableSize
	for existing := ec.table[ix]; existing != nil; existing = existing.Next {
		if existing.RequiredHash == entry.RequiredHash &&
			existing.HalfMoves == entry.HalfMoves &&
			existing.CumulativeHash == entry.CumulativeHash {
			// Collision - skip this entry
			return
		}
	}

	// Add to table
	entry.Next = ec.table[ix]
	ec.table[ix] = entry
	ec.entriesLoaded++

	// Update max half moves
	if halfMoves+ECOHalfMoveLimit > ec.maxHalfMoves {
		ec.maxHalfMoves = halfMoves + ECOHalfMoveLimit
	}
}

// ClassifyGame finds the best ECO match for a game.
// Returns the ECO entry or nil if no match found.
func (ec *ECOClassifier) ClassifyGame(game *chess.Game) *ECOEntry {
	if ec.entriesLoaded == 0 {
		return nil
	}

	// Check if game has custom start position
	var board *chess.Board
	var err error
	if fen, ok := game.Tags["FEN"]; ok {
		board, err = engine.NewBoardFromFEN(fen)
		if err != nil {
			board, _ = engine.NewBoardFromFEN(engine.InitialFEN)
		}
	} else {
		board, _ = engine.NewBoardFromFEN(engine.InitialFEN)
	}

	var bestMatch *ECOEntry
	var cumulativeHash uint64
	halfMoves := 0

	// Replay game and check each position
	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}
		halfMoves++

		// Don't bother checking if we're past max ECO depth
		if halfMoves > ec.maxHalfMoves {
			break
		}

		posHash := hashing.GenerateZobristHash(board)
		cumulativeHash ^= posHash

		// Look for match
		match := ec.findMatch(posHash, cumulativeHash, halfMoves)
		if match != nil {
			bestMatch = match
		}
	}

	return bestMatch
}

// findMatch looks up a position in the ECO table.
func (ec *ECOClassifier) findMatch(posHash, cumulativeHash uint64, halfMoves int) *ECOEntry {
	ix := posHash % ECOTableSize
	var possible *ECOEntry

	for entry := ec.table[ix]; entry != nil; entry = entry.Next {
		if entry.RequiredHash == posHash {
			// Exact match on position and cumulative hash
			if entry.HalfMoves == halfMoves && entry.CumulativeHash == cumulativeHash {
				return entry
			}
			// Partial match within limit
			if abs(halfMoves-entry.HalfMoves) <= ECOHalfMoveLimit {
				possible = entry
			}
		}
	}

	return possible
}

// AddECOTags adds ECO, Opening, and Variation tags to a game.
func (ec *ECOClassifier) AddECOTags(game *chess.Game) bool {
	match := ec.ClassifyGame(game)
	if match == nil {
		return false
	}

	if match.ECOCode != "" {
		game.Tags["ECO"] = match.ECOCode
	}
	if match.Opening != "" {
		game.Tags["Opening"] = match.Opening
	}
	if match.Variation != "" {
		game.Tags["Variation"] = match.Variation
	}
	if match.SubVariation != "" {
		game.Tags["SubVariation"] = match.SubVariation
	}

	return true
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
