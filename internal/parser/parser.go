package parser

import (
	"fmt"
	"io"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
)

// Parser parses PGN input into Game structures.
type Parser struct {
	lexer        *Lexer
	currentToken *Token
	ravLevel     uint
	cfg          *config.Config
}

// NewParser creates a new parser for the given reader.
func NewParser(r io.Reader, cfg *config.Config) *Parser {
	if cfg == nil {
		cfg = config.GlobalConfig
	}
	return &Parser{
		lexer: NewLexer(r, cfg),
		cfg:   cfg,
	}
}

// nextToken gets the next token from the lexer.
func (p *Parser) nextToken() {
	p.currentToken = p.lexer.NextToken()
}

// ParseGame parses a single game from the input.
// Returns nil if no more games are available.
func (p *Parser) ParseGame() (*chess.Game, error) {
	// Get first token if we haven't yet
	if p.currentToken == nil {
		p.nextToken()
	}

	// Skip to next game
	p.skipToNextGame()

	// Skip any prefix comments between games
	p.parseOptCommentList()

	game := chess.NewGame()
	game.StartLine = p.lexer.LineNumber()

	// Parse tags
	p.parseOptTagList(game)

	// Skip any initial NAGs (non-standard but sometimes present)
	for p.currentToken.Type == NAGToken {
		p.nextToken()
	}

	// Parse moves
	game.Moves = p.parseMoveList()

	// Handle any trailing comment
	trailingComments := p.parseOptCommentList()

	// Parse result
	result := p.parseResult()
	game.EndLine = p.lexer.LineNumber()

	if game.Moves != nil {
		// Attach trailing comment and result to last move
		lastMove := game.LastMove()
		if lastMove != nil {
			for _, c := range trailingComments {
				lastMove.Comments = append(lastMove.Comments, c)
			}
			if result != "" {
				lastMove.TerminatingResult = result
			}
		}
	}

	// Store result in tags if not present
	if result != "" {
		if game.GetTag("Result") == "" || game.GetTag("Result") == "?" {
			game.SetTag("Result", result)
		}
	}

	// Check if we got anything
	if p.currentToken.Type == EOFToken && game.Moves == nil && len(game.Tags) == 0 {
		return nil, nil
	}

	return game, nil
}

// skipToNextGame skips tokens until the start of a game is found.
func (p *Parser) skipToNextGame() {
	for {
		switch p.currentToken.Type {
		case EOFToken, TagToken, MoveToken, TerminatingResult:
			return
		default:
			p.nextToken()
		}
	}
}

// parseOptTagList parses zero or more tags.
func (p *Parser) parseOptTagList(game *chess.Game) {
	for p.parseTag(game) {
		// Continue parsing tags
	}

	// Parse any prefix comment
	comments := p.parseOptCommentList()
	game.PrefixComment = comments
}

// parseTag parses a single tag.
func (p *Parser) parseTag(game *chess.Game) bool {
	if p.currentToken.Type == TagToken {
		tagName := p.currentToken.TokenString
		p.nextToken()

		if p.currentToken.Type == StringToken {
			tagValue := p.currentToken.TokenString
			game.SetTag(tagName, tagValue)
			p.nextToken()
		} else {
			fmt.Fprintf(p.cfg.LogFile, "Missing tag string for %s.\n", tagName)
		}
		return true
	}

	if p.currentToken.Type == StringToken {
		fmt.Fprintf(p.cfg.LogFile, "Missing tag name for %s.\n", p.currentToken.TokenString)
		p.nextToken()
		return true
	}

	return false
}

// parseMoveList parses a list of moves.
func (p *Parser) parseMoveList() *chess.Move {
	var head, tail *chess.Move

	move := p.parseMoveAndVariants()
	if move != nil {
		head = move
		tail = move

		for {
			nextMove := p.parseMoveAndVariants()
			if nextMove == nil {
				break
			}
			tail.Next = nextMove
			nextMove.Prev = tail
			tail = nextMove
		}
	}

	return head
}

// parseMoveAndVariants parses a move with its variations.
func (p *Parser) parseMoveAndVariants() *chess.Move {
	move := p.parseMove()
	if move != nil {
		// Parse variations
		move.Variations = p.parseOptVariantList()

		// Parse any trailing comments
		comments := p.parseOptCommentList()
		for _, c := range comments {
			move.Comments = append(move.Comments, c)
		}
	}
	return move
}

// parseMove parses a single move.
func (p *Parser) parseMove() *chess.Move {
	// Skip optional move number
	p.parseOptMoveNumber()

	// Parse the actual move
	move := p.parseMoveUnit()
	if move != nil {
		// Parse NAGs
		p.parseOptNAGList(move)
	}
	return move
}

// parseMoveUnit parses the move itself.
func (p *Parser) parseMoveUnit() *chess.Move {
	if p.currentToken.Type == MoveToken {
		move := p.currentToken.MoveDetails
		p.nextToken()

		// Handle check symbol
		if p.currentToken.Type == CheckSymbol {
			move.Text += "+"
			p.nextToken()
			// Sometimes + is followed by #
			if p.currentToken.Type == CheckSymbol {
				p.nextToken()
			}
		}

		// Check for null move restriction
		if move.Class == chess.NullMove && p.ravLevel == 0 {
			if !p.cfg.AllowNullMoves {
				fmt.Fprintf(p.cfg.LogFile, "Null moves (--) only allowed in variations.\n")
			}
		}

		// Parse comments after the move
		move.Comments = p.parseOptCommentList()

		return move
	}
	return nil
}

// parseOptCommentList parses zero or more comments.
func (p *Parser) parseOptCommentList() []*chess.Comment {
	var comments []*chess.Comment

	for p.currentToken.Type == CommentToken {
		if p.currentToken.Comments != nil {
			comments = append(comments, p.currentToken.Comments...)
		}
		p.nextToken()
	}

	return comments
}

// parseOptMoveNumber parses an optional move number.
func (p *Parser) parseOptMoveNumber() bool {
	if p.currentToken.Type == MoveNumber {
		p.nextToken()
		return true
	}
	return false
}

// parseOptNAGList parses zero or more NAGs.
func (p *Parser) parseOptNAGList(move *chess.Move) {
	for p.currentToken.Type == NAGToken {
		nag := &chess.NAG{
			Text: []string{p.currentToken.TokenString},
		}
		p.nextToken()

		// Gather multiple consecutive NAGs
		for p.currentToken.Type == NAGToken {
			nag.Text = append(nag.Text, p.currentToken.TokenString)
			p.nextToken()
		}

		// Parse any comments following the NAGs
		comments := p.parseOptCommentList()
		for _, c := range comments {
			nag.Comments = append(nag.Comments, c)
		}

		move.NAGs = append(move.NAGs, nag)
	}
}

// parseOptVariantList parses zero or more variations.
func (p *Parser) parseOptVariantList() []*chess.Variation {
	var variations []*chess.Variation

	for {
		variation := p.parseVariant()
		if variation == nil {
			break
		}
		variations = append(variations, variation)
	}

	return variations
}

// parseVariant parses a single variation.
func (p *Parser) parseVariant() *chess.Variation {
	if p.currentToken.Type != RAVStart {
		return nil
	}

	p.ravLevel++
	p.nextToken()

	variation := &chess.Variation{}

	// Parse prefix comment
	variation.PrefixComment = p.parseOptCommentList()

	// Parse moves in variation
	variation.Moves = p.parseMoveList()

	if variation.Moves == nil {
		fmt.Fprintf(p.cfg.LogFile, "Missing move list in variation.\n")
	}

	// Parse result in variation
	result := p.parseResult()
	if result != "" && variation.Moves != nil {
		// Find last move and attach result
		lastMove := variation.Moves
		for lastMove.Next != nil {
			lastMove = lastMove.Next
		}
		lastMove.TerminatingResult = result

		// Handle trailing comment
		trailingComment := p.parseOptCommentList()
		for _, c := range trailingComment {
			lastMove.Comments = append(lastMove.Comments, c)
		}
	}

	// Expect RAV_END
	if p.currentToken.Type == RAVEnd {
		p.ravLevel--
		p.nextToken()
	} else {
		fmt.Fprintf(p.cfg.LogFile, "Missing ')' to close variation.\n")
	}

	// Parse suffix comment
	variation.SuffixComment = p.parseOptCommentList()

	return variation
}

// parseResult parses a game result.
func (p *Parser) parseResult() string {
	if p.currentToken.Type == TerminatingResult {
		result := p.currentToken.TokenString
		if p.ravLevel == 0 {
			// Set to NoToken to help skip between games
			p.currentToken = &Token{Type: NoToken}
		} else {
			p.nextToken()
		}
		return result
	}
	return ""
}

// ParseAllGames parses all games from the input.
func (p *Parser) ParseAllGames() ([]*chess.Game, error) {
	var games []*chess.Game

	for {
		game, err := p.ParseGame()
		if err != nil {
			return games, err
		}
		if game == nil {
			break
		}
		games = append(games, game)
	}

	return games, nil
}
