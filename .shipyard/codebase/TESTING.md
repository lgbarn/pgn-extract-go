# Testing Infrastructure

This document describes the test framework, coverage patterns, and testing conventions used in pgn-extract-go.

## Test Framework

**Go's Built-in Testing:**
- Standard library `testing` package
- No external test frameworks or assertion libraries
- Table-driven tests as the primary pattern

**Test Execution:**
```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./internal/parser/...

# Run specific test
go test -v ./... -run TestFunctionName

# Run with race detector
go test -race ./...

# Short mode (skip long-running tests)
go test -short ./...

# Generate coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test File Organization

### File Naming

**Test Files:**
- `*_test.go` - Unit tests (same package)
- `benchmark_test.go` - Performance benchmarks (separate file for clarity)
- Integration tests in `cmd/pgn-extract/*_test.go`

**Test Data:**
- `testdata/` - Test fixtures, sample PGN files, golden outputs
- `testdata/infiles/` - Input PGN files for integration tests
- `testdata/eco.pgn` - ECO classification database for tests

### Package Organization

```
87 total Go source files
29 test files (33% of files are tests)

Test distribution by package:
├── cmd/pgn-extract/
│   ├── clock_test.go          # Clock annotation tests
│   ├── features_test.go       # Feature flag integration tests
│   ├── filters_test.go        # Filter function unit tests
│   ├── golden_test.go         # Golden file integration tests
│   └── parallel_test.go       # Parallel processing tests
├── internal/chess/
│   └── board_test.go          # Board state and operations
├── internal/config/
│   └── config_test.go         # Configuration validation
├── internal/cql/
│   ├── advanced_test.go       # Complex CQL queries
│   ├── evaluator_test.go      # CQL evaluation logic
│   ├── lexer_test.go          # CQL tokenization
│   ├── parser_test.go         # CQL parsing
│   └── transforms_test.go     # CQL transformations
├── internal/eco/
│   ├── eco_test.go            # ECO classification
│   └── benchmark_test.go      # ECO performance
├── internal/engine/
│   ├── apply_test.go          # Move application
│   ├── fen_test.go            # FEN parsing/generation
│   └── benchmark_test.go      # Engine performance
├── internal/errors/
│   └── errors_test.go         # Error handling
├── internal/hashing/
│   ├── hashing_test.go        # Hash algorithms
│   ├── thread_safe_test.go    # Concurrent hashing
│   └── benchmark_test.go      # Hash performance
├── internal/matching/
│   ├── matching_test.go       # Game matching
│   └── benchmark_test.go      # Matching performance
├── internal/output/
│   └── writer_test.go         # Output formatting
├── internal/parser/
│   ├── parser_test.go         # PGN parsing
│   └── benchmark_test.go      # Parser performance
├── internal/processing/
│   └── processing_test.go     # Processing pipeline
├── internal/testutil/
│   └── game_test.go           # Test utility tests
└── internal/worker/
    └── pool_test.go           # Worker pool
```

## Test Coverage

**Current Coverage by Package** (as of latest run):

| Package | Coverage | Notes |
|---------|----------|-------|
| worker | 92.5% | Excellent - Worker pool thoroughly tested |
| cql | 80.0% | Good - Complex CQL queries covered |
| eco | 73.0% | Good - ECO classification logic |
| errors | 72.1% | Good - Error types and wrapping |
| testutil | 66.7% | Adequate - Test helpers |
| hashing | 65.4% | Adequate - Zobrist and duplicate detection |
| parser | 63.9% | Adequate - PGN parsing |
| engine | 62.3% | Adequate - Move validation and board state |
| chess | 52.4% | Moderate - Core types |
| config | 50.0% | Moderate - Configuration |
| output | 49.7% | Moderate - Output formatting |
| processing | 36.1% | Low - Pipeline logic |
| matching | 34.6% | Low - Game filtering |
| cmd/pgn-extract | 1.6% | Very low - CLI layer (integration tested) |

**Coverage Trends:**
- Core logic packages (cql, eco, worker) have strong coverage (70-90%)
- Infrastructure packages (parser, engine, hashing) have adequate coverage (60-70%)
- Integration points (matching, processing, cmd) have lower unit test coverage but are tested via integration tests

## Test Patterns

### Table-Driven Tests

**Primary Pattern** - Used extensively throughout the codebase:

```go
// From internal/chess/board_test.go
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
}
```

**Benefits:**
- Clear test case documentation
- Easy to add new test cases
- Self-documenting test names via `t.Run()`
- Good error messages with context

### Subtests

**Using `t.Run()` for Organization:**

```go
// From internal/config/config_test.go
func TestOutputConfig_Defaults(t *testing.T) {
    cfg := NewOutputConfig()

    if cfg.Format != SAN {
        t.Errorf("Format = %v, want %v", cfg.Format, SAN)
    }
    if cfg.MaxLineLength != 80 {
        t.Errorf("MaxLineLength = %d, want 80", cfg.MaxLineLength)
    }
}
```

Multiple related checks in a single test function, or use subtests:

```go
func TestBoard(t *testing.T) {
    t.Run("initial state", func(t *testing.T) { ... })
    t.Run("all squares empty", func(t *testing.T) { ... })
    t.Run("hedge squares are Off", func(t *testing.T) { ... })
}
```

### Test Helpers

**Using `t.Helper()`** - Mark helper functions:

```go
// From internal/testutil/game_test.go
func assertTag(t *testing.T, game interface{ GetTag(string) string }, tag, want string) {
    t.Helper()  // Ensures error lines point to caller, not helper
    if want == "" {
        return
    }
    if got := game.GetTag(tag); got != want {
        t.Errorf("game.GetTag(%q) = %q, want %q", tag, got, want)
    }
}
```

**Test Utilities Package:**

The `internal/testutil/` package provides helpers for tests:

```go
// Parse a game from PGN string for testing
game := testutil.ParseTestGame(`
[Event "Test"]
[White "A"]
[Black "B"]
[Result "*"]

1. e4 e5 *
`)

// Must-parse variant (fails test on error)
game := testutil.MustParseGame(t, pgnString)

// Parse multiple games
games := testutil.ParseTestGames(multiGamePGN)
```

### Golden File Testing

**Integration Tests with Golden Files:**

```go
// From cmd/pgn-extract/golden_test.go
func TestBasicParsing(t *testing.T) {
    stdout, _ := runPgnExtract(t, "-s", inputFile("fools-mate.pgn"))
    if stdout == "" {
        t.Error("Expected non-empty output")
    }
    if !strings.Contains(stdout, "[Event") {
        t.Error("Expected Event tag in output")
    }
    if !containsMove(stdout, "f3") || !containsMove(stdout, "Qh4") {
        t.Error("Expected fools mate moves in output")
    }
}
```

**Test Binary Management:**
- Build test binary once, reuse across tests
- Platform-specific binary names (`.exe` on Windows)
- Clean shutdown and error capture

```go
func buildTestBinary(t *testing.T) string {
    t.Helper()
    if testBinaryPath != "" {
        return testBinaryPath  // Reuse existing binary
    }

    binName := "pgn-extract-test"
    if runtime.GOOS == "windows" {
        binName += ".exe"
    }
    // Build and cache path
    cmd := exec.Command("go", "build", "-o", binPath, ".")
    // ...
    testBinaryPath = binPath
    return binPath
}
```

## Benchmark Tests

### Benchmark Organization

**Separate Files** - `benchmark_test.go` in each package:

```go
// From internal/engine/benchmark_test.go
var benchFENs = map[string]string{
    "Initial":   "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
    "Midgame":   "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
    "Endgame":   "8/5k2/8/8/8/8/5K2/4R3 w - - 0 1",
    "Complex":   "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
}

func BenchmarkNewBoardFromFEN(b *testing.B) {
    for name, fen := range benchFENs {
        b.Run(name, func(b *testing.B) {
            for i := 0; i < b.N; i++ {
                NewBoardFromFEN(fen)
            }
        })
    }
}
```

### Benchmark Patterns

**Table-Driven Benchmarks:**

```go
func BenchmarkApplyMove(b *testing.B) {
    cases := []struct {
        name string
        fen  string
        move *chess.Move
    }{
        {
            name: "PawnMove",
            fen:  benchFENs["Initial"],
            move: &chess.Move{Text: "e4", ...},
        },
        {
            name: "KingsideCastle",
            fen:  benchFENs["Castling"],
            move: &chess.Move{Text: "O-O", ...},
        },
    }

    for _, tc := range cases {
        b.Run(tc.name, func(b *testing.B) {
            board, _ := NewBoardFromFEN(tc.fen)
            b.ResetTimer()  // Don't count setup time
            for i := 0; i < b.N; i++ {
                boardCopy := board.Copy()
                ApplyMove(boardCopy, tc.move)
            }
        })
    }
}
```

**Key Practices:**
- Use `b.ResetTimer()` after setup to exclude initialization cost
- Create copies when benchmarking destructive operations
- Use `b.Run()` for subtests to get granular results
- Include memory allocation stats with `-benchmem`

### Benchmark Coverage

Benchmarks exist for performance-critical operations:

| Package | Benchmarks | Focus |
|---------|------------|-------|
| engine | FEN parsing, move application, check detection, board copy | Move validation performance |
| parser | PGN parsing, token scanning | Input processing speed |
| hashing | Zobrist hashing, duplicate detection | Hash performance |
| matching | Tag matching, position matching, variation matching | Filter performance |
| eco | ECO classification, opening lookup | Classification speed |

**Run Benchmarks:**
```bash
# All benchmarks
go test -bench=. -benchmem ./...

# Specific package
go test -bench=. -benchmem ./internal/engine/

# Specific benchmark
go test -bench=BenchmarkApplyMove -benchmem ./internal/engine/
```

## Test Naming Conventions

### Test Function Names

**Pattern:** `Test<FunctionName>` or `Test<Feature>`

```go
func TestNewBoard(t *testing.T)               // Tests NewBoard() function
func TestBoardGetSet(t *testing.T)            // Tests Get/Set methods
func TestParseElo(t *testing.T)               // Tests parseElo() function
func TestGameMatcherInterface(t *testing.T)   // Tests interface compliance
```

**Subtest Names:**
- Descriptive phrases in lowercase
- Use spaces for readability (not underscores)

```go
t.Run("initial state", func(t *testing.T) { ... })
t.Run("all squares empty", func(t *testing.T) { ... })
t.Run("modifications are independent", func(t *testing.T) { ... })
```

### Benchmark Names

**Pattern:** `Benchmark<Operation>` with variants in subtests

```go
func BenchmarkApplyMove(b *testing.B)
func BenchmarkNewBoardFromFEN(b *testing.B)
func BenchmarkPositionMatcher_MatchGame(b *testing.B)  // Method benchmarks
```

**Subtest Naming:**
```go
b.Run("NoPatterns", func(b *testing.B) { ... })
b.Run("SingleFEN", func(b *testing.B) { ... })
b.Run("WildcardPattern", func(b *testing.B) { ... })
```

## Assertion Patterns

**No External Assertion Libraries** - Use standard Go testing:

```go
// Simple equality check
if got != want {
    t.Errorf("function() = %v; want %v", got, want)
}

// Error checking
if err != nil {
    t.Fatalf("function() error = %v; want nil", err)
}

// Nil checking
if game == nil {
    t.Fatal("ParseGame() = nil, want game")
}

// Boolean conditions
if !matcher.Match(game) {
    t.Error("Expected match on player filter")
}

// String containment
if !strings.Contains(output, "expected text") {
    t.Errorf("Output does not contain expected text")
}
```

**Error Message Formatting:**
- Include function/method name in error
- Show both `got` and `want` values
- Use `%v` for values, `%q` for strings, `%d` for integers
- Include context when helpful

```go
t.Errorf("after Set(%c, %c, %v), Get() = %v; want %v",
    tt.col, tt.rank, tt.piece, got, tt.piece)

t.Errorf("ParseCriterion(%s) failed: %v", line, err)
```

## Mocking and Test Doubles

**Minimal Use of Mocks** - Real implementations preferred:

```go
// Use in-memory buffers instead of file mocks
cfg := config.NewConfig()
buf := &bytes.Buffer{}
cfg.SetOutput(buf)

// Use strings.NewReader for parser tests
p := parser.NewParser(strings.NewReader(benchPGN), cfg)

// Use temp files for file operations
tmpFile := filepath.Join(t.TempDir(), "output.pgn")
```

**Interface Compliance Tests:**

```go
// Verify all matchers implement the interface
var _ GameMatcher = NewGameFilter()
var _ GameMatcher = NewMaterialMatcher("Q:q", false)
var _ GameMatcher = NewVariationMatcher()
```

## Integration Testing

### Golden File Tests

**Full End-to-End Tests** in `cmd/pgn-extract/golden_test.go`:

```go
func TestOutputFormat(t *testing.T) {
    tests := []struct {
        name       string
        format     string
        checkMove  string
        shouldHave []string
    }{
        {"lalg", "lalg", "e2e4", []string{"e2e4", "e7e5"}},
        {"halg", "halg", "e2-e4", []string{"e2-e4", "e7-e5"}},
        {"uci", "uci", "e2e4", []string{"e2e4", "e7e5"}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            stdout, _ := runPgnExtract(t, "-W", tt.format, "-s", inputFile("test-ucW.pgn"))
            if stdout == "" {
                t.Error("Expected non-empty output")
                return
            }

            for _, expected := range tt.shouldHave {
                if !strings.Contains(stdout, expected) {
                    t.Errorf("Expected %s in %s format output", expected, tt.format)
                }
            }
        })
    }
}
```

### Parallel Testing

Tests for concurrent operations:

```go
// From cmd/pgn-extract/parallel_test.go
func TestParallelProcessing(t *testing.T) {
    stdout, _ := runPgnExtract(t, "--workers", "4", "-s", inputFile("fischer.pgn"))
    // Verify output correctness regardless of processing order
}
```

## Test Execution

### CI/CD Integration

**Pre-commit Hooks** run tests automatically:
```yaml
- id: go-test
  name: go test
  entry: bash -c 'GO111MODULE=on go test -short ./...'
  language: system
  types: [go]
  pass_filenames: false
```

**Short Mode** for quick feedback:
```bash
go test -short ./...  # Skips long-running tests
```

### Coverage Tracking

**Per-Package Coverage:**
```bash
# Individual package coverage
go test -coverprofile=coverage_engine.out ./internal/engine/
go test -coverprofile=coverage_matching.out ./internal/matching/
go tool cover -html=coverage_engine.out
```

**Full Project Coverage:**
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Test Data Management

### Testdata Organization

```
testdata/
├── infiles/              # Input PGN files
│   ├── fools-mate.pgn
│   ├── fischer.pgn
│   ├── petrosian.pgn
│   ├── test-7.pgn       # Seven tag roster test
│   ├── test-C.pgn       # Comment test
│   ├── test-N.pgn       # NAG test
│   └── test-V.pgn       # Variation test
└── eco.pgn              # ECO classification database
```

**Test File Naming:**
- Descriptive names: `fools-mate.pgn`, `nested-comment.pgn`
- Feature-specific: `test-C.pgn` (comments), `test-V.pgn` (variations)
- Real-world examples: `fischer.pgn`, `petrosian.pgn`

### Test Helpers

```go
// From cmd/pgn-extract/golden_test.go
func testdataDir() string {
    return filepath.Join("..", "..", "testdata")
}

func inputFile(name string) string {
    return filepath.Join(testdataDir(), "infiles", name)
}

func countGames(pgn string) int {
    return strings.Count(pgn, "[Event ")
}

func containsMove(output, move string) bool {
    return strings.Contains(output, move)
}
```

## Testing Best Practices

### Do's

✅ **Write table-driven tests** for multiple test cases
✅ **Use t.Helper()** in test utility functions
✅ **Test edge cases** and error conditions
✅ **Keep tests focused** - one concept per test
✅ **Use descriptive names** for tests and subtests
✅ **Reset state** between test iterations
✅ **Include benchmarks** for performance-critical code
✅ **Test interface compliance** explicitly
✅ **Use temp directories** for file operations

### Don'ts

❌ **Don't use global state** in tests
❌ **Don't ignore test failures** with `t.Skip()` without reason
❌ **Don't test implementation details** - test behavior
❌ **Don't copy-paste tests** - use table-driven approach
❌ **Don't forget to test error paths**
❌ **Don't use sleep for timing** - use proper synchronization

## Future Test Improvements

**Areas for Improvement:**
1. Increase coverage in `matching` package (currently 34.6%)
2. Increase coverage in `processing` package (currently 36.1%)
3. Add property-based testing for parser robustness
4. Add fuzzing tests for PGN parser edge cases
5. Add more integration tests for complex filter combinations
6. Add benchmarks for full game processing pipeline
7. Add stress tests for worker pool scalability
