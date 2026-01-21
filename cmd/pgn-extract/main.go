// pgn-extract is a tool for searching, manipulating, and formatting chess games in PGN format.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/cql"
	"github.com/lgbarn/pgn-extract-go/internal/eco"
	"github.com/lgbarn/pgn-extract-go/internal/hashing"
	"github.com/lgbarn/pgn-extract-go/internal/matching"
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
	applyFlags(cfg)

	// Set up logging and output files
	setupLogFile(cfg)
	setupOutputFile(cfg)
	setupDuplicateFile(cfg)

	// Set up non-matching file for -n flag
	if *negateMatch && *outputFile != "" {
		cfg.NonMatchingFile = cfg.OutputFile
		cfg.OutputFile = nil
	}

	// Create duplicate detector and load check file if needed
	detector := setupDuplicateDetector(cfg)

	// Load ECO classifier if specified
	ecoClassifier := loadECOClassifier(cfg)

	// Set up game filter with all criteria
	gameFilter := setupGameFilter()

	// Load variation matcher if specified
	variationMatcher := loadVariationMatcher()

	// Parse material match criteria
	materialMatcher := loadMaterialMatcher()

	// Parse CQL query
	cqlNode := parseCQLQuery()

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

	// Process input files or stdin
	totalGames, outputGames, duplicates := processAllInputs(ctx, splitWriter)

	// Report statistics
	if cfg.Verbosity > 0 && !*quiet && !*reportOnly {
		reportStatistics(detector, outputGames, duplicates, totalGames)
	}
}

// setupLogFile configures the log file based on command-line flags.
func setupLogFile(cfg *config.Config) {
	if *logFile != "" {
		file, err := os.Create(*logFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating log file %s: %v\n", *logFile, err)
			os.Exit(1)
		}
		cfg.LogFile = file
	}

	if *appendLog != "" {
		file, err := os.OpenFile(*appendLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //nolint:gosec // G302: 0644 is appropriate for user-created log files
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening log file %s: %v\n", *appendLog, err)
			os.Exit(1)
		}
		cfg.LogFile = file
	}
}

// setupOutputFile configures the output file based on command-line flags.
func setupOutputFile(cfg *config.Config) {
	if *outputFile == "" {
		return
	}

	var file *os.File
	var err error

	if *appendOutput {
		file, err = os.OpenFile(*outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //nolint:gosec // G302: 0644 is appropriate for user-created output files
	} else {
		file, err = os.Create(*outputFile)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file %s: %v\n", *outputFile, err)
		os.Exit(1)
	}
	cfg.OutputFile = file
}

// setupDuplicateFile configures the duplicate output file.
func setupDuplicateFile(cfg *config.Config) {
	if *duplicateFile == "" {
		return
	}

	file, err := os.Create(*duplicateFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating duplicate file %s: %v\n", *duplicateFile, err)
		os.Exit(1)
	}
	cfg.Duplicate.DuplicateFile = file
}

// setupDuplicateDetector creates and configures the duplicate detector.
func setupDuplicateDetector(cfg *config.Config) *hashing.DuplicateDetector {
	if !*suppressDuplicates && *duplicateFile == "" && !*outputDupsOnly && *checkFile == "" {
		return nil
	}

	detector := hashing.NewDuplicateDetector(false)
	cfg.Duplicate.Suppress = *suppressDuplicates
	cfg.Duplicate.SuppressOriginals = *outputDupsOnly

	// Load check file for duplicate detection
	if *checkFile != "" {
		file, err := os.Open(*checkFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening check file %s: %v\n", *checkFile, err)
			os.Exit(1)
		}
		defer file.Close()

		checkGames := processInput(file, *checkFile, cfg)
		for _, game := range checkGames {
			board := replayGame(game)
			detector.CheckAndAdd(game, board)
		}

		if cfg.Verbosity > 0 {
			fmt.Fprintf(cfg.LogFile, "Loaded %d games from check file\n", len(checkGames))
		}
	}

	return detector
}

// loadECOClassifier loads the ECO classification file if specified.
func loadECOClassifier(cfg *config.Config) *eco.ECOClassifier {
	if *ecoFile == "" {
		return nil
	}

	classifier := eco.NewECOClassifier()
	if err := classifier.LoadFromFile(*ecoFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ECO file %s: %v\n", *ecoFile, err)
		os.Exit(1)
	}

	if cfg.Verbosity > 0 {
		fmt.Fprintf(cfg.LogFile, "Loaded %d ECO entries\n", classifier.EntriesLoaded())
	}
	cfg.AddECO = true

	return classifier
}

// setupGameFilter creates and configures the game filter with all criteria.
func setupGameFilter() *matching.GameFilter {
	filter := matching.NewGameFilter()
	filter.SetUseSoundex(*useSoundex)
	filter.SetSubstringMatch(*tagSubstring)

	// Load tag criteria file if specified
	if *tagFile != "" {
		if err := filter.LoadTagFile(*tagFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading tag file %s: %v\n", *tagFile, err)
			os.Exit(1)
		}
	}

	// Add individual filter criteria
	if *playerFilter != "" {
		filter.AddPlayerFilter(*playerFilter)
	}
	if *whiteFilter != "" {
		filter.AddWhiteFilter(*whiteFilter)
	}
	if *blackFilter != "" {
		filter.AddBlackFilter(*blackFilter)
	}
	if *ecoFilter != "" {
		filter.AddECOFilter(*ecoFilter)
	}
	if *resultFilter != "" {
		filter.AddResultFilter(*resultFilter)
	}
	if *fenFilter != "" {
		if err := filter.AddFENFilter(*fenFilter); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing FEN filter: %v\n", err)
			os.Exit(1)
		}
	}

	return filter
}

// loadVariationMatcher loads variation and position files if specified.
func loadVariationMatcher() *matching.VariationMatcher {
	if *variationFile == "" && *positionFile == "" {
		return nil
	}

	matcher := matching.NewVariationMatcher()

	if *variationFile != "" {
		if err := matcher.LoadFromFile(*variationFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading variation file %s: %v\n", *variationFile, err)
			os.Exit(1)
		}
	}

	if *positionFile != "" {
		if err := matcher.LoadPositionalFromFile(*positionFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading position file %s: %v\n", *positionFile, err)
			os.Exit(1)
		}
	}

	return matcher
}

// loadMaterialMatcher creates a material matcher if specified.
func loadMaterialMatcher() *matching.MaterialMatcher {
	if *materialMatchExact != "" {
		return matching.NewMaterialMatcher(*materialMatchExact, true)
	}
	if *materialMatch != "" {
		return matching.NewMaterialMatcher(*materialMatch, false)
	}
	return nil
}

// parseCQLQuery parses the CQL query from file or command line.
func parseCQLQuery() cql.Node {
	queryStr := *cqlQuery

	if *cqlFile != "" {
		content, err := os.ReadFile(*cqlFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading CQL file %s: %v\n", *cqlFile, err)
			os.Exit(1)
		}
		queryStr = strings.TrimSpace(string(content))
	}

	if queryStr == "" {
		return nil
	}

	node, err := cql.Parse(queryStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing CQL query: %v\n", err)
		os.Exit(1)
	}

	return node
}

// processAllInputs processes all input files or stdin.
func processAllInputs(ctx *ProcessingContext, splitWriter *SplitWriter) (totalGames, outputGames, duplicates int) {
	args := flag.Args()

	if len(args) == 0 {
		games := processInput(os.Stdin, "stdin", ctx.cfg)
		totalGames = len(games)
		outputGames, duplicates = outputGamesWithProcessing(games, ctx)
	} else {
		for _, filename := range args {
			if *stopAfter > 0 && atomic.LoadInt64(&matchedCount) >= int64(*stopAfter) {
				break
			}

			file, err := os.Open(filename) //nolint:gosec // G304: CLI tool opens user-specified files
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening file %s: %v\n", filename, err)
				continue
			}

			games := processInput(file, filename, ctx.cfg)
			totalGames += len(games)
			out, dup := outputGamesWithProcessing(games, ctx)
			outputGames += out
			duplicates += dup

			file.Close() //nolint:errcheck,gosec // G104: cleanup on exit
		}
	}

	if splitWriter != nil {
		splitWriter.Close() //nolint:errcheck,gosec // G104: cleanup on exit
	}

	return totalGames, outputGames, duplicates
}

// reportStatistics prints the final statistics to stderr.
func reportStatistics(detector *hashing.DuplicateDetector, outputGames, duplicates, totalGames int) {
	if detector != nil {
		fmt.Fprintf(os.Stderr, "%d game(s) output, %d duplicate(s) out of %d.\n", outputGames, duplicates, totalGames)
	} else {
		fmt.Fprintf(os.Stderr, "%d game(s) matched out of %d.\n", outputGames, totalGames)
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
