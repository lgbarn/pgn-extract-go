package cql

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

func TestTransformFlipHorizontal(t *testing.T) {
	// Position with white king on g1 (already castled)
	board := engine.MustBoardFromFEN("r1bq1rk1/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQ1RK1 w - - 6 5")

	tests := []struct {
		cql      string
		expected bool
	}{
		// King on g1 should match with flip because b1 mirrored = g1
		{"(flip (piece K g1))", true},
		// Flip should also match K b1 because it checks both g1 (original) and b1 (flipped)
		{"(flip (piece K b1))", true},
		// King NOT on h1, and a1 has nothing after flip
		{"(flip (piece K h1))", false},
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

func TestTransformFlipVertical(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		// White king on e1, flipvertical should also match e8
		{"(flipvertical (piece K e1))", true},
		// Black king on e8, flipvertical should match e1
		{"(flipvertical (piece k e8))", true},
		// Pawn on e2, flipvertical should also match e7
		{"(flipvertical (piece P e2))", true},
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

func TestTransformFlipColor(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		// White king on e1, flipcolor should also check for black king on e1
		{"(flipcolor (piece K e1))", true},
		// Check for white pawn on e2 or black pawn on e2
		{"(flipcolor (piece P e2))", true},
		// There's no queen on e1 (neither white nor black)
		{"(flipcolor (piece Q e1))", false},
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

func TestTransformShift(t *testing.T) {
	// Position with king in corner
	board := engine.MustBoardFromFEN("7k/8/8/8/8/8/8/K7 w - - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		// King on a1, shift should find it at any corner
		{"(shift (piece K a1))", true},
		// Also matches because black king on h8 is a king in a corner
		{"(shift (piece k h8))", true},
		// No pieces in center
		{"(shift (piece K e4))", true}, // Shift can translate to any position
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

func TestTransformShiftHorizontal(t *testing.T) {
	// White king on e1
	board := engine.MustBoardFromFEN("8/8/8/8/8/8/8/4K3 w - - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		// King on e1, shifthorizontal should match any file on rank 1
		{"(shifthorizontal (piece K a1))", true},
		{"(shifthorizontal (piece K h1))", true},
		// But not on rank 2
		{"(shifthorizontal (piece K e2))", false},
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

func TestTransformShiftVertical(t *testing.T) {
	// White king on e1
	board := engine.MustBoardFromFEN("8/8/8/8/8/8/8/4K3 w - - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		// King on e1, shiftvertical should match any rank on file e
		{"(shiftvertical (piece K e4))", true},
		{"(shiftvertical (piece K e8))", true},
		// But not on file d
		{"(shiftvertical (piece K d1))", false},
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

func TestTransformCombined(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		// Check multiple transformations in combination
		{"(and (piece K e1) (flip (piece K e1)))", true},
		{"(or (flip (piece K g1)) (piece K e1))", true},
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
