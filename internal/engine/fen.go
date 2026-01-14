// Package engine provides chess move validation and board manipulation.
package engine

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/errors"
)

// InitialFEN is the FEN string for the standard starting position.
const InitialFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// SAN piece characters for FEN strings (always English).
var sanPieceChars = map[chess.Piece]byte{
	chess.Pawn:   'P',
	chess.Knight: 'N',
	chess.Bishop: 'B',
	chess.Rook:   'R',
	chess.Queen:  'Q',
	chess.King:   'K',
}

// ConvertFENCharToPiece converts a FEN character to a piece type.
func ConvertFENCharToPiece(c byte) chess.Piece {
	switch c {
	case 'K', 'k':
		return chess.King
	case 'Q', 'q':
		return chess.Queen
	case 'R', 'r':
		return chess.Rook
	case 'N', 'n':
		return chess.Knight
	case 'B', 'b':
		return chess.Bishop
	case 'P', 'p':
		return chess.Pawn
	default:
		return chess.Empty
	}
}

// SANPieceLetter returns the SAN letter for a piece.
func SANPieceLetter(piece chess.Piece) byte {
	if c, ok := sanPieceChars[piece]; ok {
		return c
	}
	return '?'
}

// ColouredPieceToSANLetter returns the SAN letter for a coloured piece.
func ColouredPieceToSANLetter(colouredPiece chess.Piece) byte {
	piece := chess.ExtractPiece(colouredPiece)
	letter := SANPieceLetter(piece)
	if chess.ExtractColour(colouredPiece) == chess.Black {
		letter = byte(unicode.ToLower(rune(letter)))
	}
	return letter
}

// NewBoardFromFEN creates a board from a FEN string.
func NewBoardFromFEN(fen string) (*chess.Board, error) {
	parts := strings.Fields(fen)
	if len(parts) < 1 {
		return nil, fmt.Errorf("empty FEN string: %w", errors.ErrInvalidFEN)
	}

	board := chess.NewBoard()

	if err := parsePiecePositions(board, parts[0]); err != nil {
		return nil, err
	}

	if err := parseSideToMove(board, parts); err != nil {
		return nil, err
	}

	parseCastlingRights(board, parts)
	parseEnPassant(board, parts)
	parseClocks(board, parts)

	return board, nil
}

// parsePiecePositions parses the piece placement field of a FEN string.
func parsePiecePositions(board *chess.Board, positions string) error {
	rank := chess.Rank('8')
	col := chess.Col('a')

	for _, c := range positions {
		switch {
		case c == '/':
			rank--
			col = 'a'
		case c >= '1' && c <= '8':
			col += chess.Col(c - '0')
		default:
			piece := ConvertFENCharToPiece(byte(c))
			if piece == chess.Empty {
				return fmt.Errorf("invalid piece character: %c: %w", c, errors.ErrInvalidFEN)
			}
			if col > 'h' || rank < '1' {
				return fmt.Errorf("position out of bounds: %w", errors.ErrInvalidFEN)
			}

			colour := chess.White
			if unicode.IsLower(c) {
				colour = chess.Black
			}

			board.Set(col, rank, chess.MakeColouredPiece(colour, piece))

			if piece == chess.King {
				if colour == chess.White {
					board.WKingCol, board.WKingRank = col, rank
				} else {
					board.BKingCol, board.BKingRank = col, rank
				}
			}
			col++
		}
	}
	return nil
}

// parseSideToMove parses the side to move field.
func parseSideToMove(board *chess.Board, parts []string) error {
	if len(parts) < 2 {
		return nil
	}
	switch parts[1] {
	case "w":
		board.ToMove = chess.White
	case "b":
		board.ToMove = chess.Black
	default:
		return fmt.Errorf("invalid side to move: %s: %w", parts[1], errors.ErrInvalidFEN)
	}
	return nil
}

// parseCastlingRights parses the castling availability field.
func parseCastlingRights(board *chess.Board, parts []string) {
	board.WKingCastle = 0
	board.WQueenCastle = 0
	board.BKingCastle = 0
	board.BQueenCastle = 0

	if len(parts) < 3 || parts[2] == "-" {
		return
	}

	for _, c := range parts[2] {
		switch c {
		case 'K':
			board.WKingCastle = 'h'
		case 'Q':
			board.WQueenCastle = 'a'
		case 'k':
			board.BKingCastle = 'h'
		case 'q':
			board.BQueenCastle = 'a'
		default:
			// Chess960 notation - column letter
			parseCastling960(board, c)
		}
	}
}

// parseCastling960 handles Chess960 castling notation.
func parseCastling960(board *chess.Board, c rune) {
	if c >= 'A' && c <= 'H' {
		col := chess.Col(unicode.ToLower(c))
		if col > board.WKingCol {
			board.WKingCastle = col
		} else {
			board.WQueenCastle = col
		}
	} else if c >= 'a' && c <= 'h' {
		col := chess.Col(c)
		if col > board.BKingCol {
			board.BKingCastle = col
		} else {
			board.BQueenCastle = col
		}
	}
}

// parseEnPassant parses the en passant target square field.
func parseEnPassant(board *chess.Board, parts []string) {
	board.EnPassant = false
	if len(parts) < 4 || parts[3] == "-" || len(parts[3]) != 2 {
		return
	}
	board.EnPassant = true
	board.EPCol = chess.Col(parts[3][0])
	board.EPRank = chess.Rank(parts[3][1])
}

// parseClocks parses the halfmove clock and fullmove number fields.
func parseClocks(board *chess.Board, parts []string) {
	if len(parts) >= 5 {
		fmt.Sscanf(parts[4], "%d", &board.HalfmoveClock)
	}
	if len(parts) >= 6 {
		fmt.Sscanf(parts[5], "%d", &board.MoveNumber)
	}
}

// BoardToFEN converts a board to a FEN string.
func BoardToFEN(board *chess.Board) string {
	var sb strings.Builder

	writePiecePositions(&sb, board)
	sb.WriteByte(' ')
	writeSideToMove(&sb, board)
	sb.WriteByte(' ')
	writeCastlingRights(&sb, board)
	sb.WriteByte(' ')
	writeEnPassant(&sb, board)
	sb.WriteByte(' ')
	fmt.Fprintf(&sb, "%d %d", board.HalfmoveClock, board.MoveNumber)

	return sb.String()
}

// writePiecePositions writes the piece placement to the builder.
func writePiecePositions(sb *strings.Builder, board *chess.Board) {
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

// writeSideToMove writes the side to move to the builder.
func writeSideToMove(sb *strings.Builder, board *chess.Board) {
	if board.ToMove == chess.White {
		sb.WriteByte('w')
	} else {
		sb.WriteByte('b')
	}
}

// writeCastlingRights writes the castling availability to the builder.
func writeCastlingRights(sb *strings.Builder, board *chess.Board) {
	hasCastling := false
	if board.WKingCastle != 0 {
		sb.WriteByte('K')
		hasCastling = true
	}
	if board.WQueenCastle != 0 {
		sb.WriteByte('Q')
		hasCastling = true
	}
	if board.BKingCastle != 0 {
		sb.WriteByte('k')
		hasCastling = true
	}
	if board.BQueenCastle != 0 {
		sb.WriteByte('q')
		hasCastling = true
	}
	if !hasCastling {
		sb.WriteByte('-')
	}
}

// writeEnPassant writes the en passant target square to the builder.
func writeEnPassant(sb *strings.Builder, board *chess.Board) {
	if board.EnPassant {
		sb.WriteByte(byte(board.EPCol))
		sb.WriteByte(byte(board.EPRank))
	} else {
		sb.WriteByte('-')
	}
}

// NewInitialBoard creates a board with the standard starting position.
func NewInitialBoard() *chess.Board {
	board, _ := NewBoardFromFEN(InitialFEN)
	return board
}

// NewBoardForGame creates a board for a game, using FEN tag if present.
// Falls back to initial position if FEN is missing or invalid.
func NewBoardForGame(game *chess.Game) *chess.Board {
	if fen, ok := game.Tags["FEN"]; ok {
		if board, err := NewBoardFromFEN(fen); err == nil {
			return board
		}
	}
	return NewInitialBoard()
}
