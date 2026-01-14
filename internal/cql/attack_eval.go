package cql

import (
	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

// evalAttack checks if a piece attacks another piece or square.
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

// evalAttackOnPiece checks if attacker pieces attack target pieces.
func (e *Evaluator) evalAttackOnPiece(attackerDesig, targetDesig string) bool {
	attackerPieces := e.parsePieceDesignator(attackerDesig)
	targetPieces := e.parsePieceDesignator(targetDesig)

	// Find all target piece locations
	for rank := chess.Rank(0); rank < 8; rank++ {
		for col := chess.Col(0); col < 8; col++ {
			piece := e.getPieceAt(col, rank)
			if piece == chess.Empty || !containsPiece(targetPieces, piece) {
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

// evalAttackOnSquare checks if attacker pieces attack given squares.
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

// isAttackedByPieces checks if a square is attacked by any of the given pieces.
func (e *Evaluator) isAttackedByPieces(targetCol chess.Col, targetRank chess.Rank, attackerPieces []chess.Piece) bool {
	// Find all attacker piece locations and check if they attack the target
	for rank := chess.Rank(0); rank < 8; rank++ {
		for col := chess.Col(0); col < 8; col++ {
			piece := e.getPieceAt(col, rank)
			if piece == chess.Empty || !containsPiece(attackerPieces, piece) {
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

// canPieceAttack checks if a piece can attack from one square to another.
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

// isPathClear checks if the path between two squares is clear of pieces.
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

// evalCheck checks if the current side to move is in check.
func (e *Evaluator) evalCheck() bool {
	return engine.IsInCheck(e.board, e.board.ToMove)
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

// isPinned checks if a piece at pinnedCol,pinnedRank is pinned to targetCol,targetRank by one of pinnerPieces.
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
