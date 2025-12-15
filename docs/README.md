# pgn-extract-go

A command-line tool for searching, manipulating, and formatting chess games in PGN (Portable Game Notation) format.

## Table of Contents

- [Introduction](#introduction)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Basic Usage](#basic-usage)
- [Filtering Games](#filtering-games)
- [Output Options](#output-options)
- [Duplicate Detection](#duplicate-detection)
- [ECO Classification](#eco-classification)
- [CQL Queries](#cql-queries)
- [Material Matching](#material-matching)
- [Variation Matching](#variation-matching)
- [Game Feature Filters](#game-feature-filters)
- [Output Splitting](#output-splitting)
- [Validation and Fixing](#validation-and-fixing)
- [Command Reference](#command-reference)
- [Examples](#examples)

---

## Introduction

### What is PGN?

PGN (Portable Game Notation) is a standard format for recording chess games. A PGN file contains one or more games, each with:

- **Tags**: Metadata like player names, date, event, and result
- **Moves**: The game's moves in algebraic notation
- **Comments**: Optional annotations explaining the moves
- **Variations**: Alternative move sequences

Here's a simple example:

```
[Event "Example Game"]
[Site "Internet"]
[Date "2024.01.15"]
[White "Player One"]
[Black "Player Two"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O 1-0
```

### What Does pgn-extract-go Do?

pgn-extract-go helps you work with PGN files by:

- **Filtering**: Find games matching specific criteria (players, openings, positions)
- **Formatting**: Convert move notation between different formats
- **Cleaning**: Remove duplicates, strip comments, standardize tags
- **Searching**: Find games with specific positions using CQL queries
- **Classifying**: Add ECO codes to identify openings

### Relationship to pgn-extract

This tool is a Go implementation inspired by David J. Barnes' [pgn-extract](https://www.cs.kent.ac.uk/people/staff/djb/pgn-extract/), a widely-used C program for PGN manipulation. While not all features are identical, pgn-extract-go provides similar functionality with some additions like CQL support.

---

## Installation

### From Source

Requires Go 1.21 or later.

```bash
git clone https://github.com/lgbarn/pgn-extract-go.git
cd pgn-extract-go
go build -o pgn-extract-go ./cmd/pgn-extract
```

### Verify Installation

```bash
./pgn-extract-go --version
```

---

## Quick Start

### View Games

Read a PGN file and output all games:

```bash
pgn-extract-go games.pgn
```

### Filter by Player

Find all games where Magnus Carlsen played:

```bash
pgn-extract-go -p "Carlsen" games.pgn
```

### Find Checkmates

Output only games that end in checkmate:

```bash
pgn-extract-go --checkmate games.pgn
```

### Remove Duplicates

Output unique games only:

```bash
pgn-extract-go -D games.pgn
```

### Save to File

Write output to a file instead of the screen:

```bash
pgn-extract-go -o output.pgn games.pgn
```

---

## Basic Usage

### Command Structure

```
pgn-extract-go [options] [input-files...]
```

- **options**: Flags that control behavior (start with `-` or `--`)
- **input-files**: One or more PGN files to process

If no input files are given, the program reads from standard input.

### Reading from Standard Input

You can pipe PGN data into the program:

```bash
cat games.pgn | pgn-extract-go -p "Fischer"
```

Or use input redirection:

```bash
pgn-extract-go -p "Fischer" < games.pgn
```

### Processing Multiple Files

Process several files at once:

```bash
pgn-extract-go file1.pgn file2.pgn file3.pgn
```

All games from all files are processed together, which is useful for duplicate detection across files.

### Silent Mode

By default, the program reports how many games were processed:

```
34 game(s) matched out of 100.
```

Use `-s` to suppress this message:

```bash
pgn-extract-go -s games.pgn
```

---

## Filtering Games

Filters let you select games based on various criteria. Multiple filters can be combined—a game must match ALL filters to be included.

### By Player Name

Find games where a player appears as either White or Black:

```bash
pgn-extract-go -p "Kasparov" games.pgn
```

The search matches partial names and is case-sensitive.

#### Filter by Color

Find games where a specific player had the white pieces:

```bash
pgn-extract-go -Tw "Kasparov" games.pgn
```

Find games where a specific player had the black pieces:

```bash
pgn-extract-go -Tb "Kasparov" games.pgn
```

### By Result

Find games with a specific result:

```bash
# White wins
pgn-extract-go -Tr "1-0" games.pgn

# Black wins
pgn-extract-go -Tr "0-1" games.pgn

# Draws
pgn-extract-go -Tr "1/2-1/2" games.pgn
```

### By ECO Code

Filter by opening classification:

```bash
# Sicilian Defense (B20-B99)
pgn-extract-go -Te "B" games.pgn

# Sicilian Najdorf specifically
pgn-extract-go -Te "B90" games.pgn
```

The filter matches ECO codes that start with the given prefix.

### By Position (FEN)

Find games that pass through a specific position:

```bash
pgn-extract-go -Tf "rnbqkb1r/pppppppp/5n2/8/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 1 2" games.pgn
```

This finds games where this exact position occurred at any point.

### By Game Ending

Find games ending in checkmate:

```bash
pgn-extract-go --checkmate games.pgn
```

Find games ending in stalemate:

```bash
pgn-extract-go --stalemate games.pgn
```

### Using a Tag File

For complex filtering, create a file with tag criteria:

```
# tags.txt
White "Carlsen"
Date >= "2020"
Result "1-0"
```

Then use it:

```bash
pgn-extract-go -t tags.txt games.pgn
```

### Combining Filters

Filters are combined with AND logic. This finds games where Kasparov played White and won:

```bash
pgn-extract-go -Tw "Kasparov" -Tr "1-0" games.pgn
```

---

## Output Options

### Output to File

Write results to a file:

```bash
pgn-extract-go -o output.pgn games.pgn
```

Without `-o`, output goes to standard output (the terminal).

### Move Notation Formats

Use `-W` to change how moves are written:

| Format | Flag | Example | Description |
|--------|------|---------|-------------|
| SAN | `-W san` | Nf3 | Standard Algebraic Notation (default) |
| Long Algebraic | `-W lalg` | g1f3 | Full from-to squares |
| Hyphenated | `-W halg` | g1-f3 | Long algebraic with hyphen |
| Enhanced | `-W elalg` | Ng1f3 | Piece letter + from-to |
| UCI | `-W uci` | g1f3 | Universal Chess Interface format |

Examples:

```bash
# Output in UCI format (useful for chess engines)
pgn-extract-go -W uci games.pgn

# Output in long algebraic notation
pgn-extract-go -W lalg games.pgn
```

### JSON Output

Output games in JSON format instead of PGN:

```bash
pgn-extract-go -J games.pgn
```

This produces structured data that's easy to process with other programs.

### Tag Options

Output only the Seven Tag Roster (Event, Site, Date, Round, White, Black, Result):

```bash
pgn-extract-go -7 games.pgn
```

Output no tags at all (moves only):

```bash
pgn-extract-go --notags games.pgn
```

### Content Options

Remove comments from output:

```bash
pgn-extract-go -C games.pgn
```

Remove NAGs (Numeric Annotation Glyphs like $1, $2):

```bash
pgn-extract-go -N games.pgn
```

Remove variations (alternative move sequences):

```bash
pgn-extract-go -V games.pgn
```

Remove game results from the move text:

```bash
pgn-extract-go --noresults games.pgn
```

### Line Length

Control the maximum line length in output (default is 80):

```bash
pgn-extract-go -w 120 games.pgn
```

---

## Duplicate Detection

When processing large collections, you often encounter the same game recorded multiple times. pgn-extract-go can identify and handle duplicates.

### How It Works

The program computes a hash of each game's moves. Games with identical move sequences are considered duplicates, even if their tags differ.

### Suppress Duplicates

Output only unique games (first occurrence of each):

```bash
pgn-extract-go -D games.pgn
```

### Save Duplicates Separately

Write duplicate games to a separate file:

```bash
pgn-extract-go -D -d duplicates.pgn games.pgn
```

This outputs unique games to stdout (or `-o` file) and duplicates to the specified file.

### Example Workflow

To deduplicate a large collection:

```bash
# Combine all files and remove duplicates
pgn-extract-go -D -o unique.pgn -d dups.pgn *.pgn

# Check results
echo "Unique games saved to unique.pgn"
echo "Duplicates saved to dups.pgn"
```

---

## ECO Classification

ECO (Encyclopaedia of Chess Openings) codes classify chess openings into categories A00-E99. pgn-extract-go can add ECO codes to games based on their opening moves.

### Using an ECO File

You need an ECO classification file in PGN format. Each game in this file represents an opening line:

```
[ECO "B20"]
[Opening "Sicilian Defense"]

1. e4 c5 *
```

Apply ECO classification:

```bash
pgn-extract-go -e eco.pgn games.pgn
```

This adds ECO and Opening tags to games that match known openings.

### How Matching Works

The program replays each game and finds the longest matching opening line from the ECO file. For example, if a game begins 1.e4 c5 2.Nf3 d6 3.d4, it will match against:

- 1.e4 (ECO B00)
- 1.e4 c5 (ECO B20)
- 1.e4 c5 2.Nf3 (ECO B27)
- 1.e4 c5 2.Nf3 d6 (ECO B50)

The longest match wins, so the game would be classified as B50.

---

## CQL Queries

CQL (Chess Query Language) lets you search for games containing specific positions or patterns. It's much more powerful than simple tag-based filtering.

### Basic Usage

```bash
# Find games with checkmate
pgn-extract-go --cql "mate" games.pgn

# Find games where white king is on g1
pgn-extract-go --cql "piece K g1" games.pgn

# Find check positions
pgn-extract-go --cql "check" games.pgn
```

### Using a CQL File

For complex queries, save them to a file:

```bash
echo "(and mate (piece [RQ] [a-h]8))" > back-rank.cql
pgn-extract-go --cql-file back-rank.cql games.pgn
```

### Learn More

CQL is a rich language with many features. See the full [CQL Documentation](CQL.md) for:

- Piece designators and square notation
- Logical operators (and, or, not)
- Counting and material filters
- Transformations (flip, shift)
- Game metadata filters (player, year, elo)
- Advanced pattern matching (pins, rays)

---

## Material Matching

Material matching lets you find games where a specific material balance occurs at any point during the game.

### Basic Material Pattern

Use `-z` for minimum material matching (at least these pieces exist):

```bash
# Find games with Q vs Q (queen endgames)
pgn-extract-go -z "Q:q" games.pgn

# Find games with two rooks vs one
pgn-extract-go -z "RR:r" games.pgn
```

### Exact Material Match

Use `-y` for exact material matching (exactly these pieces, no more):

```bash
# Find exact KQR vs KQR positions
pgn-extract-go -y "KQR:kqr" games.pgn

# Find king and pawn endgames
pgn-extract-go -y "KP:kp" games.pgn
```

### Pattern Format

Material patterns use `WhitePieces:BlackPieces` format:
- `K` = King, `Q` = Queen, `R` = Rook, `B` = Bishop, `N` = Knight, `P` = Pawn
- Use uppercase for White, lowercase for Black
- Repeat letters for multiple pieces: `RR` = two rooks

---

## Variation Matching

Variation matching finds games containing specific move sequences or position sequences.

### Move Sequence Matching

Use `-v` with a file containing move sequences:

```bash
# Create a file with opening moves
echo "1. e4 e5 2. Nf3 Nc6 3. Bb5" > italian.txt

# Find games with this opening
pgn-extract-go -v italian.txt games.pgn
```

### Position Sequence Matching

Use `-x` with a file containing FEN positions:

```bash
# Create file with FEN positions to match in order
echo "rnbqkb1r/pppp1ppp/5n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R" > setup.txt

# Find games reaching this position
pgn-extract-go -x setup.txt games.pgn
```

### File Format

Move sequence files have one sequence per line:
```
1. e4 e5 2. Nf3 Nc6
1. d4 d5 2. c4
```

Position files have one FEN per line, separated by blank lines for different sequences.

---

## Game Feature Filters

These filters find games based on specific chess characteristics.

### Game Ending Filters

```bash
# Games ending in checkmate
pgn-extract-go --checkmate games.pgn

# Games ending in stalemate
pgn-extract-go --stalemate games.pgn
```

### Draw Condition Filters

```bash
# Games where 50-move rule could be claimed
pgn-extract-go --fifty games.pgn

# Games with threefold repetition
pgn-extract-go --repetition games.pgn
```

### Special Move Filters

```bash
# Games with underpromotion (to R, B, or N instead of Q)
pgn-extract-go --underpromotion games.pgn

# Games with comments/annotations
pgn-extract-go --commented games.pgn
```

### Rating-Based Filters

```bash
# Higher-rated player won
pgn-extract-go --higherratedwinner games.pgn

# Lower-rated player won (upsets)
pgn-extract-go --lowerratedwinner games.pgn
```

### Game Length Filters

```bash
# Games with at least 40 ply (20 full moves)
pgn-extract-go --minply 40 games.pgn

# Short games (at most 20 ply)
pgn-extract-go --maxply 20 games.pgn

# Alternative: filter by move count
pgn-extract-go --minmoves 20 --maxmoves 40 games.pgn
```

### Negated Matching

Use `-n` to invert any filter:

```bash
# Games NOT ending in checkmate
pgn-extract-go -n --checkmate games.pgn

# Non-draws
pgn-extract-go -n -Tr "1/2-1/2" games.pgn
```

---

## Output Splitting

Split large databases into smaller files for easier management.

### Split by Game Count

Create files with a fixed number of games:

```bash
# Split into files of 1000 games each
pgn-extract-go -# 1000 -o output.pgn games.pgn
# Creates: output_001.pgn, output_002.pgn, ...
```

### Split by ECO Code

Organize games by opening classification:

```bash
# Split by ECO code
pgn-extract-go -E -e eco.pgn -o output.pgn games.pgn
# Creates: output_B20.pgn, output_C65.pgn, ...
```

### Limiting Output

Stop after a specific number of games:

```bash
# Output first 100 matching games
pgn-extract-go --stopafter 100 -p "Carlsen" games.pgn
```

---

## Validation and Fixing

PGN files from various sources often contain errors: missing tags, illegal moves, encoding issues, or malformed data. pgn-extract-go provides tools to handle these problems.

### Strict Mode

Use `--strict` to only output games that meet PGN standards:

```bash
# Only output games with all 7 required tags
pgn-extract-go --strict games.pgn
```

The Seven Tag Roster (required tags) are:
- Event, Site, Date, Round, White, Black, Result

Games missing any of these tags are skipped in strict mode.

### Move Validation

Use `--validate` to verify all moves are legal:

```bash
# Skip games with illegal moves
pgn-extract-go --validate games.pgn
```

This replays each game move-by-move and rejects games where any move is illegal according to chess rules. Useful for:
- Cleaning databases with corrupted games
- Verifying game integrity after format conversion
- Filtering out games with OCR or transcription errors

### Auto-Fix Mode

Use `--fixable` to automatically repair common issues:

```bash
# Fix problems and output repaired games
pgn-extract-go --fixable games.pgn
```

This fixes:
- **Missing required tags**: Adds placeholder values (e.g., `[Event "?"]`)
- **Invalid results**: Normalizes to standard format (1-0, 0-1, 1/2-1/2, *)
- **Date format**: Converts `2024/01/15` or `2024-01-15` to `2024.01.15`
- **Whitespace**: Trims leading/trailing spaces from tag values
- **Control characters**: Removes non-printable characters from tags

### Combining Options

Use `--fixable` with `--strict` to fix what can be fixed, then validate:

```bash
# Fix issues, then only output valid games
pgn-extract-go --fixable --strict games.pgn
```

Use `--fixable` with `--validate` for maximum cleanup:

```bash
# Fix tags and skip games with illegal moves
pgn-extract-go --fixable --validate games.pgn
```

### Example Workflow: Database Cleanup

```bash
# Step 1: See how many games have problems
pgn-extract-go -r --strict dirty.pgn
# Output: "Processed 1000 games, 847 matched"

# Step 2: Fix what can be fixed
pgn-extract-go --fixable --strict -o clean.pgn dirty.pgn

# Step 3: Validate moves too
pgn-extract-go --fixable --validate -o verified.pgn dirty.pgn
```

---

## Command Reference

### Output Options

| Flag | Description |
|------|-------------|
| `-o <file>` | Write output to file (default: stdout) |
| `-a` | Append to output file instead of overwriting |
| `-7` | Output only Seven Tag Roster |
| `--notags` | Don't output any tags |
| `-w <n>` | Maximum line length (default: 80) |
| `-W <format>` | Output format: san, lalg, halg, elalg, uci, epd, fen |
| `-J` | Output in JSON format |
| `-# <n>` | Split output into files of n games each |
| `-E` | Use ECO code for split file naming |
| `-l <file>` | Write log to file |
| `-L <file>` | Append log to file |
| `-r` | Report only (statistics, no game output) |

### Content Options

| Flag | Description |
|------|-------------|
| `-C` | Don't output comments |
| `-N` | Don't output NAGs |
| `-V` | Don't output variations |
| `--noresults` | Don't output results in moves |
| `--plycount` | Add PlyCount tag to games |
| `--addhashcode` | Add HashCode tag to games |
| `--fencomments` | Add FEN position as comment after each move |
| `--hashcomments` | Add position hash as comment after each move |
| `--fixresulttags` | Fix inconsistent Result tags |
| `--fixtagstrings` | Fix malformed tag strings |

### Validation Options

| Flag | Description |
|------|-------------|
| `--strict` | Only output games that parse without errors (all 7 required tags present) |
| `--validate` | Verify all moves are legal, skip games with illegal moves |
| `--fixable` | Attempt to fix common issues (missing tags, bad date format, encoding) |

### Filtering Options

| Flag | Description |
|------|-------------|
| `-t <file>` | Tag criteria file |
| `-p <name>` | Filter by player (either color) |
| `-Tw <name>` | Filter by White player |
| `-Tb <name>` | Filter by Black player |
| `-Te <code>` | Filter by ECO code prefix |
| `-Tr <result>` | Filter by result |
| `-Tf <fen>` | Filter by FEN position |
| `-Tp <name>` | Filter by player (either color, substring match) |
| `-S` | Use Soundex for player name matching |
| `-n` | Negate match (output non-matching games) |
| `--stopafter <n>` | Stop after outputting n games |

### Game Length Filters

| Flag | Description |
|------|-------------|
| `--minply <n>` | Minimum ply count (half-moves) |
| `--maxply <n>` | Maximum ply count |
| `--minmoves <n>` | Minimum move count |
| `--maxmoves <n>` | Maximum move count |

### Game Feature Filters

| Flag | Description |
|------|-------------|
| `--checkmate` | Only games ending in checkmate |
| `--stalemate` | Only games ending in stalemate |
| `--fifty` | Games with 50-move rule draw potential |
| `--repetition` | Games with threefold repetition |
| `--underpromotion` | Games with underpromotion |
| `--commented` | Only games with comments |
| `--higherratedwinner` | Higher-rated player won |
| `--lowerratedwinner` | Lower-rated player won (upset) |

### Material Matching

| Flag | Description |
|------|-------------|
| `-z <pattern>` | Match material balance (e.g., "Q:q" for Q vs Q) |
| `-y <pattern>` | Exact material match (e.g., "KQR:kqr") |

### Variation Matching

| Flag | Description |
|------|-------------|
| `-v <file>` | Match games containing move sequences from file |
| `-x <file>` | Match games containing positional sequences from file |

### CQL Options

| Flag | Description |
|------|-------------|
| `--cql <query>` | CQL query string |
| `--cql-file <file>` | File containing CQL query |

### Duplicate Detection

| Flag | Description |
|------|-------------|
| `-D` | Suppress duplicate games |
| `-d <file>` | Write duplicates to file |
| `-U` | Output only duplicate games |
| `-c <file>` | Check against games in file (don't output those) |

### Hash Matching

| Flag | Description |
|------|-------------|
| `-H <file>` | Match games reaching positions with hashes in file |

### ECO Classification

| Flag | Description |
|------|-------------|
| `-e <file>` | ECO classification file (PGN format) |

### Other Options

| Flag | Description |
|------|-------------|
| `-s` | Silent mode (no statistics) |
| `-h` | Show help |
| `--version` | Show version |

---

## Examples

### Extract a Player's Games

Get all of Bobby Fischer's games as White where he won:

```bash
pgn-extract-go -Tw "Fischer" -Tr "1-0" -o fischer-wins.pgn megabase.pgn
```

### Clean Up a PGN File

Remove comments, variations, and duplicates:

```bash
pgn-extract-go -C -V -D games.pgn > clean.pgn
```

### Convert to UCI Format

For use with chess engines:

```bash
pgn-extract-go -W uci --notags games.pgn
```

### Find Tactical Patterns

Games with knight forks (knight attacking king and another piece):

```bash
pgn-extract-go --cql "(and (attack N k) (attack N [qr]))" games.pgn
```

### Process Tournament Results

Extract decisive games from a tournament:

```bash
pgn-extract-go -Tr "1-0" -o white-wins.pgn tournament.pgn
pgn-extract-go -Tr "0-1" -o black-wins.pgn tournament.pgn
pgn-extract-go -Tr "1/2-1/2" -o draws.pgn tournament.pgn
```

### Create a Checkmate Collection

Find all checkmates with back rank patterns:

```bash
pgn-extract-go --cql "(and mate (piece k [a-h]8) (attack [RQ] k))" \
  -o back-rank-mates.pgn games.pgn
```

### Combine Multiple Databases

Merge files and remove duplicates:

```bash
pgn-extract-go -D -o combined.pgn db1.pgn db2.pgn db3.pgn
```

### Export for Analysis

Create JSON output for processing with other tools:

```bash
pgn-extract-go -J -p "Carlsen" games.pgn > carlsen.json
```

### Find Opening Transpositions

Search for games reaching the King's Indian setup:

```bash
pgn-extract-go --cql "(and (piece p d6) (piece p g6) (piece b g7) (piece n f6))" \
  games.pgn
```

### Find Upsets

Games where the lower-rated player won:

```bash
pgn-extract-go --lowerratedwinner -o upsets.pgn tournament.pgn
```

### Endgame Study

Find queen endgames:

```bash
pgn-extract-go -z "KQ:kq" -o queen-endgames.pgn games.pgn
```

Find exact rook endgames (only K+R vs K+R):

```bash
pgn-extract-go -y "KR:kr" -o pure-rook-endings.pgn games.pgn
```

### Find Similar Names

Use Soundex to find players with similar-sounding names:

```bash
# Find both "Fischer" and "Fisher"
pgn-extract-go -S -p "Fischer" games.pgn
```

### Split Large Database

Divide a database into manageable chunks:

```bash
# Split into files of 500 games each
pgn-extract-go -# 500 -o output.pgn megabase.pgn
```

### Sample Games

Get a quick sample of games:

```bash
# First 10 games matching criteria
pgn-extract-go --stopafter 10 --checkmate games.pgn
```

### Add Analysis Annotations

Add FEN positions as comments for engine analysis:

```bash
pgn-extract-go --fencomments games.pgn > annotated.pgn
```

### Report Statistics

Get statistics without outputting games:

```bash
pgn-extract-go -r games.pgn
```

### Exclude Known Games

Filter out games already in your database:

```bash
pgn-extract-go -c existing.pgn new-games.pgn -o truly-new.pgn
```

---

## Tips and Best Practices

### Performance

- Process files in batches rather than one at a time when doing duplicate detection
- Use `-s` (silent mode) when processing large files to avoid output overhead
- CQL queries check every position, so complex queries on large databases may take time

### Combining with Unix Tools

pgn-extract-go works well with standard Unix tools:

```bash
# Count games matching criteria
pgn-extract-go -s -p "Kasparov" games.pgn | grep -c "^\[Event"

# Extract just the results
pgn-extract-go -7 games.pgn | grep "Result"

# Process files in parallel
ls *.pgn | xargs -P 4 -I {} pgn-extract-go --checkmate {}
```

### Common Workflows

**Building a training database:**
```bash
# Get decisive games from strong players
pgn-extract-go -D \
  --cql "(and (> (elo \"white\") 2600) (> (elo \"black\") 2600))" \
  -Tr "1-0" -o training.pgn megabase.pgn
```

**Preparing for opening study:**
```bash
# Extract Sicilian Najdorf games
pgn-extract-go -Te "B90" -e eco.pgn -o najdorf.pgn games.pgn
```

**Quality control:**
```bash
# Find potentially problematic games (ended in mate but result is draw)
pgn-extract-go --checkmate -Tr "1/2-1/2" suspicious.pgn
```

---

## See Also

- [CQL Documentation](CQL.md) - Complete Chess Query Language reference
- [pgn-extract](https://www.cs.kent.ac.uk/people/staff/djb/pgn-extract/) - Original C implementation by David J. Barnes
- [PGN Specification](https://www.thechessdrum.net/PGN_Reference.txt) - Official PGN format standard
