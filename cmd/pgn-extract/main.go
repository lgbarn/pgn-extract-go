// pgn-extract is a tool for searching, manipulating, and formatting chess games in PGN format.
package main

import (
	"bufio"
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

	// First pass: check for -A flag to load arguments file
	// We need to do a quick scan of os.Args to find -A before full parsing
	argsFromFile := loadArgsFromFileIfSpecified()
	if len(argsFromFile) > 0 {
		// Prepend file args to os.Args (after program name, before user args)
		// This allows user args to override file args
		newArgs := make([]string, 0, 1+len(argsFromFile)+len(os.Args)-1)
		newArgs = append(newArgs, os.Args[0])
		newArgs = append(newArgs, argsFromFile...)
		newArgs = append(newArgs, os.Args[1:]...)
		os.Args = newArgs
	}

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

	// Initialize selection sets for selectOnly/skipMatching flags
	initSelectionSets()

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
		splitWriter = NewSplitWriterWithPattern(base, *splitGames, *splitPattern)
		cfg.OutputFile = splitWriter
	}

	// Set up ECO-based output splitting
	var ecoSplitWriter *ECOSplitWriter
	if *ecoSplit > 0 && *ecoSplit <= 3 {
		base := "output"
		if *outputFile != "" {
			base = strings.TrimSuffix(*outputFile, filepath.Ext(*outputFile))
		}
		ecoSplitWriter = NewECOSplitWriter(base, *ecoSplit, cfg, 128)
	}

	// Set up same-setup duplicate detection
	var setupDetector *hashing.SetupDuplicateDetector
	if *deleteSameSetup {
		setupDetector = hashing.NewSetupDuplicateDetector()
	}

	// Create processing context
	ctx := &ProcessingContext{
		cfg:              cfg,
		detector:         detector,
		setupDetector:    setupDetector,
		ecoClassifier:    ecoClassifier,
		gameFilter:       gameFilter,
		cqlNode:          cqlNode,
		variationMatcher: variationMatcher,
		materialMatcher:  materialMatcher,
		ecoSplitWriter:   ecoSplitWriter,
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
func setupDuplicateDetector(cfg *config.Config) hashing.DuplicateChecker {
	if !*suppressDuplicates && *duplicateFile == "" && !*outputDupsOnly && *checkFile == "" {
		return nil
	}

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

		// Load games into a temporary non-thread-safe detector
		tempDetector := hashing.NewDuplicateDetector(false, 0)
		checkGames := processInput(file, *checkFile, cfg)
		for _, game := range checkGames {
			board := replayGame(game)
			tempDetector.CheckAndAdd(game, board)
		}

		if cfg.Verbosity > 0 {
			fmt.Fprintf(cfg.LogFile, "Loaded %d games from check file\n", len(checkGames))
		}

		// Create thread-safe detector and load from temporary detector
		detector := hashing.NewThreadSafeDuplicateDetector(false)
		detector.LoadFromDetector(tempDetector)
		return detector
	}

	// No check file - create empty thread-safe detector
	return hashing.NewThreadSafeDuplicateDetector(false)
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

	// Set matchAnywhere option if specified
	if *varAnywhere {
		matcher.SetMatchAnywhere(true)
	}

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

	// If -f flag is specified, load file list from file
	if *fileListFile != "" {
		fileList, err := loadFileList(*fileListFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading file list %s: %v\n", *fileListFile, err)
			os.Exit(1)
		}
		// Append file list to command-line args
		args = append(args, fileList...)
	}

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

	// Close ECO split writer if used
	if ctx.ecoSplitWriter != nil {
		ctx.ecoSplitWriter.Close() //nolint:errcheck,gosec // G104: cleanup on exit
	}

	return totalGames, outputGames, duplicates
}

// reportStatistics prints the final statistics to stderr.
func reportStatistics(detector hashing.DuplicateChecker, outputGames, duplicates, totalGames int) {
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

// loadArgsFile reads command-line arguments from a file.
// Lines starting with # are treated as comments and ignored.
// Empty lines are also ignored.
func loadArgsFile(filename string) ([]string, error) {
	file, err := os.Open(filename) //nolint:gosec // G304: CLI tool opens user-specified files
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var args []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Split line into individual arguments (handles quoted strings)
		lineArgs := splitArgsLine(line)
		args = append(args, lineArgs...)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return args, nil
}

// splitArgsLine splits a line into individual arguments, respecting quotes.
func splitArgsLine(line string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range line {
		switch {
		case !inQuote && (r == '"' || r == '\''):
			inQuote = true
			quoteChar = r
		case inQuote && r == quoteChar:
			inQuote = false
			quoteChar = 0
		case !inQuote && (r == ' ' || r == '\t'):
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// loadFileList reads a list of PGN file paths from a file.
// Returns the list of file paths, skipping empty lines.
func loadFileList(filename string) ([]string, error) {
	file, err := os.Open(filename) //nolint:gosec // G304: CLI tool opens user-specified files
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var files []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		files = append(files, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return files, nil
}

// loadArgsFromFileIfSpecified scans os.Args for -A flag and loads args from file if found.
// This must happen before flag.Parse() to inject file arguments.
func loadArgsFromFileIfSpecified() []string {
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		var filename string
		if arg == "-A" && i+1 < len(os.Args) {
			filename = os.Args[i+1]
		} else if strings.HasPrefix(arg, "-A=") {
			filename = strings.TrimPrefix(arg, "-A=")
		}

		if filename == "" {
			continue
		}

		args, err := loadArgsFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading arguments file %s: %v\n", filename, err)
			os.Exit(1)
		}
		return args
	}
	return nil
}
