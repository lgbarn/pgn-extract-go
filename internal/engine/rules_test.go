package engine

import (
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
)

// TestHasInsufficientMaterial tests various material configurations
func TestHasInsufficientMaterial(t *testing.T) {
	tests := []struct {
		name string
		fen  string
		want bool // true = insufficient material
	}{
		{"K vs K", "4k3/8/8/8/8/8/8/4K3 w - - 0 1", true},
		{"K+B vs K", "4k3/8/8/8/8/8/8/4KB2 w - - 0 1", true},
		{"K+N vs K", "4k3/8/8/8/8/8/8/4KN2 w - - 0 1", true},
		{"K vs K+b", "4k1b1/8/8/8/8/8/8/4K3 w - - 0 1", true},
		{"K vs K+n", "4k1n1/8/8/8/8/8/8/4K3 w - - 0 1", true},
		{"K+B vs K+B same color", "5b2/8/8/8/8/8/8/2B1K3 w - - 0 1", true},
		{"K+R vs K", "4k3/8/8/8/8/8/8/4KR2 w - - 0 1", false},
		{"K+Q vs K", "4k3/8/8/8/8/8/8/4KQ2 w - - 0 1", false},
		{"K+P vs K", "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1", false},
		{"K+B vs K+B opposite color", "5b2/8/8/8/8/8/8/3BK3 w - - 0 1", false},
		{"K+B+B vs K", "4k3/8/8/8/8/8/8/2B1KB2 w - - 0 1", false},
		{"standard starting position", "", false}, // empty fen means use initial board
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var board *chess.Board
			if tt.fen == "" {
				board = NewInitialBoard()
			} else {
				var err error
				board, err = NewBoardFromFEN(tt.fen)
				if err != nil {
					t.Fatalf("NewBoardFromFEN(%q) error: %v", tt.fen, err)
				}
			}

			got := HasInsufficientMaterial(board)
			if got != tt.want {
				t.Errorf("HasInsufficientMaterial() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAnalyzeDrawRules_EmptyGame tests analyzing a game with no moves
func TestAnalyzeDrawRules_EmptyGame(t *testing.T) {
	game := &chess.Game{
		Tags:  map[string]string{},
		Moves: nil,
	}

	result := AnalyzeDrawRules(game)

	if result.Has75MoveRule {
		t.Errorf("AnalyzeDrawRules(empty game).Has75MoveRule = true, want false")
	}
	if result.Has5FoldRepetition {
		t.Errorf("AnalyzeDrawRules(empty game).Has5FoldRepetition = true, want false")
	}
	if result.HasInsufficientMaterial {
		t.Errorf("AnalyzeDrawRules(empty game).HasInsufficientMaterial = true, want false")
	}
}

// TestAnalyzeDrawRules_MaterialOdds tests detecting material odds
func TestAnalyzeDrawRules_MaterialOdds(t *testing.T) {
	// Game starting from a position with missing material
	game := &chess.Game{
		Tags: map[string]string{
			"FEN": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBN1 w Qkq - 0 1", // Missing white rook
		},
		Moves: nil,
	}

	result := AnalyzeDrawRules(game)

	if !result.HasMaterialOdds {
		t.Errorf("AnalyzeDrawRules(game with missing rook).HasMaterialOdds = false, want true")
	}
}

// TestNewInitialBoard tests creating a standard initial board
func TestNewInitialBoard(t *testing.T) {
	board := NewInitialBoard()

	if board == nil {
		t.Fatal("NewInitialBoard() = nil, want non-nil board")
	}

	if board.ToMove != chess.White {
		t.Errorf("NewInitialBoard().ToMove = %v, want White", board.ToMove)
	}

	// Check that pieces are present in expected positions
	if got := board.Get('e', '1'); got == chess.Empty {
		t.Errorf("NewInitialBoard().Get('e', '1') = Empty, want white king")
	}
	if got := board.Get('e', '8'); got == chess.Empty {
		t.Errorf("NewInitialBoard().Get('e', '8') = Empty, want black king")
	}
	if got := board.Get('e', '2'); got == chess.Empty {
		t.Errorf("NewInitialBoard().Get('e', '2') = Empty, want white pawn")
	}

	// Check that center is empty
	if got := board.Get('e', '4'); got != chess.Empty {
		t.Errorf("NewInitialBoard().Get('e', '4') = %v, want Empty", got)
	}
}

// TestCheckMaterialOdds tests material odds detection
func TestCheckMaterialOdds(t *testing.T) {
	// Standard position - no odds
	game1 := &chess.Game{
		Tags:  map[string]string{},
		Moves: nil,
	}
	if CheckMaterialOdds(game1) {
		t.Errorf("CheckMaterialOdds(standard position) = true, want false")
	}

	// Position with missing piece - has odds
	game2 := &chess.Game{
		Tags: map[string]string{
			"FEN": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBN1 w Qkq - 0 1",
		},
		Moves: nil,
	}
	if !CheckMaterialOdds(game2) {
		t.Errorf("CheckMaterialOdds(position with missing rook) = false, want true")
	}
}

// TestIsLightSquare tests the isLightSquare function
func TestIsLightSquare(t *testing.T) {
	// isLightSquare returns (colNum+rankNum)%2 == 1
	// where colNum = col - 'a', rankNum = rank - '1'
	tests := []struct {
		col  chess.Col
		rank chess.Rank
		want bool
	}{
		{'a', '1', false}, // (0+0)%2=0 -> dark
		{'a', '2', true},  // (0+1)%2=1 -> light
		{'h', '8', false}, // (7+7)%2=0 -> dark
		{'h', '1', true},  // (7+0)%2=1 -> light
		{'e', '4', true},  // (4+3)%2=1 -> light
		{'d', '4', false}, // (3+3)%2=0 -> dark
		{'b', '1', true},  // (1+0)%2=1 -> light
		{'c', '3', false}, // (2+2)%2=0 -> dark
	}

	for _, tt := range tests {
		name := string(tt.col) + string(tt.rank)
		t.Run(name, func(t *testing.T) {
			got := isLightSquare(tt.col, tt.rank)
			if got != tt.want {
				t.Errorf("isLightSquare(%c, %c) = %v, want %v", tt.col, tt.rank, got, tt.want)
			}
		})
	}
}

// TestIsStandardMaterial tests the isStandardMaterial function
func TestIsStandardMaterial(t *testing.T) {
	// Standard position
	board1 := NewInitialBoard()
	if !isStandardMaterial(board1) {
		t.Errorf("isStandardMaterial(initial board) = false, want true")
	}

	// Position with missing piece
	board2, _ := NewBoardFromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBN1 w Qkq - 0 1")
	if isStandardMaterial(board2) {
		t.Errorf("isStandardMaterial(position with missing rook) = true, want false")
	}

	// Endgame position
	board3, _ := NewBoardFromFEN("4k3/8/8/8/8/8/8/4K3 w - - 0 1")
	if isStandardMaterial(board3) {
		t.Errorf("isStandardMaterial(K vs K) = true, want false")
	}
}

// TestCanPieceMove tests the canPieceMove function
func TestCanPieceMove(t *testing.T) {
	// Empty board for testing
	emptyBoard, _ := NewBoardFromFEN("8/8/8/8/8/8/8/8 w - - 0 1")

	tests := []struct {
		name     string
		piece    chess.Piece
		fromCol  chess.Col
		fromRank chess.Rank
		toCol    chess.Col
		toRank   chess.Rank
		want     bool
	}{
		{"knight L-move", chess.Knight, 'g', '1', 'f', '3', true},
		{"knight invalid", chess.Knight, 'g', '1', 'g', '3', false},
		{"bishop diagonal", chess.Bishop, 'c', '1', 'h', '6', true},
		{"bishop straight", chess.Bishop, 'c', '1', 'c', '5', false},
		{"rook straight", chess.Rook, 'a', '1', 'a', '8', true},
		{"rook diagonal", chess.Rook, 'a', '1', 'h', '8', false},
		{"queen diagonal", chess.Queen, 'd', '1', 'h', '5', true},
		{"queen straight", chess.Queen, 'd', '1', 'd', '8', true},
		{"queen invalid", chess.Queen, 'd', '1', 'e', '3', false},
		{"king one square", chess.King, 'e', '1', 'f', '2', true},
		{"king two squares", chess.King, 'e', '1', 'g', '1', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canPieceMove(emptyBoard, tt.piece, tt.fromCol, tt.fromRank, tt.toCol, tt.toRank)
			if got != tt.want {
				t.Errorf("canPieceMove() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCanPieceMove_BlockedPath tests that paths are blocked correctly
func TestCanPieceMove_BlockedPath(t *testing.T) {
	board := NewInitialBoard()

	// Rook at a1 cannot move to a8 (blocked by pawn at a2)
	if canPieceMove(board, chess.Rook, 'a', '1', 'a', '8') {
		t.Errorf("canPieceMove(Rook, a1, a8) = true, want false (blocked by pawn)")
	}

	// Bishop at c1 cannot move to h6 (blocked by pawn at d2)
	if canPieceMove(board, chess.Bishop, 'c', '1', 'h', '6') {
		t.Errorf("canPieceMove(Bishop, c1, h6) = true, want false (blocked by pawn)")
	}
}

// TestIsPathClear tests the isPathClear function
func TestIsPathClear(t *testing.T) {
	// Empty board - path should always be clear
	emptyBoard, _ := NewBoardFromFEN("8/8/8/8/8/8/8/8 w - - 0 1")
	if !isPathClear(emptyBoard, 'a', '1', 'a', '8') {
		t.Errorf("isPathClear(empty board, a1, a8) = false, want true")
	}
	if !isPathClear(emptyBoard, 'a', '1', 'h', '8') {
		t.Errorf("isPathClear(empty board, a1, h8) = false, want true")
	}

	// Initial board - paths blocked
	board := NewInitialBoard()
	if isPathClear(board, 'a', '1', 'a', '8') {
		t.Errorf("isPathClear(initial board, a1, a8) = true, want false")
	}
}

// TestFindKing tests that findKing can locate kings
func TestFindKing(t *testing.T) {
	board := NewInitialBoard()

	// Find white king
	col, rank := findKing(board, chess.White)
	if col != 'e' || rank != '1' {
		t.Errorf("White king at (%c, %c), want (e, 1)", col, rank)
	}

	// Find black king
	col, rank = findKing(board, chess.Black)
	if col != 'e' || rank != '8' {
		t.Errorf("Black king at (%c, %c), want (e, 8)", col, rank)
	}
}

// TestFindKing_CustomPosition tests finding kings in various positions
func TestFindKing_CustomPosition(t *testing.T) {
	board, _ := NewBoardFromFEN("8/8/8/3K4/8/8/8/4k3 w - - 0 1")

	col, rank := findKing(board, chess.White)
	if col != 'd' || rank != '5' {
		t.Errorf("White king at (%c, %c), want (d, 5)", col, rank)
	}

	col, rank = findKing(board, chess.Black)
	if col != 'e' || rank != '1' {
		t.Errorf("Black king at (%c, %c), want (e, 1)", col, rank)
	}
}

// TestCanPieceMove_Pawn tests that pawn movement returns false (not supported)
func TestCanPieceMove_Pawn(t *testing.T) {
	emptyBoard, _ := NewBoardFromFEN("8/8/8/8/8/8/8/8 w - - 0 1")

	// Pawn movement is not handled by canPieceMove
	if canPieceMove(emptyBoard, chess.Pawn, 'e', '2', 'e', '4') {
		t.Errorf("canPieceMove(Pawn, e2, e4) = true, want false")
	}
}

// TestNewBoardForGame tests creating a board from a game's FEN
func TestNewBoardForGame(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"FEN": "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		},
	}

	board := NewBoardForGame(game)
	if board == nil {
		t.Fatal("NewBoardForGame() = nil, want non-nil board")
	}

	if board.ToMove != chess.Black {
		t.Errorf("NewBoardForGame().ToMove = %v, want Black", board.ToMove)
	}
}

// TestNewBoardForGame_Standard tests creating a board from game without FEN
func TestNewBoardForGame_Standard(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{},
	}

	board := NewBoardForGame(game)
	if board == nil {
		t.Fatal("NewBoardForGame() = nil, want non-nil board")
	}

	if board.ToMove != chess.White {
		t.Errorf("NewBoardForGame().ToMove = %v, want White", board.ToMove)
	}
}

// TestNewBoardForGame_InvalidFEN tests fallback to initial when FEN is invalid
func TestNewBoardForGame_InvalidFEN(t *testing.T) {
	game := &chess.Game{
		Tags: map[string]string{
			"FEN": "invalid fen string",
		},
	}

	board := NewBoardForGame(game)
	if board == nil {
		t.Fatal("NewBoardForGame(invalid FEN) = nil, want non-nil board")
	}

	// Should fall back to standard starting position
	if board.ToMove != chess.White {
		t.Errorf("NewBoardForGame(invalid FEN).ToMove = %v, want White (fallback)", board.ToMove)
	}
}

