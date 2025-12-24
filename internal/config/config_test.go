package config

import (
	"bytes"
	"testing"
)

// TestOutputConfig_Defaults verifies OutputConfig has sensible defaults
func TestOutputConfig_Defaults(t *testing.T) {
	cfg := NewOutputConfig()

	if cfg.Format != SAN {
		t.Errorf("Format = %v, want %v", cfg.Format, SAN)
	}
	if cfg.MaxLineLength != 80 {
		t.Errorf("MaxLineLength = %d, want 80", cfg.MaxLineLength)
	}
	if !cfg.KeepMoveNumbers {
		t.Error("KeepMoveNumbers should be true by default")
	}
	if !cfg.KeepResults {
		t.Error("KeepResults should be true by default")
	}
	if !cfg.KeepChecks {
		t.Error("KeepChecks should be true by default")
	}
	if !cfg.KeepNAGs {
		t.Error("KeepNAGs should be true by default")
	}
	if !cfg.KeepComments {
		t.Error("KeepComments should be true by default")
	}
	if !cfg.KeepVariations {
		t.Error("KeepVariations should be true by default")
	}
	if cfg.TagFormat != AllTags {
		t.Errorf("TagFormat = %v, want AllTags", cfg.TagFormat)
	}
}

// TestFilterConfig_Defaults verifies FilterConfig has sensible defaults
func TestFilterConfig_Defaults(t *testing.T) {
	cfg := NewFilterConfig()

	// Most filter options should be disabled by default
	if cfg.CheckMoveBounds {
		t.Error("CheckMoveBounds should be false by default")
	}
	if cfg.MatchCheckmate {
		t.Error("MatchCheckmate should be false by default")
	}
	if cfg.MatchStalemate {
		t.Error("MatchStalemate should be false by default")
	}
	if cfg.MatchUnderpromotion {
		t.Error("MatchUnderpromotion should be false by default")
	}
	if cfg.CheckRepetition {
		t.Error("CheckRepetition should be false by default")
	}
	if cfg.CheckFiftyMoveRule {
		t.Error("CheckFiftyMoveRule should be false by default")
	}
}

// TestFilterConfig_Validate verifies filter config validation
func TestFilterConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     FilterConfig
		wantErr bool
	}{
		{
			name:    "empty config is valid",
			cfg:     FilterConfig{},
			wantErr: false,
		},
		{
			name: "valid move bounds",
			cfg: FilterConfig{
				CheckMoveBounds: true,
				LowerMoveBound:  10,
				UpperMoveBound:  50,
			},
			wantErr: false,
		},
		{
			name: "invalid move bounds - lower > upper",
			cfg: FilterConfig{
				CheckMoveBounds: true,
				LowerMoveBound:  50,
				UpperMoveBound:  10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDuplicateConfig_Defaults verifies DuplicateConfig has sensible defaults
func TestDuplicateConfig_Defaults(t *testing.T) {
	cfg := NewDuplicateConfig()

	if cfg.Suppress {
		t.Error("Suppress should be false by default")
	}
	if cfg.SuppressOriginals {
		t.Error("SuppressOriginals should be false by default")
	}
	if cfg.FuzzyMatch {
		t.Error("FuzzyMatch should be false by default")
	}
	if cfg.FuzzyDepth != 0 {
		t.Errorf("FuzzyDepth = %d, want 0", cfg.FuzzyDepth)
	}
}

// TestAnnotationConfig_Defaults verifies AnnotationConfig has sensible defaults
func TestAnnotationConfig_Defaults(t *testing.T) {
	cfg := NewAnnotationConfig()

	// Annotation options should be disabled by default
	if cfg.AddFENComments {
		t.Error("AddFENComments should be false by default")
	}
	if cfg.AddHashComments {
		t.Error("AddHashComments should be false by default")
	}
	if cfg.AddPlyCount {
		t.Error("AddPlyCount should be false by default")
	}
	if cfg.AddHashTag {
		t.Error("AddHashTag should be false by default")
	}
	if cfg.AddMatchTag {
		t.Error("AddMatchTag should be false by default")
	}
}

// TestConfig_EmbeddedStructs verifies that Config properly embeds sub-configs
func TestConfig_EmbeddedStructs(t *testing.T) {
	cfg := NewConfig()

	// Should be able to access embedded struct fields directly
	if cfg.Output.Format != SAN {
		t.Errorf("Output.Format = %v, want %v", cfg.Output.Format, SAN)
	}
	if cfg.Filter.CheckMoveBounds {
		t.Error("Filter.CheckMoveBounds should be false")
	}
	if cfg.Duplicate.Suppress {
		t.Error("Duplicate.Suppress should be false")
	}
	if cfg.Annotation.AddFENComments {
		t.Error("Annotation.AddFENComments should be false")
	}
}

// TestConfig_SetOutput verifies output stream setting
func TestConfig_SetOutput(t *testing.T) {
	cfg := NewConfig()
	buf := &bytes.Buffer{}

	cfg.SetOutput(buf)

	if cfg.OutputFile != buf {
		t.Error("SetOutput did not set OutputFile")
	}
}

// TestConfigBuilder verifies the builder pattern works correctly
func TestConfigBuilder(t *testing.T) {
	cfg := NewConfigBuilder().
		WithOutputFormat(LALG).
		WithMaxLineLength(120).
		WithDuplicateSuppression(true).
		WithFuzzyMatch(true, 10).
		Build()

	if cfg.Output.Format != LALG {
		t.Errorf("Format = %v, want LALG", cfg.Output.Format)
	}
	if cfg.Output.MaxLineLength != 120 {
		t.Errorf("MaxLineLength = %d, want 120", cfg.Output.MaxLineLength)
	}
	if !cfg.Duplicate.Suppress {
		t.Error("Duplicate.Suppress should be true")
	}
	if !cfg.Duplicate.FuzzyMatch {
		t.Error("Duplicate.FuzzyMatch should be true")
	}
	if cfg.Duplicate.FuzzyDepth != 10 {
		t.Errorf("FuzzyDepth = %d, want 10", cfg.Duplicate.FuzzyDepth)
	}
}
