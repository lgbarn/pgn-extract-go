# Phase 4 Research: Test Coverage — Matching Package

**Date:** 2026-01-31
**Package:** `internal/matching`
**Current Coverage:** 34.6%
**Target Coverage:** >70%

## Executive Summary

The `internal/matching` package provides game filtering and matching capabilities through tags, positions, variations, and material patterns. Current test coverage is 34.6%, with significant gaps in five key files: `filter.go`, `position.go`, `variation.go`, `material.go`, and `tags.go`.

**Key Finding:** Of 50 functions in the package, 21 have 0% coverage (42% completely untested), and another 8 have partial coverage below 90%. The existing test suite uses good patterns and helpers from `internal/testutil`, making it straightforward to extend.

## Current Coverage Analysis

### Overall Package Coverage
```
ok  	github.com/lgbarn/pgn-extract-go/internal/matching	0.321s	coverage: 34.6% of statements
```

### Coverage by File

| File | Well-Tested Functions | Partially Tested | Untested | Notes |
|------|----------------------|------------------|----------|-------|
| **filter.go** | 4/15 (27%) | 1/15 (7%) | 10/15 (66%) | Core function `MatchGame` has 85.7% coverage |
| **position.go** | 4/15 (27%) | 3/15 (20%) | 8/15 (53%) | Pattern matching completely untested |
| **variation.go** | 2/14 (14%) | 0/14 (0%) | 12/14 (86%) | Almost entirely untested |
| **material.go** | 3/11 (27%) | 1/11 (9%) | 7/11 (64%) | Matching logic completely untested |
| **tags.go** | 7/13 (54%) | 3/13 (23%) | 3/13 (23%) | Best coverage, but still gaps |
| **matcher.go** | 2/6 (33%) | 2/6 (33%) | 2/6 (33%) | Core `Match()` at 91.7% |
| **soundex.go** | 2/3 (67%) | 1/3 (33%) | 0/3 (0%) | Good coverage |

## Detailed Function Coverage

### filter.go (10 functions at 0% coverage)

| Function | Coverage | Priority | Notes |
|----------|----------|----------|-------|
| `NewGameFilter` | 100.0% | - | Tested |
| `LoadTagFile` | **0.0%** | HIGH | File I/O, parsing, FEN pattern detection |
| `AddTagCriterion` | **0.0%** | LOW | Thin wrapper around TagMatcher |
| `AddPlayerFilter` | 100.0% | - | Tested |
| `AddWhiteFilter` | **0.0%** | MEDIUM | Simple but untested |
| `AddBlackFilter` | **0.0%** | MEDIUM | Simple but untested |
| `AddECOFilter` | 100.0% | - | Tested |
| `AddResultFilter` | 100.0% | - | Tested |
| `AddDateFilter` | **0.0%** | MEDIUM | Operator support untested |
| `AddFENFilter` | **0.0%** | HIGH | Position matching entry point |
| `AddPatternFilter` | **0.0%** | HIGH | Wildcard pattern support |
| `MatchGame` | 85.7% | LOW | Mostly tested, likely edge case |
| `HasCriteria` | **0.0%** | LOW | Simple getter |
| `SetUseSoundex` | **0.0%** | LOW | Simple setter |
| `SetSubstringMatch` | **0.0%** | LOW | Simple setter |
| `Match` | 100.0% | - | Interface wrapper |
| `Name` | 100.0% | - | Interface wrapper |

**Coverage Impact:** Testing `LoadTagFile`, `AddFENFilter`, and `AddPatternFilter` would significantly increase coverage.

### position.go (8 functions at 0% coverage)

| Function | Coverage | Priority | Notes |
|----------|----------|----------|-------|
| `NewPositionMatcher` | 100.0% | - | Tested |
| `AddFEN` | 87.5% | LOW | Mostly tested, error case missing? |
| `AddPattern` | **0.0%** | HIGH | Wildcard pattern support, including invert |
| `MatchGame` | 63.6% | MEDIUM | Partial coverage, edge cases needed |
| `getStartingBoard` | 60.0% | MEDIUM | FEN tag handling partially tested |
| `matchPosition` | 85.7% | LOW | Good coverage |
| `matchPattern` | **0.0%** | HIGH | Core pattern matching logic |
| `boardToRanks` | **0.0%** | MEDIUM | Board conversion utility |
| `pieceToChar` | **0.0%** | MEDIUM | FEN character conversion |
| `matchRank` | **0.0%** | HIGH | Wildcard matching (?, !, *, A, a, _, digits) |
| `invertPattern` | **0.0%** | HIGH | Color inversion for patterns |
| `PatternCount` | 100.0% | - | Tested |

**Coverage Impact:** The entire pattern matching subsystem is untested. Testing `matchPattern`, `matchRank`, and `invertPattern` would add ~200+ lines of coverage.

### variation.go (12 functions at 0% coverage)

| Function | Coverage | Priority | Notes |
|----------|----------|----------|-------|
| `NewVariationMatcher` | 100.0% | - | Tested (name only) |
| `LoadFromFile` | **0.0%** | HIGH | File I/O and move sequence parsing |
| `LoadPositionalFromFile` | **0.0%** | HIGH | File I/O and FEN sequence parsing |
| `AddMoveSequence` | **0.0%** | MEDIUM | Simple but untested |
| `MatchGame` | **0.0%** | HIGH | Core matching logic |
| `matchMoveSequence` | **0.0%** | HIGH | Move sequence matching with reset logic |
| `matchPositionSequence` | **0.0%** | HIGH | FEN position sequence matching |
| `parseMoveSequence` | **0.0%** | MEDIUM | Move text parsing |
| `normalizeMove` | **0.0%** | MEDIUM | Move normalization |
| `matchesFENPosition` | **0.0%** | MEDIUM | FEN comparison |
| `HasCriteria` | **0.0%** | LOW | Simple getter |
| `SetMatchAnywhere` | **0.0%** | LOW | Simple setter |
| `Match` | **0.0%** | MEDIUM | Interface wrapper |
| `Name` | 100.0% | - | Tested (name only) |

**Coverage Impact:** Variation matching is completely untested except for constructor/name. This is the largest coverage opportunity (~150-200 lines).

### material.go (7 functions at 0% coverage)

| Function | Coverage | Priority | Notes |
|----------|----------|----------|-------|
| `NewMaterialMatcher` | 100.0% | - | Tested |
| `parsePattern` | 100.0% | - | Tested |
| `parsePieces` | 54.5% | MEDIUM | Partial coverage, missing some piece types? |
| `MatchGame` | **0.0%** | HIGH | Core matching logic |
| `matchPosition` | **0.0%** | HIGH | Board scanning and piece counting |
| `exactMaterialMatch` | **0.0%** | HIGH | Exact material comparison |
| `minimalMaterialMatch` | **0.0%** | HIGH | Minimal material comparison |
| `HasCriteria` | **0.0%** | LOW | Simple getter |
| `Match` | **0.0%** | MEDIUM | Interface wrapper |
| `Name` | 100.0% | - | Tested |

**Coverage Impact:** Material matching is only tested at the parsing level. Runtime matching is completely untested (~80-100 lines).

### tags.go (3 functions at 0% coverage)

| Function | Coverage | Priority | Notes |
|----------|----------|----------|-------|
| `NewTagMatcher` | 100.0% | - | Tested |
| `SetMatchAll` | **0.0%** | MEDIUM | OR mode untested |
| `SetUseSoundex` | **0.0%** | LOW | Feature flag setter |
| `SetSubstringMatch` | **0.0%** | LOW | Feature flag setter |
| `AddCriterion` | 58.3% | MEDIUM | Regex and soundex branches likely untested |
| `AddSimpleCriterion` | 100.0% | - | Tested |
| `AddPlayerCriterion` | 75.0% | LOW | Soundex branch likely untested |
| `ParseCriterion` | 65.7% | MEDIUM | Some operators untested (!=, <=, >=, ~) |
| `MatchGame` | 77.8% | LOW | OR mode likely untested |
| `matchCriterion` | 87.5% | LOW | Good coverage |
| `matchValue` | 40.0% | HIGH | Regex and soundex operators untested |
| `compareValues` | 26.1% | HIGH | String comparison fallback untested |
| `parseDate` | 86.7% | LOW | Mostly tested, edge cases? |
| `CriteriaCount` | 100.0% | - | Tested |

**Coverage Impact:** Tag matching has the best coverage but still missing regex, soundex, OR mode, and some operators (~50-80 lines).

### matcher.go (2 functions at 0% coverage)

| Function | Coverage | Priority | Notes |
|----------|----------|----------|-------|
| `NewCompositeMatcher` | 100.0% | - | Tested |
| `Match` | 91.7% | LOW | Excellent coverage |
| `Name` | 77.8% | LOW | Good coverage |
| `Add` | **0.0%** | LOW | Untested but simple |
| `Matchers` | **0.0%** | LOW | Untested getter |
| `Mode` | **0.0%** | LOW | Untested getter |

**Coverage Impact:** Composite matcher is well-tested. Remaining functions are low-priority getters (~10 lines).

### soundex.go (already well-tested)

| Function | Coverage | Priority | Notes |
|----------|----------|----------|-------|
| `Soundex` | 90.5% | LOW | Excellent coverage |
| `soundexCode` | 100.0% | - | Complete |
| `SoundexMatch` | **0.0%** | LOW | Simple wrapper, 1 line |

## Existing Test Patterns

### Test File Structure
The package has a single comprehensive test file: `matching_test.go` (367 lines).

### Test Patterns Used

1. **Table-Driven Tests**
   ```go
   tests := []struct {
       name1, name2 string
       shouldMatch  bool
   }{...}
   ```

2. **Game Construction via testutil.ParseTestGame()**
   ```go
   game := testutil.ParseTestGame(`
   [Event "Test"]
   [White "Player1"]
   ...
   1. e4 e5 *
   `)
   ```

3. **Direct Assertion Style**
   ```go
   if !matcher.MatchGame(game) {
       t.Error("Expected match")
   }
   ```

4. **Interface Verification**
   ```go
   var _ GameMatcher = NewGameFilter()
   ```

### Test Helpers Available

From `internal/testutil`:
- `ParseTestGame(pgn string) *chess.Game` - Parse single game, returns nil on error
- `ParseTestGames(pgn string) []*chess.Game` - Parse multiple games
- `MustParseGame(t, pgn)` - Parse or fatal
- `MustParseGames(t, pgn)` - Parse multiple or fatal

### Common Test Game Patterns

```go
// Simple two-move game
game := testutil.ParseTestGame(`
[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 *
`)

// Ruy Lopez position (for position matching)
1. e4 e5 2. Nf3 Nc6 3. Bb5 *
// Results in FEN: r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3
```

## Dependencies and Setup

### Internal Dependencies
- `internal/chess` - Core game structures (`chess.Game`, `chess.Board`, `chess.Piece`)
- `internal/engine` - Board operations (`NewBoardFromFEN`, `ApplyMove`, `BoardToFEN`)
- `internal/hashing` - Zobrist hashing for position matching
- `internal/config` - Used by testutil for parser setup
- `internal/parser` - Used by testutil for PGN parsing

### Test Setup Requirements

**Minimal setup:**
```go
import (
    "testing"
    "github.com/lgbarn/pgn-extract-go/internal/matching"
    "github.com/lgbarn/pgn-extract-go/internal/testutil"
)
```

**No complex initialization needed** - all matchers have simple constructors.

## Untested/Under-Tested Function Analysis

### High-Impact Targets (Maximum Coverage Gain)

Testing these 12 functions would add the most coverage with the least effort:

1. **variation.go: MatchGame, matchMoveSequence, matchPositionSequence** (~80 lines)
   - Core variation matching logic
   - Test with simple move sequences: "1. e4 e5", "1. d4 d5 2. c4"
   - Test positional sequences with FEN positions

2. **position.go: matchPattern, matchRank** (~90 lines)
   - Wildcard pattern matching
   - Test patterns: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR" (exact)
   - Test wildcards: "?", "!", "*", "A", "a", "_", digits
   - Test inverted patterns

3. **material.go: MatchGame, matchPosition, exactMaterialMatch, minimalMaterialMatch** (~70 lines)
   - Material balance matching
   - Test exact: "Q:q" (white has Q, black has q, nothing else)
   - Test minimal: "QR:qr" (white has at least QR, black at least qr)

4. **filter.go: LoadTagFile, AddFENFilter, AddPatternFilter** (~40 lines)
   - File loading and FEN/pattern integration
   - Test tag file parsing with various criterion formats
   - Test FEN exact matching
   - Test pattern matching with wildcards

5. **tags.go: matchValue (regex, soundex), compareValues (string fallback)** (~30 lines)
   - Missing operators and comparison modes
   - Test OpRegex with pattern matching
   - Test OpSoundex with player name variants
   - Test string comparison fallback for non-numeric/non-date values

### Testing Strategy by Priority

#### Priority 1: Core Matching Logic (Target: +25% coverage)
- `variation.go`: MatchGame, matchMoveSequence, matchPositionSequence
- `material.go`: MatchGame, matchPosition, exactMaterialMatch, minimalMaterialMatch
- `position.go`: matchPattern, matchRank

**Estimated impact:** 240+ lines of coverage

#### Priority 2: Pattern Support (Target: +8% coverage)
- `position.go`: invertPattern, boardToRanks, pieceToChar, AddPattern
- `variation.go`: parseMoveSequence, normalizeMove, matchesFENPosition

**Estimated impact:** 100+ lines of coverage

#### Priority 3: File I/O and Integration (Target: +5% coverage)
- `filter.go`: LoadTagFile, AddFENFilter, AddPatternFilter
- `variation.go`: LoadFromFile, LoadPositionalFromFile

**Estimated impact:** 60+ lines of coverage

#### Priority 4: Missing Operators and Modes (Target: +4% coverage)
- `tags.go`: matchValue (OpRegex, OpSoundex), compareValues (string comparison)
- `tags.go`: SetMatchAll, MatchGame (OR mode)
- `tags.go`: ParseCriterion (!=, <=, >=, ~)

**Estimated impact:** 50+ lines of coverage

#### Priority 5: Simple Getters/Setters (Target: +1% coverage)
- All `HasCriteria()`, `SetMatchAnywhere()`, etc.
- `matcher.go`: Add, Matchers, Mode
- `soundex.go`: SoundexMatch

**Estimated impact:** 15+ lines of coverage

**Total Estimated Coverage Gain:** 465+ lines = ~42% additional coverage → **~77% total coverage**

## Recommended Testing Approach

### Phase 1: Core Matching Logic (Days 1-2)
Focus on getting basic game matching working for all matcher types.

**Tests to write:**
1. `TestVariationMatcher_BasicSequence` - Simple move sequence matching
2. `TestVariationMatcher_PositionalSequence` - FEN position sequence
3. `TestMaterialMatcher_ExactMatch` - Exact material balance
4. `TestMaterialMatcher_MinimalMatch` - Minimal material balance
5. `TestPositionMatcher_Patterns` - Wildcard patterns (?, !, *, A, a, _, digits)
6. `TestPositionMatcher_InvertPattern` - Color inversion

**Success criteria:** Variation, material, and pattern matching all functional

### Phase 2: Pattern Matching Details (Day 3)
Deep dive into position pattern matching edge cases.

**Tests to write:**
1. `TestMatchRank_Wildcards` - Test all wildcard types
2. `TestMatchRank_Numbers` - Test digit-based empty square matching
3. `TestMatchRank_Star` - Test * (zero or more) greedy matching
4. `TestBoardToRanks` - Verify board conversion correctness
5. `TestPieceToChar` - All piece types (PNBRQK, pnbrqk, empty)

**Success criteria:** All position pattern features working

### Phase 3: File I/O and Integration (Day 4)
Test file loading and multi-criteria filters.

**Tests to write:**
1. `TestGameFilter_LoadTagFile` - Tag criterion file parsing
2. `TestGameFilter_FENAndPattern` - Combined tag+position filtering
3. `TestVariationMatcher_LoadFromFile` - Move sequence file
4. `TestVariationMatcher_LoadPositionalFromFile` - FEN sequence file

**Success criteria:** File-based configuration working

### Phase 4: Advanced Operators (Day 5)
Fill in missing tag operators and comparison modes.

**Tests to write:**
1. `TestTagMatcher_Regex` - OpRegex matching
2. `TestTagMatcher_Soundex` - OpSoundex for player names
3. `TestTagMatcher_OrMode` - SetMatchAll(false) with multiple criteria
4. `TestTagMatcher_AllOperators` - Table-driven test for !=, <=, >=, ~
5. `TestCompareValues_StringFallback` - Non-numeric, non-date comparison

**Success criteria:** All operators and modes working

### Phase 5: Coverage Cleanup (Day 6)
Fill in remaining gaps and edge cases.

**Tests to write:**
1. Simple getter/setter tests
2. Edge cases from coverage report
3. Error handling paths
4. Boundary conditions

**Success criteria:** Coverage > 70%

## Potential Risks and Mitigations

### Risk 1: Pattern Matching Complexity
**Description:** `matchRank` has complex wildcard logic with recursive calls for `*`. Edge cases may be hard to enumerate.

**Mitigation:**
- Start with simple patterns (no wildcards)
- Add one wildcard type at a time
- Use table-driven tests with many small cases
- Test `*` separately with known edge cases (beginning, middle, end, empty)

### Risk 2: File I/O Testing
**Description:** File loading functions require actual files or mocks.

**Mitigation:**
- Use `os.CreateTemp()` to write test files in `/tmp`
- Clean up with `defer os.Remove()`
- Test both valid and invalid file contents
- Test empty files, comments, malformed lines

**Example:**
```go
func TestLoadTagFile(t *testing.T) {
    content := `White "Fischer"
# Comment
Date >= "1970.01.01"
`
    tmpfile, _ := os.CreateTemp("", "test*.txt")
    defer os.Remove(tmpfile.Name())
    tmpfile.WriteString(content)
    tmpfile.Close()

    gf := NewGameFilter()
    err := gf.LoadTagFile(tmpfile.Name())
    // assertions...
}
```

### Risk 3: FEN Position Generation
**Description:** Testing position/material matching requires games that reach specific board states.

**Mitigation:**
- Use known openings with standard FEN positions (Ruy Lopez, Sicilian, etc.)
- Use testutil.ParseTestGame() with short move sequences
- Document the expected FEN in test comments
- For complex positions, use FEN tag in PGN header

**Example:**
```go
game := testutil.ParseTestGame(`
[Event "Test"]
[FEN "rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w KQkq c6 0 2"]
...
`)
```

### Risk 4: Soundex Implementation Variations
**Description:** Soundex has multiple implementations; behavior may vary on edge cases.

**Mitigation:**
- Test against known soundex codes from existing test (Fischer, Kasparov, etc.)
- Focus on chess player names (already in Soundex implementation comments)
- Don't over-test edge cases that aren't relevant to chess names

## Test File Organization

### Current State
All tests in `matching_test.go` (367 lines).

### Recommended Organization (Optional)
Could split into focused files if test count grows significantly:
- `matching_test.go` - Core tests and helpers
- `filter_test.go` - GameFilter tests
- `position_test.go` - PositionMatcher tests
- `variation_test.go` - VariationMatcher tests
- `material_test.go` - MaterialMatcher tests
- `tags_test.go` - TagMatcher tests

**Decision:** Keep single file for now unless it exceeds ~1000 lines.

## Implementation Checklist

- [ ] Phase 1: Core Matching Logic (6 tests, ~150 LOC)
  - [ ] TestVariationMatcher_BasicSequence
  - [ ] TestVariationMatcher_PositionalSequence
  - [ ] TestMaterialMatcher_ExactMatch
  - [ ] TestMaterialMatcher_MinimalMatch
  - [ ] TestPositionMatcher_Patterns
  - [ ] TestPositionMatcher_InvertPattern

- [ ] Phase 2: Pattern Matching Details (5 tests, ~100 LOC)
  - [ ] TestMatchRank_Wildcards
  - [ ] TestMatchRank_Numbers
  - [ ] TestMatchRank_Star
  - [ ] TestBoardToRanks
  - [ ] TestPieceToChar

- [ ] Phase 3: File I/O and Integration (4 tests, ~120 LOC)
  - [ ] TestGameFilter_LoadTagFile
  - [ ] TestGameFilter_FENAndPattern
  - [ ] TestVariationMatcher_LoadFromFile
  - [ ] TestVariationMatcher_LoadPositionalFromFile

- [ ] Phase 4: Advanced Operators (5 tests, ~80 LOC)
  - [ ] TestTagMatcher_Regex
  - [ ] TestTagMatcher_Soundex
  - [ ] TestTagMatcher_OrMode
  - [ ] TestTagMatcher_AllOperators
  - [ ] TestCompareValues_StringFallback

- [ ] Phase 5: Coverage Cleanup (8 tests, ~50 LOC)
  - [ ] TestGameFilter_Setters
  - [ ] TestPositionMatcher_EdgeCases
  - [ ] TestVariationMatcher_EdgeCases
  - [ ] TestMaterialMatcher_EdgeCases
  - [ ] TestCompositeMatcher_Getters
  - [ ] TestSoundexMatch
  - [ ] Verify coverage > 70%
  - [ ] Document any remaining gaps

**Total Estimated Effort:** 28 new test functions, ~500 lines of test code

## Open Questions

1. **Pattern matching behavior:** Are there documented examples of FEN patterns with wildcards in the original pgn-extract? This would help verify correct behavior.

2. **Move sequence matching:** Should `matchMoveSequence` match only from game start, or anywhere in the game? The `matchAnywhere` field exists but isn't used yet.

3. **Material pattern format:** Are there other pattern formats beyond "QR:qrr"? Should we support patterns like "QR:qr+" for "at least" logic?

4. **File formats:** Are there example tag files, variation files, or positional variation files we can use as test fixtures?

## Conclusion

Raising test coverage from 34.6% to >70% is achievable by focusing on the core matching logic (variation, material, position patterns) which represents the largest untested surface area. The existing test patterns are clean and easy to follow, and the testutil package provides good game construction helpers.

**Recommended Priority Order:**
1. Variation matching (highest impact, completely untested)
2. Material matching (second highest impact, completely untested)
3. Position pattern matching (wildcards, inversion - high complexity)
4. File I/O integration
5. Missing operators and modes
6. Simple getters/setters

**Expected Outcome:** Following this plan should achieve 75-80% coverage with ~500 lines of well-structured test code.
