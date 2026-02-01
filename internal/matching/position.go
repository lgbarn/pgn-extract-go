package matching

import (
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/hashing"
)

// FENPattern represents a FEN pattern to match.
// Supports wildcards:
//   - ? matches any square (empty or occupied)
//   - ! matches any non-empty square
//   - * matches zero or more of anything
//   - A matches any white piece
//   - a matches any black piece
//   - _ matches empty square
type FENPattern struct {
	Pattern       string
	Label         string // optional label for matched position
	Hash          uint64 // position hash for exact FEN matches
	IsExact       bool   // true if this is an exact FEN (no wildcards)
	IncludeInvert bool   // also match color-inverted position
	ranks         []string
}

// PositionMatcher provides position-based game filtering.
type PositionMatcher struct {
	patterns    []*FENPattern
	exactHashes map[uint64]*FENPattern
}

// NewPositionMatcher creates a new position matcher.
func NewPositionMatcher() *PositionMatcher {
	return &PositionMatcher{
		exactHashes: make(map[uint64]*FENPattern),
	}
}

// AddFEN adds an exact FEN position to match.
func (pm *PositionMatcher) AddFEN(fen string, label string) error {
	board, err := engine.NewBoardFromFEN(fen)
	if err != nil {
		return err
	}

	hash := hashing.GenerateZobristHash(board)
	pattern := &FENPattern{
		Pattern: fen,
		Label:   label,
		Hash:    hash,
		IsExact: true,
	}

	pm.patterns = append(pm.patterns, pattern)
	pm.exactHashes[hash] = pattern

	return nil
}

// AddPattern adds a FEN pattern with wildcards.
func (pm *PositionMatcher) AddPattern(pattern string, label string, includeInvert bool) {
	p := &FENPattern{
		Pattern:       pattern,
		Label:         label,
		IsExact:       false,
		IncludeInvert: includeInvert,
	}

	// Parse into ranks
	p.ranks = strings.Split(pattern, "/")

	pm.patterns = append(pm.patterns, p)

	// If invert requested, also add inverted pattern
	if includeInvert {
		inverted := invertPattern(pattern)
		ip := &FENPattern{
			Pattern:       inverted,
			Label:         label,
			IsExact:       false,
			IncludeInvert: false,
		}
		ip.ranks = strings.Split(inverted, "/")
		pm.patterns = append(pm.patterns, ip)
	}
}

// MatchGame checks if any position in the game matches a pattern.
// Returns the matching pattern (with label) or nil.
func (pm *PositionMatcher) MatchGame(game *chess.Game) *FENPattern {
	if len(pm.patterns) == 0 {
		return nil
	}

	// Get starting position from FEN tag or use initial position
	board := pm.getStartingBoard(game)

	// Check initial position
	if match := pm.matchPosition(board); match != nil {
		return match
	}

	// Replay game and check each position
	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}

		if match := pm.matchPosition(board); match != nil {
			return match
		}
	}

	return nil
}

// getStartingBoard returns the starting board from FEN tag or initial position.
func (pm *PositionMatcher) getStartingBoard(game *chess.Game) *chess.Board {
	if fen, ok := game.Tags["FEN"]; ok {
		if board, err := engine.NewBoardFromFEN(fen); err == nil {
			return board
		}
	}
	board := engine.MustBoardFromFEN(engine.InitialFEN)
	return board
}

// matchPosition checks if a position matches any pattern.
func (pm *PositionMatcher) matchPosition(board *chess.Board) *FENPattern {
	// First check exact hash matches (fast)
	hash := hashing.GenerateZobristHash(board)
	if pattern, ok := pm.exactHashes[hash]; ok {
		return pattern
	}

	// Then check pattern matches
	for _, pattern := range pm.patterns {
		if !pattern.IsExact && pm.matchPattern(board, pattern) {
			return pattern
		}
	}

	return nil
}

// matchPattern checks if a board matches a FEN pattern with wildcards.
func (pm *PositionMatcher) matchPattern(board *chess.Board, pattern *FENPattern) bool {
	if len(pattern.ranks) == 0 {
		return false
	}

	// Convert board to rank strings for matching
	boardRanks := boardToRanks(board)

	// Match each rank
	for i, patternRank := range pattern.ranks {
		if i >= 8 {
			break
		}
		if !matchRank(boardRanks[7-i], patternRank) {
			return false
		}
	}

	return true
}

// boardToRanks converts a board to rank strings (rank 8 first).
func boardToRanks(board *chess.Board) [8]string {
	var ranks [8]string

	for r := 0; r < 8; r++ {
		rank := chess.Rank('1' + byte(r))
		var sb strings.Builder

		for c := chess.Col('a'); c <= 'h'; c++ {
			piece := board.Get(c, rank)
			sb.WriteByte(pieceToChar(piece))
		}

		ranks[r] = sb.String()
	}

	return ranks
}

// pieceToChar converts a piece to FEN character.
func pieceToChar(piece chess.Piece) byte {
	if piece == chess.Empty {
		return '_'
	}

	colour := chess.ExtractColour(piece)
	pieceType := chess.ExtractPiece(piece)

	var c byte
	switch pieceType {
	case chess.Pawn:
		c = 'P'
	case chess.Knight:
		c = 'N'
	case chess.Bishop:
		c = 'B'
	case chess.Rook:
		c = 'R'
	case chess.Queen:
		c = 'Q'
	case chess.King:
		c = 'K'
	default:
		return '_'
	}

	if colour == chess.Black {
		c += 32 // lowercase
	}

	return c
}

// matchRank matches a board rank string against a pattern rank.
func matchRank(boardRank, patternRank string) bool {
	bi := 0 // board index
	pi := 0 // pattern index

	for pi < len(patternRank) {
		if bi >= len(boardRank) && patternRank[pi] != '*' {
			return false
		}

		c := patternRank[pi]

		switch c {
		case '*':
			// * matches zero or more of anything
			pi++
			if pi >= len(patternRank) {
				return true // * at end matches rest
			}
			// Try matching rest of pattern at each position
			for bi <= len(boardRank) {
				if matchRank(boardRank[bi:], patternRank[pi:]) {
					return true
				}
				bi++
			}
			return false

		case '?':
			// ? matches any single square
			bi++
			pi++

		case '!':
			// ! matches any non-empty square
			if bi >= len(boardRank) || boardRank[bi] == '_' {
				return false
			}
			bi++
			pi++

		case 'A':
			// A matches any white piece (uppercase letters except _)
			if bi >= len(boardRank) || boardRank[bi] < 'A' || boardRank[bi] > 'Z' {
				return false
			}
			bi++
			pi++

		case 'a':
			// a (lowercase) matches any black piece
			if bi >= len(boardRank) || boardRank[bi] < 'a' || boardRank[bi] > 'z' {
				return false
			}
			bi++
			pi++

		case '_':
			// _ matches empty square
			if bi >= len(boardRank) || boardRank[bi] != '_' {
				return false
			}
			bi++
			pi++

		case '1', '2', '3', '4', '5', '6', '7', '8':
			// Number means N empty squares
			count := int(c - '0')
			for i := 0; i < count; i++ {
				if bi >= len(boardRank) || boardRank[bi] != '_' {
					return false
				}
				bi++
			}
			pi++

		default:
			// Exact piece match
			if bi >= len(boardRank) || boardRank[bi] != c {
				return false
			}
			bi++
			pi++
		}
	}

	return bi == len(boardRank)
}

// invertPattern inverts colors in a FEN pattern.
func invertPattern(pattern string) string {
	var result strings.Builder

	for _, c := range pattern {
		switch {
		case c >= 'A' && c <= 'Z':
			result.WriteRune(c + 32) // to lowercase
		case c >= 'a' && c <= 'z':
			result.WriteRune(c - 32) // to uppercase
		default:
			result.WriteRune(c)
		}
	}

	// Also reverse rank order
	ranks := strings.Split(result.String(), "/")
	for i, j := 0, len(ranks)-1; i < j; i, j = i+1, j-1 {
		ranks[i], ranks[j] = ranks[j], ranks[i]
	}

	return strings.Join(ranks, "/")
}

// PatternCount returns the number of patterns.
func (pm *PositionMatcher) PatternCount() int {
	return len(pm.patterns)
}
