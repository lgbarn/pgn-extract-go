// Package parser provides PGN lexing and parsing functionality.
package parser

import "github.com/lgbarn/pgn-extract-go/internal/chess"

// TokenType represents the type of a lexical token.
type TokenType int

const (
	// Tokens returned to the parser
	EOFToken TokenType = iota
	TagToken
	StringToken
	CommentToken
	NAGToken
	CheckSymbol
	MoveNumber
	RAVStart
	RAVEnd
	MoveToken
	TerminatingResult

	// Internal tokens used for identification
	Whitespace
	TagStart
	TagEnd
	DoubleQuote
	CommentStart
	CommentEnd
	Annotate
	Dot
	Percent
	Escape
	Alpha
	Digit
	Star
	Dash
	EOS
	Operator
	NoToken
	ErrorToken
)

// tokenTypeNames maps token types to their string representations.
var tokenTypeNames = [...]string{
	EOFToken:          "EOF",
	TagToken:          "TAG",
	StringToken:       "STRING",
	CommentToken:      "COMMENT",
	NAGToken:          "NAG",
	CheckSymbol:       "CHECK_SYMBOL",
	MoveNumber:        "MOVE_NUMBER",
	RAVStart:          "RAV_START",
	RAVEnd:            "RAV_END",
	MoveToken:         "MOVE",
	TerminatingResult: "TERMINATING_RESULT",
	Whitespace:        "WHITESPACE",
	TagStart:          "TAG_START",
	TagEnd:            "TAG_END",
	DoubleQuote:       "DOUBLE_QUOTE",
	CommentStart:      "COMMENT_START",
	CommentEnd:        "COMMENT_END",
	Annotate:          "ANNOTATE",
	Dot:               "DOT",
	Percent:           "PERCENT",
	Escape:            "ESCAPE",
	Alpha:             "ALPHA",
	Digit:             "DIGIT",
	Star:              "STAR",
	Dash:              "DASH",
	EOS:               "EOS",
	Operator:          "OPERATOR",
	NoToken:           "NO_TOKEN",
	ErrorToken:        "ERROR_TOKEN",
}

// String returns the string representation of a token type.
func (t TokenType) String() string {
	if int(t) < len(tokenTypeNames) {
		return tokenTypeNames[t]
	}
	return "UNKNOWN"
}

// Token represents a lexical token with its value.
type Token struct {
	Type TokenType

	// TokenString is used for tag names, results, NAGs
	TokenString string

	// MoveDetails holds parsed move information
	MoveDetails *chess.Move

	// MoveNum holds move numbers
	MoveNum uint

	// Comments holds comment text
	Comments []*chess.Comment

	// TagIndex is an index into the tag list
	TagIndex int

	// Line and column for error reporting
	Line   uint
	Column uint
}

// NewToken creates a new token of the given type.
func NewToken(tokenType TokenType) *Token {
	return &Token{Type: tokenType}
}

// Russian piece letter constants (for international support).
const (
	RussianKnightOrKing     = 0xcb // King and Knight
	RussianKingSecondLetter = 0xf0 // King (second character)
	RussianQueen            = 0xc6 // Queen
	RussianRook             = 0xcc // Rook
	RussianBishop           = 0xd3 // Bishop
)
