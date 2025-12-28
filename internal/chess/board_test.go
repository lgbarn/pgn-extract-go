package chess

import (
	"testing"
)

func TestNewBoard(t *testing.T) {
	b := NewBoard()

	t.Run("initial state", func(t *testing.T) {
		if b.ToMove != White {
			t.Errorf("ToMove = %v; want White", b.ToMove)
		}
		if b.MoveNumber != 1 {
			t.Errorf("MoveNumber = %d; want 1", b.MoveNumber)
		}
		if b.EnPassant {
			t.Error("EnPassant = true; want false")
		}
		if b.HalfmoveClock != 0 {
			t.Errorf("HalfmoveClock = %d; want 0", b.HalfmoveClock)
		}
	})

	t.Run("all squares empty", func(t *testing.T) {
		for col := Col('a'); col <= 'h'; col++ {
			for rank := Rank('1'); rank <= '8'; rank++ {
				if got := b.Get(col, rank); got != Empty {
					t.Errorf("Get(%c, %c) = %v; want Empty", col, rank, got)
				}
			}
		}
	})

	t.Run("hedge squares are Off", func(t *testing.T) {
		// Check corners of the hedge
		if b.GetByIndex(0, 0) != Off {
			t.Error("Hedge corner (0,0) is not Off")
		}
		if b.GetByIndex(1, 1) != Off {
			t.Error("Hedge corner (1,1) is not Off")
		}
		if b.GetByIndex(Hedge+BoardSize, Hedge+BoardSize) != Off {
			t.Error("Hedge corner at far edge is not Off")
		}
	})
}

func TestSetupInitialPosition(t *testing.T) {
	b := NewBoard()
	b.SetupInitialPosition()

	tests := []struct {
		name  string
		col   Col
		rank  Rank
		piece Piece
	}{
		// White back rank
		{"white rook a1", 'a', '1', W(Rook)},
		{"white knight b1", 'b', '1', W(Knight)},
		{"white bishop c1", 'c', '1', W(Bishop)},
		{"white queen d1", 'd', '1', W(Queen)},
		{"white king e1", 'e', '1', W(King)},
		{"white bishop f1", 'f', '1', W(Bishop)},
		{"white knight g1", 'g', '1', W(Knight)},
		{"white rook h1", 'h', '1', W(Rook)},
		// White pawns
		{"white pawn a2", 'a', '2', W(Pawn)},
		{"white pawn e2", 'e', '2', W(Pawn)},
		{"white pawn h2", 'h', '2', W(Pawn)},
		// Black pawns
		{"black pawn a7", 'a', '7', B(Pawn)},
		{"black pawn e7", 'e', '7', B(Pawn)},
		{"black pawn h7", 'h', '7', B(Pawn)},
		// Black back rank
		{"black rook a8", 'a', '8', B(Rook)},
		{"black knight b8", 'b', '8', B(Knight)},
		{"black bishop c8", 'c', '8', B(Bishop)},
		{"black queen d8", 'd', '8', B(Queen)},
		{"black king e8", 'e', '8', B(King)},
		{"black bishop f8", 'f', '8', B(Bishop)},
		{"black knight g8", 'g', '8', B(Knight)},
		{"black rook h8", 'h', '8', B(Rook)},
		// Empty squares
		{"empty e3", 'e', '3', Empty},
		{"empty d4", 'd', '4', Empty},
		{"empty f5", 'f', '5', Empty},
		{"empty c6", 'c', '6', Empty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.Get(tt.col, tt.rank)
			if got != tt.piece {
				t.Errorf("Get(%c, %c) = %v; want %v", tt.col, tt.rank, got, tt.piece)
			}
		})
	}

	t.Run("king positions", func(t *testing.T) {
		if b.WKingCol != 'e' || b.WKingRank != '1' {
			t.Errorf("White king position = (%c, %c); want (e, 1)", b.WKingCol, b.WKingRank)
		}
		if b.BKingCol != 'e' || b.BKingRank != '8' {
			t.Errorf("Black king position = (%c, %c); want (e, 8)", b.BKingCol, b.BKingRank)
		}
	})

	t.Run("castling rights", func(t *testing.T) {
		if b.WKingCastle != 'h' {
			t.Errorf("WKingCastle = %c; want h", b.WKingCastle)
		}
		if b.WQueenCastle != 'a' {
			t.Errorf("WQueenCastle = %c; want a", b.WQueenCastle)
		}
		if b.BKingCastle != 'h' {
			t.Errorf("BKingCastle = %c; want h", b.BKingCastle)
		}
		if b.BQueenCastle != 'a' {
			t.Errorf("BQueenCastle = %c; want a", b.BQueenCastle)
		}
	})
}

func TestBoardGetSet(t *testing.T) {
	tests := []struct {
		name  string
		col   Col
		rank  Rank
		piece Piece
	}{
		{"white pawn on e4", 'e', '4', W(Pawn)},
		{"black knight on f6", 'f', '6', B(Knight)},
		{"white queen on d1", 'd', '1', W(Queen)},
		{"black king on e8", 'e', '8', B(King)},
		{"empty square", 'a', '1', Empty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBoard()
			b.Set(tt.col, tt.rank, tt.piece)
			got := b.Get(tt.col, tt.rank)
			if got != tt.piece {
				t.Errorf("after Set(%c, %c, %v), Get() = %v; want %v",
					tt.col, tt.rank, tt.piece, got, tt.piece)
			}
		})
	}

	t.Run("invalid coordinates return Off", func(t *testing.T) {
		b := NewBoard()
		// Test invalid coordinates
		if got := b.Get('i', '1'); got != Off {
			t.Errorf("Get('i', '1') = %v; want Off", got)
		}
		if got := b.Get('a', '9'); got != Off {
			t.Errorf("Get('a', '9') = %v; want Off", got)
		}
		if got := b.Get('z', 'z'); got != Off {
			t.Errorf("Get('z', 'z') = %v; want Off", got)
		}
	})

	t.Run("Set with invalid coordinates is no-op", func(t *testing.T) {
		b := NewBoard()
		b.SetupInitialPosition()
		// Set on invalid coordinate should not crash
		b.Set('z', '9', W(Queen))
		// Board should remain unchanged
		if got := b.Get('e', '1'); got != W(King) {
			t.Errorf("Get('e', '1') = %v after invalid Set; want white king", got)
		}
	})
}

func TestBoardGetByIndexSetByIndex(t *testing.T) {
	b := NewBoard()

	// Set a piece using array indices (remember hedge offset)
	col := Hedge + 4  // 'e' column
	rank := Hedge + 3 // rank 4
	b.SetByIndex(col, rank, W(Knight))

	if got := b.GetByIndex(col, rank); got != W(Knight) {
		t.Errorf("GetByIndex(%d, %d) = %v; want white knight", col, rank, got)
	}

	// Verify consistency with Get/Set using char coords
	if got := b.Get('e', '4'); got != W(Knight) {
		t.Errorf("Get('e', '4') = %v; want white knight (consistency check)", got)
	}
}

func TestBoardCopy(t *testing.T) {
	original := NewBoard()
	original.SetupInitialPosition()
	original.ToMove = Black
	original.MoveNumber = 5
	original.EnPassant = true
	original.EPCol = 'e'
	original.EPRank = '3'

	copied := original.Copy()

	t.Run("copies all state", func(t *testing.T) {
		if copied.ToMove != original.ToMove {
			t.Errorf("ToMove = %v; want %v", copied.ToMove, original.ToMove)
		}
		if copied.MoveNumber != original.MoveNumber {
			t.Errorf("MoveNumber = %d; want %d", copied.MoveNumber, original.MoveNumber)
		}
		if copied.EnPassant != original.EnPassant {
			t.Errorf("EnPassant = %v; want %v", copied.EnPassant, original.EnPassant)
		}
		if copied.EPCol != original.EPCol || copied.EPRank != original.EPRank {
			t.Errorf("EP square = (%c, %c); want (%c, %c)",
				copied.EPCol, copied.EPRank, original.EPCol, original.EPRank)
		}
	})

	t.Run("copies piece positions", func(t *testing.T) {
		if got := copied.Get('e', '1'); got != W(King) {
			t.Errorf("Get('e', '1') = %v; want white king", got)
		}
		if got := copied.Get('e', '8'); got != B(King) {
			t.Errorf("Get('e', '8') = %v; want black king", got)
		}
	})

	t.Run("modifications are independent", func(t *testing.T) {
		// Modify the copy
		copied.Set('e', '4', W(Pawn))
		copied.ToMove = White
		copied.MoveNumber = 10

		// Original should be unchanged
		if got := original.Get('e', '4'); got != Empty {
			t.Errorf("original Get('e', '4') = %v after copy modification; want Empty", got)
		}
		if original.ToMove != Black {
			t.Errorf("original ToMove = %v after copy modification; want Black", original.ToMove)
		}
		if original.MoveNumber != 5 {
			t.Errorf("original MoveNumber = %d after copy modification; want 5", original.MoveNumber)
		}
	})
}

func TestBoardSaveRestoreState(t *testing.T) {
	b := NewBoard()
	b.SetupInitialPosition()

	// Save initial state
	savedState := b.SaveState()

	// Make some modifications
	b.Set('e', '4', W(Pawn))
	b.Set('e', '2', Empty)
	b.ToMove = Black
	b.MoveNumber = 2
	b.EnPassant = true
	b.EPCol = 'e'
	b.EPRank = '3'
	b.WKingCastle = 0 // Removed castling right

	t.Run("modifications visible before restore", func(t *testing.T) {
		if got := b.Get('e', '4'); got != W(Pawn) {
			t.Errorf("Get('e', '4') = %v; want white pawn", got)
		}
		if got := b.Get('e', '2'); got != Empty {
			t.Errorf("Get('e', '2') = %v; want Empty", got)
		}
		if b.ToMove != Black {
			t.Errorf("ToMove = %v; want Black", b.ToMove)
		}
	})

	// Restore the saved state
	b.RestoreState(savedState)

	t.Run("state restored correctly", func(t *testing.T) {
		if got := b.Get('e', '4'); got != Empty {
			t.Errorf("Get('e', '4') after restore = %v; want Empty", got)
		}
		if got := b.Get('e', '2'); got != W(Pawn) {
			t.Errorf("Get('e', '2') after restore = %v; want white pawn", got)
		}
		if b.ToMove != White {
			t.Errorf("ToMove after restore = %v; want White", b.ToMove)
		}
		if b.MoveNumber != 1 {
			t.Errorf("MoveNumber after restore = %d; want 1", b.MoveNumber)
		}
		if b.EnPassant {
			t.Error("EnPassant after restore = true; want false")
		}
		if b.WKingCastle != 'h' {
			t.Errorf("WKingCastle after restore = %c; want h", b.WKingCastle)
		}
	})

	t.Run("all castling rights restored", func(t *testing.T) {
		if b.WKingCastle != 'h' {
			t.Errorf("WKingCastle = %c; want h", b.WKingCastle)
		}
		if b.WQueenCastle != 'a' {
			t.Errorf("WQueenCastle = %c; want a", b.WQueenCastle)
		}
		if b.BKingCastle != 'h' {
			t.Errorf("BKingCastle = %c; want h", b.BKingCastle)
		}
		if b.BQueenCastle != 'a' {
			t.Errorf("BQueenCastle = %c; want a", b.BQueenCastle)
		}
	})
}

func TestMovePair(t *testing.T) {
	mp := MovePair{
		FromCol:  'e',
		FromRank: '2',
		ToCol:    'e',
		ToRank:   '4',
	}

	if mp.FromCol != 'e' || mp.FromRank != '2' {
		t.Errorf("From = (%c, %c); want (e, 2)", mp.FromCol, mp.FromRank)
	}
	if mp.ToCol != 'e' || mp.ToRank != '4' {
		t.Errorf("To = (%c, %c); want (e, 4)", mp.ToCol, mp.ToRank)
	}
}
