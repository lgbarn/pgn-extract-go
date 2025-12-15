// pgn-extract is a tool for searching, manipulating, and formatting chess games in PGN format.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/cql"
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
	appendOutput  = flag.Bool("a", false, "Append to output file instead of overwrite")
	sevenTagOnly  = flag.Bool("7", false, "Output only the seven tag roster")
	noTags        = flag.Bool("notags", false, "Don't output any tags")
	lineLength    = flag.Int("w", 80, "Maximum line length")
	outputFormat  = flag.String("W", "", "Output format: san, lalg, halg, elalg, uci, epd, fen")
	jsonOutput    = flag.Bool("J", false, "Output in JSON format")
	splitGames    = flag.Int("#", 0, "Split output into files of N games each")
	splitByECO    = flag.String("E", "", "Split output by ECO level (1-3)")

	// Content options
	noComments    = flag.Bool("C", false, "Don't output comments")
	noNAGs        = flag.Bool("N", false, "Don't output NAGs")
	noVariations  = flag.Bool("V", false, "Don't output variations")
	noResults     = flag.Bool("noresults", false, "Don't output results")

	// Duplicate detection
	suppressDuplicates = flag.Bool("D", false, "Suppress duplicate games")
	duplicateFile      = flag.String("d", "", "Output duplicates to this file")
	outputDupsOnly     = flag.Bool("U", false, "Output only duplicates (suppress unique games)")
	checkFile          = flag.String("c", "", "Check file for duplicate detection")

	// ECO classification
	ecoFile = flag.String("e", "", "ECO classification file (PGN format)")

	// Filtering options
	tagFile          = flag.String("t", "", "Tag criteria file for filtering")
	playerFilter     = flag.String("p", "", "Filter by player name (either color)")
	whiteFilter      = flag.String("Tw", "", "Filter by White player")
	blackFilter      = flag.String("Tb", "", "Filter by Black player")
	ecoFilter        = flag.String("Te", "", "Filter by ECO code prefix")
	resultFilter     = flag.String("Tr", "", "Filter by result (1-0, 0-1, 1/2-1/2)")
	fenFilter        = flag.String("Tf", "", "Filter by FEN position")
	negateMatch      = flag.Bool("n", false, "Output games that DON'T match criteria")
	useSoundex       = flag.Bool("S", false, "Use Soundex for player name matching")
	tagSubstring     = flag.Bool("tagsubstr", false, "Match tag values anywhere (substring)")

	// Ply/move bounds
	minPly    = flag.Int("minply", 0, "Minimum ply count")
	maxPly    = flag.Int("maxply", 0, "Maximum ply count (0 = no limit)")
	minMoves  = flag.Int("minmoves", 0, "Minimum number of moves")
	maxMoves  = flag.Int("maxmoves", 0, "Maximum number of moves (0 = no limit)")
	stopAfter = flag.Int("stopafter", 0, "Stop after matching N games")

	// Ending filters
	checkmateFilter = flag.Bool("checkmate", false, "Only output games ending in checkmate")
	stalemateFilter = flag.Bool("stalemate", false, "Only output games ending in stalemate")

	// Game feature filters
	fiftyMoveFilter       = flag.Bool("fifty", false, "Games with 50-move rule")
	repetitionFilter      = flag.Bool("repetition", false, "Games with 3-fold repetition")
	underpromotionFilter  = flag.Bool("underpromotion", false, "Games with underpromotion")
	commentedFilter       = flag.Bool("commented", false, "Only games with comments")
	higherRatedWinner     = flag.Bool("higherratedwinner", false, "Higher-rated player won")
	lowerRatedWinner      = flag.Bool("lowerratedwinner", false, "Lower-rated player won")

	// CQL filter
	cqlQuery = flag.String("cql", "", "CQL query to filter games by position patterns")
	cqlFile  = flag.String("cql-file", "", "File containing CQL query")

	// Variation matching
	variationFile = flag.String("v", "", "File with move sequences to match")
	positionFile  = flag.String("x", "", "File with positional variations to match")

	// Material matching
	materialMatch      = flag.String("z", "", "Material balance to match (e.g., 'QR:qrr')")
	materialMatchExact = flag.String("y", "", "Exact material balance to match")

	// Annotations
	addPlyCount    = flag.Bool("plycount", false, "Add PlyCount tag")
	addFENComments = flag.Bool("fencomments", false, "Add FEN comment after each move")
	addHashComments = flag.Bool("hashcomments", false, "Add position hash after each move")
	addHashcodeTag = flag.Bool("addhashcode", false, "Add HashCode tag")

	// Tag management
	fixResultTags  = flag.Bool("fixresulttags", false, "Fix inconsistent result tags")
	fixTagStrings  = flag.Bool("fixtagstrings", false, "Fix malformed tag strings")

	// Validation
	strictMode   = flag.Bool("strict", false, "Only output games that parse without errors")
	validateMode = flag.Bool("validate", false, "Verify all moves are legal")
	fixableMode  = flag.Bool("fixable", false, "Attempt to fix common issues")

	// Logging
	logFile    = flag.String("l", "", "Write diagnostics to log file")
	appendLog  = flag.String("L", "", "Append diagnostics to log file")
	reportOnly = flag.Bool("r", false, "Report errors without extracting games")

	// Polyglot hash
	hashMatch = flag.String("H", "", "Match positions by polyglot hashcode")

	// Other options
	quiet         = flag.Bool("s", false, "Silent mode (no game count)")
	help          = flag.Bool("h", false, "Show help")
	version       = flag.Bool("version", false, "Show version")
)

const programVersion = "0.1.0"

// Global state for stopAfter
var matchedCount int

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

	// Set up logging
	if *logFile != "" {
		file, err := os.Create(*logFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating log file %s: %v\n", *logFile, err)
			os.Exit(1)
		}
		defer file.Close()
		cfg.LogFile = file
	}
	if *appendLog != "" {
		file, err := os.OpenFile(*appendLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening log file %s: %v\n", *appendLog, err)
			os.Exit(1)
		}
		defer file.Close()
		cfg.LogFile = file
	}

	// Set up output file
	if *outputFile != "" {
		var file *os.File
		var err error
		if *appendOutput {
			file, err = os.OpenFile(*outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			file, err = os.Create(*outputFile)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file %s: %v\n", *outputFile, err)
			os.Exit(1)
		}
		defer file.Close()
		cfg.OutputFile = file
	}

	// Set up non-matching file for -n flag
	var nonMatchFile *os.File
	if *negateMatch && *outputFile != "" {
		cfg.NonMatchingFile = cfg.OutputFile
		cfg.OutputFile = nil // Don't output matching games
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
	if *suppressDuplicates || *duplicateFile != "" || *outputDupsOnly || *checkFile != "" {
		detector = hashing.NewDuplicateDetector(false)
		cfg.SuppressDuplicates = *suppressDuplicates
		cfg.SuppressOriginals = *outputDupsOnly
	}

	// Load check file for duplicate detection
	if *checkFile != "" && detector != nil {
		file, err := os.Open(*checkFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening check file %s: %v\n", *checkFile, err)
			os.Exit(1)
		}
		checkGames := processInput(file, *checkFile, cfg)
		file.Close()
		// Add all games from checkfile to detector
		for _, game := range checkGames {
			board := replayGame(game)
			detector.CheckAndAdd(game, board)
		}
		if cfg.Verbosity > 0 {
			fmt.Fprintf(cfg.LogFile, "Loaded %d games from check file\n", len(checkGames))
		}
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
			fmt.Fprintf(cfg.LogFile, "Loaded %d ECO entries\n", ecoClassifier.EntriesLoaded())
		}
		cfg.AddECO = true
	}

	// Set up game filter
	gameFilter := matching.NewGameFilter()
	gameFilter.SetUseSoundex(*useSoundex)
	gameFilter.SetSubstringMatch(*tagSubstring)

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

	// Load variation file if specified
	var variationMatcher *matching.VariationMatcher
	if *variationFile != "" {
		variationMatcher = matching.NewVariationMatcher()
		if err := variationMatcher.LoadFromFile(*variationFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading variation file %s: %v\n", *variationFile, err)
			os.Exit(1)
		}
	}
	if *positionFile != "" {
		if variationMatcher == nil {
			variationMatcher = matching.NewVariationMatcher()
		}
		if err := variationMatcher.LoadPositionalFromFile(*positionFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading position file %s: %v\n", *positionFile, err)
			os.Exit(1)
		}
	}

	// Parse material match criteria
	var materialMatcher *matching.MaterialMatcher
	if *materialMatch != "" {
		materialMatcher = matching.NewMaterialMatcher(*materialMatch, false)
	}
	if *materialMatchExact != "" {
		materialMatcher = matching.NewMaterialMatcher(*materialMatchExact, true)
	}

	// Parse CQL query if specified (from file or command line)
	var cqlNode cql.Node
	cqlQueryStr := *cqlQuery
	if *cqlFile != "" {
		content, err := os.ReadFile(*cqlFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading CQL file %s: %v\n", *cqlFile, err)
			os.Exit(1)
		}
		cqlQueryStr = strings.TrimSpace(string(content))
	}
	if cqlQueryStr != "" {
		var err error
		cqlNode, err = cql.Parse(cqlQueryStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing CQL query: %v\n", err)
			os.Exit(1)
		}
	}

	// Set up output splitting
	var splitWriter *SplitWriter
	if *splitGames > 0 {
		base := "output"
		if *outputFile != "" {
			base = strings.TrimSuffix(*outputFile, filepath.Ext(*outputFile))
		}
		splitWriter = NewSplitWriter(base, *splitGames)
		cfg.OutputFile = splitWriter
	}

	// Process input files or stdin
	args := flag.Args()
	var totalGames, outputGames, duplicates int

	// Create processing context
	ctx := &ProcessingContext{
		cfg:              cfg,
		detector:         detector,
		ecoClassifier:    ecoClassifier,
		gameFilter:       gameFilter,
		cqlNode:          cqlNode,
		variationMatcher: variationMatcher,
		materialMatcher:  materialMatcher,
	}

	if len(args) == 0 {
		// Read from stdin
		games := processInput(os.Stdin, "stdin", cfg)
		totalGames = len(games)
		outputGames, duplicates = outputGamesWithProcessing(games, ctx)
	} else {
		// Process each input file
		for _, filename := range args {
			// Check if we should stop
			if *stopAfter > 0 && matchedCount >= *stopAfter {
				break
			}

			file, err := os.Open(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening file %s: %v\n", filename, err)
				continue
			}

			games := processInput(file, filename, cfg)
			totalGames += len(games)
			out, dup := outputGamesWithProcessing(games, ctx)
			outputGames += out
			duplicates += dup

			file.Close()
		}
	}

	// Close split writer if used
	if splitWriter != nil {
		splitWriter.Close()
	}

	// Close non-match file
	if nonMatchFile != nil {
		nonMatchFile.Close()
	}

	// Report statistics
	if cfg.Verbosity > 0 && !*quiet && !*reportOnly {
		if detector != nil {
			fmt.Fprintf(os.Stderr, "%d game(s) output, %d duplicate(s) out of %d.\n", outputGames, duplicates, totalGames)
		} else {
			fmt.Fprintf(os.Stderr, "%d game(s) matched out of %d.\n", outputGames, totalGames)
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
	fmt.Fprintf(os.Stderr, "  epd    Extended Position Description\n")
	fmt.Fprintf(os.Stderr, "  fen    FEN sequence\n")
}

// ProcessingContext holds all processing state
type ProcessingContext struct {
	cfg              *config.Config
	detector         *hashing.DuplicateDetector
	ecoClassifier    *eco.ECOClassifier
	gameFilter       *matching.GameFilter
	cqlNode          cql.Node
	variationMatcher *matching.VariationMatcher
	materialMatcher  *matching.MaterialMatcher
}

// SplitWriter handles writing to multiple output files
type SplitWriter struct {
	baseName    string
	gamesPerFile int
	currentFile *os.File
	fileNumber  int
	gameCount   int
}

// NewSplitWriter creates a new split writer
func NewSplitWriter(baseName string, gamesPerFile int) *SplitWriter {
	return &SplitWriter{
		baseName:    baseName,
		gamesPerFile: gamesPerFile,
		fileNumber:  1,
	}
}

// Write implements io.Writer
func (sw *SplitWriter) Write(p []byte) (n int, err error) {
	if sw.currentFile == nil || sw.gameCount >= sw.gamesPerFile {
		if sw.currentFile != nil {
			sw.currentFile.Close()
			sw.fileNumber++
		}
		filename := fmt.Sprintf("%s_%d.pgn", sw.baseName, sw.fileNumber)
		sw.currentFile, err = os.Create(filename)
		if err != nil {
			return 0, err
		}
		sw.gameCount = 0
	}
	return sw.currentFile.Write(p)
}

// IncrementGameCount should be called after each game is written
func (sw *SplitWriter) IncrementGameCount() {
	sw.gameCount++
}

// Close closes the current file
func (sw *SplitWriter) Close() error {
	if sw.currentFile != nil {
		return sw.currentFile.Close()
	}
	return nil
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
	case "epd":
		cfg.OutputFormat = config.EPD
	case "fen":
		cfg.OutputFormat = config.FEN
	default:
		cfg.OutputFormat = config.SAN
	}

	// Verbosity
	if *quiet {
		cfg.Verbosity = 0
	}

	// JSON output
	cfg.JSONFormat = *jsonOutput

	// Ply/move bounds
	if *minPly > 0 || *maxPly > 0 || *minMoves > 0 || *maxMoves > 0 {
		cfg.CheckMoveBounds = true
		if *minMoves > 0 {
			cfg.LowerMoveBound = uint(*minMoves)
		}
		if *maxMoves > 0 {
			cfg.UpperMoveBound = uint(*maxMoves)
		}
	}

	// Annotations
	cfg.OutputPlycount = *addPlyCount
	cfg.AddFENComments = *addFENComments
	cfg.AddHashcodeComments = *addHashComments
	cfg.AddHashcodeTag = *addHashcodeTag

	// Tag fixing
	cfg.FixResultTags = *fixResultTags
	cfg.FixTagStrings = *fixTagStrings

	// Game feature matching
	cfg.MatchOnlyCheckmate = *checkmateFilter
	cfg.MatchOnlyStalemate = *stalemateFilter
	cfg.CheckForFiftyMoveRule = *fiftyMoveFilter
	cfg.CheckForRepetition = *repetitionFilter
	cfg.MatchUnderpromotion = *underpromotionFilter

	// Soundex
	cfg.UseSoundex = *useSoundex

	// Report only mode
	cfg.CheckOnly = *reportOnly
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
func outputGamesWithProcessing(games []*chess.Game, ctx *ProcessingContext) (int, int) {
	cfg := ctx.cfg
	detector := ctx.detector
	ecoClassifier := ctx.ecoClassifier
	gameFilter := ctx.gameFilter
	cqlNode := ctx.cqlNode

	outputCount := 0
	duplicateCount := 0

	// For JSON output, collect games first then output all at once
	var jsonGames []*chess.Game

	for _, game := range games {
		// Check if we should stop
		if *stopAfter > 0 && matchedCount >= *stopAfter {
			break
		}

		// Apply fixes if requested (do this before validation)
		if *fixableMode {
			fixGame(game)
		}

		// Validation checks
		if *strictMode || *validateMode {
			validResult := validateGame(game)

			// In strict mode, skip games with any parse errors
			if *strictMode && len(validResult.ParseErrors) > 0 {
				if !*quiet {
					for _, err := range validResult.ParseErrors {
						fmt.Fprintf(os.Stderr, "Skipping game (strict): %s\n", err)
					}
				}
				continue
			}

			// In validate mode, skip games with illegal moves
			if *validateMode && !validResult.Valid {
				if !*quiet {
					fmt.Fprintf(os.Stderr, "Skipping game (invalid): %s\n", validResult.ErrorMsg)
				}
				continue
			}
		}

		// Add ECO tags if classifier is available
		if ecoClassifier != nil {
			ecoClassifier.AddECOTags(game)
		}

		// Check filter criteria
		matched := true
		if gameFilter != nil && gameFilter.HasCriteria() {
			matched = gameFilter.MatchGame(game)
		}

		// Check CQL filter - evaluate against every position in the game
		if matched && cqlNode != nil {
			matched = matchesCQL(game, cqlNode)
		}

		// Check variation matcher
		if matched && ctx.variationMatcher != nil {
			matched = ctx.variationMatcher.MatchGame(game)
		}

		// Check material matcher
		if matched && ctx.materialMatcher != nil {
			matched = ctx.materialMatcher.MatchGame(game)
		}

		// Calculate ply count for bounds checking
		plyCount := countPlies(game)

		// Check ply bounds
		if matched && *minPly > 0 && plyCount < *minPly {
			matched = false
		}
		if matched && *maxPly > 0 && plyCount > *maxPly {
			matched = false
		}

		// Check move bounds (moves = plies / 2, rounded up)
		moveCount := (plyCount + 1) / 2
		if matched && *minMoves > 0 && moveCount < *minMoves {
			matched = false
		}
		if matched && *maxMoves > 0 && moveCount > *maxMoves {
			matched = false
		}

		// Replay the game to get final position and game info
		var board *chess.Board
		var gameInfo *GameAnalysis
		needsReplay := *checkmateFilter || *stalemateFilter || detector != nil ||
			*fiftyMoveFilter || *repetitionFilter || *underpromotionFilter ||
			*higherRatedWinner || *lowerRatedWinner || cfg.AddFENComments || cfg.AddHashcodeComments ||
			cfg.AddHashcodeTag

		if needsReplay {
			board, gameInfo = analyzeGame(game)
		}

		// Check checkmate filter
		if matched && *checkmateFilter {
			if !engine.IsCheckmate(board) {
				matched = false
			}
		}

		// Check stalemate filter
		if matched && *stalemateFilter {
			if !engine.IsStalemate(board) {
				matched = false
			}
		}

		// Check fifty-move rule filter
		if matched && *fiftyMoveFilter {
			if !gameInfo.HasFiftyMoveRule {
				matched = false
			}
		}

		// Check repetition filter
		if matched && *repetitionFilter {
			if !gameInfo.HasRepetition {
				matched = false
			}
		}

		// Check underpromotion filter
		if matched && *underpromotionFilter {
			if !gameInfo.HasUnderpromotion {
				matched = false
			}
		}

		// Check commented filter
		if matched && *commentedFilter {
			if !hasComments(game) {
				matched = false
			}
		}

		// Check rating-based winner filters
		if matched && (*higherRatedWinner || *lowerRatedWinner) {
			whiteElo := parseElo(game.Tags["WhiteElo"])
			blackElo := parseElo(game.Tags["BlackElo"])
			result := game.Tags["Result"]

			if whiteElo > 0 && blackElo > 0 {
				if *higherRatedWinner {
					higherWon := (whiteElo > blackElo && result == "1-0") ||
						(blackElo > whiteElo && result == "0-1")
					if !higherWon {
						matched = false
					}
				}
				if *lowerRatedWinner {
					lowerWon := (whiteElo < blackElo && result == "1-0") ||
						(blackElo < whiteElo && result == "0-1")
					if !lowerWon {
						matched = false
					}
				}
			} else {
				matched = false // No rating info available
			}
		}

		// Handle negated matching
		if *negateMatch {
			matched = !matched
		}

		// If not matched, skip or output to non-matching file
		if !matched {
			if cfg.NonMatchingFile != nil {
				originalOutput := cfg.OutputFile
				cfg.OutputFile = cfg.NonMatchingFile
				output.OutputGame(game, cfg)
				cfg.OutputFile = originalOutput
			}
			continue
		}

		// Report-only mode - don't output games
		if *reportOnly {
			matchedCount++
			outputCount++
			continue
		}

		// Add plycount tag if requested
		if cfg.OutputPlycount {
			game.Tags["PlyCount"] = strconv.Itoa(plyCount)
		}

		// Add hashcode tag if requested
		if cfg.AddHashcodeTag && board != nil {
			hash := hashing.GenerateZobristHash(board)
			game.Tags["HashCode"] = fmt.Sprintf("%016x", hash)
		}

		// Handle duplicate detection
		if detector != nil {
			if board == nil {
				board = replayGame(game)
			}

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
				// If outputting only duplicates
				if cfg.SuppressOriginals {
					outputGameWithAnnotations(game, cfg, gameInfo, &jsonGames)
					matchedCount++
					outputCount++
				}
			} else {
				// Not a duplicate - output normally unless suppressing duplicates
				if !cfg.SuppressDuplicates {
					outputGameWithAnnotations(game, cfg, gameInfo, &jsonGames)
					matchedCount++
					outputCount++
				} else if !cfg.SuppressOriginals {
					outputGameWithAnnotations(game, cfg, gameInfo, &jsonGames)
					matchedCount++
					outputCount++
				}
			}
		} else {
			// No duplicate detection
			outputGameWithAnnotations(game, cfg, gameInfo, &jsonGames)
			matchedCount++
			outputCount++
		}
	}

	// Output JSON array if JSON mode
	if cfg.JSONFormat && len(jsonGames) > 0 {
		output.OutputGamesJSON(jsonGames, cfg, cfg.OutputFile)
	}

	return outputCount, duplicateCount
}

// GameAnalysis holds analysis results from replaying a game
type GameAnalysis struct {
	FinalBoard        *chess.Board
	HasFiftyMoveRule  bool
	HasRepetition     bool
	HasUnderpromotion bool
	Positions         []uint64 // Zobrist hashes for repetition detection
}

// analyzeGame replays a game and analyzes it for various features
func analyzeGame(game *chess.Game) (*chess.Board, *GameAnalysis) {
	var board *chess.Board
	var err error

	// Check if game has a custom starting position (FEN tag)
	if fen, ok := game.Tags["FEN"]; ok {
		board, err = engine.NewBoardFromFEN(fen)
		if err != nil {
			board, _ = engine.NewBoardFromFEN(engine.InitialFEN)
		}
	} else {
		board, _ = engine.NewBoardFromFEN(engine.InitialFEN)
	}

	analysis := &GameAnalysis{
		Positions: make([]uint64, 0),
	}

	// Track initial position
	posHash := hashing.GenerateZobristHash(board)
	analysis.Positions = append(analysis.Positions, posHash)
	positionCount := make(map[uint64]int)
	positionCount[posHash] = 1

	// Apply all moves
	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}

		// Check for fifty-move rule
		if board.HalfmoveClock >= 100 {
			analysis.HasFiftyMoveRule = true
		}

		// Check for underpromotion
		if move.PromotedPiece != chess.Empty && move.PromotedPiece != chess.Queen {
			analysis.HasUnderpromotion = true
		}

		// Track position for repetition
		posHash = hashing.GenerateZobristHash(board)
		analysis.Positions = append(analysis.Positions, posHash)
		positionCount[posHash]++
		if positionCount[posHash] >= 3 {
			analysis.HasRepetition = true
		}
	}

	analysis.FinalBoard = board
	return board, analysis
}

// outputGameWithAnnotations outputs a game with optional annotations
func outputGameWithAnnotations(game *chess.Game, cfg *config.Config, gameInfo *GameAnalysis, jsonGames *[]*chess.Game) {
	// Handle split writer
	if sw, ok := cfg.OutputFile.(*SplitWriter); ok {
		defer sw.IncrementGameCount()
	}

	if cfg.JSONFormat {
		*jsonGames = append(*jsonGames, game)
	} else {
		// TODO: Add FEN comments and hash comments if requested
		output.OutputGame(game, cfg)
	}
}

// countPlies counts the number of half-moves in a game
func countPlies(game *chess.Game) int {
	count := 0
	for move := game.Moves; move != nil; move = move.Next {
		count++
	}
	return count
}

// hasComments checks if a game has any comments
func hasComments(game *chess.Game) bool {
	for move := game.Moves; move != nil; move = move.Next {
		if move.HasComments() {
			return true
		}
	}
	return false
}

// parseElo parses an Elo rating string to int
func parseElo(s string) int {
	if s == "" || s == "-" || s == "?" {
		return 0
	}
	elo, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return elo
}

// matchesCQL checks if any position in the game matches the CQL query.
func matchesCQL(game *chess.Game, cqlNode cql.Node) bool {
	var board *chess.Board
	var err error

	// Check if game has a custom starting position (FEN tag)
	if fen, ok := game.Tags["FEN"]; ok {
		board, err = engine.NewBoardFromFEN(fen)
		if err != nil {
			board, _ = engine.NewBoardFromFEN(engine.InitialFEN)
		}
	} else {
		board, _ = engine.NewBoardFromFEN(engine.InitialFEN)
	}

	// Check starting position
	eval := cql.NewEvaluator(board)
	if eval.Evaluate(cqlNode) {
		return true
	}

	// Check each position after a move
	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}
		eval = cql.NewEvaluator(board)
		if eval.Evaluate(cqlNode) {
			return true
		}
	}

	return false
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

// ValidationResult holds the result of game validation
type ValidationResult struct {
	Valid       bool
	ErrorPly    int
	ErrorMsg    string
	ParseErrors []string
}

// validateGame validates all moves in a game are legal
func validateGame(game *chess.Game) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Check for missing required tags
	requiredTags := []string{"Event", "Site", "Date", "Round", "White", "Black", "Result"}
	for _, tag := range requiredTags {
		if game.GetTag(tag) == "" {
			result.ParseErrors = append(result.ParseErrors, fmt.Sprintf("missing required tag: %s", tag))
		}
	}

	// Check for valid result
	resultTag := game.GetTag("Result")
	if resultTag != "" && resultTag != "1-0" && resultTag != "0-1" && resultTag != "1/2-1/2" && resultTag != "*" {
		result.ParseErrors = append(result.ParseErrors, fmt.Sprintf("invalid result: %s", resultTag))
	}

	// If we have no moves, game is valid (just tags)
	if game.Moves == nil {
		return result
	}

	// Replay game to validate moves
	var board *chess.Board
	var err error

	if fen, ok := game.Tags["FEN"]; ok {
		board, err = engine.NewBoardFromFEN(fen)
		if err != nil {
			result.Valid = false
			result.ErrorMsg = fmt.Sprintf("invalid FEN: %s", fen)
			return result
		}
	} else {
		board, _ = engine.NewBoardFromFEN(engine.InitialFEN)
	}

	plyCount := 0
	for move := game.Moves; move != nil; move = move.Next {
		plyCount++
		if !engine.ApplyMove(board, move) {
			result.Valid = false
			result.ErrorPly = plyCount
			result.ErrorMsg = fmt.Sprintf("illegal move at ply %d: %s", plyCount, move.Text)
			return result
		}
	}

	// Mark game as validated
	game.MovesChecked = true
	game.MovesOK = true

	return result
}

// fixGame attempts to fix common issues in a game
func fixGame(game *chess.Game) bool {
	fixed := false

	// Fix missing required tags with placeholder values
	if game.GetTag("Event") == "" {
		game.SetTag("Event", "?")
		fixed = true
	}
	if game.GetTag("Site") == "" {
		game.SetTag("Site", "?")
		fixed = true
	}
	if game.GetTag("Date") == "" {
		game.SetTag("Date", "????.??.??")
		fixed = true
	}
	if game.GetTag("Round") == "" {
		game.SetTag("Round", "?")
		fixed = true
	}
	if game.GetTag("White") == "" {
		game.SetTag("White", "?")
		fixed = true
	}
	if game.GetTag("Black") == "" {
		game.SetTag("Black", "?")
		fixed = true
	}
	if game.GetTag("Result") == "" {
		game.SetTag("Result", "*")
		fixed = true
	}

	// Fix invalid result tag
	resultTag := game.GetTag("Result")
	validResults := map[string]bool{"1-0": true, "0-1": true, "1/2-1/2": true, "*": true}
	if !validResults[resultTag] {
		// Try to normalize common variations
		switch strings.ToLower(strings.TrimSpace(resultTag)) {
		case "1-0", "white", "white wins":
			game.SetTag("Result", "1-0")
			fixed = true
		case "0-1", "black", "black wins":
			game.SetTag("Result", "0-1")
			fixed = true
		case "1/2", "draw", "1/2-1/2", "0.5-0.5":
			game.SetTag("Result", "1/2-1/2")
			fixed = true
		default:
			game.SetTag("Result", "*")
			fixed = true
		}
	}

	// Fix common date format issues
	date := game.GetTag("Date")
	if date != "" && date != "????.??.??" {
		// Replace common separators with dots
		normalizedDate := strings.ReplaceAll(date, "/", ".")
		normalizedDate = strings.ReplaceAll(normalizedDate, "-", ".")
		if normalizedDate != date {
			game.SetTag("Date", normalizedDate)
			fixed = true
		}
	}

	// Trim whitespace from all tags
	for tag, value := range game.Tags {
		trimmed := strings.TrimSpace(value)
		if trimmed != value {
			game.Tags[tag] = trimmed
			fixed = true
		}
	}

	// Fix encoding issues - remove control characters
	for tag, value := range game.Tags {
		cleaned := cleanString(value)
		if cleaned != value {
			game.Tags[tag] = cleaned
			fixed = true
		}
	}

	return fixed
}

// cleanString removes control characters and fixes common encoding issues
func cleanString(s string) string {
	var result strings.Builder
	for _, r := range s {
		// Keep printable ASCII, space, and common Unicode
		if r >= 32 && r != 127 {
			result.WriteRune(r)
		}
	}
	return result.String()
}
