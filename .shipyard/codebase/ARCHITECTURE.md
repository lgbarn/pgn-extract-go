# Architecture

## Overview

pgn-extract-go is a high-performance PGN (Portable Game Notation) processing tool implemented in Go. The architecture follows a **pipeline pattern** with clear separation between parsing, filtering, processing, and output stages. The system is designed for scalability, supporting parallel processing of chess games through a worker pool pattern.

## Architectural Patterns

### 1. Pipeline Architecture

The application follows a classic data processing pipeline:

```
Input (PGN Files/Stdin)
    ↓
Parser (Lexer → AST)
    ↓
Filter/Matcher (Tag, Position, CQL, Variation)
    ↓
Processor (Validation, ECO Classification, Duplicate Detection)
    ↓
Output (PGN, JSON, EPD, FEN)
```

Each stage is independent and can be configured or extended without affecting others.

### 2. Layered Architecture

The codebase is organized into distinct layers:

- **Presentation Layer** (`cmd/pgn-extract`): CLI interface, flag parsing, user interaction
- **Application Layer** (`cmd/pgn-extract`): Orchestration, workflow control, parallel processing
- **Domain Layer** (`internal/chess`, `internal/engine`): Core chess logic and rules
- **Service Layer** (`internal/matching`, `internal/processing`, `internal/cql`): Business logic
- **Infrastructure Layer** (`internal/parser`, `internal/output`, `internal/hashing`): I/O and utilities

### 3. Domain-Driven Design Elements

Core chess domain concepts are encapsulated in dedicated types:

- **Game**: Represents a complete chess game with tags, moves, and metadata
- **Board**: Represents a chess position with piece placement and state
- **Move**: Represents a single chess move with metadata
- **Position**: Immutable position state for hashing and comparison

File: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/chess/types.go`
```go
type Game struct {
    Tags                map[string]string
    Moves               *Move
    FinalHashValue      HashCode
    CumulativeHashValue HashCode
    MovesChecked        bool
    MovesOK             bool
}
```

### 4. Worker Pool Pattern (Producer-Consumer)

For performance, the application uses a configurable worker pool for parallel game processing:

**Architecture:**
```
Main Thread (Producer)
    ↓ [WorkItem Channel]
Worker Pool (N goroutines)
    ↓ [ProcessResult Channel]
Main Thread (Consumer/Output)
```

File: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/worker/pool.go`
```go
type Pool struct {
    numWorkers  int
    workChan    chan WorkItem
    resultChan  chan ProcessResult
    processFunc ProcessFunc
    wg          sync.WaitGroup
}
```

**Benefits:**
- CPU-bound work (move validation, CQL evaluation) parallelized
- I/O remains sequential to preserve game order
- Configurable worker count (auto-detects CPU cores)
- Early termination support via atomic stop flag

### 5. Strategy Pattern

Multiple strategies are used for different concerns:

#### Filtering Strategy
```
GameFilter (interface)
    ├── TagFilter (player, result, ECO)
    ├── PositionFilter (FEN matching)
    ├── MaterialMatcher (piece balance)
    ├── VariationMatcher (move sequences)
    └── CQLEvaluator (complex queries)
```

#### Output Format Strategy
File: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/config/config.go`
```go
type OutputFormat int
const (
    SAN    // Standard Algebraic Notation
    LALG   // Long algebraic (e2e4)
    HALG   // Hyphenated (e2-e4)
    UCI    // Universal Chess Interface
    EPD    // Extended Position Description
    FEN    // Forsyth-Edwards Notation
)
```

#### Duplicate Detection Strategy
```
DuplicateDetector
    ├── Zobrist Hashing (position-based)
    ├── Move Sequence Hashing (game-based)
    └── Fuzzy Depth Matching (partial game)
```

### 6. Builder Pattern

Configuration is constructed using a builder pattern:

File: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/config/builder.go`
```go
ConfigBuilder
    .SetOutputFormat(format)
    .SetFilter(filter)
    .EnableDuplicateDetection()
    .Build()
```

### 7. Visitor Pattern (Implicit)

The CQL evaluator visits AST nodes to evaluate queries:

File: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/cql/evaluator.go`
```go
func (e *Evaluator) Evaluate(node Node) bool {
    switch n := node.(type) {
    case *FilterNode:
        return e.evalFilter(n)
    case *LogicalNode:
        return e.evalLogical(n)
    case *ComparisonNode:
        return e.evalComparison(n)
    }
}
```

## Data Flow

### Primary Data Flow

```
1. Input Parsing (Sequential)
   ┌─────────────────────────────┐
   │ PGN File/Stdin              │
   └──────────┬──────────────────┘
              ↓
   ┌─────────────────────────────┐
   │ Lexer (Tokenization)        │
   │ - Tags → TagToken           │
   │ - Moves → MoveToken         │
   │ - Comments → CommentToken   │
   └──────────┬──────────────────┘
              ↓
   ┌─────────────────────────────┐
   │ Parser (AST Construction)   │
   │ - Game objects              │
   │ - Move linked list          │
   └──────────┬──────────────────┘
              ↓
   []*chess.Game (in-memory)

2. Filtering/Processing (Parallel or Sequential)
   ┌─────────────────────────────┐
   │ applyFilters()              │
   │ - Tag matching              │
   │ - Position matching         │
   │ - CQL evaluation            │
   │ - Material balance          │
   │ - Variation matching        │
   └──────────┬──────────────────┘
              ↓
   ┌─────────────────────────────┐
   │ replayGame()                │
   │ - Apply moves to board      │
   │ - Validate legality         │
   │ - Track repetitions         │
   │ - Detect game features      │
   └──────────┬──────────────────┘
              ↓
   ┌─────────────────────────────┐
   │ ECO Classification          │
   │ (if enabled)                │
   └──────────┬──────────────────┘
              ↓
   ┌─────────────────────────────┐
   │ Duplicate Detection         │
   │ - Zobrist hash              │
   │ - Cumulative hash           │
   └──────────┬──────────────────┘
              ↓
   FilterResult{Matched, Board}

3. Output (Sequential)
   ┌─────────────────────────────┐
   │ Output Selection            │
   │ - Main file                 │
   │ - Duplicate file            │
   │ - ECO split files           │
   │ - Non-matching file         │
   └──────────┬──────────────────┘
              ↓
   ┌─────────────────────────────┐
   │ Format Conversion           │
   │ - PGN (default)             │
   │ - JSON                      │
   │ - UCI/LALG/HALG             │
   │ - EPD/FEN                   │
   └──────────┬──────────────────┘
              ↓
   ┌─────────────────────────────┐
   │ Output Writer               │
   │ - Line wrapping             │
   │ - Tag formatting            │
   │ - Move number insertion     │
   └──────────┬──────────────────┘
              ↓
   File/Stdout
```

### Parallel Processing Flow

File: `/Users/lgbarn/Personal/Chess/pgn-extract-go/cmd/pgn-extract/processor.go`

```
Main Thread                  Worker Pool                    Output Thread
─────────────               ────────────                   ──────────────
parseInput()
    │
    ├─> Submit WorkItem ──> worker.processFunc() ───┐
    ├─> Submit WorkItem ──> worker.processFunc() ───┤
    ├─> Submit WorkItem ──> worker.processFunc() ───┼─> Results Channel
    ├─> Submit WorkItem ──> worker.processFunc() ───┤
    └─> pool.Close()                                │
                                                     ├─> handleGameOutput()
                                                     ├─> handleGameOutput()
                                                     └─> handleGameOutput()
```

**Synchronization Points:**
- Work submission via buffered channel (non-blocking up to buffer size)
- Result collection via buffered channel (maintains output order via index)
- WaitGroup ensures all workers complete before closing result channel

### State Management

**Immutable State:**
- Parsed Game objects (read-only after parsing)
- Board positions (created fresh for each evaluation)

**Mutable State (Thread-Safe):**
- Duplicate detector (uses sync.Mutex internally)
- Match counters (atomic.Int64)
- Worker pool stop flag (atomic.Int32)

**Mutable State (Main Thread Only):**
- Config object
- Output writers
- Game counters

## Module Boundaries and Dependencies

### Dependency Graph

```
cmd/pgn-extract
    ├── internal/config (configuration)
    ├── internal/parser (PGN → AST)
    ├── internal/chess (domain types)
    ├── internal/engine (move validation)
    ├── internal/matching (filters)
    ├── internal/processing (analysis)
    ├── internal/output (formatters)
    ├── internal/cql (query language)
    ├── internal/eco (opening classification)
    ├── internal/hashing (duplicate detection)
    └── internal/worker (parallelization)

internal/parser
    ├── internal/chess (Game, Move types)
    └── internal/config (parsing options)

internal/engine
    ├── internal/chess (Board, Move)
    └── internal/errors

internal/matching
    ├── internal/chess
    ├── internal/engine (move validation)
    └── internal/config

internal/cql
    ├── internal/chess
    └── internal/engine

internal/output
    ├── internal/chess
    ├── internal/engine
    └── internal/config

internal/hashing
    └── internal/chess

internal/worker
    └── internal/chess
```

**Dependency Rules:**
1. No circular dependencies between packages
2. `internal/chess` is the foundation (no internal dependencies)
3. `internal/engine` depends only on chess domain
4. Upper layers depend on lower layers
5. Packages in same layer don't depend on each other (except via interfaces)

### Package Cohesion

**High Cohesion Examples:**
- `internal/chess`: All core chess types and constants
- `internal/parser`: Lexer, Parser, Token - all parsing concerns
- `internal/engine`: FEN parsing, move application, validation - board operations
- `internal/cql`: Lexer, Parser, Evaluator - complete query subsystem

**Low Coupling Examples:**
- Parser produces `*chess.Game` without knowing how it will be filtered
- Filters accept `*chess.Game` without knowing parser implementation
- Output formatters work with any `*chess.Game` regardless of source

## Key Design Decisions

### 1. Linked List for Moves

**Decision:** Use linked list (`Move.Next`, `Move.Prev`) instead of slices

**Rationale:**
- PGN structure is inherently recursive (variations within variations)
- Linked list allows easy insertion of variations at any point
- Easier to traverse forward/backward during replay
- Natural representation of the game tree

File: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/chess/move.go`
```go
type Move struct {
    Next       *Move
    Prev       *Move
    Variations []*Variation
    // ... move data
}
```

### 2. Zobrist Hashing for Duplicates

**Decision:** Use Polyglot-compatible Zobrist hashing

**Rationale:**
- O(1) position comparison
- Incremental updates possible (not yet implemented)
- Standard in chess programming
- Compatible with external tools

### 3. Separation of Lexer and Parser

**Decision:** Two-stage parsing (Lexer → Tokens → Parser → AST)

**Rationale:**
- PGN has complex tokenization rules (comments, NAGs, variations)
- Separation of concerns (tokenization vs structure)
- Easier to debug and test
- Standard compiler design pattern

### 4. Worker Pool with Buffered Channels

**Decision:** Use buffered channels sized to number of games or 100, whichever is smaller

**Rationale:**
- Prevents producer blocking on fast parsing
- Limits memory usage for large databases
- Maintains ordering via index tracking
- Allows graceful shutdown

### 5. Config as Dependency Injection

**Decision:** Pass `*config.Config` through the call chain

**Rationale:**
- Avoids global state
- Makes testing easier (can create test configs)
- Explicit dependencies
- Thread-safe (read-only after initialization)

### 6. In-Memory Processing

**Decision:** Load all games from a file into memory before processing

**Rationale:**
- Simplifies parallel processing (all work items available upfront)
- Allows multi-pass algorithms (duplicate detection)
- Files are typically small enough for modern memory
- Trade-off: speed over memory efficiency

**Future Improvement:** Streaming parser for very large files

## Extension Points

### Adding New Filters

Implement the filter interface in `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/matching/filter.go`:

```go
type GameMatcher interface {
    Matches(game *chess.Game, board *chess.Board) bool
}
```

### Adding New Output Formats

1. Add format constant to `config.OutputFormat`
2. Implement case in `output.formatMove()`
3. Add CLI flag in `cmd/pgn-extract/flags.go`

### Adding New CQL Filters

1. Add filter name to `cql.FilterNode` switch in `evaluator.go`
2. Implement `evalXXX()` method
3. Update parser to recognize new keyword

### Adding New Analysis

Implement in `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/processing/analyzer.go` and add to `GameAnalysis` struct.

## Performance Characteristics

### Time Complexity

- **Parsing:** O(n × m) where n = games, m = average moves per game
- **Tag Filtering:** O(1) per game (hash map lookup)
- **Position Matching:** O(m) per game (replay all moves)
- **CQL Evaluation:** O(m × c) where c = complexity of query
- **Duplicate Detection:** O(1) hash lookup + O(k) collision resolution
- **Output:** O(m) per game

### Space Complexity

- **In-Memory Games:** O(n × m) - all games loaded
- **Duplicate Detector:** O(d) where d = unique positions seen
- **Worker Pool:** O(w + b) where w = workers, b = buffer size

### Bottlenecks

1. **Single-threaded parsing** - lexer/parser is sequential
2. **Output serialization** - must maintain game order
3. **Memory growth** - large databases fully loaded

### Optimizations

1. **Parallel processing** - worker pool for CPU-bound filtering
2. **Early exit** - `--stopafter` flag halts processing
3. **Weak hash filtering** - fast pre-check before full Zobrist hash
4. **Pre-allocated slices** - parser uses capacity hints
5. **String interning** - tag names reused across games

## Error Handling Strategy

### Error Types

File: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/errors/errors.go`

```go
var (
    ErrInvalidFEN       error
    ErrInvalidMove      error
    ErrIllegalPosition  error
    ErrParseError       error
)
```

### Error Propagation

- **Parsing Errors:** Logged to stderr, game skipped
- **Validation Errors:** Marked in `Game.ErrorPly`, optionally filtered
- **I/O Errors:** Fatal (os.Exit)
- **Configuration Errors:** Fatal (os.Exit)

### Fault Tolerance

- Parser continues on malformed games
- Invalid moves can be ignored (`--strict` vs permissive mode)
- Worker errors don't crash the pool

## Concurrency Model

### Thread Safety

**Thread-Safe Components:**
- `worker.Pool` (channels + WaitGroup + atomic flags)
- `hashing.DuplicateDetector` (mutex-protected map)
- Atomic counters (`matchedCount`, `gamePosition`)

**Thread-Unsafe Components (Main Thread Only):**
- `config.Config` (read-only after init)
- `output.Writer` (sequential writes)
- File handles

**Safe Usage Pattern:**
```
1. Parse (main thread) → []*Game
2. Submit to workers → parallel filtering
3. Collect results (main thread) → sequential output
```

### Race Condition Prevention

- No shared mutable state between workers
- Results collected via channel (FIFO, ordered)
- Atomic operations for counters
- Worker pool properly synchronized with WaitGroups

## Testing Strategy

### Unit Tests

- Domain logic: `internal/chess`, `internal/engine`
- Parsing: `internal/parser`
- CQL: `internal/cql`

### Integration Tests

- Golden tests: Parse → Filter → Output comparison
- Feature tests: End-to-end CLI scenarios

### Benchmark Tests

- Move application performance
- Parsing performance
- Hashing performance

File: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/engine/benchmark_test.go`

## Summary

pgn-extract-go demonstrates a **clean architecture** with:

- **Clear boundaries** between parsing, domain, filtering, and output
- **Pluggable components** (filters, output formats, matchers)
- **Scalable design** (parallel processing, worker pools)
- **Domain-driven** core (chess concepts as first-class types)
- **Performance-oriented** (Zobrist hashing, parallelization, memory efficiency)

The architecture balances **simplicity** (straightforward pipeline) with **power** (CQL, ECO classification, multiple output formats) while maintaining **performance** through parallelization and efficient algorithms.
