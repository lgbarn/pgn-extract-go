// Package cql implements CQL (Chess Query Language) parsing and evaluation.
package cql

import (
	"strings"
	"unicode"
)

// TokenType represents the type of a lexical token.
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF

	// Delimiters
	LPAREN // (
	RPAREN // )

	// Literals
	IDENT    // and, or, piece, attack, mate, etc.
	NUMBER   // 0, 1, 42, 2500
	STRING   // "Carlsen"
	PIECE    // K, Q, R, B, N, P, k, q, r, b, n, p, A, a, _, ?
	PIECESET // [RQ], [RBN], etc.
	SQUARE   // a1, e4, h8, .
	SQUARESET // [a-h]1, a[1-8], [a-d][1-4]

	// Operators
	LT // <
	GT // >
	LE // <=
	GE // >=
	EQ // ==
)

var tokenNames = map[TokenType]string{
	ILLEGAL:   "ILLEGAL",
	EOF:       "EOF",
	LPAREN:    "LPAREN",
	RPAREN:    "RPAREN",
	IDENT:     "IDENT",
	NUMBER:    "NUMBER",
	STRING:    "STRING",
	PIECE:     "PIECE",
	PIECESET:  "PIECESET",
	SQUARE:    "SQUARE",
	SQUARESET: "SQUARESET",
	LT:        "LT",
	GT:        "GT",
	LE:        "LE",
	GE:        "GE",
	EQ:        "EQ",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}

// Token represents a lexical token.
type Token struct {
	Type    TokenType
	Literal string
	Pos     int // Position in input
}

// Lexer tokenizes CQL expressions.
type Lexer struct {
	input   string
	pos     int  // current position in input
	readPos int  // next reading position
	ch      byte // current character
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	var tok Token
	tok.Pos = l.pos

	switch l.ch {
	case 0:
		tok.Type = EOF
		tok.Literal = ""
	case '(':
		tok.Type = LPAREN
		tok.Literal = "("
		l.readChar()
	case ')':
		tok.Type = RPAREN
		tok.Literal = ")"
		l.readChar()
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = LE
			tok.Literal = "<="
		} else {
			tok.Type = LT
			tok.Literal = "<"
		}
		l.readChar()
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = GE
			tok.Literal = ">="
		} else {
			tok.Type = GT
			tok.Literal = ">"
		}
		l.readChar()
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = EQ
			tok.Literal = "=="
			l.readChar()
		} else {
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
			l.readChar()
		}
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
	case '[':
		// Could be piece set [RQ] or square range [a-h]
		tok = l.readBracketExpr()
	case '.':
		// Any square
		tok.Type = SQUARE
		tok.Literal = "."
		l.readChar()
	case '_', '?':
		// Special piece designators
		tok.Type = PIECE
		tok.Literal = string(l.ch)
		l.readChar()
	default:
		if isDigit(l.ch) {
			tok.Type = NUMBER
			tok.Literal = l.readNumber()
		} else if isLetter(l.ch) {
			tok = l.readIdentOrPieceOrSquare()
		} else {
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
			l.readChar()
		}
	}

	return tok
}

func (l *Lexer) readString() string {
	// Skip opening quote
	l.readChar()

	start := l.pos
	for l.ch != '"' && l.ch != 0 {
		l.readChar()
	}

	str := l.input[start:l.pos]

	// Skip closing quote
	if l.ch == '"' {
		l.readChar()
	}

	return str
}

func (l *Lexer) readNumber() string {
	start := l.pos
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readBracketExpr() Token {
	start := l.pos

	// Skip '['
	l.readChar()

	// Read until closing ']'
	for l.ch != ']' && l.ch != 0 {
		l.readChar()
	}

	// Include the ']'
	if l.ch == ']' {
		l.readChar()
	}

	content := l.input[start:l.pos]

	// Check if this could be followed by more square range notation
	// e.g., [a-h]1 or [a-d][1-4]
	if l.ch != 0 && (isDigit(l.ch) || l.ch == '[') {
		// Read the rest of the square range
		for l.ch != 0 && !isWhitespace(l.ch) && l.ch != '(' && l.ch != ')' {
			l.readChar()
		}
		return Token{Type: SQUARESET, Literal: l.input[start:l.pos]}
	}

	// Determine if it's a piece set or square set
	inner := content[1 : len(content)-1] // Remove brackets
	if isPieceSetContent(inner) {
		return Token{Type: PIECESET, Literal: content}
	}

	return Token{Type: SQUARESET, Literal: content}
}

func (l *Lexer) readIdentOrPieceOrSquare() Token {
	start := l.pos

	// Read first character
	firstChar := l.ch
	l.readChar()

	// Check if this is a file letter followed by [ (square range like a[1-8], e[1-8])
	// This check must come first because e, f, g, h are file letters but NOT piece chars
	if isFile(firstChar) && l.ch == '[' {
		// Read the range part: a[1-8]
		for l.ch != 0 && !isWhitespace(l.ch) && l.ch != '(' && l.ch != ')' {
			l.readChar()
		}
		return Token{Type: SQUARESET, Literal: l.input[start:l.pos]}
	}

	// Check if it forms a square (e.g., e1) - file letters can form squares
	if isFile(firstChar) && isRank(l.ch) {
		// It's a square like a1, e4
		l.readChar()
		literal := l.input[start:l.pos]

		// Check if followed by square range notation
		if l.ch == '[' {
			// Read the range part: a[1-8]
			for l.ch != 0 && !isWhitespace(l.ch) && l.ch != '(' && l.ch != ')' {
				l.readChar()
			}
			return Token{Type: SQUARESET, Literal: l.input[start:l.pos]}
		}

		return Token{Type: SQUARE, Literal: literal}
	}

	// Check if it's a single piece designator followed by non-letter
	// K, Q, R, B, N, P (white), k, q, r, b, n, p (black), A, a, _
	if isPieceChar(firstChar) {
		// Single piece designator
		if !isLetter(l.ch) && !isDigit(l.ch) {
			return Token{Type: PIECE, Literal: string(firstChar)}
		}
	}

	// Check for special single-char piece designators
	if (firstChar == '_' || firstChar == '?') && !isLetter(l.ch) {
		return Token{Type: PIECE, Literal: string(firstChar)}
	}

	// Read the rest as identifier
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '-' || l.ch == '_' {
		l.readChar()
	}

	literal := l.input[start:l.pos]

	// Check if it could be a square
	if len(literal) == 2 && isFile(literal[0]) && isRank(literal[1]) {
		return Token{Type: SQUARE, Literal: literal}
	}

	// Check if it's a single piece char that we read as part of identifier
	if len(literal) == 1 && isPieceChar(literal[0]) {
		return Token{Type: PIECE, Literal: literal}
	}

	return Token{Type: IDENT, Literal: literal}
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch))
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isFile(ch byte) bool {
	return ch >= 'a' && ch <= 'h'
}

func isRank(ch byte) bool {
	return ch >= '1' && ch <= '8'
}

func isPieceChar(ch byte) bool {
	return strings.ContainsRune("KQRBNPkqrbnpAa_?", rune(ch))
}

func isPieceSetContent(s string) bool {
	// Piece set contains only piece chars like "RQ", "RBN", "kqrbnp"
	// Square range contains a dash like "a-h" or "1-8"
	if strings.Contains(s, "-") {
		return false
	}
	for _, ch := range s {
		if !strings.ContainsRune("KQRBNPkqrbnpAa", ch) {
			return false
		}
	}
	return len(s) > 0
}
