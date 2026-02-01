package parser

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
)

// Lexer tokenizes PGN input.
type Lexer struct {
	reader   *bufio.Reader
	line     string
	pos      int
	lineNum  uint
	ravLevel uint
	lastMove string
	eof      bool
	cfg      *config.Config

	// Comment nesting depth
	commentDepth uint
}

// Character classification table
var chTab [256]TokenType

// Move character classification table
var moveChars [256]bool

func init() {
	initLexTables()
}

// initLexTables initializes the character classification tables.
func initLexTables() {
	// Initialize all to error
	for i := range chTab {
		chTab[i] = ErrorToken
	}

	// Whitespace
	for _, c := range []byte{' ', '\t', '\r', '\n'} {
		chTab[c] = Whitespace
	}

	// Brackets and quotes
	chTab['['] = TagStart
	chTab[']'] = TagEnd
	chTab['"'] = DoubleQuote
	chTab['{'] = CommentStart
	chTab['}'] = CommentEnd

	// Special symbols
	chTab['$'] = NAGToken
	chTab['!'] = Annotate
	chTab['?'] = Annotate
	chTab['+'] = CheckSymbol
	chTab['#'] = CheckSymbol
	chTab['.'] = Dot
	chTab['('] = RAVStart
	chTab[')'] = RAVEnd
	chTab['%'] = Percent
	chTab['\\'] = Escape
	chTab[0] = EOS
	chTab['*'] = Star
	chTab['-'] = Dash

	// Operators (only allowed in tag files)
	for _, c := range []byte{'<', '>', '='} {
		chTab[c] = Operator
	}

	// Digits
	for c := byte('0'); c <= '9'; c++ {
		chTab[c] = Digit
	}

	// Alpha characters (upper and lowercase)
	for c := byte('A'); c <= 'Z'; c++ {
		chTab[c] = Alpha
		chTab[c+32] = Alpha
	}
	chTab['_'] = Alpha

	// Russian piece letters
	for _, c := range []byte{RussianKnightOrKing, RussianKingSecondLetter, RussianQueen, RussianRook, RussianBishop} {
		chTab[c] = Alpha
	}

	// Initialize move character table
	initMoveChars()
}

// initMoveChars initializes the move character classification table.
func initMoveChars() {
	// Files (a-h) and ranks (1-8)
	for c := byte('a'); c <= 'h'; c++ {
		moveChars[c] = true
	}
	for c := byte('1'); c <= '8'; c++ {
		moveChars[c] = true
	}

	// Piece letters: English (upper/lower), Dutch/German, Russian
	pieceChars := []byte{
		'K', 'Q', 'R', 'N', 'B', 'k', 'q', 'r', 'n', 'b', // English
		'D', 'T', 'S', 'P', 'L', // Dutch/German
		RussianKnightOrKing, RussianKingSecondLetter, RussianQueen, RussianRook, RussianBishop,
	}
	for _, c := range pieceChars {
		moveChars[c] = true
	}

	// Capture/separators, promotion, castling, en passant
	specialChars := []byte{'x', 'X', ':', '-', '=', 'O', 'o', '0', 'p'}
	for _, c := range specialChars {
		moveChars[c] = true
	}
}

// NewLexer creates a new lexer for the given reader.
// If cfg is nil, a default config is created.
func NewLexer(r io.Reader, cfg *config.Config) *Lexer {
	if cfg == nil {
		cfg = config.NewConfig()
	}
	return &Lexer{
		reader:  bufio.NewReader(r),
		lineNum: 0,
		cfg:     cfg,
	}
}

// readLine reads the next line from input.
func (l *Lexer) readLine() bool {
	line, err := l.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && len(line) > 0 {
			l.line = line
			l.pos = 0
			l.lineNum++
			return true
		}
		l.eof = true
		return false
	}
	l.line = line
	l.pos = 0
	l.lineNum++
	return true
}

// currentChar returns the current character or 0 if at end of line.
func (l *Lexer) currentChar() byte {
	if l.pos >= len(l.line) {
		return 0
	}
	return l.line[l.pos]
}

// advance moves to the next character.
func (l *Lexer) advance() {
	if l.pos < len(l.line) {
		l.pos++
	}
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() *Token {
	for {
		token := l.getNextSymbol()
		if token.Type != NoToken {
			token.Line = l.lineNum
			return token
		}
	}
}

// getNextSymbol identifies the next symbol.
func (l *Lexer) getNextSymbol() *Token {
	// Need a new line?
	if l.line == "" || l.pos >= len(l.line) {
		if !l.readLine() {
			return &Token{Type: EOFToken}
		}
		return &Token{Type: NoToken}
	}

	ch := l.currentChar()
	symbolStart := l.pos
	l.advance()

	tokenType := chTab[ch]

	switch tokenType {
	case Whitespace:
		for l.pos < len(l.line) && chTab[l.currentChar()] == Whitespace {
			l.advance()
		}
		return &Token{Type: NoToken}

	case TagStart:
		return l.gatherTag()

	case TagEnd:
		return &Token{Type: NoToken}

	case DoubleQuote:
		return l.gatherString()

	case CommentStart:
		return l.gatherComment()

	case CommentEnd:
		if !l.cfg.SkippingCurrentGame {
			fmt.Fprintf(l.cfg.LogFile, "Unmatched comment end on line %d.\n", l.lineNum)
		}
		return &Token{Type: NoToken}

	case NAGToken:
		// Gather digits after $
		start := l.pos
		for l.pos < len(l.line) && unicode.IsDigit(rune(l.currentChar())) {
			l.advance()
		}
		text := "$" + l.line[start:l.pos]
		return &Token{Type: NAGToken, TokenString: text}

	case Annotate:
		// Gather annotation symbols (!, ?, !!, ??, !?, ?!)
		for l.pos < len(l.line) && chTab[l.currentChar()] == Annotate {
			l.advance()
		}
		text := l.line[symbolStart:l.pos]
		nagStr := annotationToNAG(text)
		return &Token{Type: NAGToken, TokenString: nagStr}

	case CheckSymbol:
		// Allow ++ for double check
		for l.pos < len(l.line) && chTab[l.currentChar()] == CheckSymbol {
			l.advance()
		}
		return &Token{Type: CheckSymbol}

	case Dot:
		// Skip dots
		for l.pos < len(l.line) && chTab[l.currentChar()] == Dot {
			l.advance()
		}
		return &Token{Type: NoToken}

	case RAVStart:
		l.ravLevel++
		return &Token{Type: RAVStart}

	case RAVEnd:
		if l.ravLevel > 0 {
			l.ravLevel--
			return &Token{Type: RAVEnd}
		}
		if !l.cfg.SkippingCurrentGame {
			fmt.Fprintf(l.cfg.LogFile, "Too many ')' found on line %d.\n", l.lineNum)
		}
		return &Token{Type: NoToken}

	case Percent:
		// Skip rest of line (comment)
		l.pos = len(l.line)
		return &Token{Type: NoToken}

	case Escape:
		// Skip next character
		if l.pos < len(l.line) {
			l.advance()
		}
		return &Token{Type: NoToken}

	case Alpha:
		return l.gatherAlpha(ch, symbolStart)

	case Digit:
		return l.gatherNumeric(ch)

	case Star:
		return &Token{Type: TerminatingResult, TokenString: "*"}

	case Dash:
		if l.pos < len(l.line) && chTab[l.currentChar()] == Dash {
			l.advance()
			return l.makeNullMoveToken()
		}
		fmt.Fprintf(l.cfg.LogFile, "Single '-' not allowed on line %d.\n", l.lineNum)
		return &Token{Type: NoToken}

	case EOS:
		// End of string, get next line
		if !l.readLine() {
			return &Token{Type: EOFToken}
		}
		return &Token{Type: NoToken}

	case Operator:
		fmt.Fprintf(l.cfg.LogFile, "Operator in illegal context on line %d.\n", l.lineNum)
		for l.pos < len(l.line) && chTab[l.currentChar()] == Operator {
			l.advance()
		}
		return &Token{Type: NoToken}

	case ErrorToken:
		if !l.cfg.SkippingCurrentGame {
			fmt.Fprintf(l.cfg.LogFile, "Unknown character %c (0x%x) on line %d.\n", ch, ch, l.lineNum)
		}
		for l.pos < len(l.line) && chTab[l.currentChar()] == ErrorToken {
			l.advance()
		}
		return &Token{Type: NoToken}

	default:
		return &Token{Type: NoToken}
	}
}

// gatherTag gathers a tag name after '['.
func (l *Lexer) gatherTag() *Token {
	// Skip whitespace
	for l.pos < len(l.line) && chTab[l.currentChar()] == Whitespace {
		l.advance()
	}

	// Gather tag name
	start := l.pos
	for l.pos < len(l.line) {
		ch := l.currentChar()
		if unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_' {
			l.advance()
		} else {
			break
		}
	}

	if l.pos > start {
		tagName := l.line[start:l.pos]
		tagIndex, ok := chess.StringToTagName[tagName]
		if !ok {
			// New tag - for now just use a high index
			tagIndex = chess.OriginalNumberOfTags
		}
		return &Token{Type: TagToken, TokenString: tagName, TagIndex: int(tagIndex)}
	}
	return &Token{Type: NoToken}
}

// gatherString gathers a quoted string.
func (l *Lexer) gatherString() *Token {
	var sb strings.Builder
	escaped := false

	for l.pos < len(l.line) {
		ch := l.currentChar()
		l.advance()

		if escaped {
			sb.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		if ch == '"' {
			// End of string
			return &Token{Type: StringToken, TokenString: sb.String()}
		}

		sb.WriteByte(ch)
	}

	// String not properly terminated
	if !l.cfg.SkippingCurrentGame {
		fmt.Fprintf(l.cfg.LogFile, "Missing closing quote on line %d.\n", l.lineNum)
	}
	return &Token{Type: StringToken, TokenString: sb.String()}
}

// gatherComment gathers a comment block.
func (l *Lexer) gatherComment() *Token {
	var sb strings.Builder
	l.commentDepth++

	for {
		for l.pos < len(l.line) {
			ch := l.currentChar()
			l.advance()

			switch {
			case ch == '{' && l.cfg.AllowNestedComments:
				l.commentDepth++
				sb.WriteByte(ch)
			case ch == '}':
				if l.cfg.AllowNestedComments && l.commentDepth > 1 {
					l.commentDepth--
					sb.WriteByte(ch)
				} else {
					l.commentDepth--
					return l.makeCommentToken(sb.String())
				}
			default:
				sb.WriteByte(ch)
			}
		}

		if !l.readLine() {
			break
		}
		sb.WriteByte('\n')
	}

	if l.commentDepth > 0 {
		fmt.Fprintf(l.cfg.LogFile, "Missing end of comment.\n")
	}

	return l.makeCommentToken(sb.String())
}

// makeCommentToken creates a comment token from the given text.
func (l *Lexer) makeCommentToken(text string) *Token {
	return &Token{
		Type:     CommentToken,
		Comments: []*chess.Comment{{Text: strings.TrimSpace(text)}},
	}
}

// gatherAlpha handles alpha characters (potential moves).
func (l *Lexer) gatherAlpha(ch byte, symbolStart int) *Token {
	// Check for null move Z0
	if ch == 'Z' && l.pos < len(l.line) && l.currentChar() == '0' {
		l.advance()
		return l.makeNullMoveToken()
	}

	if !moveChars[ch] {
		if !l.cfg.SkippingCurrentGame {
			fmt.Fprintf(l.cfg.LogFile, "Unknown character %c (0x%x) on line %d.\n", ch, ch, l.lineNum)
		}
		return &Token{Type: NoToken}
	}

	// Gather move characters
	for l.pos < len(l.line) && moveChars[l.currentChar()] {
		l.advance()
	}

	moveText := l.line[symbolStart:l.pos]

	if moveSeemValid(moveText) {
		if move := DecodeMove(moveText); move != nil {
			l.lastMove = moveText
			return &Token{Type: MoveToken, MoveDetails: move}
		}
	}

	if !l.cfg.SkippingCurrentGame {
		fmt.Fprintf(l.cfg.LogFile, "Unknown move text %s on line %d.\n", moveText, l.lineNum)
	}
	return &Token{Type: NoToken}
}

// makeNullMoveToken creates a token for a null move.
func (l *Lexer) makeNullMoveToken() *Token {
	move := chess.NewMove()
	move.Text = chess.NullMoveString
	move.Class = chess.NullMove
	l.lastMove = chess.NullMoveString
	return &Token{Type: MoveToken, MoveDetails: move}
}

// gatherNumeric handles numeric tokens (move numbers, results, castling).
func (l *Lexer) gatherNumeric(initialDigit byte) *Token {
	remaining := l.line[l.pos:]

	switch initialDigit {
	case '0':
		// Could be 0-1 (result) or 0-0 / 0-0-0 (castling)
		if strings.HasPrefix(remaining, "-1") {
			l.pos += 2
			return &Token{Type: TerminatingResult, TokenString: "0-1"}
		}
		if strings.HasPrefix(remaining, "-0-0") {
			l.pos += 4
			return l.makeCastleToken("O-O-O", chess.QueensideCastle)
		}
		if strings.HasPrefix(remaining, "-0") {
			l.pos += 2
			return l.makeCastleToken("O-O", chess.KingsideCastle)
		}
	case '1':
		if strings.HasPrefix(remaining, "-0") {
			l.pos += 2
			return &Token{Type: TerminatingResult, TokenString: "1-0"}
		}
		if strings.HasPrefix(remaining, "/2") {
			l.pos += 2
			if strings.HasPrefix(l.line[l.pos:], "-1/2") {
				l.pos += 4
			}
			return &Token{Type: TerminatingResult, TokenString: "1/2-1/2"}
		}
	}

	return l.gatherMoveNumber(initialDigit)
}

// makeCastleToken creates a castling move token.
func (l *Lexer) makeCastleToken(text string, class chess.MoveClass) *Token {
	move := chess.NewMove()
	move.Text = text
	move.Class = class
	move.PieceToMove = chess.King
	l.lastMove = text
	return &Token{Type: MoveToken, MoveDetails: move}
}

// gatherMoveNumber parses a move number token.
func (l *Lexer) gatherMoveNumber(initialDigit byte) *Token {
	start := l.pos - 1
	for l.pos < len(l.line) && unicode.IsDigit(rune(l.currentChar())) {
		l.advance()
	}

	// Skip trailing dots
	for l.pos < len(l.line) && l.currentChar() == '.' {
		l.advance()
	}

	numStr := strings.TrimRight(l.line[start:l.pos], ".")
	var moveNum uint
	fmt.Sscanf(numStr, "%d", &moveNum) //nolint:gosec // G104: default 0 is acceptable

	return &Token{Type: MoveNumber, MoveNum: moveNum}
}

// annotationToNAG converts annotation symbols to NAG strings.
func annotationToNAG(text string) string {
	switch text {
	case "!":
		return "$1"
	case "?":
		return "$2"
	case "!!":
		return "$3"
	case "??":
		return "$4"
	case "!?":
		return "$5"
	case "?!":
		return "$6"
	default:
		return "$0"
	}
}

// moveSeemValid does a basic check if the move text looks valid.
func moveSeemValid(text string) bool {
	if len(text) < 2 {
		return false
	}

	// Castling
	if text == "O-O" || text == "O-O-O" || text == "o-o" || text == "o-o-o" ||
		text == "0-0" || text == "0-0-0" {
		return true
	}

	// Must contain at least one file (a-h) and one rank (1-8)
	hasFile := false
	hasRank := false
	for _, c := range text {
		if c >= 'a' && c <= 'h' {
			hasFile = true
		}
		if c >= '1' && c <= '8' {
			hasRank = true
		}
	}

	return hasFile && hasRank
}

// RestartForNewGame resets lexer state for a new game.
func (l *Lexer) RestartForNewGame() {
	l.lastMove = ""
	l.ravLevel = 0
}

// LineNumber returns the current line number.
func (l *Lexer) LineNumber() uint {
	return l.lineNum
}

// RAVLevel returns the current RAV nesting level.
func (l *Lexer) RAVLevel() uint {
	return l.ravLevel
}
