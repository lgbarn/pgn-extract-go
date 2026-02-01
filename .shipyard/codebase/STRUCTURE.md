# Project Structure

## Directory Layout

```
pgn-extract-go/
├── cmd/                        # Command-line applications
│   └── pgn-extract/           # Main CLI entry point
├── internal/                   # Private application packages
│   ├── chess/                 # Core chess domain types
│   ├── config/                # Configuration management
│   ├── cql/                   # Chess Query Language
│   ├── eco/                   # ECO opening classification
│   ├── engine/                # Move validation and board operations
│   ├── errors/                # Error definitions
│   ├── hashing/               # Position hashing and duplicate detection
│   ├── matching/              # Game filtering and matching
│   ├── output/                # Output formatting (PGN, JSON, etc.)
│   ├── parser/                # PGN lexer and parser
│   ├── processing/            # Game analysis and processing
│   ├── testutil/              # Testing utilities
│   └── worker/                # Worker pool for parallel processing
├── testdata/                   # Test fixtures and golden files
│   ├── golden/                # Expected output for regression tests
│   └── infiles/               # Input PGN files for testing
├── docs/                       # Documentation
├── scripts/                    # Build and maintenance scripts
├── .github/workflows/          # CI/CD configuration
├── go.mod                      # Go module definition
├── justfile                    # Build automation (just command runner)
└── README.md                   # Project documentation
```

## Package Details

### `cmd/pgn-extract/` - CLI Application Layer

**Purpose:** Entry point and command-line interface for the application.

**Key Files:**
- `main.go` - Application bootstrap, argument parsing, initialization
- `flags.go` - Command-line flag definitions and parsing
- `processor.go` - Game processing orchestration and worker pool management
- `filters.go` - Filter application and game matching logic
- `analysis.go` - Game analysis (checkmate, stalemate, features)

**Responsibilities:**
- Parse command-line arguments
- Initialize configuration
- Set up input/output streams
- Orchestrate the processing pipeline
- Coordinate parallel processing
- Handle user interaction and reporting

**Dependencies:** All internal packages

**Example:**
```go
// File: cmd/pgn-extract/main.go
func main() {
    flag.Parse()
    cfg := config.NewConfig()
    applyFlags(cfg)

    ctx := &ProcessingContext{
        cfg:           cfg,
        detector:      setupDuplicateDetector(cfg),
        ecoClassifier: loadECOClassifier(cfg),
        gameFilter:    setupGameFilter(),
    }

    totalGames, outputGames, duplicates := processAllInputs(ctx, splitWriter)
}
```

### `internal/chess/` - Core Domain Layer

**Purpose:** Foundation types for chess representation. No business logic, just pure domain concepts.

**Key Files:**
- `types.go` - Core types (Colour, Piece, MoveClass, Rank, Col, HashCode)
- `board.go` - Board representation with piece placement
- `game.go` - Game structure with tags, moves, metadata
- `move.go` - Move representation with variations and annotations
- `tags.go` - Tag constants and helpers

**Responsibilities:**
- Define chess domain primitives
- Provide type-safe wrappers for chess concepts
- No dependencies on other internal packages

**Example:**
```go
// File: internal/chess/types.go
type Piece int
const (
    Empty Piece = iota
    Pawn
    Knight
    Bishop
    Rook
    Queen
    King
)

type Game struct {
    Tags                map[string]string
    Moves               *Move
    FinalHashValue      HashCode
    CumulativeHashValue HashCode
}
```

### `internal/config/` - Configuration Layer

**Purpose:** Centralized configuration management for all processing options.

**Key Files:**
- `config.go` - Main Config struct with all settings
- `output.go` - Output-related configuration
- `filter.go` - Filtering configuration
- `duplicate.go` - Duplicate detection settings
- `annotation.go` - Annotation options
- `builder.go` - Config builder pattern

**Responsibilities:**
- Centralize all configuration state
- Provide type-safe option enums
- Manage I/O streams (files, stdout)
- No business logic, just state management

**Example:**
```go
// File: internal/config/config.go
type Config struct {
    Output      *OutputConfig
    Filter      *FilterConfig
    Duplicate   *DuplicateConfig
    Annotation  *AnnotationConfig

    OutputFile  io.Writer
    LogFile     io.Writer
    Verbosity   int
}
```

### `internal/parser/` - Input Processing Layer

**Purpose:** Convert PGN text into structured Game objects.

**Key Files:**
- `lexer.go` - Tokenization of PGN text
- `tokens.go` - Token type definitions
- `parser.go` - Parse tokens into Game AST
- `decode.go` - Move notation decoding (SAN → structured move)

**Responsibilities:**
- Tokenize PGN input (tags, moves, comments, variations)
- Build Game object tree
- Handle malformed input gracefully
- Support nested variations (RAV - Recursive Annotation Variation)

**Example:**
```go
// File: internal/parser/parser.go
type Parser struct {
    lexer        *Lexer
    currentToken *Token
    ravLevel     uint
    cfg          *config.Config
}

func (p *Parser) ParseGame() (*chess.Game, error)
func (p *Parser) ParseAllGames() ([]*chess.Game, error)
```

**Data Flow:**
```
PGN Text → Lexer → Tokens → Parser → *chess.Game
```

### `internal/engine/` - Chess Rules Layer

**Purpose:** Implement chess rules, move validation, and board manipulation.

**Key Files:**
- `fen.go` - FEN string parsing and generation
- `apply.go` - Apply moves to board state
- `legal_moves.go` - Generate legal moves for a position
- `check_detection.go` - Check and checkmate detection
- `castling.go` - Castling rights and validation
- `pawn.go` - Pawn-specific logic (promotion, en passant)
- `piece.go` - Piece movement patterns
- `rules.go` - Game ending conditions (checkmate, stalemate, draw)
- `chess960.go` - Fischer Random Chess support

**Responsibilities:**
- Parse and generate FEN strings
- Apply moves to board (update piece positions, castling rights, EP)
- Validate move legality
- Detect check, checkmate, stalemate
- Calculate legal moves

**Example:**
```go
// File: internal/engine/apply.go
func ApplyMove(board *chess.Board, move *chess.Move)

// File: internal/engine/fen.go
func NewBoardFromFEN(fen string) (*chess.Board, error)
func BoardToFEN(board *chess.Board) string
```

### `internal/matching/` - Filtering Layer

**Purpose:** Filter games based on various criteria (tags, positions, variations, material).

**Key Files:**
- `matcher.go` - Base matcher interface
- `filter.go` - Game filter combining multiple criteria
- `tags.go` - Tag-based filtering (player, result, ECO)
- `position.go` - FEN position matching
- `variation.go` - Move sequence matching
- `material.go` - Material balance matching
- `soundex.go` - Fuzzy name matching

**Responsibilities:**
- Tag filtering (player names, ECO codes, results)
- Position matching (FEN)
- Move sequence matching
- Material balance checking
- Soundex fuzzy matching for names

**Example:**
```go
// File: internal/matching/filter.go
type GameFilter struct {
    tagFilters     map[string]string
    playerFilter   string
    whiteFilter    string
    blackFilter    string
    ecoFilter      string
    resultFilter   string
    fenMatcher     *FENMatcher
    useSoundex     bool
}

func (f *GameFilter) Matches(game *chess.Game) bool
```

### `internal/cql/` - Query Language Layer

**Purpose:** Implement Chess Query Language for advanced position pattern matching.

**Key Files:**
- `lexer.go` - Tokenize CQL queries
- `parser.go` - Parse CQL into AST
- `ast.go` - AST node definitions (FilterNode, LogicalNode, etc.)
- `evaluator.go` - Evaluate queries against positions
- `piece_eval.go` - Piece placement evaluation
- `attack_eval.go` - Attack detection
- `transform_eval.go` - Position transformations (flip, shift)
- `game_eval.go` - Game-level filters (result, player)

**Responsibilities:**
- Parse CQL query syntax
- Build AST representation
- Evaluate queries against board positions
- Support logical operators (and, or, not)
- Support piece placement, attacks, transformations

**Example:**
```go
// File: internal/cql/evaluator.go
type Evaluator struct {
    board *chess.Board
    game  *chess.Game
}

func (e *Evaluator) Evaluate(node Node) bool {
    switch n := node.(type) {
    case *FilterNode:
        return e.evalFilter(n)
    case *LogicalNode:
        return e.evalLogical(n)
    }
}
```

**Query Examples:**
```
mate                           # Checkmate position
piece K g1                     # King on g1
attack R k                     # Rook attacking enemy king
(and mate (piece [RQ] [a-h]8)) # Back rank mate
```

### `internal/eco/` - ECO Classification Layer

**Purpose:** Classify games by ECO (Encyclopedia of Chess Openings) codes.

**Key Files:**
- `classifier.go` - ECO code matching
- `eco_test.go` - Tests

**Responsibilities:**
- Load ECO database from PGN file
- Match game positions against ECO positions
- Assign ECO codes to games

**Note:** Requires external ECO database file in PGN format.

### `internal/hashing/` - Duplicate Detection Layer

**Purpose:** Detect duplicate games using position hashing.

**Key Files:**
- `zobrist.go` - Zobrist position hashing (Polyglot-compatible)
- `thread_safe.go` - Thread-safe duplicate detector
- `hashing_test.go` - Tests

**Responsibilities:**
- Generate Zobrist hashes for positions
- Detect duplicate games by final position
- Cumulative hashing for better disambiguation
- Thread-safe duplicate tracking
- Setup-based duplicate detection

**Example:**
```go
// File: internal/hashing/zobrist.go
func GenerateZobristHash(board *chess.Board) uint64

// File: internal/hashing/thread_safe.go
type DuplicateDetector struct {
    seenGames map[uint64]*GameEntry
    mutex     sync.Mutex
}

func (d *DuplicateDetector) CheckAndAdd(game *chess.Game, board *chess.Board) bool
```

### `internal/output/` - Output Formatting Layer

**Purpose:** Format games for output in various notations.

**Key Files:**
- `output.go` - PGN output formatting
- `json.go` - JSON output formatting
- `writer.go` - Line-wrapping output writer

**Responsibilities:**
- Format PGN with proper line wrapping
- Generate JSON output
- Convert move notation (SAN, UCI, LALG, HALG, ELALG)
- Output EPD/FEN sequences
- Handle tag formatting and escaping

**Example:**
```go
// File: internal/output/output.go
func OutputGame(game *chess.Game, cfg *config.Config)
func OutputGameJSON(game *chess.Game, cfg *config.Config)

// File: internal/output/writer.go
type OutputWriter struct {
    w             io.Writer
    lineLength    int
    maxLineLength int
}
```

### `internal/processing/` - Analysis Layer

**Purpose:** Analyze games for special properties and features.

**Key Files:**
- `analyzer.go` - Game feature detection
- `processing_test.go` - Tests

**Responsibilities:**
- Detect game ending conditions (checkmate, stalemate)
- Track fifty-move rule triggers
- Detect threefold repetition
- Find underpromotions
- Validate move sequences

**Example:**
```go
// File: internal/processing/analyzer.go
type GameAnalysis struct {
    IsCheckmate     bool
    IsStalemate     bool
    FiftyMoveRule   bool
    Repetition      bool
    Underpromotion  bool
}

func AnalyzeGame(game *chess.Game) *GameAnalysis
```

### `internal/worker/` - Concurrency Layer

**Purpose:** Parallel game processing using worker pool pattern.

**Key Files:**
- `pool.go` - Worker pool implementation
- `pool_test.go` - Tests

**Responsibilities:**
- Manage worker goroutines
- Distribute work items across workers
- Collect results in order
- Support early termination
- Thread-safe submission and result collection

**Example:**
```go
// File: internal/worker/pool.go
type Pool struct {
    numWorkers  int
    workChan    chan WorkItem
    resultChan  chan ProcessResult
    processFunc ProcessFunc
    wg          sync.WaitGroup
}

func NewPool(numWorkers, bufferSize int, processFunc ProcessFunc) *Pool
func (p *Pool) Submit(item WorkItem)
func (p *Pool) Results() <-chan ProcessResult
```

**Usage Pattern:**
```go
pool := worker.NewPool(8, 100, processFunc)
pool.Start()

// Producer
go func() {
    for _, game := range games {
        pool.Submit(WorkItem{Game: game})
    }
    pool.Close()
}()

// Consumer
for result := range pool.Results() {
    handleResult(result)
}
```

### `internal/errors/` - Error Definitions

**Purpose:** Centralized error definitions for the application.

**Responsibilities:**
- Define domain-specific error types
- Sentinel errors for common cases

### `internal/testutil/` - Testing Utilities

**Purpose:** Shared testing helpers and utilities.

**Responsibilities:**
- Test fixture creation
- Helper functions for assertions
- Common test data

## Test Organization

### Unit Tests

Located alongside source files with `_test.go` suffix:

```
internal/chess/board_test.go      # Board type tests
internal/parser/parser_test.go    # Parser tests
internal/engine/apply_test.go     # Move application tests
internal/cql/evaluator_test.go    # CQL evaluation tests
```

### Integration Tests

Located in `cmd/pgn-extract/`:

```
cmd/pgn-extract/golden_test.go    # End-to-end golden tests
cmd/pgn-extract/features_test.go  # Feature detection tests
```

### Benchmark Tests

Performance-critical paths:

```
internal/engine/benchmark_test.go
internal/parser/benchmark_test.go
internal/hashing/benchmark_test.go
internal/matching/benchmark_test.go
```

### Test Data

```
testdata/
├── golden/              # Expected output files
│   ├── basic.pgn
│   ├── filters.pgn
│   └── cql.pgn
└── infiles/             # Input test files
    ├── sample.pgn
    ├── malformed.pgn
    └── variations.pgn
```

## Build and Tooling

### `justfile` - Build Automation

Project uses [just](https://github.com/casey/just) for task running:

```make
build:         # Build binary
test:          # Run all tests
test-verbose:  # Run tests with verbose output
bench:         # Run benchmarks
lint:          # Run linters
fmt:           # Format code
```

### `.github/workflows/` - CI/CD

GitHub Actions workflows for:
- Automated testing
- Linting
- Release builds

### `scripts/` - Maintenance Scripts

Utility scripts for development and maintenance.

## File Naming Conventions

- **Source files:** `lowercase_underscore.go` (e.g., `zobrist.go`, `check_detection.go`)
- **Test files:** `<source>_test.go` (e.g., `parser_test.go`)
- **Interface files:** Often named after primary type (e.g., `matcher.go` contains `Matcher` interface)
- **Implementation files:** Descriptive names (e.g., `tags.go`, `soundex.go`)

## Module Boundaries

### Public API (cmd/pgn-extract)

Entry point for the application. Accessible to users via CLI.

### Internal Packages (internal/*)

Not importable by external packages (enforced by Go).

**Layer Dependencies (Top to Bottom):**
```
cmd/pgn-extract
    ↓
internal/{matching,processing,cql,eco,output}  (Service Layer)
    ↓
internal/engine                                (Rules Layer)
    ↓
internal/chess                                 (Domain Layer)
```

**Cross-Layer Dependencies:**
```
internal/{parser,output,hashing} → internal/chess
internal/{matching,cql} → internal/engine
internal/worker → internal/chess
```

## Package Size and Complexity

| Package | Files | Lines | Purpose |
|---------|-------|-------|---------|
| `cmd/pgn-extract` | 7 | ~1500 | CLI orchestration |
| `internal/chess` | 6 | ~800 | Core domain |
| `internal/parser` | 6 | ~1500 | PGN parsing |
| `internal/engine` | 16 | ~2000 | Chess rules |
| `internal/matching` | 10 | ~1500 | Filtering logic |
| `internal/cql` | 12 | ~2500 | Query language |
| `internal/output` | 4 | ~1200 | Output formatting |
| `internal/config` | 6 | ~500 | Configuration |
| `internal/hashing` | 4 | ~600 | Duplicate detection |
| `internal/worker` | 2 | ~200 | Parallelization |

## Growth Areas

### Adding New Features

**New filter type:**
1. Add to `internal/matching/`
2. Implement `GameMatcher` interface
3. Wire into `cmd/pgn-extract/filters.go`

**New output format:**
1. Add enum to `internal/config/config.go`
2. Implement in `internal/output/output.go`
3. Add CLI flag in `cmd/pgn-extract/flags.go`

**New CQL filter:**
1. Add to `internal/cql/evaluator.go`
2. Implement evaluation logic
3. Update parser if new syntax needed

### Refactoring Opportunities

1. **Streaming parser** - Parse games on-demand instead of loading all into memory
2. **Plugin system** - External filters/analyzers
3. **Incremental hashing** - Update hash as moves are applied (performance)
4. **Database backend** - Store processed games in SQLite for querying

## Summary

The codebase structure follows **clean architecture principles**:

- **Domain-centric** (`internal/chess` at the core)
- **Layer separation** (domain → rules → services → application)
- **Package cohesion** (each package has single responsibility)
- **Minimal coupling** (dependencies flow downward)
- **Testability** (unit tests alongside code, integration tests at boundaries)

The structure supports both **horizontal scaling** (add workers) and **vertical scaling** (add features via new packages).
