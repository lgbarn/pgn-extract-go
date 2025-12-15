// pgn-extract is a tool for searching, manipulating, and formatting chess games in PGN format.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/eco"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/hashing"
	"github.com/lgbarn/pgn-extract-go/internal/matching"
	"github.com/lgbarn/pgn-extract-go/internal/output"
	"github.com/lgbarn/pgn-extract-go/internal/parser"
)

var (
	// Output options
	outputFile    = flag.String("o", "", "Output file (default: stdout)")
	sevenTagOnly  = flag.Bool("7", false, "Output only the seven tag roster")
	noTags        = flag.Bool("notags", false, "Don't output any tags")
	lineLength    = flag.Int("w", 80, "Maximum line length")
	outputFormat  = flag.String("W", "", "Output format: san, lalg, halg, elalg, uci")
	jsonOutput    = flag.Bool("J", false, "Output in JSON format")

	// Content options
	noComments    = flag.Bool("C", false, "Don't output comments")
	noNAGs        = flag.Bool("N", false, "Don't output NAGs")
	noVariations  = flag.Bool("V", false, "Don't output variations")
	noResults     = flag.Bool("noresults", false, "Don't output results")

	// Duplicate detection
	suppressDuplicates = flag.Bool("D", false, "Suppress duplicate games")
	duplicateFile      = flag.String("d", "", "Output duplicates to this file")

	// ECO classification
	ecoFile = flag.String("e", "", "ECO classification file (PGN format)")

	// Filtering options
	tagFile      = flag.String("t", "", "Tag criteria file for filtering")
	playerFilter = flag.String("p", "", "Filter by player name (either color)")
	whiteFilter  = flag.String("Tw", "", "Filter by White player")
	blackFilter  = flag.String("Tb", "", "Filter by Black player")
	ecoFilter    = flag.String("Te", "", "Filter by ECO code prefix")
	resultFilter = flag.String("Tr", "", "Filter by result (1-0, 0-1, 1/2-1/2)")
	fenFilter    = flag.String("Tf", "", "Filter by FEN position")

	// Ending filters
	checkmateFilter = flag.Bool("checkmate", false, "Only output games ending in checkmate")
	stalemateFilter = flag.Bool("stalemate", false, "Only output games ending in stalemate")

	// Other options
	quiet         = flag.Bool("s", false, "Silent mode (no game count)")
	help          = flag.Bool("h", false, "Show help")
	version       = flag.Bool("version", false, "Show version")
)

const programVersion = "0.1.0"

func main() {
	flag.Usage = usage
	flag.Parse()

	if *help {
		usage()
		os.Exit(0)
	}

	if *version {
		fmt.Printf("pgn-extract-go version %s\n", programVersion)
		os.Exit(0)
	}

	cfg := config.NewConfig()

	// Apply flags to config
	applyFlags(cfg)

	// Set up output file
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file %s: %v\n", *outputFile, err)
			os.Exit(1)
		}
		defer file.Close()
		cfg.OutputFile = file
	}

	// Set up duplicate file
	var dupFile *os.File
	if *duplicateFile != "" {
		var err error
		dupFile, err = os.Create(*duplicateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating duplicate file %s: %v\n", *duplicateFile, err)
			os.Exit(1)
		}
		defer dupFile.Close()
		cfg.DuplicateFile = dupFile
	}

	// Create duplicate detector if needed
	var detector *hashing.DuplicateDetector
	if *suppressDuplicates || *duplicateFile != "" {
		detector = hashing.NewDuplicateDetector(false)
		cfg.SuppressDuplicates = *suppressDuplicates
	}

	// Load ECO file if specified
	var ecoClassifier *eco.ECOClassifier
	if *ecoFile != "" {
		ecoClassifier = eco.NewECOClassifier()
		if err := ecoClassifier.LoadFromFile(*ecoFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading ECO file %s: %v\n", *ecoFile, err)
			os.Exit(1)
		}
		if cfg.Verbosity > 0 {
			fmt.Fprintf(os.Stderr, "Loaded %d ECO entries\n", ecoClassifier.EntriesLoaded())
		}
		cfg.AddECO = true
	}

	// Set up game filter
	gameFilter := matching.NewGameFilter()

	// Load tag criteria file if specified
	if *tagFile != "" {
		if err := gameFilter.LoadTagFile(*tagFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading tag file %s: %v\n", *tagFile, err)
			os.Exit(1)
		}
	}

	// Add individual filter criteria
	if *playerFilter != "" {
		gameFilter.AddPlayerFilter(*playerFilter)
	}
	if *whiteFilter != "" {
		gameFilter.AddWhiteFilter(*whiteFilter)
	}
	if *blackFilter != "" {
		gameFilter.AddBlackFilter(*blackFilter)
	}
	if *ecoFilter != "" {
		gameFilter.AddECOFilter(*ecoFilter)
	}
	if *resultFilter != "" {
		gameFilter.AddResultFilter(*resultFilter)
	}
	if *fenFilter != "" {
		if err := gameFilter.AddFENFilter(*fenFilter); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing FEN filter: %v\n", err)
			os.Exit(1)
		}
	}

	// Process input files or stdin
	args := flag.Args()
	var totalGames, outputGames, duplicates int

	if len(args) == 0 {
		// Read from stdin
		games := processInput(os.Stdin, "stdin", cfg)
		totalGames = len(games)
		outputGames, duplicates = outputGamesWithProcessing(games, cfg, detector, ecoClassifier, gameFilter)
	} else {
		// Process each input file
		for _, filename := range args {
			file, err := os.Open(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening file %s: %v\n", filename, err)
				continue
			}

			games := processInput(file, filename, cfg)
			totalGames += len(games)
			out, dup := outputGamesWithProcessing(games, cfg, detector, ecoClassifier, gameFilter)
			outputGames += out
			duplicates += dup

			file.Close()
		}
	}

	// Report statistics
	if cfg.Verbosity > 0 && !*quiet {
		if detector != nil {
			fmt.Fprintf(os.Stderr, "%d game(s) output, %d duplicate(s) out of %d.\n", outputGames, duplicates, totalGames)
		} else {
			fmt.Fprintf(os.Stderr, "%d game(s) matched out of %d.\n", totalGames, totalGames)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: pgn-extract [options] [input-files...]\n\n")
	fmt.Fprintf(os.Stderr, "A tool for manipulating chess games in PGN format.\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nOutput formats (-W):\n")
	fmt.Fprintf(os.Stderr, "  san    Standard Algebraic Notation (default)\n")
	fmt.Fprintf(os.Stderr, "  lalg   Long algebraic (e2e4)\n")
	fmt.Fprintf(os.Stderr, "  halg   Hyphenated long algebraic (e2-e4)\n")
	fmt.Fprintf(os.Stderr, "  elalg  Enhanced long algebraic (Ng1f3)\n")
	fmt.Fprintf(os.Stderr, "  uci    UCI format\n")
}

func applyFlags(cfg *config.Config) {
	// Tag output
	if *sevenTagOnly {
		cfg.TagOutputFormat = config.SevenTagRoster
	}
	if *noTags {
		cfg.TagOutputFormat = config.NoTags
	}

	// Content
	cfg.KeepComments = !*noComments
	cfg.KeepNAGs = !*noNAGs
	cfg.KeepVariations = !*noVariations
	cfg.KeepResults = !*noResults

	// Line length
	cfg.MaxLineLength = uint(*lineLength)

	// Output format
	switch *outputFormat {
	case "lalg":
		cfg.OutputFormat = config.LALG
	case "halg":
		cfg.OutputFormat = config.HALG
	case "elalg":
		cfg.OutputFormat = config.ELALG
	case "uci":
		cfg.OutputFormat = config.UCI
	default:
		cfg.OutputFormat = config.SAN
	}

	// Verbosity
	if *quiet {
		cfg.Verbosity = 0
	}

	// JSON output
	cfg.JSONFormat = *jsonOutput
}

func processInput(r io.Reader, name string, cfg *config.Config) []*chess.Game {
	cfg.CurrentInputFile = name

	p := parser.NewParser(r, cfg)
	games, err := p.ParseAllGames()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", name, err)
	}

	return games
}

// outputGamesWithProcessing outputs games with optional filtering, ECO classification, and duplicate detection.
// Returns the number of games output and the number of duplicates found.
func outputGamesWithProcessing(games []*chess.Game, cfg *config.Config, detector *hashing.DuplicateDetector, ecoClassifier *eco.ECOClassifier, gameFilter *matching.GameFilter) (int, int) {
	outputCount := 0
	duplicateCount := 0

	// For JSON output, collect games first then output all at once
	var jsonGames []*chess.Game

	for _, game := range games {
		// Add ECO tags if classifier is available
		if ecoClassifier != nil {
			ecoClassifier.AddECOTags(game)
		}

		// Check filter criteria
		if gameFilter != nil && gameFilter.HasCriteria() {
			if !gameFilter.MatchGame(game) {
				continue // Skip non-matching games
			}
		}

		// Replay the game to get final position (needed for checkmate/stalemate and duplicate detection)
		var board *chess.Board
		if *checkmateFilter || *stalemateFilter || detector != nil {
			board = replayGame(game)
		}

		// Check checkmate filter
		if *checkmateFilter {
			if !engine.IsCheckmate(board) {
				continue // Skip non-checkmate games
			}
		}

		// Check stalemate filter
		if *stalemateFilter {
			if !engine.IsStalemate(board) {
				continue // Skip non-stalemate games
			}
		}

		// If no duplicate detection, just output
		if detector == nil {
			if cfg.JSONFormat {
				jsonGames = append(jsonGames, game)
			} else {
				output.OutputGame(game, cfg)
			}
			outputCount++
			continue
		}

		// board is already replayed above

		// Check for duplicate
		isDuplicate := detector.CheckAndAdd(game, board)

		if isDuplicate {
			duplicateCount++
			// Output to duplicate file if configured
			if cfg.DuplicateFile != nil {
				originalOutput := cfg.OutputFile
				cfg.OutputFile = cfg.DuplicateFile
				if cfg.JSONFormat {
					output.OutputGameJSON(game, cfg)
				} else {
					output.OutputGame(game, cfg)
				}
				cfg.OutputFile = originalOutput
			}
		} else {
			// Not a duplicate - output normally
			if !cfg.SuppressDuplicates || !isDuplicate {
				if cfg.JSONFormat {
					jsonGames = append(jsonGames, game)
				} else {
					output.OutputGame(game, cfg)
				}
				outputCount++
			}
		}
	}

	// Output JSON array if JSON mode
	if cfg.JSONFormat && len(jsonGames) > 0 {
		output.OutputGamesJSON(jsonGames, cfg, cfg.OutputFile)
	}

	return outputCount, duplicateCount
}

// replayGame replays a game from the initial position to get the final board state.
func replayGame(game *chess.Game) *chess.Board {
	var board *chess.Board
	var err error

	// Check if game has a custom starting position (FEN tag)
	if fen, ok := game.Tags["FEN"]; ok {
		board, err = engine.NewBoardFromFEN(fen)
		if err != nil {
			// Fall back to initial position
			board, _ = engine.NewBoardFromFEN(engine.InitialFEN)
		}
	} else {
		board, _ = engine.NewBoardFromFEN(engine.InitialFEN)
	}

	// Apply all moves
	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			// Move application failed - return current board state
			break
		}
	}

	return board
}
