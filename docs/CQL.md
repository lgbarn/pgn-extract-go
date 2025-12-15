# CQL: Chess Query Language

CQL (Chess Query Language) lets you search chess games by describing positions and patterns. Instead of looking through games one by one, you write a query that describes what you want to find, and CQL finds all matching games for you.

## Table of Contents

- [What is CQL?](#what-is-cql)
- [Quick Start](#quick-start)
- [Piece Designators](#piece-designators)
- [Square Notation](#square-notation)
- [Basic Filters](#basic-filters)
- [Logical Operators](#logical-operators)
- [Counting and Comparisons](#counting-and-comparisons)
- [Transformations](#transformations)
- [Game Metadata Filters](#game-metadata-filters)
- [Advanced Filters](#advanced-filters)
- [Using CQL Files](#using-cql-files)
- [Complete Examples](#complete-examples)
- [Filter Reference](#filter-reference)

---

## What is CQL?

CQL was created by Gady Costeff and Lewis Stiller to help chess composers find specific patterns in large game databases. It has since become useful for:

- **Researchers** studying opening theory or endgame patterns
- **Players** finding games with specific tactical themes
- **Authors** collecting examples for books or articles
- **Composers** searching for studies with particular characteristics

CQL works by checking each position in a game against your query. If any position matches, the game is included in the output.

### How It Works

When you run a CQL query, the program:

1. Reads each game from the input file
2. Replays the game move by move
3. Checks each position against your query
4. Outputs games where at least one position matches

This means a query like `mate` will find all games that contain a checkmate position anywhere in the game.

---

## Quick Start

### Your First Query

Find all games that end in checkmate:

```bash
pgn-extract-go --cql "mate" games.pgn
```

Find games where the white king is on g1 (typically after castling kingside):

```bash
pgn-extract-go --cql "piece K g1" games.pgn
```

Find games with a check:

```bash
pgn-extract-go --cql "check" games.pgn
```

### Understanding the Syntax

CQL uses a simple pattern: `filter arguments`

- `mate` - A filter with no arguments
- `piece K g1` - The `piece` filter with two arguments: `K` (white king) and `g1` (a square)

Multiple conditions are combined with parentheses and logical operators:

```bash
pgn-extract-go --cql "(and mate wtm)" games.pgn
```

This finds checkmate positions where it's white to move (meaning black delivered the mate).

---

## Piece Designators

Piece designators tell CQL which pieces to look for. They use standard chess notation with some additions.

### Single Pieces

| Designator | Meaning |
|------------|---------|
| `K` | White King |
| `Q` | White Queen |
| `R` | White Rook |
| `B` | White Bishop |
| `N` | White Knight |
| `P` | White Pawn |
| `k` | Black King |
| `q` | Black Queen |
| `r` | Black Rook |
| `b` | Black Bishop |
| `n` | Black Knight |
| `p` | Black Pawn |

Uppercase letters are white pieces. Lowercase letters are black pieces.

### Special Designators

| Designator | Meaning |
|------------|---------|
| `A` | Any white piece (K, Q, R, B, N, or P) |
| `a` | Any black piece (k, q, r, b, n, or p) |
| `_` | Empty square |
| `?` | Any piece or empty (matches anything) |

### Piece Sets

Use square brackets to match any piece from a set:

| Example | Meaning |
|---------|---------|
| `[RQ]` | White Rook or White Queen |
| `[rq]` | Black Rook or Black Queen |
| `[NB]` | White Knight or White Bishop |
| `[KQRBNP]` | Any white piece (same as `A`) |

### Examples

```bash
# Games with a white queen on d4
pgn-extract-go --cql "piece Q d4" games.pgn

# Games with any black piece on e5
pgn-extract-go --cql "piece a e5" games.pgn

# Games with a rook or queen on the a-file, rank 1
pgn-extract-go --cql "piece [RQ] a1" games.pgn
```

---

## Square Notation

Squares can be specified individually or as sets.

### Single Squares

Standard algebraic notation: `a1` through `h8`

```
  a   b   c   d   e   f   g   h
8 [ ] [ ] [ ] [ ] [ ] [ ] [ ] [ ]
7 [ ] [ ] [ ] [ ] [ ] [ ] [ ] [ ]
6 [ ] [ ] [ ] [ ] [ ] [ ] [ ] [ ]
5 [ ] [ ] [ ] [ ] [ ] [ ] [ ] [ ]
4 [ ] [ ] [ ] [ ] [ ] [ ] [ ] [ ]
3 [ ] [ ] [ ] [ ] [ ] [ ] [ ] [ ]
2 [ ] [ ] [ ] [ ] [ ] [ ] [ ] [ ]
1 [ ] [ ] [ ] [ ] [ ] [ ] [ ] [ ]
```

### Square Ranges

Use brackets with ranges to match multiple squares:

| Pattern | Meaning | Squares Matched |
|---------|---------|-----------------|
| `[a-h]1` | Entire first rank | a1, b1, c1, d1, e1, f1, g1, h1 |
| `a[1-8]` | Entire a-file | a1, a2, a3, a4, a5, a6, a7, a8 |
| `[a-d][1-4]` | Lower-left quadrant | a1-d1, a2-d2, a3-d3, a4-d4 (16 squares) |
| `[e-h][5-8]` | Upper-right quadrant | 16 squares |

### Special Square Designators

| Pattern | Meaning |
|---------|---------|
| `.` | Any square (all 64) |

### Examples

```bash
# White pawns on the 5th rank (advanced pawns)
pgn-extract-go --cql "piece P [a-h]5" games.pgn

# Any piece on the e-file
pgn-extract-go --cql "piece ? e[1-8]" games.pgn

# Empty central squares (e4, d4, e5, d5)
pgn-extract-go --cql "piece _ [d-e][4-5]" games.pgn
```

---

## Basic Filters

These filters check the current position for specific conditions.

### Position Filters

| Filter | Description |
|--------|-------------|
| `check` | King is in check |
| `mate` | Checkmate position |
| `stalemate` | Stalemate position |
| `wtm` | White to move |
| `btm` | Black to move |

### Piece Placement

The `piece` filter checks if a specific piece is on a specific square:

```
piece <piece-designator> <square>
```

Examples:

```bash
# White king on e1 (hasn't castled or moved)
pgn-extract-go --cql "piece K e1" games.pgn

# Black pawn on d5 (common in many openings)
pgn-extract-go --cql "piece p d5" games.pgn

# Any rook on an open file (checking if square is occupied)
pgn-extract-go --cql "piece [Rr] [a-h]1" games.pgn
```

### Attack Detection

The `attack` filter checks if one piece attacks a square (or another piece):

```
attack <attacker> <target-square>
```

Examples:

```bash
# Rook attacks the black king
pgn-extract-go --cql "attack R k" games.pgn

# Knight attacks the queen
pgn-extract-go --cql "attack N q" games.pgn

# Any piece attacks e5
pgn-extract-go --cql "attack A e5" games.pgn
```

---

## Logical Operators

Combine multiple conditions using logical operators. These must be wrapped in parentheses.

### AND - All Conditions Must Match

```
(and condition1 condition2 ...)
```

```bash
# Checkmate with white to move (black delivered mate)
pgn-extract-go --cql "(and mate wtm)" games.pgn

# White king castled kingside AND black king castled queenside
pgn-extract-go --cql "(and (piece K g1) (piece k c8))" games.pgn
```

### OR - Any Condition Can Match

```
(or condition1 condition2 ...)
```

```bash
# Checkmate or stalemate
pgn-extract-go --cql "(or mate stalemate)" games.pgn

# King on g1 or c1 (castled either side)
pgn-extract-go --cql "(or (piece K g1) (piece K c1))" games.pgn
```

### NOT - Condition Must Not Match

```
(not condition)
```

```bash
# Positions that are NOT in check
pgn-extract-go --cql "(not check)" games.pgn

# Games without castled kings
pgn-extract-go --cql "(not (or (piece K g1) (piece K c1)))" games.pgn
```

### Combining Operators

You can nest logical operators for complex queries:

```bash
# Check delivered by knight or bishop
pgn-extract-go --cql "(and check (or (attack N k) (attack B k)))" games.pgn

# Mate but not with a queen
pgn-extract-go --cql "(and mate (not (attack Q k)))" games.pgn
```

---

## Counting and Comparisons

Count pieces and compare values using numeric filters.

### The count Filter

Count how many pieces match a designator:

```
(count <piece-designator>)
```

This returns a number, which you compare using operators.

### Comparison Operators

| Operator | Meaning |
|----------|---------|
| `>` | Greater than |
| `<` | Less than |
| `>=` | Greater than or equal |
| `<=` | Less than or equal |
| `==` | Equal to |
| `!=` | Not equal to |

Comparisons use prefix notation (operator first):

```
(> (count P) 5)    # More than 5 white pawns
(< (count p) 8)    # Fewer than 8 black pawns (at least one captured)
(== (count R) 2)   # Exactly 2 white rooks
```

### Examples

```bash
# Games where white has lost pawns
pgn-extract-go --cql "(< (count P) 8)" games.pgn

# Games with a piece imbalance (not equal material)
pgn-extract-go --cql "(!= (count [QRBN]) (count [qrbn]))" games.pgn

# Endgames with few pieces (6 or fewer total)
pgn-extract-go --cql "(<= (count ?) 6)" games.pgn
```

### The material Filter

Calculate total material value for a side:

```
(material "white")
(material "black")
```

Material values: Pawn=1, Knight=3, Bishop=3, Rook=5, Queen=9

```bash
# White has more material
pgn-extract-go --cql "(> (material \"white\") (material \"black\"))" games.pgn

# Material is equal
pgn-extract-go --cql "(== (material \"white\") (material \"black\"))" games.pgn
```

---

## Transformations

Transformations modify a pattern to match equivalent positions. This is useful for finding patterns regardless of which side of the board they occur on.

### flip - Mirror Horizontally

Reflects the pattern across the center of the board (a-file ↔ h-file):

```bash
# King on g1 OR b1 (castled on either side, for both colors)
pgn-extract-go --cql "(flip (piece K g1))" games.pgn
```

The `flip` transformation takes the pattern `piece K g1` and also checks `piece K b1`.

### flipvertical - Mirror Vertically

Reflects the pattern top to bottom (rank 1 ↔ rank 8):

```bash
# Rook on first rank OR eighth rank
pgn-extract-go --cql "(flipvertical (piece R [a-h]1))" games.pgn
```

### flipcolor - Swap Colors

Swaps white and black pieces:

```bash
# Pattern for either color
pgn-extract-go --cql "(flipcolor (piece K e1))" games.pgn
```

This matches white king on e1 OR black king on e8.

### shift - Try All Translations

Tries the pattern at every possible position on the board:

```bash
# Find a piece formation anywhere on the board
pgn-extract-go --cql "(shift (and (piece N c3) (piece B c4)))" games.pgn
```

### shifthorizontal / shiftvertical

Shift pattern only left/right or only up/down:

```bash
# Rook on any square of the first rank
pgn-extract-go --cql "(shifthorizontal (piece R a1))" games.pgn
```

---

## Game Metadata Filters

Filter games by their header information (tags), not just positions.

### result - Game Outcome

```bash
# White wins
pgn-extract-go --cql "result \"1-0\"" games.pgn

# Black wins
pgn-extract-go --cql "result \"0-1\"" games.pgn

# Draws
pgn-extract-go --cql "result \"1/2-1/2\"" games.pgn
```

### player - Player Name

Searches both White and Black player names:

```bash
# Games with Fischer
pgn-extract-go --cql "player \"Fischer\"" games.pgn

# Games with Kasparov
pgn-extract-go --cql "player \"Kasparov\"" games.pgn
```

The search is case-sensitive and matches partial names.

### year - Game Year

Use with comparison operators:

```bash
# Games from 1972
pgn-extract-go --cql "(== (year) 1972)" games.pgn

# Games after 2000
pgn-extract-go --cql "(> (year) 2000)" games.pgn

# Games from the 1990s
pgn-extract-go --cql "(and (>= (year) 1990) (< (year) 2000))" games.pgn
```

### elo - Player Rating

Check player ratings:

```bash
# White player rated above 2700
pgn-extract-go --cql "(> (elo \"white\") 2700)" games.pgn

# Both players above 2600
pgn-extract-go --cql "(and (> (elo \"white\") 2600) (> (elo \"black\") 2600))" games.pgn
```

---

## Advanced Filters

These filters detect more complex positional patterns.

### pin - Pinned Pieces

The `pin` filter detects when a piece is pinned:

```
(pin <pinned-piece> <pinner> <protected-piece>)
```

```bash
# Knight pinned by bishop to king
pgn-extract-go --cql "(pin N b K)" games.pgn

# Any piece pinned by rook to queen
pgn-extract-go --cql "(pin A r Q)" games.pgn
```

### ray - Pieces Along a Line

Check if pieces lie along a straight line:

```
ray "<direction>" <square1> <square2>
```

Directions: `horizontal`, `vertical`, `diagonal`, `orthogonal`

```bash
# Rook and king on same rank
pgn-extract-go --cql "ray \"horizontal\" R K" games.pgn

# Bishop and king on same diagonal
pgn-extract-go --cql "ray \"diagonal\" B k" games.pgn
```

### between - Squares Between Two Points

Check squares between two pieces:

```
(between <square1> <square2>)
```

```bash
# Check if squares between rook and king are empty
pgn-extract-go --cql "(and (between a1 e1) (piece _ b1))" games.pgn
```

---

## Using CQL Files

For complex queries, you can save your CQL in a file and reference it:

```bash
# Create a CQL file
echo '(and mate (piece [RQ] [a-h]8))' > back-rank-mate.cql

# Use the file
pgn-extract-go --cql-file back-rank-mate.cql games.pgn
```

### File Format

CQL files are plain text. You can format them across multiple lines for readability:

```
# back-rank-mate.cql
(and
  mate
  (piece [RQ] [a-h]8)
  (piece k [a-h]8)
)
```

Note: Comments (lines starting with `#`) are not currently supported in CQL files. Only the query itself should be in the file.

### Benefits of CQL Files

- **Reusable**: Save common queries for repeated use
- **Readable**: Format complex queries across multiple lines
- **Shareable**: Exchange query files with others
- **Versionable**: Track changes in version control

---

## Complete Examples

### Finding Checkmate Patterns

**Back Rank Mate**: King trapped on back rank, mated by rook or queen:

```bash
pgn-extract-go --cql "(and mate (piece [RQ] [a-h]8) (piece k [a-h]8))" games.pgn
```

**Smothered Mate**: King surrounded by own pieces, mated by knight:

```bash
pgn-extract-go --cql "(and mate (attack N k))" games.pgn
```

### Finding Opening Patterns

**Sicilian Dragon**: Black fianchettoed bishop and pawn on d6:

```bash
pgn-extract-go --cql "(and (piece p d6) (piece b g7))" games.pgn
```

**King's Indian Defense**: Similar fianchetto with pawn on d6:

```bash
pgn-extract-go --cql "(and (piece p d6) (piece p g6) (piece b g7))" games.pgn
```

### Finding Endgame Positions

**Rook Endgames**: Only rooks and pawns remain:

```bash
pgn-extract-go --cql "(and (== (count [QBN]) 0) (== (count [qbn]) 0) (> (count [Rr]) 0))" games.pgn
```

**Opposite-Colored Bishops**: Each side has one bishop on different colors:

```bash
# Simplified: each side has exactly one bishop
pgn-extract-go --cql "(and (== (count B) 1) (== (count b) 1))" games.pgn
```

### Finding Tactical Themes

**Queen Sacrifice**: Games where white loses the queen but wins:

```bash
pgn-extract-go --cql "(and (== (count Q) 0) (> (count q) 0) (result \"1-0\"))" games.pgn
```

**Double Check**: Check by two pieces:

```bash
pgn-extract-go --cql "(and check (>= (count A) 2))" games.pgn
```

### Finding Games by Criteria

**High-Level Games from the 1970s**:

```bash
pgn-extract-go --cql "(and (>= (year) 1970) (< (year) 1980) (> (elo \"white\") 2600))" games.pgn
```

**Fischer's Wins as White**:

```bash
pgn-extract-go --cql "(and (player \"Fischer\") (result \"1-0\"))" games.pgn
```

---

## Filter Reference

### Position Filters

| Filter | Arguments | Description |
|--------|-----------|-------------|
| `check` | none | Position has check |
| `mate` | none | Position is checkmate |
| `stalemate` | none | Position is stalemate |
| `wtm` | none | White to move |
| `btm` | none | Black to move |
| `piece` | designator, square | Piece on square |
| `attack` | attacker, target | Piece attacks square |

### Numeric Functions

| Function | Arguments | Returns |
|----------|-----------|---------|
| `count` | designator | Number of matching pieces |
| `material` | `"white"` or `"black"` | Total material value |
| `year` | none | Year from Date tag |
| `elo` | `"white"` or `"black"` | Player's Elo rating |

### Comparison Operators

| Operator | Usage | Description |
|----------|-------|-------------|
| `>` | `(> a b)` | a greater than b |
| `<` | `(< a b)` | a less than b |
| `>=` | `(>= a b)` | a greater than or equal to b |
| `<=` | `(<= a b)` | a less than or equal to b |
| `==` | `(== a b)` | a equals b |
| `!=` | `(!= a b)` | a not equals b |

### Logical Operators

| Operator | Usage | Description |
|----------|-------|-------------|
| `and` | `(and a b ...)` | All conditions must match |
| `or` | `(or a b ...)` | Any condition can match |
| `not` | `(not a)` | Condition must not match |

### Transformations

| Transform | Arguments | Description |
|-----------|-----------|-------------|
| `flip` | pattern | Mirror left-right |
| `flipvertical` | pattern | Mirror top-bottom |
| `flipcolor` | pattern | Swap white/black |
| `shift` | pattern | Try all board positions |
| `shifthorizontal` | pattern | Shift left/right only |
| `shiftvertical` | pattern | Shift up/down only |

### Game Metadata

| Filter | Arguments | Description |
|--------|-----------|-------------|
| `result` | string | Match game result |
| `player` | string | Match player name |
| `year` | none | Get year for comparison |
| `elo` | `"white"` or `"black"` | Get rating for comparison |

### Advanced Filters

| Filter | Arguments | Description |
|--------|-----------|-------------|
| `pin` | piece, pinner, target | Detect pinned piece |
| `ray` | direction, sq1, sq2 | Pieces on a line |
| `between` | sq1, sq2 | Squares between two points |

---

## Further Reading

- [Original CQL Documentation](https://www.gadycosteff.com/cql/) by Gady Costeff
- [pgn-extract Documentation](https://www.cs.kent.ac.uk/people/staff/djb/pgn-extract/help.html) by David J. Barnes
- [Chess Programming Wiki - CQL](https://www.chessprogramming.org/Chess_Query_Language)
