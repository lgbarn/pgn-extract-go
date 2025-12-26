package engine

import "github.com/lgbarn/pgn-extract-go/internal/chess"

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
