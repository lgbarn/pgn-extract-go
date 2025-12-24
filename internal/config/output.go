package config

// OutputConfig holds settings related to output formatting.
type OutputConfig struct {
	// Format specifies the output notation format (SAN, LALG, etc.)
	Format OutputFormat

	// MaxLineLength is the maximum line length for PGN output
	MaxLineLength uint

	// JSONFormat enables JSON output instead of PGN
	JSONFormat bool

	// KeepMoveNumbers controls whether move numbers are included
	KeepMoveNumbers bool

	// KeepResults controls whether game results are included
	KeepResults bool

	// KeepChecks controls whether check symbols (+, #) are included
	KeepChecks bool

	// KeepNAGs controls whether Numeric Annotation Glyphs are kept
	KeepNAGs bool

	// KeepComments controls whether comments are kept in output
	KeepComments bool

	// KeepVariations controls whether variations (RAV) are kept
	KeepVariations bool

	// StripClockAnnotations removes clock/time annotations from comments
	StripClockAnnotations bool

	// TagFormat specifies which tags to output (AllTags, SevenTagRoster, NoTags)
	TagFormat TagOutputForm

	// SeparateCommentLines puts each comment on its own line
	SeparateCommentLines bool

	// OutputEvaluation includes engine evaluation annotations
	OutputEvaluation bool
}

// NewOutputConfig creates an OutputConfig with default values.
func NewOutputConfig() *OutputConfig {
	return &OutputConfig{
		Format:          SAN,
		MaxLineLength:   80,
		KeepMoveNumbers: true,
		KeepResults:     true,
		KeepChecks:      true,
		KeepNAGs:        true,
		KeepComments:    true,
		KeepVariations:  true,
		TagFormat:       AllTags,
	}
}
