---
phase: test-coverage-matching
plan: 12
wave: 1
dependencies: []
must_haves:
  - Tests for NewMaterialMatcher and parsePattern/parsePieces
  - Tests for exact vs minimal material matching (QR:qrr patterns)
  - Tests for material balance checking through game positions
  - Tests for position pattern matching with wildcards (?!*Aa_)
  - Tests for InvertPattern and pattern inversion
files_touched:
  - internal/matching/material_test.go
  - internal/matching/position_test.go
tdd: false
---

# Plan 1.2: Material and Position Matcher Test Coverage

**Goal**: Raise material.go coverage from 36% to >85% (~+8%) and position.go coverage from 47% to >85% (~+10%).

**Context**: material.go has 7/11 functions at 0%, position.go has 8/15 functions at 0%. These files handle material balance matching (QR:qrr patterns) and FEN pattern matching with wildcards. Current position tests only cover exact FEN matching.

## Tasks

<task id="1" files="internal/matching/material_test.go" tdd="false">
  <action>Create comprehensive tests for MaterialMatcher in material_test.go. Test NewMaterialMatcher with various patterns (QR:qrr, K:k, KQRBNP:kqrbnp), parsePattern/parsePieces with uppercase/lowercase handling, and HasCriteria. Test both exact matching (all pieces must match exactly) and minimal matching (at least the specified pieces). Use testutil.MustParseGame() to create games that reach specific material balances (e.g., queen trade, rook endgame).</action>
  <verify>go test -v -run TestMaterialMatcher ./internal/matching && go test -cover ./internal/matching/material.go</verify>
  <done>MaterialMatcher has passing tests covering pattern parsing, exact matching, minimal matching, and material balance detection through game positions. material.go shows >85% coverage</done>
</task>

<task id="2" files="internal/matching/position_test.go" tdd="false">
  <action>Create tests for FEN pattern matching with wildcards in position_test.go. Test AddPattern with wildcards (? matches any square, ! matches non-empty, * matches zero or more, A matches white piece, a matches black piece, _ matches empty, digits for empty square runs). Test matchRank, matchPattern, boardToRanks, and pieceToChar functions. Use table-driven tests with known board positions and patterns.</action>
  <verify>go test -v -run TestPositionMatcher_Pattern ./internal/matching && go test -v -run TestMatchRank ./internal/matching</verify>
  <done>Pattern matching functions have passing tests covering all wildcard types and edge cases (*, nested patterns, boundary conditions)</done>
</task>

<task id="3" files="internal/matching/position_test.go" tdd="false">
  <action>Create tests for InvertPattern and pattern inversion functionality. Test color inversion (uppercase to lowercase and vice versa), rank reversal, and the IncludeInvert flag on AddPattern. Test getStartingBoard with FEN tag and default initial position. Verify inverted patterns match color-flipped positions correctly.</action>
  <verify>go test -v -run TestInvertPattern ./internal/matching && go test -v -run TestPositionMatcher_Invert ./internal/matching && go test -cover ./internal/matching/position.go</verify>
  <done>InvertPattern and related functions have passing tests. position.go shows >85% coverage in test output</done>
</task>
