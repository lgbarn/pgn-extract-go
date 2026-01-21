package engine

import (
	"testing"
)

// TestFormatEvaluation tests the FormatEvaluation function for various scenarios.
func TestFormatEvaluation_PositiveCentipawns(t *testing.T) {
	eval := &Evaluation{Score: 123, IsMate: false}
	got := FormatEvaluation(eval)
	want := "+1.23"
	if got != want {
		t.Errorf("FormatEvaluation() = %q, want %q", got, want)
	}
}

func TestFormatEvaluation_NegativeCentipawns(t *testing.T) {
	eval := &Evaluation{Score: -45, IsMate: false}
	got := FormatEvaluation(eval)
	want := "-0.45"
	if got != want {
		t.Errorf("FormatEvaluation() = %q, want %q", got, want)
	}
}

func TestFormatEvaluation_ZeroCentipawns(t *testing.T) {
	eval := &Evaluation{Score: 0, IsMate: false}
	got := FormatEvaluation(eval)
	want := "+0.00"
	if got != want {
		t.Errorf("FormatEvaluation() = %q, want %q", got, want)
	}
}

func TestFormatEvaluation_LargeCentipawns(t *testing.T) {
	eval := &Evaluation{Score: 1250, IsMate: false}
	got := FormatEvaluation(eval)
	want := "+12.50"
	if got != want {
		t.Errorf("FormatEvaluation() = %q, want %q", got, want)
	}
}

func TestFormatEvaluation_PositiveMate(t *testing.T) {
	eval := &Evaluation{IsMate: true, MateIn: 3}
	got := FormatEvaluation(eval)
	want := "+M3"
	if got != want {
		t.Errorf("FormatEvaluation() = %q, want %q", got, want)
	}
}

func TestFormatEvaluation_NegativeMate(t *testing.T) {
	eval := &Evaluation{IsMate: true, MateIn: -5}
	got := FormatEvaluation(eval)
	want := "-M5"
	if got != want {
		t.Errorf("FormatEvaluation() = %q, want %q", got, want)
	}
}

func TestFormatEvaluation_MateInOne(t *testing.T) {
	eval := &Evaluation{IsMate: true, MateIn: 1}
	got := FormatEvaluation(eval)
	want := "+M1"
	if got != want {
		t.Errorf("FormatEvaluation() = %q, want %q", got, want)
	}
}

// TestFormatEvaluation_TableDriven runs comprehensive tests using a table.
func TestFormatEvaluation_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		eval *Evaluation
		want string
	}{
		{
			name: "small positive",
			eval: &Evaluation{Score: 15, IsMate: false},
			want: "+0.15",
		},
		{
			name: "small negative",
			eval: &Evaluation{Score: -8, IsMate: false},
			want: "-0.08",
		},
		{
			name: "exactly one pawn",
			eval: &Evaluation{Score: 100, IsMate: false},
			want: "+1.00",
		},
		{
			name: "exactly minus one pawn",
			eval: &Evaluation{Score: -100, IsMate: false},
			want: "-1.00",
		},
		{
			name: "very large positive",
			eval: &Evaluation{Score: 9999, IsMate: false},
			want: "+99.99",
		},
		{
			name: "very large negative",
			eval: &Evaluation{Score: -9999, IsMate: false},
			want: "-99.99",
		},
		{
			name: "mate in many",
			eval: &Evaluation{IsMate: true, MateIn: 15},
			want: "+M15",
		},
		{
			name: "getting mated in many",
			eval: &Evaluation{IsMate: true, MateIn: -20},
			want: "-M20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatEvaluation(tt.eval)
			if got != tt.want {
				t.Errorf("FormatEvaluation() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParseInfo tests the parseInfo method indirectly through an engine instance.
func TestParseInfo_Depth(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{}

	e.parseInfo("info depth 20 seldepth 25 multipv 1 score cp 125 nodes 123456", eval)

	if eval.Depth != 20 {
		t.Errorf("parseInfo depth = %d, want 20", eval.Depth)
	}
}

func TestParseInfo_ScoreCp(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{}

	e.parseInfo("info depth 15 score cp 125 nodes 100000", eval)

	if eval.Score != 125 {
		t.Errorf("parseInfo score = %d, want 125", eval.Score)
	}
	if eval.IsMate {
		t.Error("parseInfo IsMate should be false for cp score")
	}
}

func TestParseInfo_ScoreMate(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{}

	e.parseInfo("info depth 15 score mate 3 nodes 100000", eval)

	if eval.MateIn != 3 {
		t.Errorf("parseInfo MateIn = %d, want 3", eval.MateIn)
	}
	if !eval.IsMate {
		t.Error("parseInfo IsMate should be true for mate score")
	}
}

func TestParseInfo_NegativeScoreCp(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{}

	e.parseInfo("info depth 18 score cp -50 nodes 200000", eval)

	if eval.Score != -50 {
		t.Errorf("parseInfo score = %d, want -50", eval.Score)
	}
}

func TestParseInfo_NegativeScoreMate(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{}

	e.parseInfo("info depth 20 score mate -3 nodes 300000", eval)

	if eval.MateIn != -3 {
		t.Errorf("parseInfo MateIn = %d, want -3", eval.MateIn)
	}
	if !eval.IsMate {
		t.Error("parseInfo IsMate should be true for negative mate score")
	}
}

func TestParseInfo_MissingFields(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{Depth: 10, Score: 50}

	// Line missing score info should not change existing values
	e.parseInfo("info nodes 100000 time 500", eval)

	if eval.Score != 50 {
		t.Errorf("parseInfo should not change score, got %d, want 50", eval.Score)
	}
}

func TestParseInfo_EmptyLine(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{Depth: 10, Score: 25}

	e.parseInfo("", eval)

	// Empty line should not change anything
	if eval.Depth != 10 || eval.Score != 25 {
		t.Error("parseInfo should not modify eval for empty line")
	}
}

func TestParseInfo_OnlyDepth(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{}

	e.parseInfo("info depth 12", eval)

	if eval.Depth != 12 {
		t.Errorf("parseInfo depth = %d, want 12", eval.Depth)
	}
}

func TestParseInfo_ScoreAtEndOfLine(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{}

	// Edge case: score at end but missing value
	e.parseInfo("info depth 10 score", eval)

	// Should not crash, depth should still be parsed
	if eval.Depth != 10 {
		t.Errorf("parseInfo depth = %d, want 10", eval.Depth)
	}
}

func TestParseInfo_DepthAtEndOfLine(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{}

	// Edge case: depth at end but missing value
	e.parseInfo("info nodes 100000 depth", eval)

	// Should not crash
	if eval.Depth != 0 {
		t.Errorf("parseInfo depth = %d, want 0 for missing value", eval.Depth)
	}
}

// TestEvaluation_DefaultValues tests that zero-valued Evaluation struct has expected defaults.
func TestEvaluation_DefaultValues(t *testing.T) {
	eval := Evaluation{}

	if eval.Score != 0 {
		t.Errorf("default Score = %d, want 0", eval.Score)
	}
	if eval.IsMate {
		t.Error("default IsMate should be false")
	}
	if eval.MateIn != 0 {
		t.Errorf("default MateIn = %d, want 0", eval.MateIn)
	}
	if eval.Depth != 0 {
		t.Errorf("default Depth = %d, want 0", eval.Depth)
	}
	if eval.BestMove != "" {
		t.Errorf("default BestMove = %q, want empty string", eval.BestMove)
	}
}

// TestUCIEngine_Struct tests UCIEngine struct initialization.
func TestUCIEngine_Struct(t *testing.T) {
	e := &UCIEngine{depth: 15}

	if e.depth != 15 {
		t.Errorf("UCIEngine depth = %d, want 15", e.depth)
	}
}

// TestParseInfo_ComplexLine tests parsing a realistic UCI info line.
func TestParseInfo_ComplexLine(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{}

	// A realistic info line from Stockfish
	line := "info depth 22 seldepth 31 multipv 1 score cp 35 nodes 2145678 nps 2500000 hashfull 456 tbhits 0 time 858 pv e2e4 e7e5 g1f3"

	e.parseInfo(line, eval)

	if eval.Depth != 22 {
		t.Errorf("parseInfo depth = %d, want 22", eval.Depth)
	}
	if eval.Score != 35 {
		t.Errorf("parseInfo score = %d, want 35", eval.Score)
	}
	if eval.IsMate {
		t.Error("parseInfo IsMate should be false")
	}
}

// TestParseInfo_MateScore tests parsing a mate score from a complex line.
func TestParseInfo_MateScore(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{}

	line := "info depth 18 seldepth 12 multipv 1 score mate 7 nodes 500000 nps 2000000 time 250 pv e1g1"

	e.parseInfo(line, eval)

	if eval.Depth != 18 {
		t.Errorf("parseInfo depth = %d, want 18", eval.Depth)
	}
	if !eval.IsMate {
		t.Error("parseInfo IsMate should be true")
	}
	if eval.MateIn != 7 {
		t.Errorf("parseInfo MateIn = %d, want 7", eval.MateIn)
	}
}

func TestParseInfo_UpdatesOnlySpecifiedFields(t *testing.T) {
	e := &UCIEngine{}
	eval := &Evaluation{
		Depth:    5,
		Score:    100,
		IsMate:   false,
		MateIn:   0,
		BestMove: "e2e4",
	}

	// Parsing a line with only depth should update depth but not other fields
	e.parseInfo("info depth 10", eval)

	if eval.Depth != 10 {
		t.Errorf("parseInfo depth = %d, want 10", eval.Depth)
	}
	if eval.Score != 100 {
		t.Errorf("parseInfo should not change score, got %d", eval.Score)
	}
	if eval.BestMove != "e2e4" {
		t.Errorf("parseInfo should not change bestmove, got %q", eval.BestMove)
	}
}
