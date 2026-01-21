package cql

import (
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// Evaluator evaluates CQL expressions against a chess position.
type Evaluator struct {
	board *chess.Board
	game  *chess.Game // Optional, for game-level filters
}

// NewEvaluator creates a new evaluator for the given board position.
func NewEvaluator(board *chess.Board) *Evaluator {
	return &Evaluator{board: board}
}

// NewEvaluatorWithGame creates a new evaluator with both board and game context.
func NewEvaluatorWithGame(board *chess.Board, game *chess.Game) *Evaluator {
	return &Evaluator{board: board, game: game}
}

// SetBoard updates the board for this evaluator, allowing reuse across positions.
func (e *Evaluator) SetBoard(board *chess.Board) {
	e.board = board
}

// SetGame updates the game context for this evaluator.
func (e *Evaluator) SetGame(game *chess.Game) {
	e.game = game
}

// Evaluate evaluates the CQL expression and returns true if it matches.
func (e *Evaluator) Evaluate(node Node) bool {
	switch n := node.(type) {
	case *FilterNode:
		return e.evalFilter(n)
	case *LogicalNode:
		return e.evalLogical(n)
	case *ComparisonNode:
		return e.evalComparison(n)
	default:
		return false
	}
}

func (e *Evaluator) evalFilter(f *FilterNode) bool {
	switch f.Name {
	case "piece":
		return e.evalPiece(f.Args)
	case "attack":
		return e.evalAttack(f.Args)
	case "check":
		return e.evalCheck()
	case "mate":
		return e.evalMate()
	case "stalemate":
		return e.evalStalemate()
	case "wtm":
		return e.board.ToMove == chess.White
	case "btm":
		return e.board.ToMove == chess.Black
	case "count":
		// Count returns a number, handled in comparison
		return false
	// Transformation filters
	case "flip":
		return e.evalFlip(f.Args)
	case "flipvertical":
		return e.evalFlipVertical(f.Args)
	case "flipcolor":
		return e.evalFlipColor(f.Args)
	case "shift":
		return e.evalShift(f.Args)
	case "shifthorizontal":
		return e.evalShiftHorizontal(f.Args)
	case "shiftvertical":
		return e.evalShiftVertical(f.Args)
	// Game-level filters
	case "result":
		return e.evalResult(f.Args)
	case "player":
		return e.evalPlayer(f.Args)
	// Position filters
	case "between":
		return e.evalBetween(f.Args)
	case "pin":
		return e.evalPin(f.Args)
	case "ray":
		return e.evalRay(f.Args)
	default:
		return false
	}
}

func (e *Evaluator) evalLogical(l *LogicalNode) bool {
	switch l.Op {
	case "and":
		for _, child := range l.Children {
			if !e.Evaluate(child) {
				return false
			}
		}
		return true
	case "or":
		for _, child := range l.Children {
			if e.Evaluate(child) {
				return true
			}
		}
		return false
	case "not":
		if len(l.Children) == 0 {
			return false
		}
		return !e.Evaluate(l.Children[0])
	default:
		return false
	}
}

func (e *Evaluator) evalComparison(c *ComparisonNode) bool {
	left := e.evalNumeric(c.Left)
	right := e.evalNumeric(c.Right)

	switch c.Op {
	case "<":
		return left < right
	case ">":
		return left > right
	case "<=":
		return left <= right
	case ">=":
		return left >= right
	case "==":
		return left == right
	default:
		return false
	}
}

func (e *Evaluator) evalNumeric(node Node) int {
	switch n := node.(type) {
	case *NumberNode:
		return n.Value
	case *FilterNode:
		switch n.Name {
		case "count":
			return e.evalCount(n.Args)
		case "material":
			return e.evalMaterial(n.Args)
		case "year":
			return e.evalYear()
		case "elo":
			return e.evalElo(n.Args)
		}
	}
	return 0
}

// Helper types and functions

type square struct {
	col  chess.Col
	rank chess.Rank
}

func (e *Evaluator) parseSquareSet(desig string) []square {
	if desig == "." {
		// All squares
		squares := make([]square, 0, 64)
		for rank := chess.Rank(0); rank < 8; rank++ {
			for col := chess.Col(0); col < 8; col++ {
				squares = append(squares, square{col, rank})
			}
		}
		return squares
	}

	// Simple single square like "e1"
	if len(desig) == 2 && desig[0] >= 'a' && desig[0] <= 'h' && desig[1] >= '1' && desig[1] <= '8' {
		col := chess.Col(desig[0] - 'a')
		rank := chess.Rank(desig[1] - '1')
		return []square{{col, rank}}
	}

	// Range patterns like [a-h]1, a[1-8], [a-d][1-4]
	// For now, handle simple patterns
	var squares []square

	// Try to parse as range pattern
	files := "abcdefgh"
	ranks := "12345678"

	if strings.HasPrefix(desig, "[") {
		// [a-h]1 or [a-d][1-4] pattern
		return e.parseComplexSquareSet(desig)
	}

	// a[1-8] pattern
	if len(desig) > 2 && desig[1] == '[' {
		file := desig[0]
		if file >= 'a' && file <= 'h' {
			col := chess.Col(file - 'a')
			// Parse rank range
			rankRange := desig[2 : len(desig)-1] // Remove brackets
			parts := strings.Split(rankRange, "-")
			if len(parts) == 2 {
				startRank := parts[0][0] - '1'
				endRank := parts[1][0] - '1'
				for r := startRank; r <= endRank; r++ {
					squares = append(squares, square{col, chess.Rank(r)})
				}
				return squares
			}
		}
	}

	// Fallback: treat each character
	for _, r := range ranks {
		for _, f := range files {
			if strings.Contains(desig, string(f)) && strings.Contains(desig, string(r)) {
				col := chess.Col(f - 'a')
				rank := chess.Rank(r - '1')
				squares = append(squares, square{col, rank})
			}
		}
	}

	return squares
}

func (e *Evaluator) parseComplexSquareSet(desig string) []square {
	var squares []square

	// [a-h]1 pattern
	if strings.HasPrefix(desig, "[") && !strings.Contains(desig[1:], "[") {
		// Single file range with rank
		closeBracket := strings.Index(desig, "]")
		if closeBracket == -1 {
			return squares
		}
		fileRange := desig[1:closeBracket]
		rankPart := desig[closeBracket+1:]

		files := e.parseRange(fileRange, 'a', 'h')
		if len(rankPart) == 1 && rankPart[0] >= '1' && rankPart[0] <= '8' {
			rank := chess.Rank(rankPart[0] - '1')
			for _, f := range files {
				squares = append(squares, square{chess.Col(f - 'a'), rank})
			}
		}
		return squares
	}

	// [a-d][1-4] pattern
	firstClose := strings.Index(desig, "]")
	if firstClose == -1 {
		return squares
	}
	secondOpen := strings.Index(desig[firstClose:], "[")
	if secondOpen == -1 {
		return squares
	}
	secondOpen += firstClose

	fileRange := desig[1:firstClose]
	rankRange := desig[secondOpen+1 : len(desig)-1]

	files := e.parseRange(fileRange, 'a', 'h')
	ranks := e.parseRange(rankRange, '1', '8')

	for _, f := range files {
		for _, r := range ranks {
			squares = append(squares, square{chess.Col(f - 'a'), chess.Rank(r - '1')})
		}
	}

	return squares
}

func (e *Evaluator) parseRange(rangeStr string, min, max byte) []byte {
	var result []byte

	if strings.Contains(rangeStr, "-") {
		parts := strings.Split(rangeStr, "-")
		if len(parts) == 2 && len(parts[0]) == 1 && len(parts[1]) == 1 {
			start := parts[0][0]
			end := parts[1][0]
			if start >= min && end <= max && start <= end {
				for c := start; c <= end; c++ {
					result = append(result, c)
				}
			}
		}
	} else {
		// Individual characters
		for _, c := range rangeStr {
			if byte(c) >= min && byte(c) <= max {
				result = append(result, byte(c))
			}
		}
	}

	return result
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}
