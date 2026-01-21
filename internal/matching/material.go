// Package matching provides game filtering by tags and positions.
package matching

import (
	"strings"
	"unicode"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

// MaterialMatcher matches games by material balance.
type MaterialMatcher struct {
	// Pattern like "QR:qrr" means white has Q+R, black has Q+2R
	pattern     string
	exactMatch  bool
	whitePieces map[chess.Piece]int
	blackPieces map[chess.Piece]int
}

// NewMaterialMatcher creates a new material matcher.
// Pattern format: "QRN:qrn" (white pieces : black pieces)
// Use uppercase for white, lowercase for black
// K=King, Q=Queen, R=Rook, B=Bishop, N=Knight, P=Pawn
func NewMaterialMatcher(pattern string, exact bool) *MaterialMatcher {
	mm := &MaterialMatcher{
		pattern:     pattern,
		exactMatch:  exact,
		whitePieces: make(map[chess.Piece]int),
		blackPieces: make(map[chess.Piece]int),
	}
	mm.parsePattern(pattern)
	return mm
}

// parsePattern parses a material pattern like "QR:qrr"
func (mm *MaterialMatcher) parsePattern(pattern string) {
	parts := strings.Split(pattern, ":")
	if len(parts) >= 1 {
		mm.parsePieces(parts[0], chess.White)
	}
	if len(parts) >= 2 {
		mm.parsePieces(parts[1], chess.Black)
	}
}

// parsePieces parses a piece specification string for the given color.
// White pieces use uppercase (KQRBNP), black pieces use lowercase (kqrbnp).
func (mm *MaterialMatcher) parsePieces(s string, color chess.Colour) {
	target := mm.whitePieces
	if color == chess.Black {
		target = mm.blackPieces
	}

	for _, c := range s {
		switch unicode.ToUpper(c) {
		case 'K':
			target[chess.King]++
		case 'Q':
			target[chess.Queen]++
		case 'R':
			target[chess.Rook]++
		case 'B':
			target[chess.Bishop]++
		case 'N':
			target[chess.Knight]++
		case 'P':
			target[chess.Pawn]++
		}
	}
}

// MatchGame checks if any position in the game matches the material pattern.
func (mm *MaterialMatcher) MatchGame(game *chess.Game) bool {
	board, _ := engine.NewBoardFromFEN(engine.InitialFEN) //nolint:errcheck // InitialFEN is known valid

	// Check starting position
	if mm.matchPosition(board) {
		return true
	}

	// Check after each move
	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}

		if mm.matchPosition(board) {
			return true
		}
	}

	return false
}

// matchPosition checks if a position matches the material pattern.
func (mm *MaterialMatcher) matchPosition(board *chess.Board) bool {
	// Count pieces on the board
	whiteCounts := make(map[chess.Piece]int)
	blackCounts := make(map[chess.Piece]int)

	// Iterate over the board squares (accounting for hedge)
	for col := chess.Hedge; col < chess.Hedge+chess.BoardSize; col++ {
		for rank := chess.Hedge; rank < chess.Hedge+chess.BoardSize; rank++ {
			colouredPiece := board.Squares[col][rank]
			if colouredPiece == chess.Empty || colouredPiece == chess.Off {
				continue
			}

			// Extract piece type and color
			pieceType := chess.ExtractPiece(colouredPiece)
			colour := chess.ExtractColour(colouredPiece)

			if colour == chess.White {
				whiteCounts[pieceType]++
			} else {
				blackCounts[pieceType]++
			}
		}
	}

	if mm.exactMatch {
		return mm.exactMaterialMatch(whiteCounts, blackCounts)
	}
	return mm.minimalMaterialMatch(whiteCounts, blackCounts)
}

// exactMaterialMatch checks for exact material match.
func (mm *MaterialMatcher) exactMaterialMatch(whiteCounts, blackCounts map[chess.Piece]int) bool {
	// White pieces must match exactly
	for piece, count := range mm.whitePieces {
		if whiteCounts[piece] != count {
			return false
		}
	}

	// Black pieces must match exactly
	for piece, count := range mm.blackPieces {
		if blackCounts[piece] != count {
			return false
		}
	}

	// Check that there are no extra pieces beyond what's specified
	allPieces := []chess.Piece{chess.King, chess.Queen, chess.Rook, chess.Bishop, chess.Knight, chess.Pawn}
	for _, piece := range allPieces {
		if mm.whitePieces[piece] == 0 && whiteCounts[piece] > 0 {
			return false
		}
		if mm.blackPieces[piece] == 0 && blackCounts[piece] > 0 {
			return false
		}
	}

	return true
}

// minimalMaterialMatch checks that at least the specified pieces exist.
func (mm *MaterialMatcher) minimalMaterialMatch(whiteCounts, blackCounts map[chess.Piece]int) bool {
	// White must have at least the specified pieces
	for piece, count := range mm.whitePieces {
		if whiteCounts[piece] < count {
			return false
		}
	}

	// Black must have at least the specified pieces
	for piece, count := range mm.blackPieces {
		if blackCounts[piece] < count {
			return false
		}
	}

	return true
}

// HasCriteria returns true if a material pattern is set.
func (mm *MaterialMatcher) HasCriteria() bool {
	return mm.pattern != ""
}

// Match implements GameMatcher interface.
func (mm *MaterialMatcher) Match(game *chess.Game) bool {
	return mm.MatchGame(game)
}

// Name implements GameMatcher interface.
func (mm *MaterialMatcher) Name() string {
	return "MaterialMatcher"
}
