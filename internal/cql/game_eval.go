package cql

import (
	"strconv"
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

// evalMate checks if the current position is checkmate.
func (e *Evaluator) evalMate() bool {
	return engine.IsCheckmate(e.board)
}

// evalStalemate checks if the current position is stalemate.
func (e *Evaluator) evalStalemate() bool {
	return engine.IsStalemate(e.board)
}

// evalResult checks if the game result matches.
func (e *Evaluator) evalResult(args []Node) bool {
	if len(args) < 1 || e.game == nil {
		return false
	}

	resultArg, ok := args[0].(*StringNode)
	if !ok {
		return false
	}

	gameResult, ok := e.game.Tags["Result"]
	if !ok {
		return false
	}

	return gameResult == resultArg.Value
}

// evalPlayer checks if either player name contains the given substring.
func (e *Evaluator) evalPlayer(args []Node) bool {
	if len(args) < 1 || e.game == nil {
		return false
	}

	playerArg, ok := args[0].(*StringNode)
	if !ok {
		return false
	}

	white := e.game.Tags["White"]
	black := e.game.Tags["Black"]

	return strings.Contains(white, playerArg.Value) || strings.Contains(black, playerArg.Value)
}

// evalYear returns the year from the Date tag.
func (e *Evaluator) evalYear() int {
	if e.game == nil {
		return 0
	}

	date, ok := e.game.Tags["Date"]
	if !ok || len(date) < 4 {
		return 0
	}

	// Parse year from "YYYY.MM.DD" or "YYYY" format
	year, _ := strconv.Atoi(date[:4])
	return year
}

// evalElo returns the Elo rating for the specified color.
func (e *Evaluator) evalElo(args []Node) int {
	if len(args) < 1 || e.game == nil {
		return 0
	}

	// Get the color argument (can be string "white"/"black" or filter node)
	var color string
	switch arg := args[0].(type) {
	case *StringNode:
		color = arg.Value
	case *FilterNode:
		color = arg.Name
	default:
		return 0
	}

	var eloTag string
	switch color {
	case "white":
		eloTag = "WhiteElo"
	case "black":
		eloTag = "BlackElo"
	default:
		return 0
	}

	eloStr, ok := e.game.Tags[eloTag]
	if !ok {
		return 0
	}

	elo, _ := strconv.Atoi(eloStr)
	return elo
}
