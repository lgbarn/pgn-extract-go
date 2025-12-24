package config

import "io"

// ConfigBuilder provides a fluent API for building Config instances.
type ConfigBuilder struct {
	cfg *Config
}

// NewConfigBuilder creates a new ConfigBuilder with default values.
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		cfg: NewConfig(),
	}
}

// Build returns the built Config.
func (b *ConfigBuilder) Build() *Config {
	return b.cfg
}

// WithOutputFormat sets the output format.
func (b *ConfigBuilder) WithOutputFormat(format OutputFormat) *ConfigBuilder {
	b.cfg.Output.Format = format
	return b
}

// WithMaxLineLength sets the maximum line length.
func (b *ConfigBuilder) WithMaxLineLength(length uint) *ConfigBuilder {
	b.cfg.Output.MaxLineLength = length
	return b
}

// WithJSONOutput enables JSON output.
func (b *ConfigBuilder) WithJSONOutput(enabled bool) *ConfigBuilder {
	b.cfg.Output.JSONFormat = enabled
	return b
}

// WithDuplicateSuppression enables duplicate suppression.
func (b *ConfigBuilder) WithDuplicateSuppression(enabled bool) *ConfigBuilder {
	b.cfg.Duplicate.Suppress = enabled
	return b
}

// WithFuzzyMatch enables fuzzy duplicate matching.
func (b *ConfigBuilder) WithFuzzyMatch(enabled bool, depth uint) *ConfigBuilder {
	b.cfg.Duplicate.FuzzyMatch = enabled
	b.cfg.Duplicate.FuzzyDepth = depth
	return b
}

// WithMoveBounds sets move bounds for filtering.
func (b *ConfigBuilder) WithMoveBounds(lower, upper uint) *ConfigBuilder {
	b.cfg.Filter.CheckMoveBounds = true
	b.cfg.Filter.LowerMoveBound = lower
	b.cfg.Filter.UpperMoveBound = upper
	return b
}

// WithCheckmateFilter enables checkmate-only filtering.
func (b *ConfigBuilder) WithCheckmateFilter(enabled bool) *ConfigBuilder {
	b.cfg.Filter.MatchCheckmate = enabled
	return b
}

// WithFENComments enables FEN comments.
func (b *ConfigBuilder) WithFENComments(enabled bool) *ConfigBuilder {
	b.cfg.Annotation.AddFENComments = enabled
	return b
}

// WithHashTag enables hashcode tags.
func (b *ConfigBuilder) WithHashTag(enabled bool) *ConfigBuilder {
	b.cfg.Annotation.AddHashTag = enabled
	return b
}

// WithOutput sets the output writer.
func (b *ConfigBuilder) WithOutput(w io.Writer) *ConfigBuilder {
	b.cfg.OutputFile = w
	return b
}

// WithVerbosity sets the verbosity level.
func (b *ConfigBuilder) WithVerbosity(level int) *ConfigBuilder {
	b.cfg.Verbosity = level
	return b
}

// KeepComments controls whether comments are kept.
func (b *ConfigBuilder) KeepComments(keep bool) *ConfigBuilder {
	b.cfg.Output.KeepComments = keep
	return b
}

// KeepVariations controls whether variations are kept.
func (b *ConfigBuilder) KeepVariations(keep bool) *ConfigBuilder {
	b.cfg.Output.KeepVariations = keep
	return b
}

// KeepNAGs controls whether NAGs are kept.
func (b *ConfigBuilder) KeepNAGs(keep bool) *ConfigBuilder {
	b.cfg.Output.KeepNAGs = keep
	return b
}
