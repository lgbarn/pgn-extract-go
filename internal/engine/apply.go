package engine

import (
	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// ApplyMove applies a move to the board and updates the board state.
// Returns true if the move was applied successfully.
func ApplyMove(board *chess.Board, move *chess.Move) bool {
	if move == nil {
		return false
	}

	switch move.Class {
	case chess.NullMove:
		// Just switch sides
		board.ToMove = board.ToMove.Opposite()
		board.EnPassant = false
		return true

	case chess.KingsideCastle:
		return applyCastle(board, true)

	case chess.QueensideCastle:
		return applyCastle(board, false)

	case chess.PawnMove, chess.PawnMoveWithPromotion, chess.EnPassantPawnMove:
		return applyPawnMove(board, move)

	case chess.PieceMove:
		return applyPieceMove(board, move)

	default:
		return false
	}
}

// applyCastle applies a castling move.
func applyCastle(board *chess.Board, kingside bool) bool {
	colour := board.ToMove
	var rank chess.Rank
	var kingFromCol, kingToCol, rookFromCol, rookToCol chess.Col

	if colour == chess.White {
		rank = '1'
		kingFromCol = board.WKingCol
		if kingside {
			kingToCol = 'g'
			rookFromCol = board.WKingCastle
			rookToCol = 'f'
		} else {
			kingToCol = 'c'
			rookFromCol = board.WQueenCastle
			rookToCol = 'd'
		}
	} else {
		rank = '8'
		kingFromCol = board.BKingCol
		if kingside {
			kingToCol = 'g'
			rookFromCol = board.BKingCastle
			rookToCol = 'f'
		} else {
			kingToCol = 'c'
			rookFromCol = board.BQueenCastle
			rookToCol = 'd'
		}
	}

	// Move king
	king := board.Get(kingFromCol, rank)
	board.Set(kingFromCol, rank, chess.Empty)
	board.Set(kingToCol, rank, king)

	// Move rook
	rook := board.Get(rookFromCol, rank)
	board.Set(rookFromCol, rank, chess.Empty)
	board.Set(rookToCol, rank, rook)

	// Update king position
	if colour == chess.White {
		board.WKingCol = kingToCol
		board.WKingCastle = 0
		board.WQueenCastle = 0
	} else {
		board.BKingCol = kingToCol
		board.BKingCastle = 0
		board.BQueenCastle = 0
	}

	board.EnPassant = false
	board.HalfmoveClock++
	if colour == chess.Black {
		board.MoveNumber++
	}
	board.ToMove = colour.Opposite()

	return true
}

// applyPawnMove applies a pawn move.
func applyPawnMove(board *chess.Board, move *chess.Move) bool {
	colour := board.ToMove
	fromCol := move.FromCol
	fromRank := move.FromRank
	toCol := move.ToCol
	toRank := move.ToRank

	// If source square not specified, find the pawn
	if fromCol == 0 || fromRank == 0 {
		fromCol, fromRank = findPawnSource(board, move, colour)
		if fromCol == 0 {
			return false
		}
	}

	pawn := board.Get(fromCol, fromRank)

	// Handle en passant capture
	if move.Class == chess.EnPassantPawnMove {
		// Remove the captured pawn
		var capturedRank chess.Rank
		if colour == chess.White {
			capturedRank = toRank - 1
		} else {
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
			promotedPiece = chess.Queen // Default to queen
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

	board.HalfmoveClock = 0 // Pawn move resets clock
	if colour == chess.Black {
		board.MoveNumber++
	}
	board.ToMove = colour.Opposite()

	return true
}

// applyPieceMove applies a piece (non-pawn) move.
func applyPieceMove(board *chess.Board, move *chess.Move) bool {
	colour := board.ToMove
	fromCol := move.FromCol
	fromRank := move.FromRank
	toCol := move.ToCol
	toRank := move.ToRank
	pieceType := move.PieceToMove

	// If source square not specified, find the piece
	if fromCol == 0 || fromRank == 0 {
		fromCol, fromRank = findPieceSource(board, move, colour)
		if fromCol == 0 {
			return false
		}
	}

	piece := board.Get(fromCol, fromRank)
	capturedPiece := board.Get(toCol, toRank)

	// Move the piece
	board.Set(fromCol, fromRank, chess.Empty)
	board.Set(toCol, toRank, piece)

	// Update king position if king moved
	if pieceType == chess.King {
		if colour == chess.White {
			board.WKingCol = toCol
			board.WKingRank = toRank
			board.WKingCastle = 0
			board.WQueenCastle = 0
		} else {
			board.BKingCol = toCol
			board.BKingRank = toRank
			board.BKingCastle = 0
			board.BQueenCastle = 0
		}
	}

	// Update castling rights if rook moved or captured
	if pieceType == chess.Rook {
		updateCastlingRightsForRook(board, colour, fromCol, fromRank)
	}
	if capturedPiece != chess.Empty && chess.ExtractPiece(capturedPiece) == chess.Rook {
		capturedColour := chess.ExtractColour(capturedPiece)
		updateCastlingRightsForRook(board, capturedColour, toCol, toRank)
	}

	board.EnPassant = false

	// Update halfmove clock
	if capturedPiece != chess.Empty {
		board.HalfmoveClock = 0
	} else {
		board.HalfmoveClock++
	}

	if colour == chess.Black {
		board.MoveNumber++
	}
	board.ToMove = colour.Opposite()

	return true
}

// findPawnSource finds the source square of a pawn move.
func findPawnSource(board *chess.Board, move *chess.Move, colour chess.Colour) (chess.Col, chess.Rank) {
	toCol := move.ToCol
	toRank := move.ToRank
	fromCol := move.FromCol

	pawn := chess.MakeColouredPiece(colour, chess.Pawn)
	direction := chess.ColourOffset(colour)

	// If we know the from column, look for the pawn there
	if fromCol != 0 {
		// Capture - look one rank back
		fromRank := chess.Rank(byte(toRank) - byte(direction))
		if board.Get(fromCol, fromRank) == pawn {
			return fromCol, fromRank
		}
		return 0, 0
	}

	// Non-capture - same column
	fromRank := chess.Rank(byte(toRank) - byte(direction))
	if board.Get(toCol, fromRank) == pawn {
		return toCol, fromRank
	}

	// Double pawn push
	if (colour == chess.White && toRank == '4') || (colour == chess.Black && toRank == '5') {
		fromRank = chess.Rank(byte(toRank) - byte(2*direction))
		middleRank := chess.Rank(byte(toRank) - byte(direction))
		if board.Get(toCol, fromRank) == pawn && board.Get(toCol, middleRank) == chess.Empty {
			return toCol, fromRank
		}
	}

	return 0, 0
}

// findPieceSource finds the source square of a piece move.
func findPieceSource(board *chess.Board, move *chess.Move, colour chess.Colour) (chess.Col, chess.Rank) {
	toCol := move.ToCol
	toRank := move.ToRank
	pieceType := move.PieceToMove
	fromCol := move.FromCol
	fromRank := move.FromRank

	piece := chess.MakeColouredPiece(colour, pieceType)

	// Search for the piece that can move to the target square
	for col := chess.Col('a'); col <= 'h'; col++ {
		for rank := chess.Rank('1'); rank <= '8'; rank++ {
			if board.Get(col, rank) != piece {
				continue
			}

			// Check disambiguation
			if fromCol != 0 && col != fromCol {
				continue
			}
			if fromRank != 0 && rank != fromRank {
				continue
			}

			// Check if this piece can reach the target
			if canPieceMove(board, pieceType, col, rank, toCol, toRank) {
				return col, rank
			}
		}
	}

	return 0, 0
}

// canPieceMove checks if a piece can move from one square to another.
func canPieceMove(board *chess.Board, pieceType chess.Piece, fromCol chess.Col, fromRank chess.Rank, toCol chess.Col, toRank chess.Rank) bool {
	colDiff := abs(int(toCol) - int(fromCol))
	rankDiff := abs(int(toRank) - int(fromRank))

	switch pieceType {
	case chess.Knight:
		return (colDiff == 1 && rankDiff == 2) || (colDiff == 2 && rankDiff == 1)

	case chess.Bishop:
		if colDiff != rankDiff {
			return false
		}
		return isDiagonalClear(board, fromCol, fromRank, toCol, toRank)

	case chess.Rook:
		if colDiff != 0 && rankDiff != 0 {
			return false
		}
		return isStraightClear(board, fromCol, fromRank, toCol, toRank)

	case chess.Queen:
		if colDiff == rankDiff {
			return isDiagonalClear(board, fromCol, fromRank, toCol, toRank)
		}
		if colDiff == 0 || rankDiff == 0 {
			return isStraightClear(board, fromCol, fromRank, toCol, toRank)
		}
		return false

	case chess.King:
		return colDiff <= 1 && rankDiff <= 1
	}

	return false
}

// isDiagonalClear checks if the diagonal path is clear.
func isDiagonalClear(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, toCol chess.Col, toRank chess.Rank) bool {
	colDir := sign(int(toCol) - int(fromCol))
	rankDir := sign(int(toRank) - int(fromRank))

	col := chess.Col(int(fromCol) + colDir)
	rank := chess.Rank(int(fromRank) + rankDir)

	for col != toCol && rank != toRank {
		if board.Get(col, rank) != chess.Empty {
			return false
		}
		col = chess.Col(int(col) + colDir)
		rank = chess.Rank(int(rank) + rankDir)
	}

	return true
}

// isStraightClear checks if the straight path is clear.
func isStraightClear(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, toCol chess.Col, toRank chess.Rank) bool {
	colDir := sign(int(toCol) - int(fromCol))
	rankDir := sign(int(toRank) - int(fromRank))

	col := chess.Col(int(fromCol) + colDir)
	rank := chess.Rank(int(fromRank) + rankDir)

	for col != toCol || rank != toRank {
		if board.Get(col, rank) != chess.Empty {
			return false
		}
		col = chess.Col(int(col) + colDir)
		rank = chess.Rank(int(rank) + rankDir)
	}

	return true
}

// updateCastlingRightsForRook removes castling rights when a rook moves or is captured.
func updateCastlingRightsForRook(board *chess.Board, colour chess.Colour, col chess.Col, rank chess.Rank) {
	if colour == chess.White && rank == '1' {
		if col == board.WKingCastle {
			board.WKingCastle = 0
		}
		if col == board.WQueenCastle {
			board.WQueenCastle = 0
		}
	} else if colour == chess.Black && rank == '8' {
		if col == board.BKingCastle {
			board.BKingCastle = 0
		}
		if col == board.BQueenCastle {
			board.BQueenCastle = 0
		}
	}
}

// IsInCheck returns true if the given colour's king is in check.
func IsInCheck(board *chess.Board, colour chess.Colour) bool {
	// Find the king
	var kingCol chess.Col
	var kingRank chess.Rank
	if colour == chess.White {
		kingCol = board.WKingCol
		kingRank = board.WKingRank
	} else {
		kingCol = board.BKingCol
		kingRank = board.BKingRank
	}

	// If king position not tracked, search for it
	if kingCol == 0 || kingRank == 0 {
		kingCol, kingRank = findKing(board, colour)
		if kingCol == 0 {
			return false // No king found
		}
	}

	return isSquareAttacked(board, kingCol, kingRank, colour.Opposite())
}

// findKing finds the king of the given colour on the board.
func findKing(board *chess.Board, colour chess.Colour) (chess.Col, chess.Rank) {
	king := chess.MakeColouredPiece(colour, chess.King)
	for col := chess.Col('a'); col <= 'h'; col++ {
		for rank := chess.Rank('1'); rank <= '8'; rank++ {
			if board.Get(col, rank) == king {
				return col, rank
			}
		}
	}
	return 0, 0
}

// isSquareAttacked returns true if the square is attacked by the given colour.
func isSquareAttacked(board *chess.Board, col chess.Col, rank chess.Rank, byColour chess.Colour) bool {
	// Check pawn attacks
	pawn := chess.MakeColouredPiece(byColour, chess.Pawn)
	var pawnDir int
	if byColour == chess.White {
		pawnDir = -1 // White pawns attack from below
	} else {
		pawnDir = 1 // Black pawns attack from above
	}
	pawnRank := chess.Rank(int(rank) + pawnDir)
	if pawnRank >= '1' && pawnRank <= '8' {
		if col > 'a' && board.Get(col-1, pawnRank) == pawn {
			return true
		}
		if col < 'h' && board.Get(col+1, pawnRank) == pawn {
			return true
		}
	}

	// Check knight attacks
	knight := chess.MakeColouredPiece(byColour, chess.Knight)
	knightMoves := [][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}
	for _, move := range knightMoves {
		c := chess.Col(int(col) + move[0])
		r := chess.Rank(int(rank) + move[1])
		if c >= 'a' && c <= 'h' && r >= '1' && r <= '8' {
			if board.Get(c, r) == knight {
				return true
			}
		}
	}

	// Check king attacks
	king := chess.MakeColouredPiece(byColour, chess.King)
	for dc := -1; dc <= 1; dc++ {
		for dr := -1; dr <= 1; dr++ {
			if dc == 0 && dr == 0 {
				continue
			}
			c := chess.Col(int(col) + dc)
			r := chess.Rank(int(rank) + dr)
			if c >= 'a' && c <= 'h' && r >= '1' && r <= '8' {
				if board.Get(c, r) == king {
					return true
				}
			}
		}
	}

	// Check sliding pieces (bishop, rook, queen) along diagonals
	bishop := chess.MakeColouredPiece(byColour, chess.Bishop)
	queen := chess.MakeColouredPiece(byColour, chess.Queen)
	diagonalDirs := [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
	for _, dir := range diagonalDirs {
		c := chess.Col(int(col) + dir[0])
		r := chess.Rank(int(rank) + dir[1])
		for c >= 'a' && c <= 'h' && r >= '1' && r <= '8' {
			piece := board.Get(c, r)
			if piece != chess.Empty {
				if piece == bishop || piece == queen {
					return true
				}
				break // Blocked
			}
			c = chess.Col(int(c) + dir[0])
			r = chess.Rank(int(r) + dir[1])
		}
	}

	// Check sliding pieces along straight lines
	rook := chess.MakeColouredPiece(byColour, chess.Rook)
	straightDirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for _, dir := range straightDirs {
		c := chess.Col(int(col) + dir[0])
		r := chess.Rank(int(rank) + dir[1])
		for c >= 'a' && c <= 'h' && r >= '1' && r <= '8' {
			piece := board.Get(c, r)
			if piece != chess.Empty {
				if piece == rook || piece == queen {
					return true
				}
				break // Blocked
			}
			c = chess.Col(int(c) + dir[0])
			r = chess.Rank(int(r) + dir[1])
		}
	}

	return false
}

// HasLegalMoves returns true if the given colour has at least one legal move.
func HasLegalMoves(board *chess.Board, colour chess.Colour) bool {
	for col := chess.Col('a'); col <= 'h'; col++ {
		for rank := chess.Rank('1'); rank <= '8'; rank++ {
			piece := board.Get(col, rank)
			if piece == chess.Empty || piece == chess.Off {
				continue
			}
			if chess.ExtractColour(piece) != colour {
				continue
			}

			pieceType := chess.ExtractPiece(piece)
			if hasLegalMovesForPiece(board, col, rank, pieceType, colour) {
				return true
			}
		}
	}
	return false
}

// hasLegalMovesForPiece checks if a specific piece has any legal moves.
func hasLegalMovesForPiece(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, pieceType chess.Piece, colour chess.Colour) bool {
	// Generate potential target squares based on piece type
	var targets [][2]int

	switch pieceType {
	case chess.Pawn:
		dir := chess.ColourOffset(colour)
		// Forward move
		toRank := chess.Rank(int(fromRank) + dir)
		if toRank >= '1' && toRank <= '8' && board.Get(fromCol, toRank) == chess.Empty {
			if tryMove(board, fromCol, fromRank, fromCol, toRank, colour) {
				return true
			}
			// Double push from starting rank
			startRank := chess.Rank('2')
			if colour == chess.Black {
				startRank = '7'
			}
			if fromRank == startRank {
				toRank2 := chess.Rank(int(fromRank) + 2*dir)
				if board.Get(fromCol, toRank2) == chess.Empty {
					if tryMove(board, fromCol, fromRank, fromCol, toRank2, colour) {
						return true
					}
				}
			}
		}
		// Captures
		for dc := -1; dc <= 1; dc += 2 {
			toCol := chess.Col(int(fromCol) + dc)
			toRank := chess.Rank(int(fromRank) + dir)
			if toCol >= 'a' && toCol <= 'h' && toRank >= '1' && toRank <= '8' {
				target := board.Get(toCol, toRank)
				if target != chess.Empty && chess.ExtractColour(target) != colour {
					if tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
						return true
					}
				}
				// En passant
				if board.EnPassant && toCol == board.EPCol && toRank == board.EPRank {
					if tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
						return true
					}
				}
			}
		}
		return false

	case chess.Knight:
		targets = [][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}

	case chess.King:
		targets = [][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}

	case chess.Bishop:
		return hasSlidingMoves(board, fromCol, fromRank, colour, true, false)

	case chess.Rook:
		return hasSlidingMoves(board, fromCol, fromRank, colour, false, true)

	case chess.Queen:
		return hasSlidingMoves(board, fromCol, fromRank, colour, true, true)
	}

	// Check each target for knight and king
	for _, offset := range targets {
		toCol := chess.Col(int(fromCol) + offset[0])
		toRank := chess.Rank(int(fromRank) + offset[1])
		if toCol >= 'a' && toCol <= 'h' && toRank >= '1' && toRank <= '8' {
			target := board.Get(toCol, toRank)
			if target == chess.Empty || chess.ExtractColour(target) != colour {
				if tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
					return true
				}
			}
		}
	}

	return false
}

// hasSlidingMoves checks if a sliding piece (bishop, rook, queen) has legal moves.
func hasSlidingMoves(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, colour chess.Colour, diagonal, straight bool) bool {
	var dirs [][2]int
	if diagonal {
		dirs = append(dirs, [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}...)
	}
	if straight {
		dirs = append(dirs, [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}...)
	}

	for _, dir := range dirs {
		toCol := chess.Col(int(fromCol) + dir[0])
		toRank := chess.Rank(int(fromRank) + dir[1])
		for toCol >= 'a' && toCol <= 'h' && toRank >= '1' && toRank <= '8' {
			target := board.Get(toCol, toRank)
			if target != chess.Empty {
				if chess.ExtractColour(target) != colour {
					if tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
						return true
					}
				}
				break // Blocked
			}
			if tryMove(board, fromCol, fromRank, toCol, toRank, colour) {
				return true
			}
			toCol = chess.Col(int(toCol) + dir[0])
			toRank = chess.Rank(int(toRank) + dir[1])
		}
	}
	return false
}

// tryMove makes a move on a copied board and checks if it leaves the king in check.
func tryMove(board *chess.Board, fromCol chess.Col, fromRank chess.Rank, toCol chess.Col, toRank chess.Rank, colour chess.Colour) bool {
	// Make a copy of the board
	testBoard := board.Copy()

	// Make the move
	piece := testBoard.Get(fromCol, fromRank)
	testBoard.Set(fromCol, fromRank, chess.Empty)
	testBoard.Set(toCol, toRank, piece)

	// Update king position if needed
	if chess.ExtractPiece(piece) == chess.King {
		if colour == chess.White {
			testBoard.WKingCol = toCol
			testBoard.WKingRank = toRank
		} else {
			testBoard.BKingCol = toCol
			testBoard.BKingRank = toRank
		}
	}

	// Check if our king is in check after the move
	return !IsInCheck(testBoard, colour)
}

// IsCheckmate returns true if the position is checkmate for the side to move.
func IsCheckmate(board *chess.Board) bool {
	colour := board.ToMove
	return IsInCheck(board, colour) && !HasLegalMoves(board, colour)
}

// IsStalemate returns true if the position is stalemate for the side to move.
func IsStalemate(board *chess.Board) bool {
	colour := board.ToMove
	return !IsInCheck(board, colour) && !HasLegalMoves(board, colour)
}

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sign(x int) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}
