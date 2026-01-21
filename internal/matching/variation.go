// Package matching provides game filtering by tags and positions.
package matching

import (
	"bufio"
	"os"
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

// VariationMatcher matches games against move sequences.
type VariationMatcher struct {
	// Textual move sequences to match
	moveSequences [][]string
	// Positional variations (FEN positions to match in sequence)
	positionSequences [][]string
}

// NewVariationMatcher creates a new variation matcher.
func NewVariationMatcher() *VariationMatcher {
	return &VariationMatcher{}
}

// LoadFromFile loads move sequences from a file.
// Each line is a move sequence like: "1. e4 e5 2. Nf3"
func (vm *VariationMatcher) LoadFromFile(filename string) error {
	file, err := os.Open(filename) //nolint:gosec // G304: CLI tool opens user-specified files
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse move sequence
		moves := parseMoveSequence(line)
		if len(moves) > 0 {
			vm.moveSequences = append(vm.moveSequences, moves)
		}
	}

	return scanner.Err()
}

// LoadPositionalFromFile loads positional variations from a file.
// Each line is a FEN position.
func (vm *VariationMatcher) LoadPositionalFromFile(filename string) error {
	file, err := os.Open(filename) //nolint:gosec // G304: CLI tool opens user-specified files
	if err != nil {
		return err
	}
	defer file.Close()

	var currentSequence []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			// Empty line separates sequences
			if len(currentSequence) > 0 {
				vm.positionSequences = append(vm.positionSequences, currentSequence)
				currentSequence = nil
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		currentSequence = append(currentSequence, line)
	}

	// Don't forget the last sequence
	if len(currentSequence) > 0 {
		vm.positionSequences = append(vm.positionSequences, currentSequence)
	}

	return scanner.Err()
}

// AddMoveSequence adds a move sequence to match.
func (vm *VariationMatcher) AddMoveSequence(moves []string) {
	vm.moveSequences = append(vm.moveSequences, moves)
}

// MatchGame checks if a game contains any of the move sequences or positions.
func (vm *VariationMatcher) MatchGame(game *chess.Game) bool {
	// Check textual move sequences
	for _, seq := range vm.moveSequences {
		if vm.matchMoveSequence(game, seq) {
			return true
		}
	}

	// Check positional sequences
	for _, seq := range vm.positionSequences {
		if vm.matchPositionSequence(game, seq) {
			return true
		}
	}

	return len(vm.moveSequences) == 0 && len(vm.positionSequences) == 0
}

// matchMoveSequence checks if the game contains the move sequence.
func (vm *VariationMatcher) matchMoveSequence(game *chess.Game, seq []string) bool {
	if len(seq) == 0 {
		return true
	}

	seqIdx := 0
	for move := game.Moves; move != nil; move = move.Next {
		// Normalize both move texts for comparison
		gameMoveText := normalizeMove(move.Text)
		seqMoveText := normalizeMove(seq[seqIdx])

		if gameMoveText == seqMoveText {
			seqIdx++
			if seqIdx >= len(seq) {
				return true // Found complete sequence
			}
		} else {
			// Reset if this isn't a contiguous match
			seqIdx = 0
			// Check if current move starts the sequence
			if normalizeMove(move.Text) == normalizeMove(seq[0]) {
				seqIdx = 1
			}
		}
	}

	return false
}

// matchPositionSequence checks if the game passes through all positions in sequence.
func (vm *VariationMatcher) matchPositionSequence(game *chess.Game, seq []string) bool {
	if len(seq) == 0 {
		return true
	}

	board, _ := engine.NewBoardFromFEN(engine.InitialFEN) //nolint:errcheck // InitialFEN is known valid
	seqIdx := 0

	// Check initial position
	if matchesFENPosition(board, seq[seqIdx]) {
		seqIdx++
		if seqIdx >= len(seq) {
			return true
		}
	}

	// Check after each move
	for move := game.Moves; move != nil; move = move.Next {
		if !engine.ApplyMove(board, move) {
			break
		}

		if matchesFENPosition(board, seq[seqIdx]) {
			seqIdx++
			if seqIdx >= len(seq) {
				return true
			}
		}
	}

	return false
}

// parseMoveSequence parses a line of moves into individual move texts.
func parseMoveSequence(line string) []string {
	var moves []string

	for _, part := range strings.Fields(line) {
		// Skip move numbers (1. 2. etc) and ellipsis
		if len(part) > 0 && (part[len(part)-1] == '.' || strings.Contains(part, "...")) {
			continue
		}
		moves = append(moves, part)
	}

	return moves
}

// normalizeMove normalizes a move text for comparison.
func normalizeMove(text string) string {
	// Remove annotations, check symbols, etc.
	return strings.TrimRight(strings.TrimSpace(text), "+#!?")
}

// matchesFENPosition checks if the board matches a FEN position string.
// The FEN can be partial (just the piece placement).
func matchesFENPosition(board *chess.Board, fen string) bool {
	boardFEN := engine.BoardToFEN(board)

	// Compare just the piece placement part
	boardParts := strings.Split(boardFEN, " ")
	fenParts := strings.Split(fen, " ")

	if len(boardParts) == 0 || len(fenParts) == 0 {
		return false
	}

	return boardParts[0] == fenParts[0]
}

// HasCriteria returns true if any matching criteria are set.
func (vm *VariationMatcher) HasCriteria() bool {
	return len(vm.moveSequences) > 0 || len(vm.positionSequences) > 0
}

// Match implements GameMatcher interface.
func (vm *VariationMatcher) Match(game *chess.Game) bool {
	return vm.MatchGame(game)
}

// Name implements GameMatcher interface.
func (vm *VariationMatcher) Name() string {
	return "VariationMatcher"
}
