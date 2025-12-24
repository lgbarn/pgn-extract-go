// flags.go - Command-line flag definitions and configuration
package main

import (
	"flag"

	"github.com/lgbarn/pgn-extract-go/internal/config"
)

var (
	// Output options
	outputFile   = flag.String("o", "", "Output file (default: stdout)")
	appendOutput = flag.Bool("a", false, "Append to output file instead of overwrite")
	sevenTagOnly = flag.Bool("7", false, "Output only the seven tag roster")
	noTags       = flag.Bool("notags", false, "Don't output any tags")
	lineLength   = flag.Int("w", 80, "Maximum line length")
	outputFormat = flag.String("W", "", "Output format: san, lalg, halg, elalg, uci, epd, fen")
	jsonOutput   = flag.Bool("J", false, "Output in JSON format")
	splitGames   = flag.Int("#", 0, "Split output into files of N games each")
	splitByECO   = flag.String("E", "", "Split output by ECO level (1-3)")

	// Content options
	noComments   = flag.Bool("C", false, "Don't output comments")
	noNAGs       = flag.Bool("N", false, "Don't output NAGs")
	noVariations = flag.Bool("V", false, "Don't output variations")
	noResults    = flag.Bool("noresults", false, "Don't output results")
	noClocks     = flag.Bool("noclocks", false, "Strip clock annotations from comments")

	// Duplicate detection
	suppressDuplicates = flag.Bool("D", false, "Suppress duplicate games")
	duplicateFile      = flag.String("d", "", "Output duplicates to this file")
	outputDupsOnly     = flag.Bool("U", false, "Output only duplicates (suppress unique games)")
	checkFile          = flag.String("c", "", "Check file for duplicate detection")

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
	negateMatch  = flag.Bool("n", false, "Output games that DON'T match criteria")
	useSoundex   = flag.Bool("S", false, "Use Soundex for player name matching")
	tagSubstring = flag.Bool("tagsubstr", false, "Match tag values anywhere (substring)")

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
	fiftyMoveFilter      = flag.Bool("fifty", false, "Games with 50-move rule")
	repetitionFilter     = flag.Bool("repetition", false, "Games with 3-fold repetition")
	underpromotionFilter = flag.Bool("underpromotion", false, "Games with underpromotion")
	commentedFilter      = flag.Bool("commented", false, "Only games with comments")
	higherRatedWinner    = flag.Bool("higherratedwinner", false, "Higher-rated player won")
	lowerRatedWinner     = flag.Bool("lowerratedwinner", false, "Lower-rated player won")

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
	addPlyCount     = flag.Bool("plycount", false, "Add PlyCount tag")
	addFENComments  = flag.Bool("fencomments", false, "Add FEN comment after each move")
	addHashComments = flag.Bool("hashcomments", false, "Add position hash after each move")
	addHashcodeTag  = flag.Bool("addhashcode", false, "Add HashCode tag")

	// Tag management
	fixResultTags = flag.Bool("fixresulttags", false, "Fix inconsistent result tags")
	fixTagStrings = flag.Bool("fixtagstrings", false, "Fix malformed tag strings")

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
	quiet   = flag.Bool("s", false, "Silent mode (no game count)")
	help    = flag.Bool("h", false, "Show help")
	version = flag.Bool("version", false, "Show version")

	// Performance options
	workers = flag.Int("workers", 0, "Number of worker threads (0 = auto-detect based on CPU cores)")
)

// applyFlags applies command-line flags to the configuration.
func applyFlags(cfg *config.Config) {
	// Tag output
	if *sevenTagOnly {
		cfg.Output.TagFormat = config.SevenTagRoster
	}
	if *noTags {
		cfg.Output.TagFormat = config.NoTags
	}

	// Content
	cfg.Output.KeepComments = !*noComments
	cfg.Output.KeepNAGs = !*noNAGs
	cfg.Output.KeepVariations = !*noVariations
	cfg.Output.KeepResults = !*noResults
	cfg.Output.StripClockAnnotations = *noClocks

	// Line length
	cfg.Output.MaxLineLength = uint(*lineLength)

	// Output format
	switch *outputFormat {
	case "lalg":
		cfg.Output.Format = config.LALG
	case "halg":
		cfg.Output.Format = config.HALG
	case "elalg":
		cfg.Output.Format = config.ELALG
	case "uci":
		cfg.Output.Format = config.UCI
	case "epd":
		cfg.Output.Format = config.EPD
	case "fen":
		cfg.Output.Format = config.FEN
	default:
		cfg.Output.Format = config.SAN
	}

	// Verbosity
	if *quiet {
		cfg.Verbosity = 0
	}

	// JSON output
	cfg.Output.JSONFormat = *jsonOutput

	// Ply/move bounds
	if *minPly > 0 || *maxPly > 0 || *minMoves > 0 || *maxMoves > 0 {
		cfg.Filter.CheckMoveBounds = true
		if *minMoves > 0 {
			cfg.Filter.LowerMoveBound = uint(*minMoves)
		}
		if *maxMoves > 0 {
			cfg.Filter.UpperMoveBound = uint(*maxMoves)
		}
	}

	// Annotations
	cfg.Annotation.AddPlyCount = *addPlyCount
	cfg.Annotation.AddFENComments = *addFENComments
	cfg.Annotation.AddHashComments = *addHashComments
	cfg.Annotation.AddHashTag = *addHashcodeTag

	// Tag fixing
	cfg.Annotation.FixResultTags = *fixResultTags
	cfg.Annotation.FixTagStrings = *fixTagStrings

	// Game feature matching
	cfg.Filter.MatchCheckmate = *checkmateFilter
	cfg.Filter.MatchStalemate = *stalemateFilter
	cfg.Filter.CheckFiftyMoveRule = *fiftyMoveFilter
	cfg.Filter.CheckRepetition = *repetitionFilter
	cfg.Filter.MatchUnderpromotion = *underpromotionFilter

	// Soundex
	cfg.Filter.UseSoundex = *useSoundex

	// Report only mode
	cfg.CheckOnly = *reportOnly
}
