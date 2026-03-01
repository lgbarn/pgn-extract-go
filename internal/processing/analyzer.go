// Package processing provides game analysis, validation, and processing logic.
package processing

import (
	"fmt"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/hashing"
)

// GameAnalysis holds analysis results from replaying a game.
type GameAnalysis struct {
	FinalBoard        *chess.Board
	HasFiftyMoveRule  bool
	HasRepetition     bool
	HasUnderpromotion bool
	Positions         []uint64 // Zobrist hashes for repetition detection

	// Extended draw rule detection
	Has75MoveRule           bool
	Has5FoldRepetition      bool
	HasInsufficientMaterial bool
	HasMaterialOdds         bool
}

// FiftyMoveTriggered returns true if the game triggered the fifty-move rule.
func (ga *GameAnalysis) FiftyMoveTriggered() bool {
	return ga.HasFiftyMoveRule
}

// RepetitionDetected returns true if the game has a threefold repetition.
func (ga *GameAnalysis) RepetitionDetected() bool {
	return ga.HasRepetition
}

// UnderpromotionFound returns true if any pawn promoted to non-queen.
func (ga *GameAnalysis) UnderpromotionFound() bool {
	return ga.HasUnderpromotion
}

// ValidationResult holds the result of game validation.
type ValidationResult struct {
	Valid       bool
	ErrorPly    int
	ErrorMsg    string
	ParseErrors []string
}

// AnalyzeGame replays a game and analyzes it for various features.
func AnalyzeGame(game *chess.Game) (*chess.Board, *GameAnalysis) {
	board := engine.NewBoardForGame(game)
	analysis := &GameAnalysis{}

	// Check for material odds (non-standard starting position)
	if game.FEN() != "" {
		analysis.HasMaterialOdds = engine.CheckMaterialOdds(game)
	}

	posHash := hashing.GenerateZobristHash(board)
	analysis.Positions = append(analysis.Positions, posHash)
	positionCount := map[uint64]int{posHash: 1}

	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}

		// 50-move rule (100 half-moves)
		if board.HalfmoveClock >= 100 {
			analysis.HasFiftyMoveRule = true
		}

		// 75-move rule (150 half-moves - automatic draw)
		if board.HalfmoveClock >= 150 {
			analysis.Has75MoveRule = true
		}

		if move.PromotedPiece != chess.Empty && move.PromotedPiece != chess.Queen {
			analysis.HasUnderpromotion = true
		}

		posHash = hashing.GenerateZobristHash(board)
		analysis.Positions = append(analysis.Positions, posHash)
		positionCount[posHash]++

		// 3-fold repetition
		if positionCount[posHash] >= 3 {
			analysis.HasRepetition = true
		}

		// 5-fold repetition (automatic draw)
		if positionCount[posHash] >= 5 {
			analysis.Has5FoldRepetition = true
		}
	}

	// Check for insufficient material at final position
	analysis.HasInsufficientMaterial = engine.HasInsufficientMaterial(board)

	analysis.FinalBoard = board
	return board, analysis
}

// ReplayGame replays a game from the initial position to get the final board state.
func ReplayGame(game *chess.Game) *chess.Board {
	board := engine.NewBoardForGame(game)

	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}
	}

	return board
}

// ValidateGame validates all moves in a game are legal.
func ValidateGame(game *chess.Game) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Check for missing required tags
	requiredTags := []string{"Event", "Site", "Date", "Round", "White", "Black", "Result"}
	for _, tag := range requiredTags {
		if game.GetTag(tag) == "" {
			result.ParseErrors = append(result.ParseErrors, fmt.Sprintf("missing required tag: %s", tag))
		}
	}

	resultTag := game.GetTag("Result")
	if resultTag != "" && !isValidResult(resultTag) {
		result.ParseErrors = append(result.ParseErrors, fmt.Sprintf("invalid result: %s", resultTag))
	}

	// If we have no moves, game is valid (just tags)
	if game.Moves == nil {
		return result
	}

	// Replay game to validate moves
	var board *chess.Board
	var err error

	if fen, ok := game.Tags["FEN"]; ok {
		board, err = engine.NewBoardFromFEN(fen)
		if err != nil {
			result.Valid = false
			result.ErrorMsg = fmt.Sprintf("invalid FEN: %s", fen)
			return result
		}
	} else {
		board = engine.MustBoardFromFEN(engine.InitialFEN)
	}

	plyCount := 0
	for move := game.Moves; move != nil; move = move.Next {
		plyCount++
		if !engine.ApplyMove(board, move) {
			result.Valid = false
			result.ErrorPly = plyCount
			result.ErrorMsg = fmt.Sprintf("illegal move at ply %d: %s", plyCount, move.Text)
			return result
		}
	}

	// Mark game as validated
	game.MovesChecked = true
	game.MovesOK = true

	return result
}

// CountPlies counts the number of plies (half-moves) in a game.
func CountPlies(game *chess.Game) int {
	count := 0
	for move := game.Moves; move != nil; move = move.Next {
		count++
	}
	return count
}

// HasComments checks if a game has any comments.
func HasComments(game *chess.Game) bool {
	for move := game.Moves; move != nil; move = move.Next {
		if move.HasComments() {
			return true
		}
	}
	return false
}

// isValidResult checks if a result string is a valid PGN result.
func isValidResult(result string) bool {
	switch result {
	case "1-0", "0-1", "1/2-1/2", "*":
		return true
	default:
		return false
	}
}

// SplitVariations splits a game with variations into multiple games.
// Returns the main line game plus a game for each variation.
// Each split game has the same header tags as the original.
func SplitVariations(game *chess.Game) []*chess.Game {
	games := make([]*chess.Game, 0, 1)

	// First, output the main line
	mainGame := copyGameHeaders(game)
	mainGame.Moves = copyMainLine(game.Moves)
	games = append(games, mainGame)

	// Then recursively extract variations
	extractVariations(game.Moves, game, &games, nil)

	return games
}

// copyGameHeaders creates a new game with copied header tags.
func copyGameHeaders(original *chess.Game) *chess.Game {
	newGame := chess.NewGame()
	// Copy all tags
	for key, value := range original.Tags {
		newGame.Tags[key] = value
	}
	return newGame
}

// copyMainLine creates a copy of the main line moves (without variations).
func copyMainLine(moves *chess.Move) *chess.Move {
	if moves == nil {
		return nil
	}

	var head, tail *chess.Move
	for m := moves; m != nil; m = m.Next {
		newMove := copyMoveWithoutVariations(m)
		if head == nil {
			head = newMove
			tail = newMove
		} else {
			tail.Next = newMove
			newMove.Prev = tail
			tail = newMove
		}
	}
	return head
}

// copyMoveWithoutVariations copies a move without its variations.
func copyMoveWithoutVariations(m *chess.Move) *chess.Move {
	newMove := chess.NewMove()
	newMove.Text = m.Text
	newMove.Class = m.Class
	newMove.PieceToMove = m.PieceToMove
	newMove.PromotedPiece = m.PromotedPiece
	newMove.FromCol = m.FromCol
	newMove.FromRank = m.FromRank
	newMove.ToCol = m.ToCol
	newMove.ToRank = m.ToRank
	newMove.CapturedPiece = m.CapturedPiece
	// Copy NAGs
	for _, nag := range m.NAGs {
		if nag != nil {
			newNag := &chess.NAG{
				Text: append([]string{}, nag.Text...),
			}
			for _, c := range nag.Comments {
				if c != nil {
					newNag.Comments = append(newNag.Comments, &chess.Comment{Text: c.Text})
				}
			}
			newMove.NAGs = append(newMove.NAGs, newNag)
		}
	}
	// Copy comments
	for _, c := range m.Comments {
		if c != nil {
			newMove.Comments = append(newMove.Comments, &chess.Comment{Text: c.Text})
		}
	}
	return newMove
}

// extractVariations recursively extracts all variations from a move list.
// prefix is the moves leading up to the current position.
func extractVariations(moves *chess.Move, original *chess.Game, games *[]*chess.Game, prefix []*chess.Move) {
	for m := moves; m != nil; m = m.Next {
		// Check if this move has variations
		for _, variation := range m.Variations {
			if variation == nil || variation.Moves == nil {
				continue
			}

			// Create a new game for this variation
			varGame := copyGameHeaders(original)

			// Build move list: prefix + variation moves
			var head, tail *chess.Move
			for _, pm := range prefix {
				newMove := copyMoveWithoutVariations(pm)
				if head == nil {
					head = newMove
					tail = newMove
				} else {
					tail.Next = newMove
					newMove.Prev = tail
					tail = newMove
				}
			}

			// Add the variation moves
			for vm := variation.Moves; vm != nil; vm = vm.Next {
				newMove := copyMoveWithoutVariations(vm)
				if head == nil {
					head = newMove
					tail = newMove
				} else {
					tail.Next = newMove
					newMove.Prev = tail
					tail = newMove
				}
			}

			varGame.Moves = head
			*games = append(*games, varGame)

			// Recursively extract nested variations
			extractVariations(variation.Moves, original, games, append(prefix, copyMoveWithoutVariations(m)))
		}

		// Add current move to prefix for nested variations
		prefix = append(prefix, m)
	}
}
