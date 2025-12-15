package cql

// Node is the interface for all AST nodes.
type Node interface {
	node()
	String() string
}

// FilterNode represents a CQL filter like "piece", "mate", "check".
type FilterNode struct {
	Name string
	Args []Node
}

func (f *FilterNode) node() {}
func (f *FilterNode) String() string {
	if len(f.Args) == 0 {
		return f.Name
	}
	result := f.Name
	for _, arg := range f.Args {
		result += " " + arg.String()
	}
	return result
}

// LogicalNode represents logical operators (and, or, not).
type LogicalNode struct {
	Op       string // "and", "or", "not"
	Children []Node
}

func (l *LogicalNode) node() {}
func (l *LogicalNode) String() string {
	result := "(" + l.Op
	for _, child := range l.Children {
		result += " " + child.String()
	}
	result += ")"
	return result
}

// ComparisonNode represents comparison operations.
type ComparisonNode struct {
	Op    string // "<", ">", "<=", ">=", "=="
	Left  Node
	Right Node
}

func (c *ComparisonNode) node() {}
func (c *ComparisonNode) String() string {
	return "(" + c.Op + " " + c.Left.String() + " " + c.Right.String() + ")"
}

// PieceNode represents a piece designator.
type PieceNode struct {
	Designator string // K, Q, R, B, N, P, k, q, r, b, n, p, A, a, _, ?, or [RQ] etc.
}

func (p *PieceNode) node() {}
func (p *PieceNode) String() string {
	return p.Designator
}

// SquareNode represents a square or square set.
type SquareNode struct {
	Designator string // a1, e4, [a-h]1, a[1-8], [a-d][1-4], .
}

func (s *SquareNode) node() {}
func (s *SquareNode) String() string {
	return s.Designator
}

// NumberNode represents a numeric value.
type NumberNode struct {
	Value int
}

func (n *NumberNode) node() {}
func (n *NumberNode) String() string {
	return string(rune('0' + n.Value)) // Simple for single digits
}

// StringNode represents a string literal.
type StringNode struct {
	Value string
}

func (s *StringNode) node() {}
func (s *StringNode) String() string {
	return `"` + s.Value + `"`
}
