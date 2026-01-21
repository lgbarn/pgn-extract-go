// Package engine provides chess move validation and board manipulation.
package engine

import (
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// IsChess960Game returns true if the game is a Chess960 game.
// This is detected by the Variant tag or non-standard castling rights.
func IsChess960Game(game *chess.Game) bool {
	variant := game.GetTag("Variant")
	variant = strings.ToLower(variant)
	if strings.Contains(variant, "960") || strings.Contains(variant, "fischerandom") {
		return true
	}
	return false
}

// IsChess960Position returns true if the board has non-standard castling positions.
func IsChess960Position(board *chess.Board) bool {
	// Standard positions: king on e-file, rooks on a and h files
	standardKingCol := chess.Col('e')
	standardKingSideRook := chess.Col('h')
	standardQueenSideRook := chess.Col('a')

	// Check if white has non-standard castling
	if board.WKingCol != standardKingCol {
		return true
	}
	if board.WKingCastle != 0 && board.WKingCastle != standardKingSideRook {
		return true
	}
	if board.WQueenCastle != 0 && board.WQueenCastle != standardQueenSideRook {
		return true
	}

	// Check if black has non-standard castling
	if board.BKingCol != standardKingCol {
		return true
	}
	if board.BKingCastle != 0 && board.BKingCastle != standardKingSideRook {
		return true
	}
	if board.BQueenCastle != 0 && board.BQueenCastle != standardQueenSideRook {
		return true
	}

	return false
}

// BoardToShredderFEN converts a board to a FEN string using Shredder notation for castling.
// This is used for Chess960 games where castling rights are indicated by rook file letters.
func BoardToShredderFEN(board *chess.Board) string {
	var sb strings.Builder

	writePiecePositionsToBuilder(&sb, board)
	sb.WriteByte(' ')
	writeSideToMoveToBuilder(&sb, board)
	sb.WriteByte(' ')
	writeShredderCastlingRights(&sb, board)
	sb.WriteByte(' ')
	writeEnPassantToBuilder(&sb, board)
	sb.WriteString(" ")
	sb.WriteString(formatClocks(board))

	return sb.String()
}

// writePiecePositionsToBuilder writes piece positions to a string builder.
func writePiecePositionsToBuilder(sb *strings.Builder, board *chess.Board) {
	for rank := chess.Rank('8'); rank >= '1'; rank-- {
		emptyCount := 0
		for col := chess.Col('a'); col <= 'h'; col++ {
			piece := board.Get(col, rank)
			if piece == chess.Empty {
				emptyCount++
				continue
			}
			if emptyCount > 0 {
				sb.WriteByte(byte('0' + emptyCount))
				emptyCount = 0
			}
			sb.WriteByte(ColouredPieceToSANLetter(piece))
		}
		if emptyCount > 0 {
			sb.WriteByte(byte('0' + emptyCount))
		}
		if rank > '1' {
			sb.WriteByte('/')
		}
	}
}

// writeSideToMoveToBuilder writes the side to move.
func writeSideToMoveToBuilder(sb *strings.Builder, board *chess.Board) {
	if board.ToMove == chess.White {
		sb.WriteByte('w')
	} else {
		sb.WriteByte('b')
	}
}

// writeShredderCastlingRights writes castling rights using Shredder notation (file letters).
func writeShredderCastlingRights(sb *strings.Builder, board *chess.Board) {
	hasCastling := false

	// White castling rights (uppercase file letters)
	if board.WKingCastle != 0 {
		sb.WriteByte(byte(board.WKingCastle - 'a' + 'A'))
		hasCastling = true
	}
	if board.WQueenCastle != 0 {
		sb.WriteByte(byte(board.WQueenCastle - 'a' + 'A'))
		hasCastling = true
	}

	// Black castling rights (lowercase file letters)
	if board.BKingCastle != 0 {
		sb.WriteByte(byte(board.BKingCastle))
		hasCastling = true
	}
	if board.BQueenCastle != 0 {
		sb.WriteByte(byte(board.BQueenCastle))
		hasCastling = true
	}

	if !hasCastling {
		sb.WriteByte('-')
	}
}

// writeEnPassantToBuilder writes the en passant square.
func writeEnPassantToBuilder(sb *strings.Builder, board *chess.Board) {
	if board.EnPassant {
		sb.WriteByte(byte(board.EPCol))
		sb.WriteByte(byte(board.EPRank))
	} else {
		sb.WriteByte('-')
	}
}

// formatClocks formats the halfmove clock and move number.
func formatClocks(board *chess.Board) string {
	var sb strings.Builder
	// Write halfmove clock
	writeUint(&sb, board.HalfmoveClock)
	sb.WriteByte(' ')
	// Write move number
	writeUint(&sb, board.MoveNumber)
	return sb.String()
}

// writeUint writes an unsigned integer to a string builder.
func writeUint(sb *strings.Builder, n uint) {
	if n == 0 {
		sb.WriteByte('0')
		return
	}
	// Convert to string manually to avoid fmt import
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	// Reverse
	for i := len(digits) - 1; i >= 0; i-- {
		sb.WriteByte(digits[i])
	}
}

// GetFENForGame returns the appropriate FEN string for a game.
// Uses Shredder notation for Chess960 games, standard notation otherwise.
func GetFENForGame(board *chess.Board, game *chess.Game, forceChess960 bool) string {
	if forceChess960 || IsChess960Game(game) || IsChess960Position(board) {
		return BoardToShredderFEN(board)
	}
	return BoardToFEN(board)
}
