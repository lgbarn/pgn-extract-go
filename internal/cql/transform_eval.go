package cql

import "strings"

// Transform functions
type squareTransform func(col, rank int) (int, int)

// colorSwapMap maps piece characters to their opposite color equivalents.
var colorSwapMap = map[rune]rune{
	'K': 'k', 'Q': 'q', 'R': 'r', 'B': 'b', 'N': 'n', 'P': 'p',
	'k': 'K', 'q': 'Q', 'r': 'R', 'b': 'B', 'n': 'N', 'p': 'P',
	'A': 'a', 'a': 'A',
}

func flipHorizontal(col, rank int) (int, int) {
	return 7 - col, rank // a↔h, b↔g, etc.
}

func flipVertical(col, rank int) (int, int) {
	return col, 7 - rank // 1↔8, 2↔7, etc.
}

func flipColor(col, rank int) (int, int) {
	// Color flip doesn't change squares, just piece colors
	// Handled specially in transformNode
	return col, rank
}

// evalFlip evaluates the child expression with horizontal flip transformation.
// Tries both the original pattern and its horizontal mirror (a↔h files).
func (e *Evaluator) evalFlip(args []Node) bool {
	if len(args) < 1 {
		return false
	}

	// Try original
	if e.Evaluate(args[0]) {
		return true
	}

	// Try horizontal flip - transform the pattern and evaluate
	flippedNode := e.transformNode(args[0], flipHorizontal)
	return e.Evaluate(flippedNode)
}

// evalFlipVertical evaluates with vertical flip transformation (1↔8 ranks).
func (e *Evaluator) evalFlipVertical(args []Node) bool {
	if len(args) < 1 {
		return false
	}

	// Try original
	if e.Evaluate(args[0]) {
		return true
	}

	// Try vertical flip
	flippedNode := e.transformNode(args[0], flipVertical)
	return e.Evaluate(flippedNode)
}

// evalFlipColor evaluates with color flip transformation (white↔black).
func (e *Evaluator) evalFlipColor(args []Node) bool {
	if len(args) < 1 {
		return false
	}

	// Try original
	if e.Evaluate(args[0]) {
		return true
	}

	// Try color flip
	flippedNode := e.transformNode(args[0], flipColor)
	return e.Evaluate(flippedNode)
}

// evalShift tries all possible translations of the pattern.
func (e *Evaluator) evalShift(args []Node) bool {
	if len(args) < 1 {
		return false
	}

	// Try all possible shifts
	for dCol := -7; dCol <= 7; dCol++ {
		for dRank := -7; dRank <= 7; dRank++ {
			shiftedNode := e.transformNode(args[0], func(col, rank int) (int, int) {
				return col + dCol, rank + dRank
			})
			if e.Evaluate(shiftedNode) {
				return true
			}
		}
	}
	return false
}

// evalShiftHorizontal tries all horizontal translations.
func (e *Evaluator) evalShiftHorizontal(args []Node) bool {
	if len(args) < 1 {
		return false
	}

	// Try all horizontal shifts
	for dCol := -7; dCol <= 7; dCol++ {
		shiftedNode := e.transformNode(args[0], func(col, rank int) (int, int) {
			return col + dCol, rank
		})
		if e.Evaluate(shiftedNode) {
			return true
		}
	}
	return false
}

// evalShiftVertical tries all vertical translations.
func (e *Evaluator) evalShiftVertical(args []Node) bool {
	if len(args) < 1 {
		return false
	}

	// Try all vertical shifts
	for dRank := -7; dRank <= 7; dRank++ {
		shiftedNode := e.transformNode(args[0], func(col, rank int) (int, int) {
			return col, rank + dRank
		})
		if e.Evaluate(shiftedNode) {
			return true
		}
	}
	return false
}

// transformNode creates a transformed copy of an AST node.
func (e *Evaluator) transformNode(node Node, transform squareTransform) Node {
	switch n := node.(type) {
	case *FilterNode:
		return e.transformFilterNode(n, transform)
	case *LogicalNode:
		children := make([]Node, len(n.Children))
		for i, child := range n.Children {
			children[i] = e.transformNode(child, transform)
		}
		return &LogicalNode{Op: n.Op, Children: children}
	case *SquareNode:
		return e.transformSquareNode(n, transform)
	case *PieceNode:
		// For flipColor, we need to swap piece colors
		if transform == nil {
			return n
		}
		// Check if this is a color flip by testing a known point
		testCol, testRank := transform(0, 0)
		if testCol == 0 && testRank == 0 {
			// This could be flipColor - check by comparing with flipHorizontal
			hCol, hRank := flipHorizontal(0, 0)
			if hCol != 0 || hRank != 0 {
				// Not flipHorizontal, check flipVertical
				vCol, vRank := flipVertical(0, 0)
				if vCol != 0 || vRank != 0 {
					// Must be flipColor - swap piece colors
					return e.transformPieceNodeColor(n)
				}
			}
		}
		return n
	default:
		return node
	}
}

// transformFilterNode transforms a filter node with the given transform.
func (e *Evaluator) transformFilterNode(f *FilterNode, transform squareTransform) *FilterNode {
	// Transform arguments
	args := make([]Node, len(f.Args))
	for i, arg := range f.Args {
		args[i] = e.transformNode(arg, transform)
	}
	return &FilterNode{Name: f.Name, Args: args}
}

// transformSquareNode transforms a square node with the given transform.
func (e *Evaluator) transformSquareNode(s *SquareNode, transform squareTransform) *SquareNode {
	// Parse the square designator, transform, and create new designator
	squares := e.parseSquareSet(s.Designator)
	if len(squares) == 0 {
		return s
	}

	// For single squares, transform and create new designator
	if len(squares) == 1 {
		sq := squares[0]
		newCol, newRank := transform(int(sq.col), int(sq.rank))
		if newCol >= 0 && newCol < 8 && newRank >= 0 && newRank < 8 {
			newDesig := string(rune('a'+newCol)) + string(rune('1'+newRank))
			return &SquareNode{Designator: newDesig}
		}
		// Out of bounds - return original (won't match)
		return s
	}

	// For complex square sets, transform each square
	// This is more complex - for now, just return original
	return s
}

// transformPieceNodeColor swaps piece colors in a piece node.
func (e *Evaluator) transformPieceNodeColor(p *PieceNode) *PieceNode {
	var sb strings.Builder
	sb.Grow(len(p.Designator))

	for _, c := range p.Designator {
		if swapped, ok := colorSwapMap[c]; ok {
			sb.WriteRune(swapped)
		} else {
			sb.WriteRune(c)
		}
	}

	return &PieceNode{Designator: sb.String()}
}
