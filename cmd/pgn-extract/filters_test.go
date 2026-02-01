package main

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/lgbarn/pgn-extract-go/internal/chess"
	"github.com/lgbarn/pgn-extract-go/internal/config"
	"github.com/lgbarn/pgn-extract-go/internal/engine"
	"github.com/lgbarn/pgn-extract-go/internal/matching"
	"github.com/lgbarn/pgn-extract-go/internal/processing"
	"github.com/lgbarn/pgn-extract-go/internal/testutil"
)

func TestParseElo(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		// Valid ratings
		{"typical elo", "2500", 2500},
		{"low rating", "1200", 1200},
		{"high rating", "2850", 2850},
		{"beginner", "600", 600},
		{"four digits", "1500", 1500},

		// Edge cases - return 0
		{"empty string", "", 0},
		{"dash", "-", 0},
		{"question mark", "?", 0},

		// Invalid formats - return 0
		{"letters", "abc", 0},
		{"mixed", "12a5", 0},
		{"float", "2500.5", 0},
		{"negative", "-100", 0}, // strconv.Atoi returns -100, but this is valid negative int
		{"spaces", " 2500", 0},
		{"trailing spaces", "2500 ", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseElo(tt.input)
			// Note: negative numbers are valid per strconv.Atoi
			if tt.input == "-100" {
				if got != -100 {
					t.Errorf("parseElo(%q) = %d; want -100", tt.input, got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("parseElo(%q) = %d; want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Normal strings remain unchanged
		{"simple text", "Hello World", "Hello World"},
		{"with numbers", "Player123", "Player123"},
		{"with punctuation", "Fischer, Robert J.", "Fischer, Robert J."},

		// Control characters removed
		{"null byte", "Hello\x00World", "HelloWorld"},
		{"tab character", "Hello\tWorld", "HelloWorld"}, // tab (0x09) is < 32, removed
		{"newline", "Hello\nWorld", "HelloWorld"},
		{"carriage return", "Hello\rWorld", "HelloWorld"},
		{"bell", "Hello\x07World", "HelloWorld"},

		// DEL character (127) removed
		{"del character", "Hello\x7fWorld", "HelloWorld"},

		// Unicode preserved
		{"accented chars", "Müller", "Müller"},
		{"cyrillic", "Карпов", "Карпов"},
		{"chinese", "象棋", "象棋"},
		{"emoji", "♔♕♖", "♔♕♖"},

		// Edge cases
		{"empty string", "", ""},
		{"only spaces", "   ", "   "},
		{"only control chars", "\x00\x01\x02", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanString(tt.input)
			if got != tt.want {
				t.Errorf("cleanString(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchedCountOperations(t *testing.T) {
	// Reset to zero state first (since tests run in order)
	// We can't reset matchedCount directly, so we test incremental behavior

	initialCount := GetMatchedCount()

	IncrementMatchedCount()
	afterFirst := GetMatchedCount()
	if afterFirst != initialCount+1 {
		t.Errorf("after first increment: GetMatchedCount() = %d; want %d", afterFirst, initialCount+1)
	}

	IncrementMatchedCount()
	IncrementMatchedCount()
	afterThree := GetMatchedCount()
	if afterThree != initialCount+3 {
		t.Errorf("after three increments: GetMatchedCount() = %d; want %d", afterThree, initialCount+3)
	}
}

// ============================================================
// Task 1: Pure helper function tests
// ============================================================

func TestParseIntSet(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKeys []int
		wantLen  int
	}{
		{"empty string", "", nil, 0},
		{"single value", "5", []int{5}, 1},
		{"multiple values", "1,2,3", []int{1, 2, 3}, 3},
		{"with whitespace", " 1 , 2 , 3 ", []int{1, 2, 3}, 3},
		{"invalid entries skipped", "1,abc,3", []int{1, 3}, 2},
		{"all invalid", "abc,def", nil, 0},
		{"duplicates", "1,1,2", []int{1, 2}, 2},
		{"negative values", "-1,2", []int{-1, 2}, 2},
		{"zero", "0", []int{0}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIntSet(tt.input)
			if len(got) != tt.wantLen {
				t.Errorf("parseIntSet(%q) has %d entries; want %d", tt.input, len(got), tt.wantLen)
			}
			for _, k := range tt.wantKeys {
				if !got[k] {
					t.Errorf("parseIntSet(%q) missing key %d", tt.input, k)
				}
			}
		})
	}
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [2]int
	}{
		{"valid range", "20-40", [2]int{20, 40}},
		{"single digit", "1-5", [2]int{1, 5}},
		{"with spaces", " 10 - 20 ", [2]int{10, 20}},
		{"missing dash", "2040", [2]int{0, 0}},
		{"empty string", "", [2]int{0, 0}},
		{"extra parts", "1-2-3", [2]int{0, 0}},
		{"zero range", "0-0", [2]int{0, 0}},
		{"same values", "10-10", [2]int{10, 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRange(tt.input)
			if got != tt.want {
				t.Errorf("parseRange(%q) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCheckPlyBounds(t *testing.T) {
	oldExactPly := *exactPly
	oldMinPly := *minPly
	oldMaxPly := *maxPly
	oldPlyRange := parsedPlyRange
	defer func() {
		*exactPly = oldExactPly
		*minPly = oldMinPly
		*maxPly = oldMaxPly
		parsedPlyRange = oldPlyRange
	}()

	tests := []struct {
		name     string
		plyCount int
		matched  bool
		exact    int
		min      int
		max      int
		plyRng   [2]int
		want     bool
	}{
		{"already false", 10, false, 0, 0, 0, [2]int{0, 0}, false},
		{"no bounds", 10, true, 0, 0, 0, [2]int{0, 0}, true},
		{"exact match", 20, true, 20, 0, 0, [2]int{0, 0}, true},
		{"exact no match", 21, true, 20, 0, 0, [2]int{0, 0}, false},
		{"min ply pass", 15, true, 0, 10, 0, [2]int{0, 0}, true},
		{"min ply fail", 5, true, 0, 10, 0, [2]int{0, 0}, false},
		{"max ply pass", 15, true, 0, 0, 20, [2]int{0, 0}, true},
		{"max ply fail", 25, true, 0, 0, 20, [2]int{0, 0}, false},
		{"range pass", 30, true, 0, 0, 0, [2]int{20, 40}, true},
		{"range fail low", 10, true, 0, 0, 0, [2]int{20, 40}, false},
		{"range fail high", 50, true, 0, 0, 0, [2]int{20, 40}, false},
		{"min and range uses higher", 15, true, 0, 10, 0, [2]int{20, 40}, false},
		{"max and range uses lower", 35, true, 0, 0, 50, [2]int{20, 30}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			*exactPly = tt.exact
			*minPly = tt.min
			*maxPly = tt.max
			parsedPlyRange = tt.plyRng

			got := checkPlyBounds(tt.plyCount, tt.matched)
			if got != tt.want {
				t.Errorf("checkPlyBounds(%d, %v) = %v; want %v", tt.plyCount, tt.matched, got, tt.want)
			}
		})
	}
}

func TestCheckMoveBounds(t *testing.T) {
	oldExactMove := *exactMove
	oldMinMoves := *minMoves
	oldMaxMoves := *maxMoves
	oldMoveRange := parsedMoveRange
	defer func() {
		*exactMove = oldExactMove
		*minMoves = oldMinMoves
		*maxMoves = oldMaxMoves
		parsedMoveRange = oldMoveRange
	}()

	tests := []struct {
		name     string
		plyCount int
		matched  bool
		exact    int
		min      int
		max      int
		moveRng  [2]int
		want     bool
	}{
		{"already false", 20, false, 0, 0, 0, [2]int{0, 0}, false},
		{"no bounds", 20, true, 0, 0, 0, [2]int{0, 0}, true},
		// (20+1)/2 = 10
		{"exact match 10 moves from 19 plies", 19, true, 10, 0, 0, [2]int{0, 0}, true},
		{"exact no match", 20, true, 11, 0, 0, [2]int{0, 0}, false},
		{"exact match 11 moves from 21 plies", 21, true, 11, 0, 0, [2]int{0, 0}, true},
		{"min moves pass", 20, true, 0, 5, 0, [2]int{0, 0}, true},
		{"min moves fail", 4, true, 0, 5, 0, [2]int{0, 0}, false},
		{"max moves pass", 10, true, 0, 0, 10, [2]int{0, 0}, true},
		{"max moves fail", 30, true, 0, 0, 10, [2]int{0, 0}, false},
		{"range pass", 20, true, 0, 0, 0, [2]int{5, 15}, true},
		{"range fail", 40, true, 0, 0, 0, [2]int{5, 15}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			*exactMove = tt.exact
			*minMoves = tt.min
			*maxMoves = tt.max
			parsedMoveRange = tt.moveRng

			got := checkMoveBounds(tt.plyCount, tt.matched)
			if got != tt.want {
				t.Errorf("checkMoveBounds(%d, %v) = %v; want %v", tt.plyCount, tt.matched, got, tt.want)
			}
		})
	}
}

func TestCheckGamePosition(t *testing.T) {
	oldSelectOnly := selectOnlySet
	oldSkipMatching := skipMatchingSet
	defer func() {
		selectOnlySet = oldSelectOnly
		skipMatchingSet = oldSkipMatching
	}()

	t.Run("both empty returns true", func(t *testing.T) {
		selectOnlySet = nil
		skipMatchingSet = nil
		if !checkGamePosition(1) {
			t.Error("expected true when both sets empty")
		}
	})

	t.Run("selectOnly includes position", func(t *testing.T) {
		selectOnlySet = map[int]bool{1: true, 3: true, 5: true}
		skipMatchingSet = nil
		if !checkGamePosition(3) {
			t.Error("expected true for position in selectOnlySet")
		}
		if checkGamePosition(2) {
			t.Error("expected false for position not in selectOnlySet")
		}
	})

	t.Run("skipMatching excludes position", func(t *testing.T) {
		selectOnlySet = nil
		skipMatchingSet = map[int]bool{2: true, 4: true}
		if !checkGamePosition(1) {
			t.Error("expected true for position not in skipMatchingSet")
		}
		if checkGamePosition(2) {
			t.Error("expected false for position in skipMatchingSet")
		}
	})

	t.Run("selectOnly takes precedence over skipMatching", func(t *testing.T) {
		selectOnlySet = map[int]bool{1: true}
		skipMatchingSet = map[int]bool{1: true}
		// selectOnly is checked first due to len > 0
		if !checkGamePosition(1) {
			t.Error("expected true: selectOnly takes precedence")
		}
	})
}

func TestCountPieces(t *testing.T) {
	board, err := engine.NewBoardFromFEN(engine.InitialFEN)
	if err != nil {
		t.Fatalf("failed to create board: %v", err)
	}
	got := countPieces(board)
	if got != 32 {
		t.Errorf("countPieces(initial) = %d; want 32", got)
	}
}

func TestCheckRatingWinner(t *testing.T) {
	oldHigher := *higherRatedWinner
	oldLower := *lowerRatedWinner
	defer func() {
		*higherRatedWinner = oldHigher
		*lowerRatedWinner = oldLower
	}()

	makeGame := func(whiteElo, blackElo, result string) *chess.Game {
		g := chess.NewGame()
		g.Tags["WhiteElo"] = whiteElo
		g.Tags["BlackElo"] = blackElo
		g.Tags["Result"] = result
		return g
	}

	tests := []struct {
		name   string
		game   *chess.Game
		higher bool
		lower  bool
		want   bool
	}{
		{"higher rated white wins", makeGame("2700", "2500", "1-0"), true, false, true},
		{"higher rated black wins", makeGame("2500", "2700", "0-1"), true, false, true},
		{"higher rated but lower wins", makeGame("2700", "2500", "0-1"), true, false, false},
		{"lower rated white wins", makeGame("2500", "2700", "1-0"), false, true, true},
		{"lower rated black wins", makeGame("2700", "2500", "0-1"), false, true, true},
		{"lower rated but higher wins", makeGame("2500", "2700", "0-1"), false, true, false},
		{"draw with higher filter", makeGame("2700", "2500", "1/2-1/2"), true, false, false},
		{"missing white elo", makeGame("", "2500", "1-0"), true, false, false},
		{"missing black elo", makeGame("2500", "", "1-0"), true, false, false},
		{"equal ratings higher filter", makeGame("2500", "2500", "1-0"), true, false, false},
		{"neither filter enabled", makeGame("2700", "2500", "1-0"), false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			*higherRatedWinner = tt.higher
			*lowerRatedWinner = tt.lower
			got := checkRatingWinner(tt.game)
			if got != tt.want {
				t.Errorf("checkRatingWinner() = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestFindCommentPly(t *testing.T) {
	t.Run("game with matching comment", func(t *testing.T) {
		pgn := `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 {book move} e5 {good response} 2. Nf3 *
`
		game := testutil.MustParseGame(t, pgn)
		got := findCommentPly(game, "good response")
		if got != 2 {
			t.Errorf("findCommentPly() = %d; want 2", got)
		}
	})

	t.Run("pattern not found", func(t *testing.T) {
		pgn := `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 *
`
		game := testutil.MustParseGame(t, pgn)
		got := findCommentPly(game, "nonexistent")
		if got != 0 {
			t.Errorf("findCommentPly() = %d; want 0", got)
		}
	})

	t.Run("no moves", func(t *testing.T) {
		game := chess.NewGame()
		got := findCommentPly(game, "test")
		if got != 0 {
			t.Errorf("findCommentPly() = %d; want 0", got)
		}
	})

	t.Run("first move comment", func(t *testing.T) {
		pgn := `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 {found it} e5 *
`
		game := testutil.MustParseGame(t, pgn)
		got := findCommentPly(game, "found it")
		if got != 1 {
			t.Errorf("findCommentPly() = %d; want 1", got)
		}
	})
}

func TestTruncateMoveList(t *testing.T) {
	// Helper to build a linked list of N moves
	buildMoveList := func(n int) *chess.Move {
		if n == 0 {
			return nil
		}
		head := chess.NewMove()
		head.Text = "m1"
		current := head
		for i := 2; i <= n; i++ {
			m := chess.NewMove()
			m.Text = fmt.Sprintf("m%d", i)
			m.Prev = current
			current.Next = m
			current = m
		}
		return head
	}

	countMoves := func(m *chess.Move) int {
		count := 0
		for ; m != nil; m = m.Next {
			count++
		}
		return count
	}

	t.Run("nil moves", func(t *testing.T) {
		got := truncateMoveList(nil, 0, 0)
		if got != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("skip 0 limit 0", func(t *testing.T) {
		moves := buildMoveList(5)
		got := truncateMoveList(moves, 0, 0)
		if countMoves(got) != 5 {
			t.Errorf("expected 5 moves; got %d", countMoves(got))
		}
	})

	t.Run("skip 2", func(t *testing.T) {
		moves := buildMoveList(5)
		got := truncateMoveList(moves, 2, 0)
		if countMoves(got) != 3 {
			t.Errorf("expected 3 moves; got %d", countMoves(got))
		}
		if got.Prev != nil {
			t.Error("expected new head to have nil Prev")
		}
	})

	t.Run("limit 2", func(t *testing.T) {
		moves := buildMoveList(5)
		got := truncateMoveList(moves, 0, 2)
		if countMoves(got) != 2 {
			t.Errorf("expected 2 moves; got %d", countMoves(got))
		}
	})

	t.Run("skip 1 limit 2", func(t *testing.T) {
		moves := buildMoveList(5)
		got := truncateMoveList(moves, 1, 2)
		if countMoves(got) != 2 {
			t.Errorf("expected 2 moves; got %d", countMoves(got))
		}
	})

	t.Run("skip past end", func(t *testing.T) {
		moves := buildMoveList(3)
		got := truncateMoveList(moves, 10, 0)
		if got != nil {
			t.Error("expected nil when skip exceeds length")
		}
	})

	t.Run("limit exceeds remaining", func(t *testing.T) {
		moves := buildMoveList(3)
		got := truncateMoveList(moves, 0, 10)
		if countMoves(got) != 3 {
			t.Errorf("expected 3 moves; got %d", countMoves(got))
		}
	})

	t.Run("skip exactly all", func(t *testing.T) {
		moves := buildMoveList(3)
		got := truncateMoveList(moves, 3, 0)
		if got != nil {
			t.Error("expected nil when skip equals length")
		}
	})

	t.Run("limit 1", func(t *testing.T) {
		moves := buildMoveList(5)
		got := truncateMoveList(moves, 0, 1)
		if countMoves(got) != 1 {
			t.Errorf("expected 1 move; got %d", countMoves(got))
		}
	})
}

// ============================================================
// Task 2: Filter sub-pipeline tests
// ============================================================

func TestApplyEndingFilters(t *testing.T) {
	oldCheckmate := *checkmateFilter
	oldStalemate := *stalemateFilter
	defer func() {
		*checkmateFilter = oldCheckmate
		*stalemateFilter = oldStalemate
	}()

	t.Run("no filters pass through with nil board", func(t *testing.T) {
		*checkmateFilter = false
		*stalemateFilter = false
		// With no filters active, nil board should pass
		got := applyEndingFilters(nil)
		if !got {
			t.Error("expected true when no ending filters enabled")
		}
	})

	t.Run("checkmate filter with initial board", func(t *testing.T) {
		*checkmateFilter = true
		*stalemateFilter = false
		board, _ := engine.NewBoardFromFEN(engine.InitialFEN)
		got := applyEndingFilters(board)
		if got {
			t.Error("expected false: initial position is not checkmate")
		}
	})

	t.Run("stalemate filter with initial board", func(t *testing.T) {
		*checkmateFilter = false
		*stalemateFilter = true
		board, _ := engine.NewBoardFromFEN(engine.InitialFEN)
		got := applyEndingFilters(board)
		if got {
			t.Error("expected false: initial position is not stalemate")
		}
	})

	t.Run("no filters with initial board passes", func(t *testing.T) {
		*checkmateFilter = false
		*stalemateFilter = false
		board, _ := engine.NewBoardFromFEN(engine.InitialFEN)
		got := applyEndingFilters(board)
		if !got {
			t.Error("expected true when no filters active")
		}
	})
}

func TestApplyGameInfoFilters(t *testing.T) {
	oldFifty := *fiftyMoveFilter
	oldRep := *repetitionFilter
	oldUnder := *underpromotionFilter
	old75 := *seventyFiveMoveFilter
	old5fold := *fiveFoldRepFilter
	oldInsuf := *insufficientFilter
	oldOdds := *materialOddsFilter
	defer func() {
		*fiftyMoveFilter = oldFifty
		*repetitionFilter = oldRep
		*underpromotionFilter = oldUnder
		*seventyFiveMoveFilter = old75
		*fiveFoldRepFilter = old5fold
		*insufficientFilter = oldInsuf
		*materialOddsFilter = oldOdds
	}()

	resetFlags := func() {
		*fiftyMoveFilter = false
		*repetitionFilter = false
		*underpromotionFilter = false
		*seventyFiveMoveFilter = false
		*fiveFoldRepFilter = false
		*insufficientFilter = false
		*materialOddsFilter = false
	}

	t.Run("nil info no filters", func(t *testing.T) {
		resetFlags()
		if !applyGameInfoFilters(nil) {
			t.Error("expected true: nil info, no filters")
		}
	})

	t.Run("nil info with fifty filter", func(t *testing.T) {
		resetFlags()
		*fiftyMoveFilter = true
		if applyGameInfoFilters(nil) {
			t.Error("expected false: nil info but filter enabled")
		}
	})

	t.Run("nil info with repetition filter", func(t *testing.T) {
		resetFlags()
		*repetitionFilter = true
		if applyGameInfoFilters(nil) {
			t.Error("expected false: nil info but repetition filter enabled")
		}
	})

	t.Run("info with fifty move rule", func(t *testing.T) {
		resetFlags()
		*fiftyMoveFilter = true
		info := &processing.GameAnalysis{HasFiftyMoveRule: true}
		if !applyGameInfoFilters(info) {
			t.Error("expected true: info has fifty move rule")
		}
	})

	t.Run("info without fifty move rule", func(t *testing.T) {
		resetFlags()
		*fiftyMoveFilter = true
		info := &processing.GameAnalysis{HasFiftyMoveRule: false}
		if applyGameInfoFilters(info) {
			t.Error("expected false: info lacks fifty move rule")
		}
	})

	t.Run("repetition filter pass", func(t *testing.T) {
		resetFlags()
		*repetitionFilter = true
		info := &processing.GameAnalysis{HasRepetition: true}
		if !applyGameInfoFilters(info) {
			t.Error("expected true")
		}
	})

	t.Run("underpromotion filter pass", func(t *testing.T) {
		resetFlags()
		*underpromotionFilter = true
		info := &processing.GameAnalysis{HasUnderpromotion: true}
		if !applyGameInfoFilters(info) {
			t.Error("expected true")
		}
	})

	t.Run("75 move filter pass", func(t *testing.T) {
		resetFlags()
		*seventyFiveMoveFilter = true
		info := &processing.GameAnalysis{Has75MoveRule: true}
		if !applyGameInfoFilters(info) {
			t.Error("expected true")
		}
	})

	t.Run("5fold rep filter pass", func(t *testing.T) {
		resetFlags()
		*fiveFoldRepFilter = true
		info := &processing.GameAnalysis{Has5FoldRepetition: true}
		if !applyGameInfoFilters(info) {
			t.Error("expected true")
		}
	})

	t.Run("insufficient filter pass", func(t *testing.T) {
		resetFlags()
		*insufficientFilter = true
		info := &processing.GameAnalysis{HasInsufficientMaterial: true}
		if !applyGameInfoFilters(info) {
			t.Error("expected true")
		}
	})

	t.Run("material odds filter pass", func(t *testing.T) {
		resetFlags()
		*materialOddsFilter = true
		info := &processing.GameAnalysis{HasMaterialOdds: true}
		if !applyGameInfoFilters(info) {
			t.Error("expected true")
		}
	})

	t.Run("info without matching info fails", func(t *testing.T) {
		resetFlags()
		*materialOddsFilter = true
		info := &processing.GameAnalysis{HasMaterialOdds: false}
		if applyGameInfoFilters(info) {
			t.Error("expected false")
		}
	})
}

func TestApplyFeatureFilters(t *testing.T) {
	oldCommented := *commentedFilter
	oldNoSetup := *noSetupTags
	oldOnlySetup := *onlySetupTags
	oldPieceCount := *pieceCount
	oldCheckmate := *checkmateFilter
	oldStalemate := *stalemateFilter
	oldHigher := *higherRatedWinner
	oldLower := *lowerRatedWinner
	oldFifty := *fiftyMoveFilter
	oldRep := *repetitionFilter
	oldUnder := *underpromotionFilter
	old75 := *seventyFiveMoveFilter
	old5fold := *fiveFoldRepFilter
	oldInsuf := *insufficientFilter
	oldOdds := *materialOddsFilter
	defer func() {
		*commentedFilter = oldCommented
		*noSetupTags = oldNoSetup
		*onlySetupTags = oldOnlySetup
		*pieceCount = oldPieceCount
		*checkmateFilter = oldCheckmate
		*stalemateFilter = oldStalemate
		*higherRatedWinner = oldHigher
		*lowerRatedWinner = oldLower
		*fiftyMoveFilter = oldFifty
		*repetitionFilter = oldRep
		*underpromotionFilter = oldUnder
		*seventyFiveMoveFilter = old75
		*fiveFoldRepFilter = old5fold
		*insufficientFilter = oldInsuf
		*materialOddsFilter = oldOdds
	}()

	resetAllFlags := func() {
		*commentedFilter = false
		*noSetupTags = false
		*onlySetupTags = false
		*pieceCount = 0
		*checkmateFilter = false
		*stalemateFilter = false
		*higherRatedWinner = false
		*lowerRatedWinner = false
		*fiftyMoveFilter = false
		*repetitionFilter = false
		*underpromotionFilter = false
		*seventyFiveMoveFilter = false
		*fiveFoldRepFilter = false
		*insufficientFilter = false
		*materialOddsFilter = false
	}

	t.Run("already false", func(t *testing.T) {
		resetAllFlags()
		result := &FilterResult{}
		game := chess.NewGame()
		if applyFeatureFilters(result, game, false) {
			t.Error("expected false when matched=false")
		}
	})

	t.Run("no filters pass through", func(t *testing.T) {
		resetAllFlags()
		result := &FilterResult{}
		game := chess.NewGame()
		if !applyFeatureFilters(result, game, true) {
			t.Error("expected true with no filters")
		}
	})

	t.Run("commented filter with comments", func(t *testing.T) {
		resetAllFlags()
		*commentedFilter = true
		pgn := `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 {a comment} e5 *
`
		game := testutil.MustParseGame(t, pgn)
		result := &FilterResult{}
		if !applyFeatureFilters(result, game, true) {
			t.Error("expected true: game has comments")
		}
	})

	t.Run("commented filter without comments", func(t *testing.T) {
		resetAllFlags()
		*commentedFilter = true
		pgn := `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 *
`
		game := testutil.MustParseGame(t, pgn)
		result := &FilterResult{}
		if applyFeatureFilters(result, game, true) {
			t.Error("expected false: game has no comments")
		}
	})

	t.Run("noSetupTags excludes games with SetUp", func(t *testing.T) {
		resetAllFlags()
		*noSetupTags = true
		game := chess.NewGame()
		game.Tags["SetUp"] = "1"
		result := &FilterResult{}
		if applyFeatureFilters(result, game, true) {
			t.Error("expected false: game has SetUp tag")
		}
	})

	t.Run("noSetupTags allows game without SetUp", func(t *testing.T) {
		resetAllFlags()
		*noSetupTags = true
		game := chess.NewGame()
		result := &FilterResult{}
		if !applyFeatureFilters(result, game, true) {
			t.Error("expected true: game has no SetUp tag")
		}
	})

	t.Run("onlySetupTags requires SetUp", func(t *testing.T) {
		resetAllFlags()
		*onlySetupTags = true
		game := chess.NewGame()
		result := &FilterResult{}
		if applyFeatureFilters(result, game, true) {
			t.Error("expected false: game lacks SetUp tag")
		}
	})

	t.Run("onlySetupTags with SetUp", func(t *testing.T) {
		resetAllFlags()
		*onlySetupTags = true
		game := chess.NewGame()
		game.Tags["SetUp"] = "1"
		result := &FilterResult{}
		if !applyFeatureFilters(result, game, true) {
			t.Error("expected true: game has SetUp tag")
		}
	})
}

func TestAddAnnotations(t *testing.T) {
	t.Run("add ply count", func(t *testing.T) {
		game := chess.NewGame()
		result := &FilterResult{PlyCount: 42}
		cfg := config.NewConfig()
		cfg.Annotation.AddPlyCount = true
		addAnnotations(game, result, cfg)
		if game.Tags["PlyCount"] != "42" {
			t.Errorf("PlyCount tag = %q; want %q", game.Tags["PlyCount"], "42")
		}
	})

	t.Run("add hash tag with board", func(t *testing.T) {
		game := chess.NewGame()
		board, _ := engine.NewBoardFromFEN(engine.InitialFEN)
		result := &FilterResult{Board: board}
		cfg := config.NewConfig()
		cfg.Annotation.AddHashTag = true
		addAnnotations(game, result, cfg)
		hash := game.Tags["HashCode"]
		if hash == "" {
			t.Error("expected HashCode tag to be set")
		}
		if len(hash) != 16 {
			t.Errorf("HashCode length = %d; want 16 hex chars", len(hash))
		}
	})

	t.Run("add hash tag nil board", func(t *testing.T) {
		game := chess.NewGame()
		result := &FilterResult{Board: nil}
		cfg := config.NewConfig()
		cfg.Annotation.AddHashTag = true
		addAnnotations(game, result, cfg)
		if _, ok := game.Tags["HashCode"]; ok {
			t.Error("expected no HashCode tag when board is nil")
		}
	})

	t.Run("no annotations", func(t *testing.T) {
		game := chess.NewGame()
		result := &FilterResult{PlyCount: 10}
		cfg := config.NewConfig()
		addAnnotations(game, result, cfg)
		if _, ok := game.Tags["PlyCount"]; ok {
			t.Error("PlyCount should not be set when disabled")
		}
	})

	t.Run("both annotations", func(t *testing.T) {
		game := chess.NewGame()
		board, _ := engine.NewBoardFromFEN(engine.InitialFEN)
		result := &FilterResult{PlyCount: 10, Board: board}
		cfg := config.NewConfig()
		cfg.Annotation.AddPlyCount = true
		cfg.Annotation.AddHashTag = true
		addAnnotations(game, result, cfg)
		if game.Tags["PlyCount"] != "10" {
			t.Errorf("PlyCount = %q; want %q", game.Tags["PlyCount"], "10")
		}
		if game.Tags["HashCode"] == "" {
			t.Error("expected HashCode to be set")
		}
	})
}

func TestApplyTagFilters(t *testing.T) {
	t.Run("already false", func(t *testing.T) {
		game := chess.NewGame()
		ctx := &ProcessingContext{cfg: config.NewConfig()}
		if applyTagFilters(game, ctx, false) {
			t.Error("expected false when matched=false")
		}
	})

	t.Run("nil game filter passes", func(t *testing.T) {
		game := chess.NewGame()
		ctx := &ProcessingContext{cfg: config.NewConfig()}
		if !applyTagFilters(game, ctx, true) {
			t.Error("expected true with nil gameFilter")
		}
	})

	t.Run("game filter no criteria passes", func(t *testing.T) {
		game := chess.NewGame()
		gf := matching.NewGameFilter()
		ctx := &ProcessingContext{cfg: config.NewConfig(), gameFilter: gf}
		if !applyTagFilters(game, ctx, true) {
			t.Error("expected true: gameFilter has no criteria")
		}
	})

	t.Run("game filter with non-matching criteria", func(t *testing.T) {
		game := chess.NewGame()
		game.Tags["White"] = "Carlsen"
		gf := matching.NewGameFilter()
		gf.AddTagCriterion("White", "Kasparov", matching.OpEqual)
		ctx := &ProcessingContext{cfg: config.NewConfig(), gameFilter: gf}
		if applyTagFilters(game, ctx, true) {
			t.Error("expected false: White doesn't match Kasparov")
		}
	})

	t.Run("game filter with matching criteria", func(t *testing.T) {
		game := chess.NewGame()
		game.Tags["White"] = "Carlsen"
		gf := matching.NewGameFilter()
		gf.AddTagCriterion("White", "Carlsen", matching.OpEqual)
		ctx := &ProcessingContext{cfg: config.NewConfig(), gameFilter: gf}
		if !applyTagFilters(game, ctx, true) {
			t.Error("expected true: White matches Carlsen")
		}
	})
}

func TestApplyPatternFilters(t *testing.T) {
	t.Run("returns matched as-is true", func(t *testing.T) {
		game := chess.NewGame()
		ctx := &ProcessingContext{cfg: config.NewConfig()}
		if !applyPatternFilters(game, ctx, true) {
			t.Error("expected true passthrough")
		}
	})

	t.Run("returns matched as-is false", func(t *testing.T) {
		game := chess.NewGame()
		ctx := &ProcessingContext{cfg: config.NewConfig()}
		if applyPatternFilters(game, ctx, false) {
			t.Error("expected false passthrough")
		}
	})
}

// ============================================================
// Task 3: Orchestration tests
// ============================================================

func TestTruncateMoves(t *testing.T) {
	oldDropPly := *dropPly
	oldStartPly := *startPly
	oldPlyLimit := *plyLimit
	oldDropBefore := *dropBefore
	defer func() {
		*dropPly = oldDropPly
		*startPly = oldStartPly
		*plyLimit = oldPlyLimit
		*dropBefore = oldDropBefore
	}()

	countMoves := func(g *chess.Game) int {
		count := 0
		for m := g.Moves; m != nil; m = m.Next {
			count++
		}
		return count
	}

	basePGN := `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 *
`

	t.Run("no flags does nothing", func(t *testing.T) {
		*dropPly = 0
		*startPly = 0
		*plyLimit = 0
		*dropBefore = ""
		game := testutil.MustParseGame(t, basePGN)
		original := countMoves(game)
		truncateMoves(game)
		if countMoves(game) != original {
			t.Errorf("moves changed from %d to %d; expected no change", original, countMoves(game))
		}
	})

	t.Run("dropPly removes first N plies", func(t *testing.T) {
		*dropPly = 2
		*startPly = 0
		*plyLimit = 0
		*dropBefore = ""
		game := testutil.MustParseGame(t, basePGN)
		truncateMoves(game)
		got := countMoves(game)
		if got != 3 {
			t.Errorf("after dropPly=2, moves = %d; want 3", got)
		}
	})

	t.Run("plyLimit limits output", func(t *testing.T) {
		*dropPly = 0
		*startPly = 0
		*plyLimit = 2
		*dropBefore = ""
		game := testutil.MustParseGame(t, basePGN)
		truncateMoves(game)
		got := countMoves(game)
		if got != 2 {
			t.Errorf("after plyLimit=2, moves = %d; want 2", got)
		}
	})

	t.Run("startPly and plyLimit combined", func(t *testing.T) {
		*dropPly = 0
		*startPly = 1
		*plyLimit = 2
		*dropBefore = ""
		game := testutil.MustParseGame(t, basePGN)
		truncateMoves(game)
		got := countMoves(game)
		if got != 2 {
			t.Errorf("after startPly=1 plyLimit=2, moves = %d; want 2", got)
		}
	})

	t.Run("dropPly larger than startPly used", func(t *testing.T) {
		*dropPly = 3
		*startPly = 1
		*plyLimit = 0
		*dropBefore = ""
		game := testutil.MustParseGame(t, basePGN)
		truncateMoves(game)
		got := countMoves(game)
		if got != 2 {
			t.Errorf("after dropPly=3 startPly=1, moves = %d; want 2", got)
		}
	})
}

func TestInitSelectionSets(t *testing.T) {
	oldSelectOnly := *selectOnly
	oldSkipMatching := *skipMatching
	oldPlyRange := *plyRange
	oldMoveRange := *moveRange
	oldSelectOnlySet := selectOnlySet
	oldSkipMatchingSet := skipMatchingSet
	oldParsedPlyRange := parsedPlyRange
	oldParsedMoveRange := parsedMoveRange
	defer func() {
		*selectOnly = oldSelectOnly
		*skipMatching = oldSkipMatching
		*plyRange = oldPlyRange
		*moveRange = oldMoveRange
		selectOnlySet = oldSelectOnlySet
		skipMatchingSet = oldSkipMatchingSet
		parsedPlyRange = oldParsedPlyRange
		parsedMoveRange = oldParsedMoveRange
	}()

	t.Run("populates selectOnlySet", func(t *testing.T) {
		*selectOnly = "1,3,5"
		*skipMatching = ""
		*plyRange = ""
		*moveRange = ""
		selectOnlySet = nil
		skipMatchingSet = nil
		initSelectionSets()
		if len(selectOnlySet) != 3 {
			t.Errorf("selectOnlySet has %d entries; want 3", len(selectOnlySet))
		}
		if !selectOnlySet[1] || !selectOnlySet[3] || !selectOnlySet[5] {
			t.Errorf("selectOnlySet missing expected values: %v", selectOnlySet)
		}
	})

	t.Run("populates skipMatchingSet", func(t *testing.T) {
		*selectOnly = ""
		*skipMatching = "2,4"
		*plyRange = ""
		*moveRange = ""
		selectOnlySet = nil
		skipMatchingSet = nil
		initSelectionSets()
		if len(skipMatchingSet) != 2 {
			t.Errorf("skipMatchingSet has %d entries; want 2", len(skipMatchingSet))
		}
	})

	t.Run("populates plyRange", func(t *testing.T) {
		*selectOnly = ""
		*skipMatching = ""
		*plyRange = "10-20"
		*moveRange = ""
		selectOnlySet = nil
		skipMatchingSet = nil
		parsedPlyRange = [2]int{0, 0}
		initSelectionSets()
		if parsedPlyRange != [2]int{10, 20} {
			t.Errorf("parsedPlyRange = %v; want [10, 20]", parsedPlyRange)
		}
	})

	t.Run("populates moveRange", func(t *testing.T) {
		*selectOnly = ""
		*skipMatching = ""
		*plyRange = ""
		*moveRange = "5-15"
		selectOnlySet = nil
		skipMatchingSet = nil
		parsedMoveRange = [2]int{0, 0}
		initSelectionSets()
		if parsedMoveRange != [2]int{5, 15} {
			t.Errorf("parsedMoveRange = %v; want [5, 15]", parsedMoveRange)
		}
	})

	t.Run("empty flags leave sets nil", func(t *testing.T) {
		*selectOnly = ""
		*skipMatching = ""
		*plyRange = ""
		*moveRange = ""
		selectOnlySet = nil
		skipMatchingSet = nil
		initSelectionSets()
		if selectOnlySet != nil {
			t.Error("expected nil selectOnlySet")
		}
		if skipMatchingSet != nil {
			t.Error("expected nil skipMatchingSet")
		}
	})
}

func TestApplyValidation(t *testing.T) {
	oldStrict := *strictMode
	oldValidate := *validateMode
	defer func() {
		*strictMode = oldStrict
		*validateMode = oldValidate
	}()

	t.Run("both off returns nil", func(t *testing.T) {
		*strictMode = false
		*validateMode = false
		game := chess.NewGame()
		if applyValidation(game) != nil {
			t.Error("expected nil when both modes off")
		}
	})

	t.Run("strict mode with valid game", func(t *testing.T) {
		*strictMode = true
		*validateMode = false
		pgn := `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 *
`
		game := testutil.MustParseGame(t, pgn)
		result := applyValidation(game)
		if result != nil {
			t.Errorf("expected nil for valid game in strict mode; got %+v", result)
		}
	})

	t.Run("validate mode with valid game", func(t *testing.T) {
		*strictMode = false
		*validateMode = true
		pgn := `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 *
`
		game := testutil.MustParseGame(t, pgn)
		result := applyValidation(game)
		if result != nil {
			t.Errorf("expected nil for valid game in validate mode; got %+v", result)
		}
	})

	t.Run("both modes with valid game", func(t *testing.T) {
		*strictMode = true
		*validateMode = true
		pgn := `[Event "Test"]
[Site "?"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 *
`
		game := testutil.MustParseGame(t, pgn)
		result := applyValidation(game)
		if result != nil {
			t.Errorf("expected nil for valid game with both modes; got %+v", result)
		}
	})
}

func TestNeedsGameAnalysis(t *testing.T) {
	oldCheckmate := *checkmateFilter
	oldStalemate := *stalemateFilter
	oldFifty := *fiftyMoveFilter
	oldRep := *repetitionFilter
	oldUnder := *underpromotionFilter
	oldHigher := *higherRatedWinner
	oldLower := *lowerRatedWinner
	old75 := *seventyFiveMoveFilter
	old5fold := *fiveFoldRepFilter
	oldInsuf := *insufficientFilter
	oldOdds := *materialOddsFilter
	defer func() {
		*checkmateFilter = oldCheckmate
		*stalemateFilter = oldStalemate
		*fiftyMoveFilter = oldFifty
		*repetitionFilter = oldRep
		*underpromotionFilter = oldUnder
		*higherRatedWinner = oldHigher
		*lowerRatedWinner = oldLower
		*seventyFiveMoveFilter = old75
		*fiveFoldRepFilter = old5fold
		*insufficientFilter = oldInsuf
		*materialOddsFilter = oldOdds
	}()

	resetFlags := func() {
		*checkmateFilter = false
		*stalemateFilter = false
		*fiftyMoveFilter = false
		*repetitionFilter = false
		*underpromotionFilter = false
		*higherRatedWinner = false
		*lowerRatedWinner = false
		*seventyFiveMoveFilter = false
		*fiveFoldRepFilter = false
		*insufficientFilter = false
		*materialOddsFilter = false
	}

	t.Run("no flags returns false", func(t *testing.T) {
		resetFlags()
		cfg := config.NewConfig()
		ctx := &ProcessingContext{cfg: cfg}
		if needsGameAnalysis(ctx) {
			t.Error("expected false with no flags")
		}
	})

	flagTests := []struct {
		name    string
		setFlag func()
	}{
		{"checkmate", func() { *checkmateFilter = true }},
		{"stalemate", func() { *stalemateFilter = true }},
		{"fifty move", func() { *fiftyMoveFilter = true }},
		{"repetition", func() { *repetitionFilter = true }},
		{"underpromotion", func() { *underpromotionFilter = true }},
		{"higher rated winner", func() { *higherRatedWinner = true }},
		{"lower rated winner", func() { *lowerRatedWinner = true }},
		{"75 move", func() { *seventyFiveMoveFilter = true }},
		{"5fold rep", func() { *fiveFoldRepFilter = true }},
		{"insufficient", func() { *insufficientFilter = true }},
		{"material odds", func() { *materialOddsFilter = true }},
	}

	for _, tt := range flagTests {
		t.Run(tt.name+" filter", func(t *testing.T) {
			resetFlags()
			tt.setFlag()
			cfg := config.NewConfig()
			ctx := &ProcessingContext{cfg: cfg}
			if !needsGameAnalysis(ctx) {
				t.Errorf("expected true with %s filter", tt.name)
			}
		})
	}

	t.Run("AddFENComments annotation", func(t *testing.T) {
		resetFlags()
		cfg := config.NewConfig()
		cfg.Annotation.AddFENComments = true
		ctx := &ProcessingContext{cfg: cfg}
		if !needsGameAnalysis(ctx) {
			t.Error("expected true with AddFENComments")
		}
	})

	t.Run("AddHashComments annotation", func(t *testing.T) {
		resetFlags()
		cfg := config.NewConfig()
		cfg.Annotation.AddHashComments = true
		ctx := &ProcessingContext{cfg: cfg}
		if !needsGameAnalysis(ctx) {
			t.Error("expected true with AddHashComments")
		}
	})

	t.Run("AddHashTag annotation", func(t *testing.T) {
		resetFlags()
		cfg := config.NewConfig()
		cfg.Annotation.AddHashTag = true
		ctx := &ProcessingContext{cfg: cfg}
		if !needsGameAnalysis(ctx) {
			t.Error("expected true with AddHashTag")
		}
	})
}

func TestIncrementGamePosition(t *testing.T) {
	oldCounter := atomic.LoadInt64(&gamePositionCounter)
	defer atomic.StoreInt64(&gamePositionCounter, oldCounter)

	atomic.StoreInt64(&gamePositionCounter, 0)
	pos1 := IncrementGamePosition()
	if pos1 != 1 {
		t.Errorf("first IncrementGamePosition() = %d; want 1", pos1)
	}
	pos2 := IncrementGamePosition()
	if pos2 != 2 {
		t.Errorf("second IncrementGamePosition() = %d; want 2", pos2)
	}
}
