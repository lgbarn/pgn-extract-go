package chess

// PositionCount tracks how many times a position has been reached (for repetition detection).
type PositionCount struct {
	HashValue      HashCode
	ToMove         Colour
	CastlingRights uint16
	EPRank         Rank
	EPCol          Col
	Count          uint
}

// Game represents a complete chess game with tags, moves, and metadata.
type Game struct {
	// Tags for this game (e.g., Event, Site, Date, White, Black, Result).
	Tags map[string]string

	// Any comment prefixing the game, between the tags and the moves.
	PrefixComment []*Comment

	// The hash value of the final position.
	FinalHashValue HashCode

	// An accumulated hash value, used to disambiguate false clashes of FinalHashValue.
	CumulativeHashValue HashCode

	// Board hash value at fuzzy_match_depth, if required.
	FuzzyDuplicateHash HashCode

	// The move list of the game.
	Moves *Move

	// Whether the moves have been checked.
	MovesChecked bool

	// Whether the moves are valid.
	MovesOK bool

	// If !MovesOK, the first ply at which an error was found (0 = no error).
	ErrorPly int

	// Counts of the number of times each position has been reached.
	PositionCounts map[HashCode]*PositionCount

	// Line numbers of the start and end of the game in the input file.
	StartLine uint
	EndLine   uint
}

// NewGame creates a new empty game.
func NewGame() *Game {
	return &Game{
		Tags:           make(map[string]string),
		PositionCounts: make(map[HashCode]*PositionCount),
	}
}

// GetTag returns a tag value, or empty string if not present.
func (g *Game) GetTag(name string) string {
	return g.Tags[name]
}

// SetTag sets a tag value.
func (g *Game) SetTag(name, value string) {
	g.ensureTags()
	g.Tags[name] = value
}

// HasTag returns true if the tag is present.
func (g *Game) HasTag(name string) bool {
	_, ok := g.Tags[name]
	return ok
}

// ensureTags initializes the Tags map if it is nil.
func (g *Game) ensureTags() {
	if g.Tags == nil {
		g.Tags = make(map[string]string)
	}
}

// White returns the White player name.
func (g *Game) White() string {
	return g.GetTag("White")
}

// Black returns the Black player name.
func (g *Game) Black() string {
	return g.GetTag("Black")
}

// Result returns the game result.
func (g *Game) Result() string {
	return g.GetTag("Result")
}

// Event returns the event name.
func (g *Game) Event() string {
	return g.GetTag("Event")
}

// Site returns the site name.
func (g *Game) Site() string {
	return g.GetTag("Site")
}

// Date returns the date string.
func (g *Game) Date() string {
	return g.GetTag("Date")
}

// Round returns the round string.
func (g *Game) Round() string {
	return g.GetTag("Round")
}

// ECO returns the ECO code.
func (g *Game) ECO() string {
	return g.GetTag("ECO")
}

// FEN returns the FEN string if present.
func (g *Game) FEN() string {
	return g.GetTag("FEN")
}

// PlyCount returns the number of half-moves in the game.
func (g *Game) PlyCount() int {
	count := 0
	for move := g.Moves; move != nil; move = move.Next {
		count++
	}
	return count
}

// LastMove returns the last move in the game, or nil if no moves.
func (g *Game) LastMove() *Move {
	if g.Moves == nil {
		return nil
	}
	move := g.Moves
	for move.Next != nil {
		move = move.Next
	}
	return move
}

// AppendMove adds a move to the end of the game.
func (g *Game) AppendMove(m *Move) {
	if g.Moves == nil {
		g.Moves = m
		return
	}
	last := g.LastMove()
	last.Next = m
	m.Prev = last
}

// AppendPrefixComment adds a prefix comment to the game.
func (g *Game) AppendPrefixComment(text string) {
	g.PrefixComment = append(g.PrefixComment, &Comment{Text: text})
}
