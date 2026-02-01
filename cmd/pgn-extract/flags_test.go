package main

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/config"
)

// saveRestoreFlag is a helper to save and defer-restore a bool flag pointer.
// Usage: defer saveRestoreFlag(t, noTags, false)()
func saveRestoreBool(ptr *bool, val bool) func() {
	old := *ptr
	*ptr = val
	return func() { *ptr = old }
}

func saveRestoreInt(ptr *int, val int) func() {
	old := *ptr
	*ptr = val
	return func() { *ptr = old }
}

func saveRestoreString(ptr *string, val string) func() {
	old := *ptr
	*ptr = val
	return func() { *ptr = old }
}

// ---------------------------------------------------------------------------
// applyTagOutputFlags
// ---------------------------------------------------------------------------

func TestApplyTagOutputFlags(t *testing.T) {
	t.Run("noTags sets NoTags", func(t *testing.T) {
		defer saveRestoreBool(noTags, true)()
		defer saveRestoreBool(sevenTagOnly, false)()
		cfg := config.NewConfig()
		applyTagOutputFlags(cfg)
		if cfg.Output.TagFormat != config.NoTags {
			t.Errorf("TagFormat = %d; want NoTags (%d)", cfg.Output.TagFormat, config.NoTags)
		}
	})

	t.Run("sevenTagOnly sets SevenTagRoster", func(t *testing.T) {
		defer saveRestoreBool(noTags, false)()
		defer saveRestoreBool(sevenTagOnly, true)()
		cfg := config.NewConfig()
		applyTagOutputFlags(cfg)
		if cfg.Output.TagFormat != config.SevenTagRoster {
			t.Errorf("TagFormat = %d; want SevenTagRoster (%d)", cfg.Output.TagFormat, config.SevenTagRoster)
		}
	})

	t.Run("defaults to AllTags", func(t *testing.T) {
		defer saveRestoreBool(noTags, false)()
		defer saveRestoreBool(sevenTagOnly, false)()
		cfg := config.NewConfig()
		applyTagOutputFlags(cfg)
		if cfg.Output.TagFormat != config.AllTags {
			t.Errorf("TagFormat = %d; want AllTags (%d)", cfg.Output.TagFormat, config.AllTags)
		}
	})
}

// ---------------------------------------------------------------------------
// applyContentFlags
// ---------------------------------------------------------------------------

func TestApplyContentFlags(t *testing.T) {
	tests := []struct {
		name         string
		noComm       bool
		noNAG        bool
		noVar        bool
		noRes        bool
		noClock      bool
		json         bool
		wantComments bool
		wantNAGs     bool
		wantVar      bool
		wantResults  bool
		wantStrip    bool
		wantJSON     bool
	}{
		{
			name:         "all defaults (nothing suppressed)",
			wantComments: true, wantNAGs: true, wantVar: true,
			wantResults: true,
		},
		{
			name: "noComments", noComm: true,
			wantNAGs: true, wantVar: true, wantResults: true,
		},
		{
			name: "noNAGs", noNAG: true,
			wantComments: true, wantVar: true, wantResults: true,
		},
		{
			name: "noVariations", noVar: true,
			wantComments: true, wantNAGs: true, wantResults: true,
		},
		{
			name: "noResults", noRes: true,
			wantComments: true, wantNAGs: true, wantVar: true,
		},
		{
			name: "noClocks", noClock: true,
			wantComments: true, wantNAGs: true, wantVar: true,
			wantResults: true, wantStrip: true,
		},
		{
			name: "jsonOutput", json: true,
			wantComments: true, wantNAGs: true, wantVar: true,
			wantResults: true, wantJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer saveRestoreBool(noComments, tt.noComm)()
			defer saveRestoreBool(noNAGs, tt.noNAG)()
			defer saveRestoreBool(noVariations, tt.noVar)()
			defer saveRestoreBool(noResults, tt.noRes)()
			defer saveRestoreBool(noClocks, tt.noClock)()
			defer saveRestoreBool(jsonOutput, tt.json)()
			defer saveRestoreInt(lineLength, 80)()
			defer saveRestoreInt(ecoMaxHandles, 128)()

			cfg := config.NewConfig()
			applyContentFlags(cfg)

			if cfg.Output.KeepComments != tt.wantComments {
				t.Errorf("KeepComments = %v; want %v", cfg.Output.KeepComments, tt.wantComments)
			}
			if cfg.Output.KeepNAGs != tt.wantNAGs {
				t.Errorf("KeepNAGs = %v; want %v", cfg.Output.KeepNAGs, tt.wantNAGs)
			}
			if cfg.Output.KeepVariations != tt.wantVar {
				t.Errorf("KeepVariations = %v; want %v", cfg.Output.KeepVariations, tt.wantVar)
			}
			if cfg.Output.KeepResults != tt.wantResults {
				t.Errorf("KeepResults = %v; want %v", cfg.Output.KeepResults, tt.wantResults)
			}
			if cfg.Output.StripClockAnnotations != tt.wantStrip {
				t.Errorf("StripClockAnnotations = %v; want %v", cfg.Output.StripClockAnnotations, tt.wantStrip)
			}
			if cfg.Output.JSONFormat != tt.wantJSON {
				t.Errorf("JSONFormat = %v; want %v", cfg.Output.JSONFormat, tt.wantJSON)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// applyOutputFormatFlags
// ---------------------------------------------------------------------------

func TestApplyOutputFormatFlags(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   config.OutputFormat
	}{
		{"lalg", "lalg", config.LALG},
		{"halg", "halg", config.HALG},
		{"elalg", "elalg", config.ELALG},
		{"uci", "uci", config.UCI},
		{"epd", "epd", config.EPD},
		{"fen", "fen", config.FEN},
		{"unknown defaults to SAN", "xyz", config.SAN},
		{"empty defaults to SAN", "", config.SAN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer saveRestoreString(outputFormat, tt.format)()
			cfg := config.NewConfig()
			applyOutputFormatFlags(cfg)
			if cfg.Output.Format != tt.want {
				t.Errorf("Format = %d; want %d", cfg.Output.Format, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// applyMoveBoundsFlags
// ---------------------------------------------------------------------------

func TestApplyMoveBoundsFlags(t *testing.T) {
	t.Run("no bounds set", func(t *testing.T) {
		defer saveRestoreInt(minPly, 0)()
		defer saveRestoreInt(maxPly, 0)()
		defer saveRestoreInt(minMoves, 0)()
		defer saveRestoreInt(maxMoves, 0)()
		cfg := config.NewConfig()
		applyMoveBoundsFlags(cfg)
		if cfg.Filter.CheckMoveBounds {
			t.Error("CheckMoveBounds = true; want false")
		}
	})

	t.Run("minMoves set", func(t *testing.T) {
		defer saveRestoreInt(minPly, 0)()
		defer saveRestoreInt(maxPly, 0)()
		defer saveRestoreInt(minMoves, 10)()
		defer saveRestoreInt(maxMoves, 0)()
		cfg := config.NewConfig()
		applyMoveBoundsFlags(cfg)
		if !cfg.Filter.CheckMoveBounds {
			t.Error("CheckMoveBounds = false; want true")
		}
		if cfg.Filter.LowerMoveBound != 10 {
			t.Errorf("LowerMoveBound = %d; want 10", cfg.Filter.LowerMoveBound)
		}
	})

	t.Run("maxMoves set", func(t *testing.T) {
		defer saveRestoreInt(minPly, 0)()
		defer saveRestoreInt(maxPly, 0)()
		defer saveRestoreInt(minMoves, 0)()
		defer saveRestoreInt(maxMoves, 50)()
		cfg := config.NewConfig()
		applyMoveBoundsFlags(cfg)
		if !cfg.Filter.CheckMoveBounds {
			t.Error("CheckMoveBounds = false; want true")
		}
		if cfg.Filter.UpperMoveBound != 50 {
			t.Errorf("UpperMoveBound = %d; want 50", cfg.Filter.UpperMoveBound)
		}
	})

	t.Run("minPly triggers bounds check", func(t *testing.T) {
		defer saveRestoreInt(minPly, 5)()
		defer saveRestoreInt(maxPly, 0)()
		defer saveRestoreInt(minMoves, 0)()
		defer saveRestoreInt(maxMoves, 0)()
		cfg := config.NewConfig()
		applyMoveBoundsFlags(cfg)
		if !cfg.Filter.CheckMoveBounds {
			t.Error("CheckMoveBounds = false; want true")
		}
	})
}

// ---------------------------------------------------------------------------
// applyAnnotationFlags
// ---------------------------------------------------------------------------

func TestApplyAnnotationFlags(t *testing.T) {
	defer saveRestoreBool(addPlyCount, true)()
	defer saveRestoreBool(addFENComments, true)()
	defer saveRestoreBool(addHashComments, false)()
	defer saveRestoreBool(addHashcodeTag, true)()
	defer saveRestoreBool(fixResultTags, true)()
	defer saveRestoreBool(fixTagStrings, false)()

	cfg := config.NewConfig()
	applyAnnotationFlags(cfg)

	if !cfg.Annotation.AddPlyCount {
		t.Error("AddPlyCount = false; want true")
	}
	if !cfg.Annotation.AddFENComments {
		t.Error("AddFENComments = false; want true")
	}
	if cfg.Annotation.AddHashComments {
		t.Error("AddHashComments = true; want false")
	}
	if !cfg.Annotation.AddHashTag {
		t.Error("AddHashTag = false; want true")
	}
	if !cfg.Annotation.FixResultTags {
		t.Error("FixResultTags = false; want true")
	}
	if cfg.Annotation.FixTagStrings {
		t.Error("FixTagStrings = true; want false")
	}
}

// ---------------------------------------------------------------------------
// applyFilterFlags
// ---------------------------------------------------------------------------

func TestApplyFilterFlags(t *testing.T) {
	defer saveRestoreBool(checkmateFilter, true)()
	defer saveRestoreBool(stalemateFilter, false)()
	defer saveRestoreBool(fiftyMoveFilter, true)()
	defer saveRestoreBool(repetitionFilter, false)()
	defer saveRestoreBool(underpromotionFilter, true)()
	defer saveRestoreBool(useSoundex, true)()

	cfg := config.NewConfig()
	applyFilterFlags(cfg)

	if !cfg.Filter.MatchCheckmate {
		t.Error("MatchCheckmate = false; want true")
	}
	if cfg.Filter.MatchStalemate {
		t.Error("MatchStalemate = true; want false")
	}
	if !cfg.Filter.CheckFiftyMoveRule {
		t.Error("CheckFiftyMoveRule = false; want true")
	}
	if cfg.Filter.CheckRepetition {
		t.Error("CheckRepetition = true; want false")
	}
	if !cfg.Filter.MatchUnderpromotion {
		t.Error("MatchUnderpromotion = false; want true")
	}
	if !cfg.Filter.UseSoundex {
		t.Error("UseSoundex = false; want true")
	}
}

// ---------------------------------------------------------------------------
// applyDuplicateFlags
// ---------------------------------------------------------------------------

func TestApplyDuplicateFlags(t *testing.T) {
	defer saveRestoreInt(duplicateCapacity, 500)()

	cfg := config.NewConfig()
	applyDuplicateFlags(cfg)

	if cfg.Duplicate.MaxCapacity != 500 {
		t.Errorf("MaxCapacity = %d; want 500", cfg.Duplicate.MaxCapacity)
	}
}

// ---------------------------------------------------------------------------
// applyPhase4Flags
// ---------------------------------------------------------------------------

func TestApplyPhase4Flags(t *testing.T) {
	defer saveRestoreBool(nestedComments, true)()
	defer saveRestoreBool(splitVariants, true)()
	defer saveRestoreBool(chess960Mode, true)()
	defer saveRestoreInt(fuzzyDepth, 12)()

	cfg := config.NewConfig()
	applyPhase4Flags(cfg)

	if !cfg.AllowNestedComments {
		t.Error("AllowNestedComments = false; want true")
	}
	if !cfg.SplitVariants {
		t.Error("SplitVariants = false; want true")
	}
	if !cfg.Chess960Mode {
		t.Error("Chess960Mode = false; want true")
	}
	if cfg.FuzzyDepth != 12 {
		t.Errorf("FuzzyDepth = %d; want 12", cfg.FuzzyDepth)
	}
}

// ---------------------------------------------------------------------------
// applyFlags (integration)
// ---------------------------------------------------------------------------

func TestApplyFlags(t *testing.T) {
	// Save/restore all flags that applyFlags touches
	defer saveRestoreBool(noTags, false)()
	defer saveRestoreBool(sevenTagOnly, false)()
	defer saveRestoreBool(noComments, true)()
	defer saveRestoreBool(noNAGs, false)()
	defer saveRestoreBool(noVariations, false)()
	defer saveRestoreBool(noResults, false)()
	defer saveRestoreBool(noClocks, false)()
	defer saveRestoreBool(jsonOutput, false)()
	defer saveRestoreInt(lineLength, 80)()
	defer saveRestoreInt(ecoMaxHandles, 128)()
	defer saveRestoreString(outputFormat, "lalg")()
	defer saveRestoreInt(minPly, 0)()
	defer saveRestoreInt(maxPly, 0)()
	defer saveRestoreInt(minMoves, 5)()
	defer saveRestoreInt(maxMoves, 0)()
	defer saveRestoreBool(addPlyCount, false)()
	defer saveRestoreBool(addFENComments, false)()
	defer saveRestoreBool(addHashComments, false)()
	defer saveRestoreBool(addHashcodeTag, false)()
	defer saveRestoreBool(fixResultTags, false)()
	defer saveRestoreBool(fixTagStrings, false)()
	defer saveRestoreBool(checkmateFilter, false)()
	defer saveRestoreBool(stalemateFilter, false)()
	defer saveRestoreBool(fiftyMoveFilter, false)()
	defer saveRestoreBool(repetitionFilter, false)()
	defer saveRestoreBool(underpromotionFilter, false)()
	defer saveRestoreBool(useSoundex, false)()
	defer saveRestoreBool(nestedComments, false)()
	defer saveRestoreBool(splitVariants, false)()
	defer saveRestoreBool(chess960Mode, false)()
	defer saveRestoreInt(fuzzyDepth, 0)()
	defer saveRestoreInt(duplicateCapacity, 0)()
	defer saveRestoreBool(quiet, false)()
	defer saveRestoreBool(reportOnly, false)()

	cfg := config.NewConfig()
	applyFlags(cfg)

	// Verify a few flags were actually applied
	if cfg.Output.KeepComments {
		t.Error("KeepComments should be false when noComments=true")
	}
	if cfg.Output.Format != config.LALG {
		t.Errorf("Format = %d; want LALG (%d)", cfg.Output.Format, config.LALG)
	}
	if !cfg.Filter.CheckMoveBounds {
		t.Error("CheckMoveBounds should be true when minMoves=5")
	}
	if cfg.Filter.LowerMoveBound != 5 {
		t.Errorf("LowerMoveBound = %d; want 5", cfg.Filter.LowerMoveBound)
	}
}

func TestApplyFlags_QuietAndReportOnly(t *testing.T) {
	defer saveRestoreBool(noTags, false)()
	defer saveRestoreBool(sevenTagOnly, false)()
	defer saveRestoreBool(noComments, false)()
	defer saveRestoreBool(noNAGs, false)()
	defer saveRestoreBool(noVariations, false)()
	defer saveRestoreBool(noResults, false)()
	defer saveRestoreBool(noClocks, false)()
	defer saveRestoreBool(jsonOutput, false)()
	defer saveRestoreInt(lineLength, 80)()
	defer saveRestoreInt(ecoMaxHandles, 128)()
	defer saveRestoreString(outputFormat, "")()
	defer saveRestoreInt(minPly, 0)()
	defer saveRestoreInt(maxPly, 0)()
	defer saveRestoreInt(minMoves, 0)()
	defer saveRestoreInt(maxMoves, 0)()
	defer saveRestoreBool(addPlyCount, false)()
	defer saveRestoreBool(addFENComments, false)()
	defer saveRestoreBool(addHashComments, false)()
	defer saveRestoreBool(addHashcodeTag, false)()
	defer saveRestoreBool(fixResultTags, false)()
	defer saveRestoreBool(fixTagStrings, false)()
	defer saveRestoreBool(checkmateFilter, false)()
	defer saveRestoreBool(stalemateFilter, false)()
	defer saveRestoreBool(fiftyMoveFilter, false)()
	defer saveRestoreBool(repetitionFilter, false)()
	defer saveRestoreBool(underpromotionFilter, false)()
	defer saveRestoreBool(useSoundex, false)()
	defer saveRestoreBool(nestedComments, false)()
	defer saveRestoreBool(splitVariants, false)()
	defer saveRestoreBool(chess960Mode, false)()
	defer saveRestoreInt(fuzzyDepth, 0)()
	defer saveRestoreInt(duplicateCapacity, 0)()
	defer saveRestoreBool(quiet, true)()
	defer saveRestoreBool(reportOnly, true)()

	cfg := config.NewConfig()
	applyFlags(cfg)

	if cfg.Verbosity != 0 {
		t.Errorf("Verbosity = %d; want 0 when quiet=true", cfg.Verbosity)
	}
	if !cfg.CheckOnly {
		t.Error("CheckOnly = false; want true when reportOnly=true")
	}
}
