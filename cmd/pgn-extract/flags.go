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
	duplicateCapacity  = flag.Int("duplicate-capacity", 0, "Maximum duplicate hash table entries (0 = unlimited)")

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
	exactPly  = flag.Int("exactply", 0, "Exact ply count to match")
	exactMove = flag.Int("exactmoves", 0, "Exact move count to match")
	plyRange  = flag.String("plyrange", "", "Ply range to match (e.g., '20-40')")
	moveRange = flag.String("moverange", "", "Move range to match (e.g., '10-20')")
	stopAfter = flag.Int("stopafter", 0, "Stop after matching N games")

	// Move truncation and range
	dropPly    = flag.Int("dropply", 0, "Remove first N plies from output")
	plyLimit   = flag.Int("plylimit", 0, "Limit output to first N plies")
	startPly   = flag.Int("startply", 0, "Begin output at ply N (skip earlier moves)")
	dropBefore = flag.String("dropbefore", "", "Drop moves before comment matching this string")

	// Game selection controls
	selectOnly   = flag.String("selectonly", "", "Output only games at these positions (comma-separated, 1-indexed)")
	skipMatching = flag.String("skipmatching", "", "Skip games at these positions (comma-separated, 1-indexed)")

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

	// Extended draw rules
	seventyFiveMoveFilter = flag.Bool("75", false, "Games with 75-move rule (automatic draw)")
	fiveFoldRepFilter     = flag.Bool("repetition5", false, "Games with 5-fold repetition")
	insufficientFilter    = flag.Bool("insufficient", false, "Games ending with insufficient mating material")

	// Material odds detection
	materialOddsFilter = flag.Bool("odds", false, "Games played at material odds (unequal starting material)")

	// Setup tag filtering
	noSetupTags   = flag.Bool("nosetuptags", false, "Exclude games with SetUp tag")
	onlySetupTags = flag.Bool("onlysetuptags", false, "Only match games with SetUp tag")

	// Same setup deletion
	deleteSameSetup = flag.Bool("deletesamesetup", false, "Remove games with identical starting positions")

	// CQL filter
	cqlQuery = flag.String("cql", "", "CQL query to filter games by position patterns")
	cqlFile  = flag.String("cql-file", "", "File containing CQL query")

	// Variation matching
	variationFile = flag.String("v", "", "File with move sequences to match")
	positionFile  = flag.String("x", "", "File with positional variations to match")

	// Material matching
	materialMatch      = flag.String("z", "", "Material balance to match (e.g., 'QR:qrr')")
	materialMatchExact = flag.String("y", "", "Exact material balance to match")
	pieceCount         = flag.Int("piececount", 0, "Match games reaching exactly N pieces on board")

	// Variation matching options
	varAnywhere = flag.Bool("vanywhere", false, "Match variation patterns throughout entire game")

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

	// Other options
	quiet   = flag.Bool("s", false, "Silent mode (no game count)")
	help    = flag.Bool("h", false, "Show help")
	version = flag.Bool("version", false, "Show version")

	// Performance options
	workers = flag.Int("workers", 0, "Number of worker threads (0 = auto-detect based on CPU cores)")

	// File input options
	fileListFile = flag.String("f", "", "File containing list of PGN files to process (one per line)")
	// Note: -A flag is handled manually before flag.Parse() in loadArgsFromFileIfSpecified
	_ = flag.String("A", "", "File containing command-line arguments (one per line, # for comments)")

	// ECO-based output splitting
	ecoSplit      = flag.Int("E", 0, "Split output by ECO code: 1=A-E, 2=A0-E9, 3=A00-E99")
	ecoMaxHandles = flag.Int("eco-max-handles", 128, "Maximum open file handles for ECO splitting")

	// Split output filename pattern
	splitPattern = flag.String("splitpattern", "%s_%d.pgn", "Filename pattern for split output (use %s for base, %d for number)")

	// Chess960 support
	chess960Mode = flag.Bool("chess960", false, "Enable Chess960 mode (use Shredder-FEN for castling)")

	// Nested comments
	nestedComments = flag.Bool("nestedcomments", false, "Allow nested comments in PGN parsing")

	// Fuzzy duplicate detection
	fuzzyDepth = flag.Int("fuzzydepth", 0, "Match duplicates at this ply depth (positional)")

	// Variation splitting
	splitVariants = flag.Bool("splitvariants", false, "Output each variation as a separate game")
)

// applyFlags applies command-line flags to the configuration.
func applyFlags(cfg *config.Config) {
	applyTagOutputFlags(cfg)
	applyContentFlags(cfg)
	applyOutputFormatFlags(cfg)
	applyMoveBoundsFlags(cfg)
	applyAnnotationFlags(cfg)
	applyFilterFlags(cfg)
	applyPhase4Flags(cfg)
	applyDuplicateFlags(cfg)

	if *quiet {
		cfg.Verbosity = 0
	}
	cfg.CheckOnly = *reportOnly
}

// applyPhase4Flags applies Phase 4 feature flags.
func applyPhase4Flags(cfg *config.Config) {
	cfg.AllowNestedComments = *nestedComments
	cfg.SplitVariants = *splitVariants
	cfg.Chess960Mode = *chess960Mode
	cfg.FuzzyDepth = *fuzzyDepth
}

// applyTagOutputFlags configures tag output settings.
func applyTagOutputFlags(cfg *config.Config) {
	switch {
	case *noTags:
		cfg.Output.TagFormat = config.NoTags
	case *sevenTagOnly:
		cfg.Output.TagFormat = config.SevenTagRoster
	}
}

// applyContentFlags configures content output settings.
func applyContentFlags(cfg *config.Config) {
	cfg.Output.KeepComments = !*noComments
	cfg.Output.KeepNAGs = !*noNAGs
	cfg.Output.KeepVariations = !*noVariations
	cfg.Output.KeepResults = !*noResults
	cfg.Output.StripClockAnnotations = *noClocks
	cfg.Output.JSONFormat = *jsonOutput
	cfg.Output.MaxLineLength = uint(*lineLength)
	cfg.Output.ECOMaxHandles = *ecoMaxHandles
}

// applyOutputFormatFlags configures the output format.
func applyOutputFormatFlags(cfg *config.Config) {
	formatMap := map[string]config.OutputFormat{
		"lalg":  config.LALG,
		"halg":  config.HALG,
		"elalg": config.ELALG,
		"uci":   config.UCI,
		"epd":   config.EPD,
		"fen":   config.FEN,
	}

	if format, ok := formatMap[*outputFormat]; ok {
		cfg.Output.Format = format
	} else {
		cfg.Output.Format = config.SAN
	}
}

// applyMoveBoundsFlags configures ply and move bounds.
func applyMoveBoundsFlags(cfg *config.Config) {
	hasMoveBounds := *minPly > 0 || *maxPly > 0 || *minMoves > 0 || *maxMoves > 0
	if !hasMoveBounds {
		return
	}

	cfg.Filter.CheckMoveBounds = true
	if *minMoves > 0 {
		cfg.Filter.LowerMoveBound = uint(*minMoves)
	}
	if *maxMoves > 0 {
		cfg.Filter.UpperMoveBound = uint(*maxMoves)
	}
}

// applyAnnotationFlags configures annotation and tag fixing settings.
func applyAnnotationFlags(cfg *config.Config) {
	cfg.Annotation.AddPlyCount = *addPlyCount
	cfg.Annotation.AddFENComments = *addFENComments
	cfg.Annotation.AddHashComments = *addHashComments
	cfg.Annotation.AddHashTag = *addHashcodeTag
	cfg.Annotation.FixResultTags = *fixResultTags
	cfg.Annotation.FixTagStrings = *fixTagStrings
}

// applyFilterFlags configures game filter settings.
func applyFilterFlags(cfg *config.Config) {
	cfg.Filter.MatchCheckmate = *checkmateFilter
	cfg.Filter.MatchStalemate = *stalemateFilter
	cfg.Filter.CheckFiftyMoveRule = *fiftyMoveFilter
	cfg.Filter.CheckRepetition = *repetitionFilter
	cfg.Filter.MatchUnderpromotion = *underpromotionFilter
	cfg.Filter.UseSoundex = *useSoundex
}

// applyDuplicateFlags configures duplicate detection settings.
func applyDuplicateFlags(cfg *config.Config) {
	cfg.Duplicate.MaxCapacity = *duplicateCapacity
}
