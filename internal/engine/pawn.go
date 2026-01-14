package engine

import "github.com/lgbarn/pgn-extract-go/internal/chess"

// applyPawnMove applies a pawn move.
func applyPawnMove(board *chess.Board, move *chess.Move) bool {
	colour := board.ToMove
	fromCol, fromRank := move.FromCol, move.FromRank
	toCol, toRank := move.ToCol, move.ToRank

	if fromCol == 0 || fromRank == 0 {
		fromCol, fromRank = findPawnSource(board, move, colour)
		if fromCol == 0 {
			return false
		}
	}

	pawn := board.Get(fromCol, fromRank)

	// Handle en passant capture
	if move.Class == chess.EnPassantPawnMove {
		capturedRank := toRank - 1
		if colour == chess.Black {
			capturedRank = toRank + 1
		}
		board.Set(toCol, capturedRank, chess.Empty)
	}

	// Move the pawn
	board.Set(fromCol, fromRank, chess.Empty)

	// Handle promotion
	if move.Class == chess.PawnMoveWithPromotion {
		promotedPiece := move.PromotedPiece
		if promotedPiece == chess.Empty {
			promotedPiece = chess.Queen
		}
		board.Set(toCol, toRank, chess.MakeColouredPiece(colour, promotedPiece))
	} else {
		board.Set(toCol, toRank, pawn)
	}

	// Set en passant square if double pawn push
	board.EnPassant = false
	if colour == chess.White && fromRank == '2' && toRank == '4' {
		board.EnPassant = true
		board.EPCol = toCol
		board.EPRank = '3'
	} else if colour == chess.Black && fromRank == '7' && toRank == '5' {
		board.EnPassant = true
		board.EPCol = toCol
		board.EPRank = '6'
	}

	board.HalfmoveClock = 0
	if colour == chess.Black {
		board.MoveNumber++
	}
	board.ToMove = colour.Opposite()

	return true
}

// findPawnSource finds the source square of a pawn move.
func findPawnSource(board *chess.Board, move *chess.Move, colour chess.Colour) (chess.Col, chess.Rank) {
	toCol, toRank := move.ToCol, move.ToRank
	fromCol := move.FromCol
	pawn := chess.MakeColouredPiece(colour, chess.Pawn)
	direction := chess.ColourOffset(colour)

	// Capture - look one rank back in the specified column
	if fromCol != 0 {
		fromRank := chess.Rank(byte(toRank) - byte(direction))
		if board.Get(fromCol, fromRank) == pawn {
			return fromCol, fromRank
		}
		return 0, 0
	}

	// Non-capture - same column, one rank back
	fromRank := chess.Rank(byte(toRank) - byte(direction))
	if board.Get(toCol, fromRank) == pawn {
		return toCol, fromRank
	}

	// Double pawn push - two ranks back
	isDoublePushRank := (colour == chess.White && toRank == '4') || (colour == chess.Black && toRank == '5')
	if isDoublePushRank {
		fromRank = chess.Rank(byte(toRank) - byte(2*direction))
		middleRank := chess.Rank(byte(toRank) - byte(direction))
		if board.Get(toCol, fromRank) == pawn && board.Get(toCol, middleRank) == chess.Empty {
			return toCol, fromRank
		}
	}

	return 0, 0
}
