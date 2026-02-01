---
phase: test-coverage-matching
plan: 13
wave: 1
dependencies: []
must_haves:
  - Tests for GameFilter file loading and tag file parsing
  - Tests for filter composition (tag AND position criteria)
  - Tests for all filter helper methods (AddWhiteFilter, AddBlackFilter, AddECOFilter, etc)
  - Tests for TagMatcher regex and soundex modes
  - Tests for TagMatcher OR mode (SetMatchAll false)
files_touched:
  - internal/matching/filter_test.go
  - internal/matching/tags_test.go
tdd: false
---

# Plan 1.3: Filter and Tag Matcher Test Coverage

**Goal**: Raise filter.go coverage from 33% to >75% (~+5%) and tags.go coverage from 75% to >90% (~+4%).

**Context**: filter.go has 10/15 functions at 0%, covering GameFilter composition and file loading. tags.go is missing tests for regex matching (OpRegex), soundex mode, and OR mode (SetMatchAll false). These are integration points for the matching system.

## Tasks

<task id="1" files="internal/matching/filter_test.go" tdd="false">
  <action>Create comprehensive tests for GameFilter in filter_test.go. Test LoadTagFile with mixed tag criteria and FEN patterns, HasCriteria, and all helper methods (AddWhiteFilter, AddBlackFilter, AddECOFilter, AddResultFilter, AddDateFilter, AddFENFilter, AddPatternFilter, AddTagCriterion). Test RequireBoth mode for AND logic between tags and positions. Use t.TempDir() for temporary tag files with various criterion formats.</action>
  <verify>go test -v -run TestGameFilter_Load ./internal/matching && go test -v -run TestGameFilter_Helpers ./internal/matching && go test -cover ./internal/matching/filter.go</verify>
  <done>GameFilter has passing tests for file loading, all helper methods, and tag+position composition. filter.go shows >75% coverage</done>
</task>

<task id="2" files="internal/matching/tags_test.go" tdd="false">
  <action>Create tests for TagMatcher regex and soundex functionality in tags_test.go. Test OpRegex with compiled regex patterns (test both matches and non-matches), regex compilation errors. Test soundex matching mode with SetUseSoundex(true) and verify phonetic matching for player names (Fischer/Fisher, Carlsen/Carlson). Use table-driven tests with known regex patterns and soundex pairs.</action>
  <verify>go test -v -run TestTagMatcher_Regex ./internal/matching && go test -v -run TestTagMatcher_Soundex ./internal/matching</verify>
  <done>OpRegex and soundex matching have passing tests covering valid regexes, invalid regexes, and phonetic name matching</done>
</task>

<task id="3" files="internal/matching/tags_test.go" tdd="false">
  <action>Create tests for TagMatcher OR mode and edge cases. Test SetMatchAll(false) for OR logic where any criterion matching is sufficient, test OpNotEqual with missing tags, test substringMatch mode with SetSubstringMatch(true), and test numeric comparison with non-date values. Verify AND vs OR behavior with multiple criteria on different tags.</action>
  <verify>go test -v -run TestTagMatcher_OR ./internal/matching && go test -v -run TestTagMatcher_NotEqual ./internal/matching && go test -cover ./internal/matching/tags.go</verify>
  <done>OR mode, OpNotEqual, and substringMatch have passing tests. tags.go shows >90% coverage in test output</done>
</task>
