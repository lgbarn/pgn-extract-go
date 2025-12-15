package cql

import (
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

// Evaluator evaluates CQL expressions against a chess position.
type Evaluator struct {
	board *chess.Board
	game  *chess.Game // Optional, for game-level filters
}

// NewEvaluator creates a new evaluator for the given board position.
func NewEvaluator(board *chess.Board) *Evaluator {
	return &Evaluator{board: board}
}

// NewEvaluatorWithGame creates a new evaluator with both board and game context.
func NewEvaluatorWithGame(board *chess.Board, game *chess.Game) *Evaluator {
	return &Evaluator{board: board, game: game}
}

// Evaluate evaluates the CQL expression and returns true if it matches.
func (e *Evaluator) Evaluate(node Node) bool {
	switch n := node.(type) {
	case *FilterNode:
		return e.evalFilter(n)
	case *LogicalNode:
		return e.evalLogical(n)
	case *ComparisonNode:
		return e.evalComparison(n)
	default:
		return false
	}
}

func (e *Evaluator) evalFilter(f *FilterNode) bool {
	switch f.Name {
	case "piece":
		return e.evalPiece(f.Args)
	case "attack":
		return e.evalAttack(f.Args)
	case "check":
		return e.evalCheck()
	case "mate":
		return e.evalMate()
	case "stalemate":
		return e.evalStalemate()
	case "wtm":
		return e.board.ToMove == chess.White
	case "btm":
		return e.board.ToMove == chess.Black
	case "count":
		// Count returns a number, handled in comparison
		return false
	// Transformation filters
	case "flip":
		return e.evalFlip(f.Args)
	case "flipvertical":
		return e.evalFlipVertical(f.Args)
	case "flipcolor":
		return e.evalFlipColor(f.Args)
	case "shift":
		return e.evalShift(f.Args)
	case "shifthorizontal":
		return e.evalShiftHorizontal(f.Args)
	case "shiftvertical":
		return e.evalShiftVertical(f.Args)
	// Game-level filters
	case "result":
		return e.evalResult(f.Args)
	case "player":
		return e.evalPlayer(f.Args)
	// Position filters
	case "between":
		return e.evalBetween(f.Args)
	case "pin":
		return e.evalPin(f.Args)
	case "ray":
		return e.evalRay(f.Args)
	default:
		return false
	}
}

func (e *Evaluator) evalLogical(l *LogicalNode) bool {
	switch l.Op {
	case "and":
		for _, child := range l.Children {
			if !e.Evaluate(child) {
				return false
			}
		}
		return true
	case "or":
		for _, child := range l.Children {
			if e.Evaluate(child) {
				return true
			}
		}
		return false
	case "not":
		if len(l.Children) == 0 {
			return false
		}
		return !e.Evaluate(l.Children[0])
	default:
		return false
	}
}

func (e *Evaluator) evalComparison(c *ComparisonNode) bool {
	left := e.evalNumeric(c.Left)
	right := e.evalNumeric(c.Right)

	switch c.Op {
	case "<":
		return left < right
	case ">":
		return left > right
	case "<=":
		return left <= right
	case ">=":
		return left >= right
	case "==":
		return left == right
	default:
		return false
	}
}

func (e *Evaluator) evalNumeric(node Node) int {
	switch n := node.(type) {
	case *NumberNode:
		return n.Value
	case *FilterNode:
		if n.Name == "count" {
			return e.evalCount(n.Args)
		}
		if n.Name == "material" {
			return e.evalMaterial(n.Args)
		}
		if n.Name == "year" {
			return e.evalYear()
		}
		if n.Name == "elo" {
			return e.evalElo(n.Args)
		}
	}
	return 0
}

func (e *Evaluator) evalPiece(args []Node) bool {
	if len(args) < 2 {
		return false
	}

	pieceArg, ok := args[0].(*PieceNode)
	if !ok {
		return false
	}

	squareArg, ok := args[1].(*SquareNode)
	if !ok {
		return false
	}

	// Get squares to check
	squares := e.parseSquareSet(squareArg.Designator)
	if len(squares) == 0 {
		return false
	}

	// Get pieces to match
	pieces := e.parsePieceDesignator(pieceArg.Designator)

	// Check if any piece matches on any square
	for _, sq := range squares {
		piece := e.getPieceAt(sq.col, sq.rank)
		for _, p := range pieces {
			if piece == p {
				return true
			}
		}
	}

	return false
}

func (e *Evaluator) evalAttack(args []Node) bool {
	if len(args) < 2 {
		return false
	}

	// First arg is the attacking piece type
	attackerArg, ok := args[0].(*PieceNode)
	if !ok {
		return false
	}

	// Second arg is the target (piece or square)
	targetArg, ok := args[1].(*PieceNode)
	if !ok {
		// Could be a square
		sqArg, ok := args[1].(*SquareNode)
		if !ok {
			return false
		}
		return e.evalAttackOnSquare(attackerArg.Designator, sqArg.Designator)
	}

	return e.evalAttackOnPiece(attackerArg.Designator, targetArg.Designator)
}

func (e *Evaluator) evalAttackOnPiece(attackerDesig, targetDesig string) bool {
	attackerPieces := e.parsePieceDesignator(attackerDesig)
	targetPieces := e.parsePieceDesignator(targetDesig)

	// Find all target piece locations
	for rank := chess.Rank(0); rank < 8; rank++ {
		for col := chess.Col(0); col < 8; col++ {
			piece := e.getPieceAt(col, rank)
			if piece == chess.Empty {
				continue
			}

			// Is this a target piece?
			isTarget := false
			for _, tp := range targetPieces {
				if piece == tp {
					isTarget = true
					break
				}
			}

			if !isTarget {
				continue
			}

			// Check if any attacker piece can attack this square
			if e.isAttackedByPieces(col, rank, attackerPieces) {
				return true
			}
		}
	}

	return false
}

func (e *Evaluator) evalAttackOnSquare(attackerDesig, squareDesig string) bool {
	attackerPieces := e.parsePieceDesignator(attackerDesig)
	squares := e.parseSquareSet(squareDesig)

	for _, sq := range squares {
		if e.isAttackedByPieces(sq.col, sq.rank, attackerPieces) {
			return true
		}
	}

	return false
}

func (e *Evaluator) isAttackedByPieces(targetCol chess.Col, targetRank chess.Rank, attackerPieces []chess.Piece) bool {
	// Find all attacker piece locations and check if they attack the target
	for rank := chess.Rank(0); rank < 8; rank++ {
		for col := chess.Col(0); col < 8; col++ {
			piece := e.getPieceAt(col, rank)
			if piece == chess.Empty {
				continue
			}

			// Is this one of the attacker pieces?
			isAttacker := false
			for _, ap := range attackerPieces {
				if piece == ap {
					isAttacker = true
					break
				}
			}

			if !isAttacker {
				continue
			}

			// Check if this piece can attack the target
			if e.canPieceAttack(piece, col, rank, targetCol, targetRank) {
				return true
			}
		}
	}

	return false
}

func (e *Evaluator) canPieceAttack(piece chess.Piece, fromCol chess.Col, fromRank chess.Rank, toCol chess.Col, toRank chess.Rank) bool {
	// Use the engine's attack detection if possible
	pieceType := chess.ExtractPiece(piece)
	colour := chess.ExtractColour(piece)

	dCol := int(toCol) - int(fromCol)
	dRank := int(toRank) - int(fromRank)

	switch pieceType {
	case chess.Pawn:
		// Pawns attack diagonally
		if colour == chess.White {
			return dRank == 1 && (dCol == 1 || dCol == -1)
		}
		return dRank == -1 && (dCol == 1 || dCol == -1)

	case chess.Knight:
		absCol := abs(dCol)
		absRank := abs(dRank)
		return (absCol == 1 && absRank == 2) || (absCol == 2 && absRank == 1)

	case chess.Bishop:
		if abs(dCol) != abs(dRank) || dCol == 0 {
			return false
		}
		return e.isPathClear(fromCol, fromRank, toCol, toRank)

	case chess.Rook:
		if dCol != 0 && dRank != 0 {
			return false
		}
		if dCol == 0 && dRank == 0 {
			return false
		}
		return e.isPathClear(fromCol, fromRank, toCol, toRank)

	case chess.Queen:
		isDiagonal := abs(dCol) == abs(dRank) && dCol != 0
		isStraight := (dCol == 0 || dRank == 0) && (dCol != 0 || dRank != 0)
		if !isDiagonal && !isStraight {
			return false
		}
		return e.isPathClear(fromCol, fromRank, toCol, toRank)

	case chess.King:
		return abs(dCol) <= 1 && abs(dRank) <= 1 && (dCol != 0 || dRank != 0)
	}

	return false
}

func (e *Evaluator) isPathClear(fromCol chess.Col, fromRank chess.Rank, toCol chess.Col, toRank chess.Rank) bool {
	dCol := sign(int(toCol) - int(fromCol))
	dRank := sign(int(toRank) - int(fromRank))

	col := int(fromCol) + dCol
	rank := int(fromRank) + dRank

	for col != int(toCol) || rank != int(toRank) {
		if e.getPieceAt(chess.Col(col), chess.Rank(rank)) != chess.Empty {
			return false
		}
		col += dCol
		rank += dRank
	}

	return true
}

func (e *Evaluator) evalCheck() bool {
	return engine.IsInCheck(e.board, e.board.ToMove)
}

func (e *Evaluator) evalMate() bool {
	return engine.IsCheckmate(e.board)
}

func (e *Evaluator) evalStalemate() bool {
	return engine.IsStalemate(e.board)
}

func (e *Evaluator) evalCount(args []Node) int {
	if len(args) < 1 {
		return 0
	}

	pieceArg, ok := args[0].(*PieceNode)
	if !ok {
		return 0
	}

	pieces := e.parsePieceDesignator(pieceArg.Designator)
	count := 0

	for rank := chess.Rank(0); rank < 8; rank++ {
		for col := chess.Col(0); col < 8; col++ {
			piece := e.getPieceAt(col, rank)
			for _, p := range pieces {
				if piece == p {
					count++
					break
				}
			}
		}
	}

	return count
}

func (e *Evaluator) evalMaterial(args []Node) int {
	// Material value of pieces for one side
	// Standard values: P=1, N=3, B=3, R=5, Q=9
	if len(args) < 1 {
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

	var targetColour chess.Colour
	switch color {
	case "white":
		targetColour = chess.White
	case "black":
		targetColour = chess.Black
	default:
		return 0
	}

	material := 0
	for rank := chess.Rank(0); rank < 8; rank++ {
		for col := chess.Col(0); col < 8; col++ {
			piece := e.getPieceAt(col, rank)
			if piece == chess.Empty {
				continue
			}

			pieceColour := chess.ExtractColour(piece)
			if pieceColour != targetColour {
				continue
			}

			pieceType := chess.ExtractPiece(piece)
			switch pieceType {
			case chess.Pawn:
				material += 1
			case chess.Knight:
				material += 3
			case chess.Bishop:
				material += 3
			case chess.Rook:
				material += 5
			case chess.Queen:
				material += 9
			// King has no material value
			}
		}
	}

	return material
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
	yearStr := date[:4]
	year := 0
	for _, c := range yearStr {
		if c >= '0' && c <= '9' {
			year = year*10 + int(c-'0')
		}
	}
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

	// Parse Elo rating
	elo := 0
	for _, c := range eloStr {
		if c >= '0' && c <= '9' {
			elo = elo*10 + int(c-'0')
		}
	}
	return elo
}

// evalBetween checks if there are squares between two given squares.
func (e *Evaluator) evalBetween(args []Node) bool {
	if len(args) < 2 {
		return false
	}

	sq1Arg, ok := args[0].(*SquareNode)
	if !ok {
		return false
	}
	sq2Arg, ok := args[1].(*SquareNode)
	if !ok {
		return false
	}

	squares1 := e.parseSquareSet(sq1Arg.Designator)
	squares2 := e.parseSquareSet(sq2Arg.Designator)

	if len(squares1) == 0 || len(squares2) == 0 {
		return false
	}

	sq1 := squares1[0]
	sq2 := squares2[0]

	// Check if squares are on same rank, file, or diagonal
	dCol := int(sq2.col) - int(sq1.col)
	dRank := int(sq2.rank) - int(sq1.rank)

	// Must be on same rank, file, or diagonal
	if dCol != 0 && dRank != 0 && abs(dCol) != abs(dRank) {
		return false
	}

	// At least one square apart
	if abs(dCol) <= 1 && abs(dRank) <= 1 {
		return false
	}

	return true
}

// evalPin checks if a piece is pinned.
// Format: pin <pinned piece> <pinner piece> <piece pinned to>
func (e *Evaluator) evalPin(args []Node) bool {
	if len(args) < 3 {
		return false
	}

	pinnedArg, ok := args[0].(*PieceNode)
	if !ok {
		return false
	}
	pinnerArg, ok := args[1].(*PieceNode)
	if !ok {
		return false
	}
	targetArg, ok := args[2].(*PieceNode)
	if !ok {
		return false
	}

	pinnedPieces := e.parsePieceDesignator(pinnedArg.Designator)
	pinnerPieces := e.parsePieceDesignator(pinnerArg.Designator)
	targetPieces := e.parsePieceDesignator(targetArg.Designator)

	// Find all pinned piece locations
	for pRank := chess.Rank(0); pRank < 8; pRank++ {
		for pCol := chess.Col(0); pCol < 8; pCol++ {
			piece := e.getPieceAt(pCol, pRank)
			if !containsPiece(pinnedPieces, piece) {
				continue
			}

			// Find target piece locations
			for tRank := chess.Rank(0); tRank < 8; tRank++ {
				for tCol := chess.Col(0); tCol < 8; tCol++ {
					targetPiece := e.getPieceAt(tCol, tRank)
					if !containsPiece(targetPieces, targetPiece) {
						continue
					}

					// Check if there's a pinner along the line from target through pinned
					if e.isPinned(pCol, pRank, tCol, tRank, pinnerPieces) {
						return true
					}
				}
			}
		}
	}

	return false
}

func (e *Evaluator) isPinned(pinnedCol chess.Col, pinnedRank chess.Rank, targetCol chess.Col, targetRank chess.Rank, pinnerPieces []chess.Piece) bool {
	// Get direction from target to pinned
	dCol := int(pinnedCol) - int(targetCol)
	dRank := int(pinnedRank) - int(targetRank)

	// Must be on same rank, file, or diagonal
	if dCol != 0 && dRank != 0 && abs(dCol) != abs(dRank) {
		return false
	}

	// Normalize direction
	stepCol := sign(dCol)
	stepRank := sign(dRank)

	// Check if path from target to pinned is clear (except for pinned piece)
	col := int(targetCol) + stepCol
	rank := int(targetRank) + stepRank
	for col != int(pinnedCol) || rank != int(pinnedRank) {
		if e.getPieceAt(chess.Col(col), chess.Rank(rank)) != chess.Empty {
			return false // Blocked
		}
		col += stepCol
		rank += stepRank
	}

	// Continue in same direction to find pinner
	col = int(pinnedCol) + stepCol
	rank = int(pinnedRank) + stepRank
	for col >= 0 && col < 8 && rank >= 0 && rank < 8 {
		piece := e.getPieceAt(chess.Col(col), chess.Rank(rank))
		if piece != chess.Empty {
			// Check if this is a pinner piece
			if containsPiece(pinnerPieces, piece) {
				// Verify it can actually attack along this line
				pieceType := chess.ExtractPiece(piece)
				isDiagonal := abs(stepCol) == 1 && abs(stepRank) == 1
				isStraight := (stepCol == 0) != (stepRank == 0)

				if isDiagonal && (pieceType == chess.Bishop || pieceType == chess.Queen) {
					return true
				}
				if isStraight && (pieceType == chess.Rook || pieceType == chess.Queen) {
					return true
				}
			}
			return false // Blocked by non-pinner
		}
		col += stepCol
		rank += stepRank
	}

	return false
}

func containsPiece(pieces []chess.Piece, piece chess.Piece) bool {
	for _, p := range pieces {
		if p == piece {
			return true
		}
	}
	return false
}

// evalRay checks if there's a ray (line) between two squares.
// Format: ray <direction> <from> <to>
func (e *Evaluator) evalRay(args []Node) bool {
	if len(args) < 3 {
		return false
	}

	// Get direction (can be string or filter node)
	var direction string
	switch arg := args[0].(type) {
	case *StringNode:
		direction = arg.Value
	case *FilterNode:
		direction = arg.Name
	default:
		return false
	}

	fromArg, ok := args[1].(*SquareNode)
	if !ok {
		return false
	}
	toArg, ok := args[2].(*SquareNode)
	if !ok {
		return false
	}

	from := e.parseSquareSet(fromArg.Designator)
	to := e.parseSquareSet(toArg.Designator)

	if len(from) == 0 || len(to) == 0 {
		return false
	}

	sq1 := from[0]
	sq2 := to[0]

	dCol := int(sq2.col) - int(sq1.col)
	dRank := int(sq2.rank) - int(sq1.rank)

	switch direction {
	case "horizontal":
		return dRank == 0 && dCol != 0
	case "vertical":
		return dCol == 0 && dRank != 0
	case "diagonal":
		return abs(dCol) == abs(dRank) && dCol != 0
	case "orthogonal":
		return (dCol == 0 || dRank == 0) && (dCol != 0 || dRank != 0)
	}

	return false
}

// Helper types and functions

type square struct {
	col  chess.Col
	rank chess.Rank
}

func (e *Evaluator) parseSquareSet(desig string) []square {
	if desig == "." {
		// All squares
		var squares []square
		for rank := chess.Rank(0); rank < 8; rank++ {
			for col := chess.Col(0); col < 8; col++ {
				squares = append(squares, square{col, rank})
			}
		}
		return squares
	}

	// Simple single square like "e1"
	if len(desig) == 2 && desig[0] >= 'a' && desig[0] <= 'h' && desig[1] >= '1' && desig[1] <= '8' {
		col := chess.Col(desig[0] - 'a')
		rank := chess.Rank(desig[1] - '1')
		return []square{{col, rank}}
	}

	// Range patterns like [a-h]1, a[1-8], [a-d][1-4]
	// For now, handle simple patterns
	var squares []square

	// Try to parse as range pattern
	files := "abcdefgh"
	ranks := "12345678"

	if strings.HasPrefix(desig, "[") {
		// [a-h]1 or [a-d][1-4] pattern
		return e.parseComplexSquareSet(desig)
	}

	// a[1-8] pattern
	if len(desig) > 2 && desig[1] == '[' {
		file := desig[0]
		if file >= 'a' && file <= 'h' {
			col := chess.Col(file - 'a')
			// Parse rank range
			rankRange := desig[2 : len(desig)-1] // Remove brackets
			parts := strings.Split(rankRange, "-")
			if len(parts) == 2 {
				startRank := parts[0][0] - '1'
				endRank := parts[1][0] - '1'
				for r := startRank; r <= endRank; r++ {
					squares = append(squares, square{col, chess.Rank(r)})
				}
				return squares
			}
		}
	}

	// Fallback: treat each character
	for _, r := range ranks {
		for _, f := range files {
			if strings.Contains(desig, string(f)) && strings.Contains(desig, string(r)) {
				col := chess.Col(f - 'a')
				rank := chess.Rank(r - '1')
				squares = append(squares, square{col, rank})
			}
		}
	}

	return squares
}

func (e *Evaluator) parseComplexSquareSet(desig string) []square {
	var squares []square

	// [a-h]1 pattern
	if strings.HasPrefix(desig, "[") && !strings.Contains(desig[1:], "[") {
		// Single file range with rank
		closeBracket := strings.Index(desig, "]")
		if closeBracket == -1 {
			return squares
		}
		fileRange := desig[1:closeBracket]
		rankPart := desig[closeBracket+1:]

		files := e.parseRange(fileRange, 'a', 'h')
		if len(rankPart) == 1 && rankPart[0] >= '1' && rankPart[0] <= '8' {
			rank := chess.Rank(rankPart[0] - '1')
			for _, f := range files {
				squares = append(squares, square{chess.Col(f - 'a'), rank})
			}
		}
		return squares
	}

	// [a-d][1-4] pattern
	firstClose := strings.Index(desig, "]")
	if firstClose == -1 {
		return squares
	}
	secondOpen := strings.Index(desig[firstClose:], "[")
	if secondOpen == -1 {
		return squares
	}
	secondOpen += firstClose

	fileRange := desig[1:firstClose]
	rankRange := desig[secondOpen+1 : len(desig)-1]

	files := e.parseRange(fileRange, 'a', 'h')
	ranks := e.parseRange(rankRange, '1', '8')

	for _, f := range files {
		for _, r := range ranks {
			squares = append(squares, square{chess.Col(f - 'a'), chess.Rank(r - '1')})
		}
	}

	return squares
}

func (e *Evaluator) parseRange(rangeStr string, min, max byte) []byte {
	var result []byte

	if strings.Contains(rangeStr, "-") {
		parts := strings.Split(rangeStr, "-")
		if len(parts) == 2 && len(parts[0]) == 1 && len(parts[1]) == 1 {
			start := parts[0][0]
			end := parts[1][0]
			if start >= min && end <= max && start <= end {
				for c := start; c <= end; c++ {
					result = append(result, c)
				}
			}
		}
	} else {
		// Individual characters
		for _, c := range rangeStr {
			if byte(c) >= min && byte(c) <= max {
				result = append(result, byte(c))
			}
		}
	}

	return result
}

func (e *Evaluator) parsePieceDesignator(desig string) []chess.Piece {
	var pieces []chess.Piece

	// Handle piece sets like [RQ]
	if strings.HasPrefix(desig, "[") && strings.HasSuffix(desig, "]") {
		inner := desig[1 : len(desig)-1]
		for _, c := range inner {
			pieces = append(pieces, e.charToPieces(byte(c))...)
		}
		return pieces
	}

	// Single character designator
	if len(desig) == 1 {
		return e.charToPieces(desig[0])
	}

	return pieces
}

func (e *Evaluator) charToPieces(c byte) []chess.Piece {
	switch c {
	case 'K':
		return []chess.Piece{chess.W(chess.King)}
	case 'Q':
		return []chess.Piece{chess.W(chess.Queen)}
	case 'R':
		return []chess.Piece{chess.W(chess.Rook)}
	case 'B':
		return []chess.Piece{chess.W(chess.Bishop)}
	case 'N':
		return []chess.Piece{chess.W(chess.Knight)}
	case 'P':
		return []chess.Piece{chess.W(chess.Pawn)}
	case 'k':
		return []chess.Piece{chess.B(chess.King)}
	case 'q':
		return []chess.Piece{chess.B(chess.Queen)}
	case 'r':
		return []chess.Piece{chess.B(chess.Rook)}
	case 'b':
		return []chess.Piece{chess.B(chess.Bishop)}
	case 'n':
		return []chess.Piece{chess.B(chess.Knight)}
	case 'p':
		return []chess.Piece{chess.B(chess.Pawn)}
	case 'A':
		// Any white piece
		return []chess.Piece{chess.W(chess.King), chess.W(chess.Queen), chess.W(chess.Rook), chess.W(chess.Bishop), chess.W(chess.Knight), chess.W(chess.Pawn)}
	case 'a':
		// Any black piece
		return []chess.Piece{chess.B(chess.King), chess.B(chess.Queen), chess.B(chess.Rook), chess.B(chess.Bishop), chess.B(chess.Knight), chess.B(chess.Pawn)}
	case '_':
		// Empty square
		return []chess.Piece{chess.Empty}
	case '?':
		// Any piece or empty
		return []chess.Piece{
			chess.Empty,
			chess.W(chess.King), chess.W(chess.Queen), chess.W(chess.Rook), chess.W(chess.Bishop), chess.W(chess.Knight), chess.W(chess.Pawn),
			chess.B(chess.King), chess.B(chess.Queen), chess.B(chess.Rook), chess.B(chess.Bishop), chess.B(chess.Knight), chess.B(chess.Pawn),
		}
	}
	return nil
}

func (e *Evaluator) getPieceAt(col chess.Col, rank chess.Rank) chess.Piece {
	// Board uses hedged 12x12 array with offset 2
	return e.board.Squares[col+chess.Hedge][rank+chess.Hedge]
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

// Transformation implementations

// evalFlip evaluates the child expression with horizontal flip transformation.
// Tries both the original pattern and its horizontal mirror (a↔h files).
func (e *Evaluator) evalFlip(args []Node) bool {
	if len(args) < 1 {
		return false
	}

	// Try original
	if e.Evaluate(args[0]) {
		return true
	}

	// Try horizontal flip - transform the pattern and evaluate
	flippedNode := e.transformNode(args[0], flipHorizontal)
	return e.Evaluate(flippedNode)
}

// evalFlipVertical evaluates with vertical flip transformation (1↔8 ranks).
func (e *Evaluator) evalFlipVertical(args []Node) bool {
	if len(args) < 1 {
		return false
	}

	// Try original
	if e.Evaluate(args[0]) {
		return true
	}

	// Try vertical flip
	flippedNode := e.transformNode(args[0], flipVertical)
	return e.Evaluate(flippedNode)
}

// evalFlipColor evaluates with color flip transformation (white↔black).
func (e *Evaluator) evalFlipColor(args []Node) bool {
	if len(args) < 1 {
		return false
	}

	// Try original
	if e.Evaluate(args[0]) {
		return true
	}

	// Try color flip
	flippedNode := e.transformNode(args[0], flipColor)
	return e.Evaluate(flippedNode)
}

// evalShift tries all possible translations of the pattern.
func (e *Evaluator) evalShift(args []Node) bool {
	if len(args) < 1 {
		return false
	}

	// Try all possible shifts
	for dCol := -7; dCol <= 7; dCol++ {
		for dRank := -7; dRank <= 7; dRank++ {
			shiftedNode := e.transformNode(args[0], func(col, rank int) (int, int) {
				return col + dCol, rank + dRank
			})
			if e.Evaluate(shiftedNode) {
				return true
			}
		}
	}
	return false
}

// evalShiftHorizontal tries all horizontal translations.
func (e *Evaluator) evalShiftHorizontal(args []Node) bool {
	if len(args) < 1 {
		return false
	}

	// Try all horizontal shifts
	for dCol := -7; dCol <= 7; dCol++ {
		shiftedNode := e.transformNode(args[0], func(col, rank int) (int, int) {
			return col + dCol, rank
		})
		if e.Evaluate(shiftedNode) {
			return true
		}
	}
	return false
}

// evalShiftVertical tries all vertical translations.
func (e *Evaluator) evalShiftVertical(args []Node) bool {
	if len(args) < 1 {
		return false
	}

	// Try all vertical shifts
	for dRank := -7; dRank <= 7; dRank++ {
		shiftedNode := e.transformNode(args[0], func(col, rank int) (int, int) {
			return col, rank + dRank
		})
		if e.Evaluate(shiftedNode) {
			return true
		}
	}
	return false
}

// Transform functions
type squareTransform func(col, rank int) (int, int)

func flipHorizontal(col, rank int) (int, int) {
	return 7 - col, rank // a↔h, b↔g, etc.
}

func flipVertical(col, rank int) (int, int) {
	return col, 7 - rank // 1↔8, 2↔7, etc.
}

func flipColor(col, rank int) (int, int) {
	// Color flip doesn't change squares, just piece colors
	// Handled specially in transformNode
	return col, rank
}

// transformNode creates a transformed copy of an AST node.
func (e *Evaluator) transformNode(node Node, transform squareTransform) Node {
	switch n := node.(type) {
	case *FilterNode:
		return e.transformFilterNode(n, transform)
	case *LogicalNode:
		children := make([]Node, len(n.Children))
		for i, child := range n.Children {
			children[i] = e.transformNode(child, transform)
		}
		return &LogicalNode{Op: n.Op, Children: children}
	case *SquareNode:
		return e.transformSquareNode(n, transform)
	case *PieceNode:
		// For flipColor, we need to swap piece colors
		if transform == nil {
			return n
		}
		// Check if this is a color flip by testing a known point
		testCol, testRank := transform(0, 0)
		if testCol == 0 && testRank == 0 {
			// This could be flipColor - check by comparing with flipHorizontal
			hCol, hRank := flipHorizontal(0, 0)
			if hCol != 0 || hRank != 0 {
				// Not flipHorizontal, check flipVertical
				vCol, vRank := flipVertical(0, 0)
				if vCol != 0 || vRank != 0 {
					// Must be flipColor - swap piece colors
					return e.transformPieceNodeColor(n)
				}
			}
		}
		return n
	default:
		return node
	}
}

func (e *Evaluator) transformFilterNode(f *FilterNode, transform squareTransform) *FilterNode {
	// Transform arguments
	args := make([]Node, len(f.Args))
	for i, arg := range f.Args {
		args[i] = e.transformNode(arg, transform)
	}
	return &FilterNode{Name: f.Name, Args: args}
}

func (e *Evaluator) transformSquareNode(s *SquareNode, transform squareTransform) *SquareNode {
	// Parse the square designator, transform, and create new designator
	squares := e.parseSquareSet(s.Designator)
	if len(squares) == 0 {
		return s
	}

	// For single squares, transform and create new designator
	if len(squares) == 1 {
		sq := squares[0]
		newCol, newRank := transform(int(sq.col), int(sq.rank))
		if newCol >= 0 && newCol < 8 && newRank >= 0 && newRank < 8 {
			newDesig := string(rune('a'+newCol)) + string(rune('1'+newRank))
			return &SquareNode{Designator: newDesig}
		}
		// Out of bounds - return original (won't match)
		return s
	}

	// For complex square sets, transform each square
	// This is more complex - for now, just return original
	return s
}

func (e *Evaluator) transformPieceNodeColor(p *PieceNode) *PieceNode {
	// Swap piece colors in the designator
	desig := p.Designator
	var newDesig string

	for _, c := range desig {
		switch c {
		case 'K':
			newDesig += "k"
		case 'Q':
			newDesig += "q"
		case 'R':
			newDesig += "r"
		case 'B':
			newDesig += "b"
		case 'N':
			newDesig += "n"
		case 'P':
			newDesig += "p"
		case 'k':
			newDesig += "K"
		case 'q':
			newDesig += "Q"
		case 'r':
			newDesig += "R"
		case 'b':
			newDesig += "B"
		case 'n':
			newDesig += "N"
		case 'p':
			newDesig += "P"
		case 'A':
			newDesig += "a"
		case 'a':
			newDesig += "A"
		default:
			newDesig += string(c)
		}
	}

	return &PieceNode{Designator: newDesig}
}
