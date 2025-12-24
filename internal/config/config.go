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
	// Processing state
	SkippingCurrentGame bool
	CheckOnly           bool
	Verbosity           int // 0=nothing, 1=game count, 2=running commentary

	// Content filtering
	KeepNAGs              bool
	KeepComments          bool
	KeepVariations        bool
	StripClockAnnotations bool
	TagOutputFormat       TagOutputForm
	MatchPermutations     bool
	PositionalVariations  bool
	UseSoundex            bool

	// Duplicate handling
	SuppressDuplicates   bool
	SuppressOriginals    bool
	FuzzyMatchDuplicates bool
	FuzzyMatchDepth      uint

	// Tag checking
	CheckTags bool

	// ECO
	AddECO         bool
	ParsingECOFile bool
	ECOLevel       EcoDivision

	// Output
	OutputFormat  OutputFormat
	MaxLineLength uint
	JSONFormat    bool

	// Virtual hash table
	UseVirtualHashTable bool

	// Move bounds
	CheckMoveBounds bool
	LowerMoveBound  uint
	UpperMoveBound  uint
	OutputPlyLimit  int

	// Match conditions
	MatchOnlyCheckmate    bool
	MatchOnlyStalemate    bool
	MatchUnderpromotion   bool
	CheckForRepetition    bool
	CheckForFiftyMoveRule bool
	TagMatchAnywhere      bool

	// Output options
	KeepMoveNumbers         bool
	KeepResults             bool
	KeepChecks              bool
	OutputEvaluation        bool
	KeepBrokenGames         bool
	SuppressRedundantEPInfo bool

	// Positional search
	DepthOfPositionalSearch uint

	// Counters
	NumGamesProcessed uint
	NumGamesMatched   uint
	GamesPerFile      uint
	NextFileNumber    uint

	// Quiescence
	QuiescenceThreshold uint
	MaximumMatches      uint

	// Ply manipulation
	DropPlyNumber int
	StartPly      uint

	// FEN options
	OutputFENString           bool
	AddFENComments            bool
	AddHashcodeComments       bool
	AddPositionMatchComments  bool
	OutputPlycount            bool
	OutputTotalPlycount       bool
	AddHashcodeTag            bool
	FixResultTags             bool
	FixTagStrings             bool
	AddFENCastling            bool
	SeparateCommentLines      bool
	SplitVariants             bool
	RejectInconsistentResults bool
	AllowNullMoves            bool
	AllowNestedComments       bool
	AddMatchTag               bool
	AddMatchLabelTag          bool
	OnlyOutputWantedTags      bool
	DeleteSameSetup           bool

	// Split depth limit (0 = no limit)
	SplitDepthLimit uint

	// Current file type
	CurrentFileType SourceFileType

	// Setup tag handling
	SetupStatus SetupOutputStatus

	// For positional matches
	WhoseMove chess.WhoseMove

	// Comment patterns
	PositionMatchComment string
	FENCommentPattern    string
	DropCommentPattern   string
	LineNumberMarker     string

	// File handling
	CurrentInputFile string
	ECOFile          string
	OutputFilename   string

	// Output streams
	OutputFile      io.Writer
	LogFile         io.Writer
	DuplicateFile   io.Writer
	NonMatchingFile io.Writer

	// Game number selection
	MatchingGameNumbers    *GameNumber
	NextGameNumberToOutput *GameNumber
	SkipGameNumbers        *GameNumber
	NextGameNumberToSkip   *GameNumber
}

// GlobalConfig is the global configuration instance.
var GlobalConfig *Config

// NewConfig creates a new Config with default values.
func NewConfig() *Config {
	return &Config{
		Verbosity:       1,
		KeepNAGs:        true,
		KeepComments:    true,
		KeepVariations:  true,
		TagOutputFormat: AllTags,
		OutputFormat:    SAN,
		MaxLineLength:   80,
		KeepMoveNumbers: true,
		KeepResults:     true,
		KeepChecks:      true,
		OutputFile:      os.Stdout,
		LogFile:         os.Stderr,
		WhoseMove:       chess.EitherToMove,
		SetupStatus:     SetupTagOK,
	}
}

// Init initializes the global configuration.
func Init() {
	GlobalConfig = NewConfig()
}

func init() {
	Init()
}
