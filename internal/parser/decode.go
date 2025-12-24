package parser

import (
	"strings"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// isCol returns true if c is a valid column (file) character.
func isCol(c byte) bool {
	return c >= chess.FirstCol && c <= chess.LastCol
}

// isRank returns true if c is a valid rank character.
func isRank(c byte) bool {
	return c >= chess.FirstRank && c <= chess.LastRank
}

// isPiece returns the piece type represented by the character(s) at the start of move.
func isPiece(move string) chess.Piece {
	if len(move) == 0 {
		return chess.Empty
	}

	switch move[0] {
	case 'K', 'k':
		return chess.King
	case 'Q', 'q', 'D': // D = Dutch/German Queen
		return chess.Queen
	case 'R', 'r', 'T': // T = Dutch/German Rook
		return chess.Rook
	case 'N', 'n', 'P', 'S': // P = Dutch Knight, S = German Knight
		return chess.Knight
	case 'B', 'L': // L = Dutch/German Bishop
		// Note: lowercase 'b' is most likely a pawn reference
		return chess.Bishop
	case RussianQueen:
		return chess.Queen
	case RussianRook:
		return chess.Rook
	case RussianBishop:
		return chess.Bishop
	case RussianKnightOrKing:
		// Check for two-character Russian King
		if len(move) > 1 && move[1] == RussianKingSecondLetter {
			return chess.King
		}
		return chess.Knight
	}
	return chess.Empty
}

// isCapture returns true if c is a capture or separator character.
func isCapture(c byte) bool {
	return c == 'x' || c == 'X' || c == ':' || c == '-'
}

// isCastlingChar returns true if c is a castling character.
func isCastlingChar(c byte) bool {
	return c == 'O' || c == '0' || c == 'o'
}

// isCheck returns true if c is a check indicator.
func isCheck(c byte) bool {
	return c == '+' || c == '#'
}

// DecodeMove parses a move string and returns a Move structure with decoded information.
func DecodeMove(moveString string) *chess.Move {
	move := chess.NewMove()
	move.Text = moveString

	var fromRank, toRank chess.Rank
	var fromCol, toCol chess.Col
	var class chess.MoveClass
	ok := true

	// Temporary locations
	var col chess.Col
	var rank chess.Rank

	pos := 0
	pieceToMove := chess.Empty
	promotedPiece := chess.Empty

	// Get current character helper
	currentChar := func() byte {
		if pos >= len(moveString) {
			return 0
		}
		return moveString[pos]
	}

	advance := func() {
		if pos < len(moveString) {
			pos++
		}
	}

	remaining := func() string {
		if pos >= len(moveString) {
			return ""
		}
		return moveString[pos:]
	}

	// Make an initial distinction between pawn moves and piece moves
	if isCol(currentChar()) {
		// Pawn move
		class = chess.PawnMove
		pieceToMove = chess.Pawn
		col = chess.Col(currentChar())
		advance()

		if isRank(currentChar()) {
			// e4, e2e4
			rank = chess.Rank(currentChar())
			advance()

			if isCapture(currentChar()) {
				advance()
			}

			if isCol(currentChar()) {
				fromCol = col
				fromRank = rank
				toCol = chess.Col(currentChar())
				advance()

				if isRank(currentChar()) {
					toRank = chess.Rank(currentChar())
					advance()
				}
			} else {
				toCol = col
				toRank = rank
			}
		} else {
			if isCapture(currentChar()) {
				// axb
				advance()
			}

			if isCol(currentChar()) {
				// ab, or bg8
				fromCol = col
				toCol = chess.Col(currentChar())
				advance()

				if isRank(currentChar()) {
					toRank = chess.Rank(currentChar())
					advance()

					// Sanity check
					if fromCol != 'b' && fromCol != chess.Col(byte(toCol)+1) && fromCol != chess.Col(byte(toCol)-1) {
						ok = false
					}
				} else {
					// Sanity check
					if fromCol != chess.Col(byte(toCol)+1) && fromCol != chess.Col(byte(toCol)-1) {
						ok = false
					}
				}
			} else {
				ok = false
			}
		}

		if ok {
			// Look for promotions
			if currentChar() == '=' {
				advance()
			}
			// Allow trailing 'b' as Bishop promotion
			if piece := isPiece(remaining()); piece != chess.Empty {
				class = chess.PawnMoveWithPromotion
				promotedPiece = piece
				advance()
			} else if currentChar() == 'b' {
				class = chess.PawnMoveWithPromotion
				promotedPiece = chess.Bishop
				advance()
			}
		}
	} else if pieceToMove = isPiece(remaining()); pieceToMove != chess.Empty {
		class = chess.PieceMove

		// Check for two-character Russian King
		if currentChar() == RussianKnightOrKing && pieceToMove == chess.King {
			advance()
		}
		advance()

		if isRank(currentChar()) {
			// Disambiguating rank: R1e1, R1xe3
			fromRank = chess.Rank(currentChar())
			advance()

			if isCapture(currentChar()) {
				advance()
			}

			if isCol(currentChar()) {
				toCol = chess.Col(currentChar())
				advance()

				if isRank(currentChar()) {
					toRank = chess.Rank(currentChar())
					advance()
				}
			} else {
				ok = false
			}
		} else {
			if isCapture(currentChar()) {
				// Rxe1
				advance()

				if isCol(currentChar()) {
					toCol = chess.Col(currentChar())
					advance()

					if isRank(currentChar()) {
						toRank = chess.Rank(currentChar())
						advance()
					} else {
						ok = false
					}
				} else {
					ok = false
				}
			} else if isCol(currentChar()) {
				col = chess.Col(currentChar())
				advance()

				if isCapture(currentChar()) {
					advance()
				}

				if isRank(currentChar()) {
					// Re1, Re1d1, Re1xd1
					rank = chess.Rank(currentChar())
					advance()

					if isCapture(currentChar()) {
						advance()
					}

					if isCol(currentChar()) {
						// Re1d1
						fromCol = col
						fromRank = rank
						toCol = chess.Col(currentChar())
						advance()

						if isRank(currentChar()) {
							toRank = chess.Rank(currentChar())
							advance()
						} else {
							ok = false
						}
					} else {
						toCol = col
						toRank = rank
					}
				} else if isCol(currentChar()) {
					// Rae1
					fromCol = col
					toCol = chess.Col(currentChar())
					advance()

					if isRank(currentChar()) {
						toRank = chess.Rank(currentChar())
						advance()
					} else {
						ok = false
					}
				} else {
					ok = false
				}
			} else {
				ok = false
			}
		}
	} else if isCastlingChar(currentChar()) {
		// Castling
		advance()

		// Allow optional separator
		if currentChar() == '-' {
			advance()
		}

		if isCastlingChar(currentChar()) {
			advance()

			if currentChar() == '-' {
				advance()
			}

			if isCastlingChar(currentChar()) {
				class = chess.QueensideCastle
				advance()
			} else {
				class = chess.KingsideCastle
			}
			pieceToMove = chess.King
		} else {
			ok = false
		}
	} else if moveString == chess.NullMoveString {
		class = chess.NullMove
	} else {
		ok = false
	}

	if ok && class != chess.NullMove {
		// Allow trailing checks
		for isCheck(currentChar()) {
			advance()
		}

		if currentChar() == 0 {
			// Nothing more to check
		} else if (strings.HasSuffix(remaining(), "ep") || strings.HasSuffix(remaining(), "e.p.")) &&
			class == chess.PawnMove {
			class = chess.EnPassantPawnMove
		} else {
			ok = false
		}
	}

	// Store all details
	if !ok {
		class = chess.UnknownMove
	}

	move.Class = class
	move.PieceToMove = pieceToMove
	move.PromotedPiece = promotedPiece
	move.FromCol = fromCol
	move.FromRank = fromRank
	move.ToCol = toCol
	move.ToRank = toRank

	return move
}

// DecodeAlgebraic refines move details using board context.
func DecodeAlgebraic(move *chess.Move, board *chess.Board) *chess.Move {
	fromR := chess.RankConvert(move.FromRank)
	fromC := chess.ColConvert(move.FromCol)

	if fromR == 0 || fromC == 0 {
		return move
	}

	colouredPiece := board.GetByIndex(fromC, fromR)
	pieceToMove := chess.ExtractPiece(colouredPiece)

	if pieceToMove != chess.Empty {
		// Check for castling
		if pieceToMove == chess.King && move.FromCol == 'e' {
			if move.ToCol == 'g' {
				move.Class = chess.KingsideCastle
			} else if move.ToCol == 'c' {
				move.Class = chess.QueensideCastle
			} else {
				move.Class = chess.PieceMove
				move.PieceToMove = pieceToMove
			}
		} else {
			if pieceToMove == chess.Pawn {
				move.Class = chess.PawnMove
			} else {
				move.Class = chess.PieceMove
			}
			move.PieceToMove = pieceToMove
		}
	}

	return move
}
