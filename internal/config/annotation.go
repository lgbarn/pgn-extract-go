package config

// AnnotationConfig holds settings for adding annotations to games.
type AnnotationConfig struct {
	// FEN annotations
	OutputFEN      bool   // Output FEN string instead of moves
	AddFENComments bool   // Add FEN as comments
	AddFENCastling bool   // Include castling rights in FEN
	FENPattern     string // Pattern for FEN comments

	// Hash annotations
	AddHashComments bool // Add position hash as comments
	AddHashTag      bool // Add hashcode tag to game

	// Ply count annotations
	AddPlyCount      bool // Add ply count to moves
	AddTotalPlyCount bool // Add total ply count tag

	// Match annotations
	AddMatchTag      bool   // Add tag indicating match
	AddMatchLabelTag bool   // Add label to match tag
	MatchCommentText string // Text for position match comments
	AddMatchComments bool   // Add comments at match positions

	// Fix options
	FixResultTags bool // Fix inconsistent result tags
	FixTagStrings bool // Fix malformed tag strings
}

// NewAnnotationConfig creates an AnnotationConfig with default values.
func NewAnnotationConfig() *AnnotationConfig {
	return &AnnotationConfig{
		OutputFEN:        false,
		AddFENComments:   false,
		AddHashComments:  false,
		AddPlyCount:      false,
		AddTotalPlyCount: false,
		AddHashTag:       false,
		AddMatchTag:      false,
		AddMatchLabelTag: false,
		FixResultTags:    false,
		FixTagStrings:    false,
	}
}
