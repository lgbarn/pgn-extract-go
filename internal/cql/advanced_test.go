package cql

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

func TestEvalResult(t *testing.T) {
	// Create a game with a specific result
	game := &chess.Game{
		Tags: map[string]string{
			"Result": "1-0",
			"White":  "Fischer",
			"Black":  "Spassky",
		},
	}
	board := setupBoard("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{`result "1-0"`, true},
		{`result "0-1"`, false},
		{`result "1/2-1/2"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluatorWithGame(board, game)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalPlayer(t *testing.T) {
	// Create a game with specific players
	game := &chess.Game{
		Tags: map[string]string{
			"White": "Fischer, Bobby",
			"Black": "Spassky, Boris",
		},
	}
	board := setupBoard("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{`player "Fischer"`, true},
		{`player "Spassky"`, true},
		{`player "Carlsen"`, false},
		{`player "Bobby"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluatorWithGame(board, game)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalYear(t *testing.T) {
	// Create a game with a specific date
	game := &chess.Game{
		Tags: map[string]string{
			"Date": "1972.07.11",
		},
	}
	board := setupBoard("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{`(== (year) 1972)`, true},
		{`(> (year) 1970)`, true},
		{`(< (year) 1980)`, true},
		{`(>= (year) 1972)`, true},
		{`(<= (year) 1972)`, true},
		{`(> (year) 1980)`, false},
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluatorWithGame(board, game)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalElo(t *testing.T) {
	// Create a game with Elo ratings
	game := &chess.Game{
		Tags: map[string]string{
			"WhiteElo": "2785",
			"BlackElo": "2660",
		},
	}
	board := setupBoard("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{`(> (elo "white") 2700)`, true},
		{`(> (elo "black") 2700)`, false},
		{`(< (elo "white") 2800)`, true},
		{`(>= (elo "white") 2785)`, true},
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluatorWithGame(board, game)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalBetween(t *testing.T) {
	// Position with pieces to test between filter
	board := setupBoard("8/8/8/3q4/8/8/8/R3K3 w - - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		// Squares between a1 and e1 (b1, c1, d1)
		{"(between a1 e1)", true},
		// Check for empty squares between
		{"(and (between a1 e1) (piece _ b1))", true},
		{"(and (between a1 e1) (piece _ c1))", true},
		{"(and (between a1 e1) (piece _ d1))", true},
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalPin(t *testing.T) {
	// Position with pinned piece: black bishop on c6 pins white knight on d5 to white king on e4
	board := setupBoard("8/8/2b5/3N4/4K3/8/8/8 w - - 0 1")

	tests := []struct {
		name     string
		cql      string
		expected bool
	}{
		// Knight on d5 is pinned to king on e4 through bishop on c6
		{"knight pinned", "(pin N b K)", true},
		// No queen pinning anything
		{"no queen pin", "(pin N q K)", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalRay(t *testing.T) {
	// Position with pieces along a ray
	board := setupBoard("8/8/8/8/R3K3/8/8/8 w - - 0 1")

	tests := []struct {
		name     string
		cql      string
		expected bool
	}{
		// Ray from a4 to e4 (horizontal)
		{"horizontal ray", `ray "horizontal" a4 e4`, true},
		// No vertical ray from a4 to e4
		{"no vertical ray", `ray "vertical" a4 e4`, false},
		// Diagonal ray test
		{"diagonal ray", `ray "diagonal" a1 d4`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
