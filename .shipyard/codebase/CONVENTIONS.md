# Code Conventions

This document describes the coding standards, naming patterns, and style conventions used throughout the pgn-extract-go project.

## Language & Version

- **Go Version**: 1.21
- **Module**: `github.com/lgbarn/pgn-extract-go`
- **No external dependencies**: Pure Go standard library implementation

## Code Organization

### Package Structure

The project follows standard Go package organization with clear separation of concerns:

```
cmd/pgn-extract/      # Main application entry point
internal/             # Internal packages (not importable externally)
  ├── chess/         # Core domain types
  ├── config/        # Configuration management
  ├── cql/           # Chess Query Language
  ├── eco/           # ECO classification
  ├── engine/        # Chess engine (move validation, FEN)
  ├── errors/        # Structured errors
  ├── hashing/       # Zobrist hashing, duplicate detection
  ├── matching/      # Game filtering and matching
  ├── output/        # Output formatting
  ├── parser/        # PGN parsing (lexer + parser)
  ├── processing/    # Game processing pipeline
  ├── testutil/      # Test utilities
  └── worker/        # Parallel processing worker pool
testdata/             # Test fixtures and golden files
```

**Package Naming Conventions:**
- All lowercase, single-word package names (`chess`, `parser`, `engine`)
- No underscores or camelCase in package names
- Package name should reflect primary purpose (not generic names like `util` or `common`)

### File Organization

**File Naming Patterns:**
- `<feature>.go` - Main implementation files
- `<feature>_test.go` - Unit tests
- `benchmark_test.go` - Benchmark tests (separate file)
- `tokens.go`, `types.go` - Type definitions
- `errors.go` - Package-specific errors

**Example from `internal/engine/`:**
```
apply.go             # Move application logic
apply_test.go        # Tests for move application
benchmark_test.go    # Benchmarks
castling.go          # Castling logic
check_detection.go   # Check/checkmate detection
fen.go              # FEN parsing/generation
fen_test.go         # FEN tests
game_state.go       # Game state functions
```

## Naming Conventions

### Variables

**Local Variables:**
- Short, descriptive names in camelCase
- Single-letter variables for loop indices: `i`, `j`
- Short names for common concepts: `err`, `cfg`, `game`, `board`

```go
// Good examples from codebase
var matchedCount int64
var game *chess.Game
var board *chess.Board
cfg := config.NewConfig()
```

**Package-Level Variables:**
- CamelCase for exported variables
- Use var blocks for related declarations
- Constants in CamelCase (not SCREAMING_CASE)

```go
// From internal/errors/errors.go
var (
    ErrInvalidFEN     = errors.New("invalid FEN string")
    ErrIllegalMove    = errors.New("illegal move")
    ErrParseFailure   = errors.New("parse failure")
    ErrCQLSyntax      = errors.New("CQL syntax error")
)
```

### Functions & Methods

**Function Names:**
- Exported functions use PascalCase: `NewBoard()`, `ParseGame()`, `ApplyMove()`
- Unexported functions use camelCase: `parseTag()`, `nextToken()`, `skipToNextGame()`
- Factory functions start with `New`: `NewParser()`, `NewConfig()`, `NewGameFilter()`
- Boolean predicates use `Is`, `Has`, or `Should` prefixes:
  - `IsCheckmate(board *chess.Board) bool`
  - `IsStalemate(board *chess.Board) bool`
  - `HasLegalMoves(board *chess.Board, colour Colour) bool`

**Method Receivers:**
- Short, consistent names (1-2 letters): `p` for Parser, `b` for Board, `g` for Game
- Use pointer receivers for methods that modify state
- Use value receivers for small, immutable types

```go
// From internal/parser/parser.go
func (p *Parser) nextToken()
func (p *Parser) ParseGame() (*chess.Game, error)

// From internal/chess/board.go
func (b *Board) Get(col Col, rank Rank) Piece
func (b *Board) Copy() *Board
```

### Types

**Type Names:**
- PascalCase for exported types
- Avoid stutter: `chess.Game` not `chess.ChessGame`
- Use descriptive names that reflect purpose

```go
// From internal/chess/
type Game struct { ... }
type Board struct { ... }
type Move struct { ... }
type Piece byte
type Colour int

// From internal/config/
type OutputFormat int
type FilterConfig struct { ... }
type DuplicateConfig struct { ... }
```

**Enum-Style Constants:**
- Type-safe enums using custom types and `const` blocks
- Use `iota` for sequential values
- Group related constants together

```go
// From internal/config/config.go
type OutputFormat int

const (
    SAN OutputFormat = iota  // Standard Algebraic Notation
    LALG                     // Long algebraic
    HALG                     // Hyphenated long algebraic
    ELALG                    // Enhanced long algebraic
    UCI                      // Universal Chess Interface
    EPD                      // Extended Position Description
    FEN                      // Forsyth-Edwards Notation
)

// From internal/chess/piece.go
const (
    Empty Piece = iota
    Pawn
    Knight
    Bishop
    Rook
    Queen
    King
    Off  // Represents off-board squares
)
```

### Interfaces

**Interface Naming:**
- Describe behavior, often ending in `-er`: `Reader`, `Writer`, `Matcher`
- Single-method interfaces preferred (Go idiom)

```go
// From internal/matching/matcher.go
type GameMatcher interface {
    Match(game *chess.Game) bool
    Name() string
}

// Standard io interfaces used throughout
io.Reader
io.Writer
io.Closer
```

## Code Style

### Formatting

**Enforced by `gofmt` and `goimports`:**
- Tabs for indentation (not spaces)
- No trailing whitespace
- One statement per line
- Import grouping: standard library, blank line, third-party, blank line, local packages

```go
// From cmd/pgn-extract/main.go
import (
    "bufio"
    "flag"
    "fmt"
    "os"
    "path/filepath"

    "github.com/lgbarn/pgn-extract-go/internal/config"
    "github.com/lgbarn/pgn-extract-go/internal/cql"
    "github.com/lgbarn/pgn-extract-go/internal/eco"
)
```

### Comments & Documentation

**Package Comments:**
- Every package has a package-level comment
- Describes the package's purpose and scope

```go
// Package matching provides game filtering and matching capabilities.
package matching

// Package errors provides sentinel errors and error types for the pgn-extract tool.
// It defines common error conditions and structured error types that preserve
// context while allowing error inspection with errors.Is() and errors.As().
package errors
```

**Function Comments:**
- Exported functions must have doc comments
- Start with function name
- Describe purpose, parameters, and return values when non-obvious

```go
// ParseGame parses a single game from the input.
// Returns nil if no more games are available.
func (p *Parser) ParseGame() (*chess.Game, error)

// NewBoardFromFEN creates a board from a FEN string.
// Returns an error if the FEN is invalid.
func NewBoardFromFEN(fen string) (*Board, error)

// Wrap adds context to an error while preserving the underlying error
// for inspection with errors.Is() and errors.As().
func Wrap(err error, context string) error
```

**Inline Comments:**
- Explain "why" not "what"
- Used sparingly for complex logic
- British English spelling in chess terminology comments ("colour" not "color" per chess tradition)

```go
// Skip to next game
p.skipToNextGame()

// Set matchAnywhere option if specified
if *varAnywhere {
    matcher.SetMatchAnywhere(true)
}

// Empty composite in AND mode should match all (vacuously true)
composite := NewCompositeMatcher(MatchAll)
```

### Error Handling

**Patterns:**

1. **Sentinel Errors** - For well-known error conditions
```go
var (
    ErrInvalidFEN    = errors.New("invalid FEN string")
    ErrIllegalMove   = errors.New("illegal move")
)
```

2. **Wrapped Errors** - Add context while preserving original error
```go
func Wrap(err error, context string) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("%s: %w", context, err)
}
```

3. **Structured Errors** - Custom error types with context
```go
type GameError struct {
    Err      error
    GameNum  int
    PlyNum   int
    MoveText string
    File     string
    Line     int
}

func (e *GameError) Error() string { ... }
func (e *GameError) Unwrap() error { return e.Err }
```

4. **Error Checking** - Always check errors, but permit ignoring specific cases
```go
// Explicit ignoring with nolint comment
file.Close() //nolint:errcheck,gosec // G104: cleanup on exit

// Function returns are checked
if err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    os.Exit(1)
}
```

### Concurrency Patterns

**Atomic Operations:**
- Use `sync/atomic` for counters
```go
var matchedCount int64

func IncrementMatchedCount() {
    atomic.AddInt64(&matchedCount, 1)
}

func GetMatchedCount() int64 {
    return atomic.LoadInt64(&matchedCount)
}
```

**Worker Pools:**
- Channel-based work distribution
- Graceful shutdown with WaitGroup
- Configurable worker count
```go
// From internal/worker/pool.go
type Pool struct {
    workers   int
    jobs      chan Job
    wg        sync.WaitGroup
}
```

## Linting & Code Quality

### Enabled Linters

Configuration in `.golangci.yml`:

**Bug Detection:**
- `errcheck` - Unchecked errors
- `govet` - Suspicious constructs
- `staticcheck` - Static analysis
- `bodyclose` - Unclosed HTTP bodies
- `durationcheck` - Duration arithmetic issues
- `nilerr` - Nil error returns

**Code Style:**
- `misspell` - Spelling errors (US English, with chess-specific exceptions for British spellings like "colour")
- `gofmt` - Standard formatting
- `goimports` - Import organization

**Complexity:**
- `cyclop` - Cyclomatic complexity (max 35 per function, 15 avg per package)
- `gocognit` - Cognitive complexity (min 50)
- `nakedret` - Naked returns (max 30 lines)

**Error Handling:**
- `errorlint` - Error wrapping issues

**Performance:**
- `prealloc` - Slice preallocation opportunities

**Security:**
- `gosec` - Security issues (comprehensive rule set enabled)

### Exclusions & Exceptions

```yaml
exclusions:
  rules:
    # Test files exempt from error checking (intentional error ignoring)
    - path: ".*_test\\.go$"
      linters:
        - errcheck
        - gosec

    # Lexer is inherently complex, exempt from complexity checks
    - path: internal/parser/lexer.go
      linters:
        - cyclop
```

### Security Annotations

Security linter warnings are disabled in specific cases with inline comments:

```go
// G302: File permissions 0644 are appropriate for user-created files
file, err := os.OpenFile(*outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //nolint:gosec

// G304: CLI tool needs to open user-specified files
file, err := os.Open(filename) //nolint:gosec

// G204: Test code builds and runs its own binary
cmd := exec.Command("go", "build", "-o", binPath, ".") //nolint:gosec,noctx
```

## Pre-Commit Hooks

Automated quality checks via `.pre-commit-config.yaml`:

1. **File Checks:** Large files, merge conflicts, trailing whitespace
2. **Go Formatting:** `go fmt ./...`
3. **Go Vet:** `go vet ./...`
4. **Module Tidiness:** `go mod tidy` with verification
5. **Static Analysis:** `staticcheck ./...` (if installed)
6. **Comprehensive Linting:** `golangci-lint run` (if installed)
7. **Build Verification:** `go build ./cmd/pgn-extract/`
8. **Quick Tests:** `go test -short ./...`
9. **Branch Protection:** Prevent direct commits to main/master

## Builder Pattern

Configuration uses the builder pattern for fluent API:

```go
cfg := NewConfigBuilder().
    WithOutputFormat(LALG).
    WithMaxLineLength(120).
    WithDuplicateSuppression(true).
    WithFuzzyMatch(true, 10).
    Build()
```

## Testing Conventions

See [TESTING.md](TESTING.md) for detailed testing patterns and conventions.

## Build & Development Tools

**Just Command Runner** (`justfile`):
- `just build` - Build binary
- `just test` - Run tests
- `just lint` - Run linters
- `just check` - Format, lint, test
- `just bench` - Run benchmarks

**Go Module:**
- No external dependencies (pure standard library)
- Module path: `github.com/lgbarn/pgn-extract-go`
- Go 1.21 required

## Performance Considerations

**Benchmark Naming:**
- `Benchmark<Operation>` for simple cases
- `Benchmark<Operation>_<Variant>` for subtests
- Use table-driven benchmarks with `b.Run()`

**Memory Efficiency:**
- Preallocation encouraged (`prealloc` linter)
- Pointer receivers for large structs
- Copy-on-write for immutable operations

```go
// From internal/engine/benchmark_test.go
func BenchmarkApplyMove(b *testing.B) {
    for _, tc := range cases {
        b.Run(tc.name, func(b *testing.B) {
            board, _ := NewBoardFromFEN(tc.fen)
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                boardCopy := board.Copy()
                ApplyMove(boardCopy, tc.move)
            }
        })
    }
}
```

## Compatibility & Portability

**Cross-Platform:**
- Pure Go implementation (no C dependencies)
- Path handling uses `filepath` package
- Platform-specific code uses build tags when needed

**Standard Library Only:**
- No external dependencies
- Maximum portability and minimal maintenance burden
