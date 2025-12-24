// Package engine provides chess move validation and board manipulation.
package engine

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/errors"
)

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
	}
	return chess.Empty
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
	board := chess.NewBoard()

	parts := strings.Fields(fen)
	if len(parts) < 1 {
		return nil, fmt.Errorf("empty FEN string: %w", errors.ErrInvalidFEN)
	}

	// Parse piece positions
	rank := chess.Rank('8')
	col := chess.Col('a')

	for _, c := range parts[0] {
		if c == '/' {
			rank--
			col = 'a'
			continue
		}

		if c >= '1' && c <= '8' {
			// Empty squares
			col += chess.Col(c - '0')
			continue
		}

		piece := ConvertFENCharToPiece(byte(c))
		if piece == chess.Empty {
			return nil, fmt.Errorf("invalid piece character: %c: %w", c, errors.ErrInvalidFEN)
		}

		var colour chess.Colour
		if unicode.IsUpper(c) {
			colour = chess.White
		} else {
			colour = chess.Black
		}

		if col > 'h' || rank < '1' {
			return nil, fmt.Errorf("position out of bounds: %w", errors.ErrInvalidFEN)
		}

		board.Set(col, rank, chess.MakeColouredPiece(colour, piece))

		// Track king positions
		if piece == chess.King {
			if colour == chess.White {
				board.WKingCol = col
				board.WKingRank = rank
			} else {
				board.BKingCol = col
				board.BKingRank = rank
			}
		}

		col++
	}

	// Parse side to move
	if len(parts) >= 2 {
		switch parts[1] {
		case "w":
			board.ToMove = chess.White
		case "b":
			board.ToMove = chess.Black
		default:
			return nil, fmt.Errorf("invalid side to move: %s: %w", parts[1], errors.ErrInvalidFEN)
		}
	}

	// Parse castling rights
	board.WKingCastle = 0
	board.WQueenCastle = 0
	board.BKingCastle = 0
	board.BQueenCastle = 0

	if len(parts) >= 3 && parts[2] != "-" {
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
		}
	}

	// Parse en passant square
	board.EnPassant = false
	if len(parts) >= 4 && parts[3] != "-" {
		if len(parts[3]) == 2 {
			board.EnPassant = true
			board.EPCol = chess.Col(parts[3][0])
			board.EPRank = chess.Rank(parts[3][1])
		}
	}

	// Parse halfmove clock
	if len(parts) >= 5 {
		fmt.Sscanf(parts[4], "%d", &board.HalfmoveClock)
	}

	// Parse fullmove number
	if len(parts) >= 6 {
		fmt.Sscanf(parts[5], "%d", &board.MoveNumber)
	}

	return board, nil
}

// BoardToFEN converts a board to a FEN string.
func BoardToFEN(board *chess.Board) string {
	var sb strings.Builder

	// Piece positions
	for rank := chess.Rank('8'); rank >= '1'; rank-- {
		emptyCount := 0
		for col := chess.Col('a'); col <= 'h'; col++ {
			piece := board.Get(col, rank)
			if piece == chess.Empty {
				emptyCount++
			} else {
				if emptyCount > 0 {
					sb.WriteByte(byte('0' + emptyCount))
					emptyCount = 0
				}
				sb.WriteByte(ColouredPieceToSANLetter(piece))
			}
		}
		if emptyCount > 0 {
			sb.WriteByte(byte('0' + emptyCount))
		}
		if rank > '1' {
			sb.WriteByte('/')
		}
	}

	sb.WriteByte(' ')

	// Side to move
	if board.ToMove == chess.White {
		sb.WriteByte('w')
	} else {
		sb.WriteByte('b')
	}

	sb.WriteByte(' ')

	// Castling rights
	castling := ""
	if board.WKingCastle != 0 {
		castling += "K"
	}
	if board.WQueenCastle != 0 {
		castling += "Q"
	}
	if board.BKingCastle != 0 {
		castling += "k"
	}
	if board.BQueenCastle != 0 {
		castling += "q"
	}
	if castling == "" {
		castling = "-"
	}
	sb.WriteString(castling)

	sb.WriteByte(' ')

	// En passant
	if board.EnPassant {
		sb.WriteByte(byte(board.EPCol))
		sb.WriteByte(byte(board.EPRank))
	} else {
		sb.WriteByte('-')
	}

	sb.WriteByte(' ')

	// Halfmove clock
	sb.WriteString(fmt.Sprintf("%d", board.HalfmoveClock))

	sb.WriteByte(' ')

	// Fullmove number
	sb.WriteString(fmt.Sprintf("%d", board.MoveNumber))

	return sb.String()
}

// InitialFEN is the FEN string for the standard starting position.
const InitialFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

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
