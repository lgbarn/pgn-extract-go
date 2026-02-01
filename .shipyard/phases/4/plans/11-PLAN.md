---
phase: test-coverage-matching
plan: 11
wave: 1
dependencies: []
must_haves:
  - Tests for LoadFromFile and LoadPositionalFromFile
  - Tests for parseMoveSequence, matchMoveSequence, matchPositionSequence
  - Tests for VariationMatcher with move sequences and positional sequences
  - Tests for SetMatchAnywhere and HasCriteria
files_touched:
  - internal/matching/variation_test.go
tdd: false
---

# Plan 1.1: Variation Matcher Test Coverage

**Goal**: Raise variation.go coverage from ~14% to >85% (target ~+25% overall coverage gain).

**Context**: variation.go has 12/14 functions at 0% coverage. This file handles move sequence matching and positional variation matching, which are critical filtering features. Current tests only cover Name() and Match() interface methods.

## Tasks

<task id="1" files="internal/matching/variation_test.go" tdd="false">
  <action>Create comprehensive tests for LoadFromFile and LoadPositionalFromFile in variation_test.go. Test file parsing with move sequences (1. e4 e5 2. Nf3), comments (#), empty lines, and positional sequences (FEN strings separated by blank lines). Use t.TempDir() for temporary test files.</action>
  <verify>go test -v -run TestVariationMatcher_LoadFromFile ./internal/matching && go test -v -run TestVariationMatcher_LoadPositionalFromFile ./internal/matching</verify>
  <done>Both LoadFromFile and LoadPositionalFromFile have passing tests covering valid files, empty lines, comments, and error cases</done>
</task>

<task id="2" files="internal/matching/variation_test.go" tdd="false">
  <action>Create tests for move sequence matching functions (parseMoveSequence, matchMoveSequence, normalizeMove). Test move number filtering (1. 2. ...), annotation removal (+#!?), contiguous sequence matching, and sequence reset on mismatch. Use testutil.MustParseGame() for test games with known move sequences like Scholar's Mate or Italian Opening.</action>
  <verify>go test -v -run TestVariationMatcher_MoveSequence ./internal/matching && go test -v -run TestParseMoveSequence ./internal/matching && go test -v -run TestNormalizeMove ./internal/matching</verify>
  <done>parseMoveSequence, matchMoveSequence, and normalizeMove functions have passing tests with table-driven test cases covering edge cases (annotations, move numbers, partial matches)</done>
</task>

<task id="3" files="internal/matching/variation_test.go" tdd="false">
  <action>Create tests for positional sequence matching (matchPositionSequence, matchesFENPosition) and matcher configuration (SetMatchAnywhere, HasCriteria, AddMoveSequence). Test FEN position comparison with partial FEN (piece placement only), position sequences through game moves, and criteria detection. Use games that reach known FEN positions.</action>
  <verify>go test -v -run TestVariationMatcher_PositionSequence ./internal/matching && go test -v -run TestVariationMatcher_Config ./internal/matching && go test -cover ./internal/matching/variation.go</verify>
  <done>All positional matching and configuration functions have passing tests. variation.go shows >85% coverage in test output</done>
</task>
