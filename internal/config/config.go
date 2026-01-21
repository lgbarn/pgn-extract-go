// Package config provides configuration and global state for pgn-extract.
package config

import (
	"io"
	"os"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// OutputFormat represents different output notation formats.
type OutputFormat int

const (
	Source OutputFormat = iota // Original source notation
	SAN                        // Standard Algebraic Notation
	EPD                        // Extended Position Description
	FEN                        // Forsyth-Edwards Notation
	CM                         // Chess Master format
	LALG                       // Long algebraic (e2e4)
	HALG                       // Hyphenated long algebraic (e2-e4)
	ELALG                      // Enhanced long algebraic (Ng1f3)
	XLALG                      // Extended long algebraic with capture notation
	XOLALG                     // XLALG with O-O castling notation
	UCI                        // UCI format (same as LALG)
)

// EcoDivision specifies how to divide output by ECO code.
type EcoDivision int

const (
	DontDivide  EcoDivision = 0
	MinECOLevel EcoDivision = 1
	MaxECOLevel EcoDivision = 10
)

// TagOutputForm specifies which tags to output.
type TagOutputForm int

const (
	AllTags        TagOutputForm = 0
	SevenTagRoster TagOutputForm = 1
	NoTags         TagOutputForm = 2
)

// SetupOutputStatus specifies how to handle games with Setup tags.
type SetupOutputStatus int

const (
	SetupTagOK SetupOutputStatus = iota
	NoSetupTag
	SetupTagOnly
)

// SourceFileType distinguishes between different types of input files.
type SourceFileType int

const (
	NormalFile SourceFileType = iota
	CheckFile
	EcoFile
)

// GameNumber represents a range of game numbers.
type GameNumber struct {
	Min  uint
	Max  uint
	Next *GameNumber
}

// Config holds all program configuration and state.
// This replaces the C StateInfo struct.
type Config struct {
	// Embedded sub-configs for organized access
	Output     *OutputConfig
	Filter     *FilterConfig
	Duplicate  *DuplicateConfig
	Annotation *AnnotationConfig

	// Processing state
	SkippingCurrentGame bool
	CheckOnly           bool
	Verbosity           int // 0=nothing, 1=game count, 2=running commentary

	// Tag checking
	CheckTags bool

	// ECO
	AddECO         bool
	ParsingECOFile bool
	ECOLevel       EcoDivision

	// Parsing options
	AllowNullMoves      bool
	AllowNestedComments bool

	// Chess960 support
	Chess960Mode bool

	// Fuzzy duplicate detection depth
	FuzzyDepth int

	// Split options
	SplitVariants   bool
	SplitDepthLimit uint

	// Consistency checks
	RejectInconsistentResults bool
	SuppressRedundantEPInfo   bool
	OnlyOutputWantedTags      bool
	DeleteSameSetup           bool

	// Current file type
	CurrentFileType SourceFileType

	// Setup tag handling
	SetupStatus SetupOutputStatus

	// For positional matches
	WhoseMove chess.WhoseMove

	// Comment patterns
	DropCommentPattern string
	LineNumberMarker   string

	// File handling
	CurrentInputFile string
	ECOFile          string
	OutputFilename   string

	// Output streams
	OutputFile      io.Writer
	LogFile         io.Writer
	NonMatchingFile io.Writer

	// Game number selection
	MatchingGameNumbers    *GameNumber
	NextGameNumberToOutput *GameNumber
	SkipGameNumbers        *GameNumber
	NextGameNumberToSkip   *GameNumber

	// Counters (runtime state - consider moving out of config)
	NumGamesProcessed uint
	NumGamesMatched   uint
	GamesPerFile      uint
	NextFileNumber    uint
}

// GlobalConfig is the global configuration instance.
var GlobalConfig *Config

// NewConfig creates a new Config with default values.
func NewConfig() *Config {
	return &Config{
		Output:      NewOutputConfig(),
		Filter:      NewFilterConfig(),
		Duplicate:   NewDuplicateConfig(),
		Annotation:  NewAnnotationConfig(),
		Verbosity:   1,
		OutputFile:  os.Stdout,
		LogFile:     os.Stderr,
		WhoseMove:   chess.EitherToMove,
		SetupStatus: SetupTagOK,
	}
}

// SetOutput sets the output writer.
func (c *Config) SetOutput(w io.Writer) {
	c.OutputFile = w
}

// Init initializes the global configuration.
func Init() {
	GlobalConfig = NewConfig()
}

func init() {
	Init()
}
