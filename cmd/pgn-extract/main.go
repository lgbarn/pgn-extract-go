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
		cfg.Duplicate.DuplicateFile = dupFile
	}

	// Create duplicate detector if needed
	var detector *hashing.DuplicateDetector
	if *suppressDuplicates || *duplicateFile != "" || *outputDupsOnly || *checkFile != "" {
		detector = hashing.NewDuplicateDetector(false)
		cfg.Duplicate.Suppress = *suppressDuplicates
		cfg.Duplicate.SuppressOriginals = *outputDupsOnly
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
			if *stopAfter > 0 && atomic.LoadInt64(&matchedCount) >= int64(*stopAfter) {
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
