# pgn-extract-go

A Go implementation of the classic [pgn-extract](https://www.cs.kent.ac.uk/people/staff/djb/pgn-extract/) tool for searching, manipulating, and formatting chess games in PGN format.

## Overview

pgn-extract-go is a complete rewrite of David J. Barnes' pgn-extract tool from C to Go. It provides comprehensive functionality for working with chess games stored in Portable Game Notation (PGN) format, including:

- **Filtering** games by player, opening, position, result, or custom criteria
- **Searching** with Chess Query Language (CQL) for complex positional patterns
- **Converting** move notation between formats (SAN, UCI, Long Algebraic, etc.)
- **Cleaning** databases by removing duplicates, stripping content, or fixing tags
- **Classifying** openings using ECO codes
- **Validating** game integrity and move legality

## Features

### Filtering & Matching
- Filter by player name (with optional Soundex fuzzy matching)
- Filter by ECO code prefix
- Filter by game result
- Filter by FEN position
- Material balance matching (exact or minimum)
- Move sequence/variation matching
- Game feature detection (checkmate, stalemate, fifty-move rule, repetition, underpromotion)
- Rating-based filters (higher/lower rated winner)
- Ply and move count bounds

### Chess Query Language (CQL)
Full CQL implementation for advanced position pattern matching:
- Piece placement queries (`piece K g1`)
- Attack detection (`attack R k`)
- Position transformations (flip, mirror, shift)
- Logical operators (and, or, not)
- Material counting and comparisons
- Game metadata filters (result, player, year, rating)

See [docs/CQL.md](docs/CQL.md) for complete CQL documentation.

### Output Formats
- **PGN** - Standard Portable Game Notation
- **JSON** - Structured JSON format
- **EPD** - Extended Position Description
- **FEN** - Forsyth-Edwards Notation sequence

### Move Notation Formats
- SAN - Standard Algebraic Notation (default)
- LALG - Long Algebraic (e2e4)
- HALG - Hyphenated Long Algebraic (e2-e4)
- ELALG - Enhanced Long Algebraic (Ng1f3)
- UCI - Universal Chess Interface format

### Duplicate Detection
- Move sequence hashing
- Zobrist position hashing
- Configurable output for duplicates vs. unique games
- Check file support for cross-database deduplication

### Game Validation & Fixing
- Strict mode for PGN compliance
- Move legality validation
- Automatic fixing of common issues (missing tags, invalid results, date formats)

## Installation

```bash
go install github.com/lgbarn/pgn-extract-go/cmd/pgn-extract@latest
```

Or build from source:

```bash
git clone git@github.com:lgbarn/pgn-extract-go.git
cd pgn-extract-go
go build -o pgn-extract ./cmd/pgn-extract
```

## Quick Start

### Basic Usage

```bash
# Process games and output to stdout
pgn-extract games.pgn

# Write output to a file
pgn-extract -o output.pgn games.pgn

# Filter by player name
pgn-extract -p Fischer games.pgn

# Filter by ECO code
pgn-extract -Te B90 games.pgn

# Find checkmate games
pgn-extract --checkmate games.pgn

# Use CQL to find specific positions
pgn-extract --cql "mate" games.pgn
```

### Common Operations

```bash
# Remove comments and variations
pgn-extract -C -V games.pgn

# Convert to JSON format
pgn-extract -J games.pgn

# Convert to UCI notation
pgn-extract -W uci games.pgn

# Remove duplicate games
pgn-extract -D games.pgn

# Add ECO classification
pgn-extract -e eco.pgn games.pgn

# Validate all moves
pgn-extract --validate games.pgn
```

## Command-Line Reference

### Output Options

| Flag | Description |
|------|-------------|
| `-o file` | Output file (default: stdout) |
| `-a` | Append to output file instead of overwrite |
| `-7` | Output only the Seven Tag Roster |
| `--notags` | Don't output any tags |
| `-w N` | Maximum line length (default: 80) |
| `-W format` | Output format: san, lalg, halg, elalg, uci, epd, fen |
| `-J` | Output in JSON format |
| `-# N` | Split output into files of N games each |
| `-E level` | Split output by ECO level (1-3) |

### Content Options

| Flag | Description |
|------|-------------|
| `-C` | Don't output comments |
| `-N` | Don't output NAGs (Numeric Annotation Glyphs) |
| `-V` | Don't output variations |
| `--noresults` | Don't output results |

### Filtering Options

| Flag | Description |
|------|-------------|
| `-t file` | Tag criteria file for filtering |
| `-p name` | Filter by player name (either color) |
| `-Tw name` | Filter by White player |
| `-Tb name` | Filter by Black player |
| `-Te code` | Filter by ECO code prefix |
| `-Tr result` | Filter by result (1-0, 0-1, 1/2-1/2) |
| `-Tf fen` | Filter by FEN position |
| `-n` | Negate match (output games that DON'T match) |
| `-S` | Use Soundex for player name matching |
| `--tagsubstr` | Match tag values as substring |
| `--stopafter N` | Stop after matching N games |

### Game Feature Filters

| Flag | Description |
|------|-------------|
| `--checkmate` | Only output games ending in checkmate |
| `--stalemate` | Only output games ending in stalemate |
| `--fifty` | Games with fifty-move rule |
| `--repetition` | Games with threefold repetition |
| `--underpromotion` | Games with underpromotion |
| `--commented` | Only games with comments |
| `--higherratedwinner` | Higher-rated player won |
| `--lowerratedwinner` | Lower-rated player won |

### Ply/Move Bounds

| Flag | Description |
|------|-------------|
| `--minply N` | Minimum ply count |
| `--maxply N` | Maximum ply count |
| `--minmoves N` | Minimum number of moves |
| `--maxmoves N` | Maximum number of moves |

### CQL (Chess Query Language)

| Flag | Description |
|------|-------------|
| `--cql query` | CQL query to filter games by position patterns |
| `--cql-file file` | File containing CQL query |

### Material & Variation Matching

| Flag | Description |
|------|-------------|
| `-z pattern` | Material balance to match (e.g., 'QR:qrr') |
| `-y pattern` | Exact material balance to match |
| `-v file` | File with move sequences to match |
| `-x file` | File with positional variations to match |

### Duplicate Detection

| Flag | Description |
|------|-------------|
| `-D` | Suppress duplicate games |
| `-d file` | Output duplicates to this file |
| `-U` | Output only duplicates (suppress unique games) |
| `-c file` | Check file for duplicate detection |
| `-H hashcode` | Match positions by Polyglot hashcode |

### ECO Classification

| Flag | Description |
|------|-------------|
| `-e file` | ECO classification file (PGN format) |

### Annotations

| Flag | Description |
|------|-------------|
| `--plycount` | Add PlyCount tag |
| `--fencomments` | Add FEN comment after each move |
| `--hashcomments` | Add position hash after each move |
| `--addhashcode` | Add HashCode tag |

### Tag Management

| Flag | Description |
|------|-------------|
| `--fixresulttags` | Fix inconsistent result tags |
| `--fixtagstrings` | Fix malformed tag strings |

### Validation

| Flag | Description |
|------|-------------|
| `--strict` | Only output games that parse without errors |
| `--validate` | Verify all moves are legal |
| `--fixable` | Attempt to fix common issues |

### Logging & Other

| Flag | Description |
|------|-------------|
| `-l file` | Write diagnostics to log file |
| `-L file` | Append diagnostics to log file |
| `-r` | Report errors without extracting games |
| `-s` | Silent mode (no game count) |
| `-h` | Show help |
| `--version` | Show version |

## Usage Examples

### Filtering Games

```bash
# Find all Fischer games
pgn-extract -p "Fischer" games.pgn

# Find Fischer's wins as White
pgn-extract -Tw "Fischer" -Tr "1-0" games.pgn

# Find Sicilian Najdorf games
pgn-extract -Te "B90" games.pgn

# Find short games (under 20 moves)
pgn-extract --maxmoves 20 games.pgn

# Find games with underpromotion
pgn-extract --underpromotion games.pgn
```

### Using CQL

```bash
# Find checkmate positions
pgn-extract --cql "mate" games.pgn

# Find games where White castled kingside
pgn-extract --cql "piece K g1" games.pgn

# Find back rank mates
pgn-extract --cql "(and mate (piece [RQ] [a-h]8) (piece k [a-h]8))" games.pgn

# Find positions with more than 2 queens
pgn-extract --cql "(> (count [Qq]) 2)" games.pgn
```

### Database Cleaning

```bash
# Remove duplicate games
pgn-extract -D -o unique.pgn games.pgn

# Separate duplicates
pgn-extract -D -d duplicates.pgn -o unique.pgn games.pgn

# Fix common issues and validate
pgn-extract --fixable --validate games.pgn

# Add ECO codes to games
pgn-extract -e eco.pgn -o classified.pgn games.pgn
```

### Format Conversion

```bash
# Convert to JSON
pgn-extract -J -o games.json games.pgn

# Convert to UCI notation
pgn-extract -W uci games.pgn

# Output only positions (EPD)
pgn-extract -W epd games.pgn

# Strip to seven tag roster only
pgn-extract -7 games.pgn
```

### Material Matching

```bash
# Find queen vs rook+bishop endgames
pgn-extract -z "Q:rb" games.pgn

# Find exact material balance
pgn-extract -y "QRRBBNN:qrrbbnn" games.pgn
```

## Output Formats

### PGN (Default)

Standard PGN format with configurable tag and move output:

```
[Event "World Championship"]
[Site "Reykjavik"]
[Date "1972.07.11"]
[Round "1"]
[White "Spassky, Boris"]
[Black "Fischer, Robert"]
[Result "1-0"]

1. d4 Nf6 2. c4 e6 3. Nf3 d5 4. Nc3 Bb4 1-0
```

### JSON

Structured JSON with full game data:

```json
{
  "tags": {
    "Event": "World Championship",
    "White": "Spassky, Boris",
    "Black": "Fischer, Robert"
  },
  "moves": [
    {"san": "d4", "uci": "d2d4", "piece": "P"},
    {"san": "Nf6", "uci": "g8f6", "piece": "N"}
  ]
}
```

## Project Structure

```
pgn-extract-go/
├── cmd/pgn-extract/     # Command-line application
│   └── main.go          # Entry point and CLI handling
├── internal/
│   ├── chess/           # Core chess types (Board, Game, Move)
│   ├── config/          # Configuration management
│   ├── cql/             # Chess Query Language implementation
│   ├── eco/             # ECO classification system
│   ├── engine/          # Move validation and board operations
│   ├── hashing/         # Zobrist hashing and duplicate detection
│   ├── matching/        # Game filtering and matching
│   ├── output/          # Output formatting (PGN, JSON)
│   └── parser/          # PGN lexer and parser
├── docs/
│   └── CQL.md           # CQL documentation
├── testdata/            # Test files and golden outputs
└── go.mod
```

## Testing

Run the test suite:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test -v ./...
```

Run specific package tests:

```bash
go test ./internal/cql/...
go test ./internal/parser/...
```

## Credits

This is a Go port of [pgn-extract](https://www.cs.kent.ac.uk/people/staff/djb/pgn-extract/) by David J. Barnes.

The original pgn-extract has been an invaluable tool for the chess community since its creation. This port aims to provide the same functionality with the benefits of Go's cross-platform compilation and modern tooling.

CQL (Chess Query Language) was created by Gady Costeff and Lewis Stiller.

## License

MIT License

Copyright (c) 2024

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
