package cql

import (
	"testing"
)

func TestParserSimpleFilters(t *testing.T) {
	tests := []struct {
		input        string
		expectedName string
		expectedArgs int
	}{
		{"mate", "mate", 0},
		{"stalemate", "stalemate", 0},
		{"check", "check", 0},
		{"wtm", "wtm", 0},
		{"btm", "btm", 0},
		{"piece K e1", "piece", 2},
		{"attack R k", "attack", 2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			filter, ok := node.(*FilterNode)
			if !ok {
				t.Fatalf("expected FilterNode, got %T", node)
			}

			if filter.Name != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, filter.Name)
			}

			if len(filter.Args) != tt.expectedArgs {
				t.Errorf("expected %d args, got %d", tt.expectedArgs, len(filter.Args))
			}
		})
	}
}

func TestParserPieceFilter(t *testing.T) {
	node, err := Parse("piece K e1")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	filter, ok := node.(*FilterNode)
	if !ok {
		t.Fatalf("expected FilterNode, got %T", node)
	}

	if filter.Name != "piece" {
		t.Errorf("expected name 'piece', got %q", filter.Name)
	}

	if len(filter.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(filter.Args))
	}

	// First arg should be a piece
	pieceArg, ok := filter.Args[0].(*PieceNode)
	if !ok {
		t.Errorf("expected PieceNode, got %T", filter.Args[0])
	} else if pieceArg.Designator != "K" {
		t.Errorf("expected piece 'K', got %q", pieceArg.Designator)
	}

	// Second arg should be a square
	squareArg, ok := filter.Args[1].(*SquareNode)
	if !ok {
		t.Errorf("expected SquareNode, got %T", filter.Args[1])
	} else if squareArg.Designator != "e1" {
		t.Errorf("expected square 'e1', got %q", squareArg.Designator)
	}
}

func TestParserLogicalAnd(t *testing.T) {
	node, err := Parse("(and mate wtm)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	logical, ok := node.(*LogicalNode)
	if !ok {
		t.Fatalf("expected LogicalNode, got %T", node)
	}

	if logical.Op != "and" {
		t.Errorf("expected op 'and', got %q", logical.Op)
	}

	if len(logical.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(logical.Children))
	}
}

func TestParserLogicalOr(t *testing.T) {
	node, err := Parse("(or mate stalemate)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	logical, ok := node.(*LogicalNode)
	if !ok {
		t.Fatalf("expected LogicalNode, got %T", node)
	}

	if logical.Op != "or" {
		t.Errorf("expected op 'or', got %q", logical.Op)
	}

	if len(logical.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(logical.Children))
	}
}

func TestParserLogicalNot(t *testing.T) {
	node, err := Parse("(not check)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	logical, ok := node.(*LogicalNode)
	if !ok {
		t.Fatalf("expected LogicalNode, got %T", node)
	}

	if logical.Op != "not" {
		t.Errorf("expected op 'not', got %q", logical.Op)
	}

	if len(logical.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(logical.Children))
	}
}

func TestParserNestedLogical(t *testing.T) {
	node, err := Parse("(and (piece K e1) (piece k e8))")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	logical, ok := node.(*LogicalNode)
	if !ok {
		t.Fatalf("expected LogicalNode, got %T", node)
	}

	if logical.Op != "and" {
		t.Errorf("expected op 'and', got %q", logical.Op)
	}

	if len(logical.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(logical.Children))
	}

	// Both children should be FilterNodes
	for i, child := range logical.Children {
		filter, ok := child.(*FilterNode)
		if !ok {
			t.Errorf("child %d: expected FilterNode, got %T", i, child)
		} else if filter.Name != "piece" {
			t.Errorf("child %d: expected name 'piece', got %q", i, filter.Name)
		}
	}
}

func TestParserComparison(t *testing.T) {
	tests := []struct {
		input string
		op    string
	}{
		{"(> (count P) 3)", ">"},
		{"(< (count R) 2)", "<"},
		{"(>= (count Q) 1)", ">="},
		{"(<= (count N) 2)", "<="},
		{"(== (count B) 2)", "=="},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			comp, ok := node.(*ComparisonNode)
			if !ok {
				t.Fatalf("expected ComparisonNode, got %T", node)
			}

			if comp.Op != tt.op {
				t.Errorf("expected op %q, got %q", tt.op, comp.Op)
			}
		})
	}
}

func TestParserImplicitAnd(t *testing.T) {
	// Multiple filters without explicit "and" should be implicitly ANDed
	node, err := Parse("mate wtm")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	logical, ok := node.(*LogicalNode)
	if !ok {
		t.Fatalf("expected LogicalNode for implicit AND, got %T", node)
	}

	if logical.Op != "and" {
		t.Errorf("expected implicit 'and', got %q", logical.Op)
	}

	if len(logical.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(logical.Children))
	}
}

func TestParserPlayerFilter(t *testing.T) {
	node, err := Parse(`player "Carlsen"`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	filter, ok := node.(*FilterNode)
	if !ok {
		t.Fatalf("expected FilterNode, got %T", node)
	}

	if filter.Name != "player" {
		t.Errorf("expected name 'player', got %q", filter.Name)
	}

	if len(filter.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(filter.Args))
	}

	strArg, ok := filter.Args[0].(*StringNode)
	if !ok {
		t.Errorf("expected StringNode, got %T", filter.Args[0])
	} else if strArg.Value != "Carlsen" {
		t.Errorf("expected value 'Carlsen', got %q", strArg.Value)
	}
}

func TestParserResultFilter(t *testing.T) {
	node, err := Parse(`result "1-0"`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	filter, ok := node.(*FilterNode)
	if !ok {
		t.Fatalf("expected FilterNode, got %T", node)
	}

	if filter.Name != "result" {
		t.Errorf("expected name 'result', got %q", filter.Name)
	}
}

func TestParserPieceSet(t *testing.T) {
	node, err := Parse("piece [RQ] a1")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	filter, ok := node.(*FilterNode)
	if !ok {
		t.Fatalf("expected FilterNode, got %T", node)
	}

	pieceArg, ok := filter.Args[0].(*PieceNode)
	if !ok {
		t.Errorf("expected PieceNode, got %T", filter.Args[0])
	} else if pieceArg.Designator != "[RQ]" {
		t.Errorf("expected piece '[RQ]', got %q", pieceArg.Designator)
	}
}

func TestParserSquareRange(t *testing.T) {
	node, err := Parse("piece K [a-h]1")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	filter, ok := node.(*FilterNode)
	if !ok {
		t.Fatalf("expected FilterNode, got %T", node)
	}

	squareArg, ok := filter.Args[1].(*SquareNode)
	if !ok {
		t.Errorf("expected SquareNode, got %T", filter.Args[1])
	} else if squareArg.Designator != "[a-h]1" {
		t.Errorf("expected square '[a-h]1', got %q", squareArg.Designator)
	}
}

func TestParserErrors(t *testing.T) {
	tests := []string{
		"(",         // Unclosed paren
		"(and",      // Unclosed paren with content
		"(and mate", // Unclosed nested
		")",         // Unexpected close paren
		"(and )",    // Empty logical
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := Parse(input)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParserComplexQuery(t *testing.T) {
	// Find back rank mates
	input := "(and mate (piece [RQ] [a-h]8) (piece k [a-h]8))"

	node, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	logical, ok := node.(*LogicalNode)
	if !ok {
		t.Fatalf("expected LogicalNode, got %T", node)
	}

	if logical.Op != "and" {
		t.Errorf("expected 'and', got %q", logical.Op)
	}

	if len(logical.Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(logical.Children))
	}
}
