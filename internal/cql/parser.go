package cql

import (
	"fmt"
	"strconv"

	"github.com/lgbarn/pgn-extract-go/internal/errors"
)

// Parser parses CQL expressions into an AST.
type Parser struct {
	lexer   *Lexer
	current Token
	peek    Token
}

// NewParser creates a new parser for the given input.
func NewParser(input string) *Parser {
	p := &Parser{lexer: NewLexer(input)}
	// Read two tokens to initialize current and peek
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.current = p.peek
	p.peek = p.lexer.NextToken()
}

// Parse parses a CQL expression and returns the AST.
func Parse(input string) (Node, error) {
	parser := NewParser(input)
	return parser.ParseExpression()
}

// ParseExpression parses the complete expression.
func (p *Parser) ParseExpression() (Node, error) {
	nodes, err := p.parseExpressionList()
	if err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("empty expression: %w", errors.ErrCQLSyntax)
	}

	// If single node, return it directly
	if len(nodes) == 1 {
		return nodes[0], nil
	}

	// Multiple top-level expressions → implicit AND
	return &LogicalNode{
		Op:       "and",
		Children: nodes,
	}, nil
}

func (p *Parser) parseExpressionList() ([]Node, error) {
	var nodes []Node

	for p.current.Type != EOF && p.current.Type != RPAREN {
		node, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (p *Parser) parsePrimary() (Node, error) {
	switch p.current.Type {
	case LPAREN:
		return p.parseParenExpr()
	case IDENT:
		return p.parseFilter()
	case PIECE, PIECESET:
		node := &PieceNode{Designator: p.current.Literal}
		p.nextToken()
		return node, nil
	case SQUARE, SQUARESET:
		node := &SquareNode{Designator: p.current.Literal}
		p.nextToken()
		return node, nil
	case NUMBER:
		val, err := strconv.Atoi(p.current.Literal)
		if err != nil {
			return nil, fmt.Errorf("invalid number: %s: %w", p.current.Literal, errors.ErrCQLSyntax)
		}
		node := &NumberNode{Value: val}
		p.nextToken()
		return node, nil
	case STRING:
		node := &StringNode{Value: p.current.Literal}
		p.nextToken()
		return node, nil
	case LT, GT, LE, GE, EQ:
		return p.parseComparison()
	default:
		return nil, fmt.Errorf("unexpected token: %v (%q): %w", p.current.Type, p.current.Literal, errors.ErrCQLSyntax)
	}
}

func (p *Parser) parseParenExpr() (Node, error) {
	// Skip '('
	p.nextToken()

	// Check for logical operators or comparisons
	switch p.current.Type {
	case IDENT:
		switch p.current.Literal {
		case "and", "or", "not":
			return p.parseLogical()
		default:
			return p.parseParenFilter()
		}
	case LT, GT, LE, GE, EQ:
		return p.parseComparison()
	default:
		return nil, fmt.Errorf("unexpected token after '(': %v: %w", p.current.Type, errors.ErrCQLSyntax)
	}
}

func (p *Parser) parseLogical() (Node, error) {
	op := p.current.Literal
	p.nextToken()

	var children []Node
	for p.current.Type != RPAREN && p.current.Type != EOF {
		child, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}

	if p.current.Type != RPAREN {
		return nil, fmt.Errorf("expected ')', got %v: %w", p.current.Type, errors.ErrCQLSyntax)
	}
	p.nextToken() // Skip ')'

	if len(children) == 0 {
		return nil, fmt.Errorf("logical operator %q requires at least one operand: %w", op, errors.ErrCQLSyntax)
	}

	return &LogicalNode{
		Op:       op,
		Children: children,
	}, nil
}

func (p *Parser) parseParenFilter() (Node, error) {
	// Parse filter inside parentheses
	filter, err := p.parseFilter()
	if err != nil {
		return nil, err
	}

	if p.current.Type != RPAREN {
		return nil, fmt.Errorf("expected ')', got %v: %w", p.current.Type, errors.ErrCQLSyntax)
	}
	p.nextToken() // Skip ')'

	return filter, nil
}

func (p *Parser) parseFilter() (Node, error) {
	name := p.current.Literal
	p.nextToken()

	// Zero-argument filters
	if isZeroArgFilter(name) {
		return &FilterNode{Name: name, Args: nil}, nil
	}

	// Collect arguments until we hit EOF, RPAREN, or another filter
	var args []Node
	expectedArgs := filterArgCount(name)

	for {
		// Stop if we hit end of input, close paren
		if p.current.Type == EOF || p.current.Type == RPAREN {
			break
		}

		// Stop if we've collected expected number of arguments
		if expectedArgs > 0 && len(args) >= expectedArgs {
			break
		}

		// Check if this looks like another top-level filter
		if p.current.Type == IDENT && isFilterName(p.current.Literal) {
			// This is the start of another filter, not an argument
			break
		}

		// Check if this is a logical operator starting a new expression
		if p.current.Type == LPAREN {
			// Peek inside - if it's a logical op, it's a new expression
			if p.peek.Type == IDENT && (p.peek.Literal == "and" || p.peek.Literal == "or" || p.peek.Literal == "not") {
				break
			}
			if p.peek.Type == LT || p.peek.Type == GT || p.peek.Type == LE || p.peek.Type == GE || p.peek.Type == EQ {
				break
			}
		}

		arg, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	return &FilterNode{
		Name: name,
		Args: args,
	}, nil
}

func (p *Parser) parseComparison() (Node, error) {
	op := p.current.Literal
	p.nextToken()

	left, err := p.parsePrimary()
	if err != nil {
		return nil, fmt.Errorf("expected left operand: %w", err)
	}

	right, err := p.parsePrimary()
	if err != nil {
		return nil, fmt.Errorf("expected right operand: %w", err)
	}

	if p.current.Type != RPAREN {
		return nil, fmt.Errorf("expected ')', got %v: %w", p.current.Type, errors.ErrCQLSyntax)
	}
	p.nextToken() // Skip ')'

	return &ComparisonNode{
		Op:    op,
		Left:  left,
		Right: right,
	}, nil
}

// isFilterName returns true if the identifier is a known CQL filter name.
func isFilterName(name string) bool {
	filters := map[string]bool{
		"piece":           true,
		"attack":          true,
		"check":           true,
		"mate":            true,
		"stalemate":       true,
		"wtm":             true,
		"btm":             true,
		"count":           true,
		"material":        true,
		"result":          true,
		"player":          true,
		"elo":             true,
		"year":            true,
		"pin":             true,
		"ray":             true,
		"between":         true,
		"flip":            true,
		"flipvertical":    true,
		"flipcolor":       true,
		"shift":           true,
		"shifthorizontal": true,
		"shiftvertical":   true,
		"controls":        true,
		"power":           true,
		// Direction keywords for ray
		"horizontal": true,
		"vertical":   true,
		"diagonal":   true,
		"orthogonal": true,
		// Color keywords for elo
		"white": true,
		"black": true,
	}
	return filters[name]
}

// isZeroArgFilter returns true if the filter takes no arguments.
func isZeroArgFilter(name string) bool {
	zeroArg := map[string]bool{
		"check":     true,
		"mate":      true,
		"stalemate": true,
		"wtm":       true,
		"btm":       true,
		"year":      true,
		// Direction keywords are zero-arg identifiers used as arguments
		"horizontal": true,
		"vertical":   true,
		"diagonal":   true,
		"orthogonal": true,
		"white":      true,
		"black":      true,
	}
	return zeroArg[name]
}

// filterArgCount returns the expected number of arguments for a filter.
// Returns -1 for variable argument filters.
func filterArgCount(name string) int {
	counts := map[string]int{
		"piece":           2, // piece <designator> <square>
		"attack":          2, // attack <piece> <square>
		"count":           1, // count <designator>
		"material":        1, // material <color>
		"result":          1, // result <value>
		"player":          1, // player <name>
		"elo":             3, // elo <color> <op> <value>
		"year":            2, // year <op> <value>
		"pin":             3, // pin <piece> <through> <to>
		"ray":             4, // ray <dir> <from> <through> <to>
		"between":         2, // between <sq1> <sq2>
		"flip":            1, // flip <expr>
		"flipvertical":    1, // flipvertical <expr>
		"flipcolor":       1, // flipcolor <expr>
		"shift":           1, // shift <expr>
		"shifthorizontal": 1, // shifthorizontal <expr>
		"shiftvertical":   1, // shiftvertical <expr>
		"controls":        2, // controls <piece> <square>
		"power":           2, // power <piece> <op>
	}
	if c, ok := counts[name]; ok {
		return c
	}
	return -1 // Variable or unknown
}
