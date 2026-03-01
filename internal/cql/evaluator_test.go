package cql

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/engine"
)

func TestEvalPieceOnSquare(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"piece K e1", true},  // White king on e1
		{"piece K e2", false}, // No king on e2
		{"piece k e8", true},  // Black king on e8
		{"piece Q d1", true},  // White queen on d1
		{"piece q d8", true},  // Black queen on d8
		{"piece R a1", true},  // White rook on a1
		{"piece R h1", true},  // White rook on h1
		{"piece P e2", true},  // White pawn on e2
		{"piece p e7", true},  // Black pawn on e7
		{"piece N b1", true},  // White knight on b1
		{"piece B c1", true},  // White bishop on c1
		{"piece K d1", false}, // No king on d1 (queen is there)
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalSideToMove(t *testing.T) {
	tests := []struct {
		fen      string
		cql      string
		expected bool
	}{
		{"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", "wtm", true},
		{"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", "btm", false},
		{"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1", "btm", true},
		{"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1", "wtm", false},
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			board := engine.MustBoardFromFEN(tt.fen)
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("FEN %q, CQL %q: expected %v, got %v", tt.fen, tt.cql, tt.expected, result)
			}
		})
	}
}

func TestEvalCheck(t *testing.T) {
	tests := []struct {
		name     string
		fen      string
		expected bool
	}{
		{"starting position", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", false},
		{"scholars mate", "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4", false},
		{"black not in check", "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1", false},       // After 1.e4, black not in check
		{"white in check by queen", "rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3", true}, // White in check (fool's mate position)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board := engine.MustBoardFromFEN(tt.fen)
			node, err := Parse("check")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalMate(t *testing.T) {
	tests := []struct {
		name     string
		fen      string
		expected bool
	}{
		{"starting position", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", false},
		{"fools mate", "rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3", true},
		{"back rank mate", "R5k1/5ppp/8/8/8/8/8/4K3 b - - 0 1", true},              // Rook on a8 checking king on g8
		{"not mate - king can escape", "6k1/5pp1/8/8/8/8/8/R3K3 b - - 0 1", false}, // King can go to h7
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board := engine.MustBoardFromFEN(tt.fen)
			node, err := Parse("mate")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalStalemate(t *testing.T) {
	tests := []struct {
		name     string
		fen      string
		expected bool
	}{
		{"starting position", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", false},
		{"classic stalemate", "8/8/8/8/8/6k1/5q2/7K w - - 0 1", true},       // White king h1, black queen f2, black king g3 - king trapped but not in check
		{"checkmate not stalemate", "k7/8/1K6/8/8/8/8/R7 b - - 0 1", false}, // This is checkmate (rook gives check)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board := engine.MustBoardFromFEN(tt.fen)
			node, err := Parse("stalemate")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalLogicalAnd(t *testing.T) {
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"(and wtm (piece K e1))", true},          // White to move AND king on e1
		{"(and wtm (piece K e8))", false},         // White to move AND king on e8 (false)
		{"(and btm (piece K e1))", false},         // Black to move (false) AND king on e1
		{"(and (piece K e1) (piece k e8))", true}, // Both kings in starting squares
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalLogicalOr(t *testing.T) {
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"(or wtm btm)", true},                    // Either white or black to move
		{"(or (piece K e1) (piece K e8))", true},  // King on e1 OR e8
		{"(or (piece K d1) (piece K d8))", false}, // King on d1 OR d8 (neither)
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalLogicalNot(t *testing.T) {
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"(not btm)", true},          // NOT black to move = white to move
		{"(not wtm)", false},         // NOT white to move = false
		{"(not check)", true},        // NOT in check
		{"(not (piece K d1))", true}, // NOT king on d1
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalAttack(t *testing.T) {
	tests := []struct {
		name     string
		fen      string
		cql      string
		expected bool
	}{
		{"rook attacks king horizontally", "4k3/8/8/8/8/8/8/r3K3 w - - 0 1", "attack r K", true}, // Rook a1, King e1 - same rank
		{"queen attacks king diagonally", "4k3/8/8/8/8/8/1q6/K7 w - - 0 1", "attack q K", true},  // Queen b2, King a1 - diagonal
		{"knight attacks king", "4k3/8/8/8/8/3n4/8/4K3 w - - 0 1", "attack n K", true},           // Knight d3, King e1 - knight jump
		{"bishop attacks king", "4k3/8/8/8/8/2b5/8/4K3 w - - 0 1", "attack b K", true},           // Bishop c3, King e1 - diagonal
		{"no attack", "4k3/8/8/8/8/8/8/4K3 w - - 0 1", "attack r K", false},                      // No rook on board
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board := engine.MustBoardFromFEN(tt.fen)
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalComplexQuery(t *testing.T) {
	// Fool's mate position
	board := engine.MustBoardFromFEN("rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"mate", true},
		{"(and mate wtm)", true},
		{"(and mate (piece q h4))", true},  // Checkmate AND queen on h4
		{"(and mate (piece q e4))", false}, // Checkmate but queen not on e4
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Phase 5 tests: Piece designators and square sets

func TestEvalPieceDesignatorAnyWhite(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"piece A e1", true},  // Any white piece on e1 (king)
		{"piece A d1", true},  // Any white piece on d1 (queen)
		{"piece A e2", true},  // Any white piece on e2 (pawn)
		{"piece A e4", false}, // No white piece on e4
		{"piece A e8", false}, // Black piece on e8, not white
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalPieceDesignatorAnyBlack(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"piece a e8", true},  // Any black piece on e8 (king)
		{"piece a d8", true},  // Any black piece on d8 (queen)
		{"piece a e7", true},  // Any black piece on e7 (pawn)
		{"piece a e4", false}, // No black piece on e4
		{"piece a e1", false}, // White piece on e1, not black
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalPieceDesignatorEmpty(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"piece _ e4", true},  // e4 is empty
		{"piece _ d5", true},  // d5 is empty
		{"piece _ e1", false}, // e1 has king
		{"piece _ e7", false}, // e7 has pawn
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalPieceDesignatorAny(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"piece ? e4", true}, // e4 is empty - ? matches
		{"piece ? e1", true}, // e1 has white king - ? matches
		{"piece ? e8", true}, // e8 has black king - ? matches
		{"piece ? d5", true}, // d5 is empty - ? matches
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalPieceSet(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"piece [RQ] d1", true},  // Queen on d1 matches [RQ]
		{"piece [RQ] a1", true},  // Rook on a1 matches [RQ]
		{"piece [RQ] e1", false}, // King on e1 doesn't match [RQ]
		{"piece [RBN] c1", true}, // Bishop on c1 matches [RBN]
		{"piece [RBN] b1", true}, // Knight on b1 matches [RBN]
		{"piece [rq] d8", true},  // Black queen on d8 matches [rq]
		{"piece [rq] a8", true},  // Black rook on a8 matches [rq]
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalSquareSetRank(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"piece P [a-h]2", true},  // Pawns on rank 2
		{"piece R [a-h]1", true},  // Rooks on rank 1 (a1 and h1)
		{"piece P [a-h]4", false}, // No pawns on rank 4
		{"piece p [a-h]7", true},  // Black pawns on rank 7
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalSquareSetFile(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"piece P e[1-8]", true},  // White pawn on e-file (e2)
		{"piece p e[1-8]", true},  // Black pawn on e-file (e7)
		{"piece K e[1-8]", true},  // White king on e-file (e1)
		{"piece k e[1-8]", true},  // Black king on e-file (e8)
		{"piece Q e[1-8]", false}, // No queen on e-file
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalSquareSetQuadrant(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"piece R [a-d][1-4]", true},  // Rook a1 is in queenside lower quadrant
		{"piece R [e-h][1-4]", true},  // Rook h1 is in kingside lower quadrant
		{"piece r [a-d][5-8]", true},  // Rook a8 is in queenside upper quadrant
		{"piece K [e-h][1-4]", true},  // King e1 is in kingside lower quadrant
		{"piece K [a-d][1-4]", false}, // King e1 is NOT in queenside quadrant
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalSquareSetAny(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"piece K .", true}, // King exists somewhere
		{"piece Q .", true}, // Queen exists somewhere
		{"piece P .", true}, // Pawn exists somewhere
		{"piece k .", true}, // Black king exists
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalCount(t *testing.T) {
	// Standard starting position
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{"(== (count P) 8)", true},     // 8 white pawns
		{"(== (count p) 8)", true},     // 8 black pawns
		{"(== (count R) 2)", true},     // 2 white rooks
		{"(== (count K) 1)", true},     // 1 white king
		{"(> (count P) 5)", true},      // More than 5 white pawns
		{"(< (count P) 3)", false},     // Less than 3 white pawns (false)
		{"(>= (count [RBN]) 6)", true}, // At least 6 white minor/major pieces
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalMaterial(t *testing.T) {
	// Standard starting position
	// White material: 8*1 (pawns) + 2*3 (knights) + 2*3 (bishops) + 2*5 (rooks) + 1*9 (queen) = 8+6+6+10+9 = 39
	board := engine.MustBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{`(== (material "white") 39)`, true},                 // Standard white material
		{`(== (material "black") 39)`, true},                 // Standard black material
		{`(== (material "white") (material "black"))`, true}, // Equal material
		{`(> (material "white") 30)`, true},                  // More than 30 material
		{`(< (material "white") 50)`, true},                  // Less than 50 material
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalMaterialImbalance(t *testing.T) {
	// Position with material imbalance: white is up a queen
	board := engine.MustBoardFromFEN("rnb1kbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	tests := []struct {
		cql      string
		expected bool
	}{
		{`(== (material "white") 39)`, true},                // White has full material
		{`(== (material "black") 30)`, true},                // Black missing queen (39-9=30)
		{`(> (material "white") (material "black"))`, true}, // White has more
	}

	for _, tt := range tests {
		t.Run(tt.cql, func(t *testing.T) {
			node, err := Parse(tt.cql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			eval := NewEvaluator(board)
			result := eval.Evaluate(node)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
