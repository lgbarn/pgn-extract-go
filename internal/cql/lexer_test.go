// Package cql implements CQL (Chess Query Language) parsing and evaluation.
package cql

import (
	"testing"
)

func TestLexerBasicTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{"", []TokenType{EOF}},
		{"()", []TokenType{LPAREN, RPAREN, EOF}},
		{"piece", []TokenType{IDENT, EOF}},
		{"piece K e1", []TokenType{IDENT, PIECE, SQUARE, EOF}},
		{"(and x y)", []TokenType{LPAREN, IDENT, IDENT, IDENT, RPAREN, EOF}},
		{"mate", []TokenType{IDENT, EOF}},
		{"wtm btm", []TokenType{IDENT, IDENT, EOF}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			for i, expected := range tt.expected {
				tok := lexer.NextToken()
				if tok.Type != expected {
					t.Errorf("token %d: expected %v, got %v (literal: %q)", i, expected, tok.Type, tok.Literal)
				}
			}
		})
	}
}

func TestLexerPieceDesignators(t *testing.T) {
	// Individual white pieces
	whitePieces := []string{"K", "Q", "R", "B", "N", "P"}
	for _, p := range whitePieces {
		lexer := NewLexer(p)
		tok := lexer.NextToken()
		if tok.Type != PIECE {
			t.Errorf("expected PIECE for %q, got %v", p, tok.Type)
		}
		if tok.Literal != p {
			t.Errorf("expected literal %q, got %q", p, tok.Literal)
		}
	}

	// Individual black pieces
	blackPieces := []string{"k", "q", "r", "b", "n", "p"}
	for _, p := range blackPieces {
		lexer := NewLexer(p)
		tok := lexer.NextToken()
		if tok.Type != PIECE {
			t.Errorf("expected PIECE for %q, got %v", p, tok.Type)
		}
	}

	// Special designators
	specials := []string{"A", "a", "_", "?"}
	for _, s := range specials {
		lexer := NewLexer(s)
		tok := lexer.NextToken()
		if tok.Type != PIECE {
			t.Errorf("expected PIECE for %q, got %v", s, tok.Type)
		}
	}
}

func TestLexerPieceSets(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"[RQ]", "[RQ]"},
		{"[RBN]", "[RBN]"},
		{"[rq]", "[rq]"},
		{"[KQRBNP]", "[KQRBNP]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tok := lexer.NextToken()
			if tok.Type != PIECESET {
				t.Errorf("expected PIECESET for %q, got %v", tt.input, tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tok.Literal)
			}
		})
	}
}

func TestLexerSquares(t *testing.T) {
	// Test all 64 squares
	files := "abcdefgh"
	ranks := "12345678"

	for _, f := range files {
		for _, r := range ranks {
			sq := string(f) + string(r)
			lexer := NewLexer(sq)
			tok := lexer.NextToken()
			if tok.Type != SQUARE {
				t.Errorf("expected SQUARE for %q, got %v", sq, tok.Type)
			}
			if tok.Literal != sq {
				t.Errorf("expected literal %q, got %q", sq, tok.Literal)
			}
		}
	}
}

func TestLexerSquareRanges(t *testing.T) {
	tests := []struct {
		input   string
		tokType TokenType
	}{
		{"[a-h]1", SQUARESET},
		{"a[1-8]", SQUARESET},
		{"[a-d][1-4]", SQUARESET},
		{".", SQUARE}, // Any square
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tok := lexer.NextToken()
			if tok.Type != tt.tokType {
				t.Errorf("expected %v for %q, got %v", tt.tokType, tt.input, tok.Type)
			}
		})
	}
}

func TestLexerNumbers(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"0", "0"},
		{"1", "1"},
		{"42", "42"},
		{"100", "100"},
		{"2500", "2500"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tok := lexer.NextToken()
			if tok.Type != NUMBER {
				t.Errorf("expected NUMBER for %q, got %v", tt.input, tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tok.Literal)
			}
		})
	}
}

func TestLexerStrings(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{`"Carlsen"`, "Carlsen"},
		{`"Fischer"`, "Fischer"},
		{`"1-0"`, "1-0"},
		{`"hello world"`, "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tok := lexer.NextToken()
			if tok.Type != STRING {
				t.Errorf("expected STRING for %q, got %v", tt.input, tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tok.Literal)
			}
		})
	}
}

func TestLexerOperators(t *testing.T) {
	tests := []struct {
		input   string
		tokType TokenType
	}{
		{"<", LT},
		{">", GT},
		{"<=", LE},
		{">=", GE},
		{"==", EQ},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tok := lexer.NextToken()
			if tok.Type != tt.tokType {
				t.Errorf("expected %v for %q, got %v", tt.tokType, tt.input, tok.Type)
			}
		})
	}
}

func TestLexerComplexExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{
			"(and (piece K e1) (piece k e8))",
			[]TokenType{LPAREN, IDENT, LPAREN, IDENT, PIECE, SQUARE, RPAREN, LPAREN, IDENT, PIECE, SQUARE, RPAREN, RPAREN, EOF},
		},
		{
			"(or mate stalemate)",
			[]TokenType{LPAREN, IDENT, IDENT, IDENT, RPAREN, EOF},
		},
		{
			"(> (count P) 3)",
			[]TokenType{LPAREN, GT, LPAREN, IDENT, PIECE, RPAREN, NUMBER, RPAREN, EOF},
		},
		{
			`player "Carlsen"`,
			[]TokenType{IDENT, STRING, EOF},
		},
		{
			"attack [RQ] k",
			[]TokenType{IDENT, PIECESET, PIECE, EOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			for i, expected := range tt.expected {
				tok := lexer.NextToken()
				if tok.Type != expected {
					t.Errorf("token %d: expected %v, got %v (literal: %q)", i, expected, tok.Type, tok.Literal)
				}
			}
		})
	}
}

func TestLexerWhitespace(t *testing.T) {
	// Various whitespace should be skipped
	inputs := []string{
		"piece  K  e1",   // extra spaces
		"piece\tK\te1",   // tabs
		"piece\nK\ne1",   // newlines
		"  piece K e1  ", // leading/trailing
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			lexer := NewLexer(input)
			expected := []TokenType{IDENT, PIECE, SQUARE, EOF}
			for i, exp := range expected {
				tok := lexer.NextToken()
				if tok.Type != exp {
					t.Errorf("token %d: expected %v, got %v", i, exp, tok.Type)
				}
			}
		})
	}
}
