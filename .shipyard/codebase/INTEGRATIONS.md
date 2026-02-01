# External Integrations

## Overview

pgn-extract-go is a **standalone, zero-dependency command-line tool** with no external service integrations, APIs, or databases. All data processing happens locally on the filesystem.

## External Services: None

This project has:
- **No external API calls** - All processing is local
- **No database connections** - Input/output via PGN files
- **No network requests** - Pure offline tool
- **No cloud services** - Entirely filesystem-based
- **No authentication** - No user accounts or credentials

## Data Sources

### Input: PGN Files

**Format**: Portable Game Notation (text-based chess game format)
**Source**: User-provided local files
**Encoding**: UTF-8 text
**Size**: Handles files of arbitrary size via streaming parser

**Example Input:**
```
[Event "World Championship"]
[White "Kasparov, Garry"]
[Black "Karpov, Anatoly"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 1-0
```

**File Locations:**
- Command-line arguments: `pgn-extract games.pgn`
- ECO classification file: `-e eco.pgn`
- Tag criteria file: `-t criteria.txt`
- Variation match file: `-v variations.txt`
- Position match file: `-x positions.txt`
- Check file (duplicate detection): `-c checkfile.txt`
- CQL query file: `--cql-file query.cql`

### Output: Local Files

**Formats Supported:**
1. **PGN** (default) - Standard chess game notation
2. **JSON** (`-J`) - Structured game data
3. **EPD** (`-W epd`) - Extended Position Description
4. **FEN** (`-W fen`) - Forsyth-Edwards Notation

**Output Destinations:**
- `stdout` (default) - Pipeline-friendly
- File (`-o output.pgn`) - Overwrite mode
- File (`-a -o output.pgn`) - Append mode
- Split files (`-# N`) - Multiple files of N games each
- ECO-based split (`-E 1|2|3`) - Split by opening classification

**Example Output (JSON):**
```json
{
  "tags": {
    "Event": "World Championship",
    "White": "Kasparov, Garry",
    "Black": "Karpov, Anatoly",
    "Result": "1-0"
  },
  "moves": [
    {"san": "e4", "uci": "e2e4", "piece": "P"},
    {"san": "e5", "uci": "e7e5", "piece": "P"}
  ]
}
```

## Standard Library I/O

All I/O operations use Go's standard library (no external libraries):

### File Operations

**Packages Used:**
- `os` - File opening, reading, writing
- `bufio` - Buffered reading for large files
- `io` - Generic I/O interfaces

**Key Patterns:**
```go
// Reading PGN files
file, err := os.Open("games.pgn")
reader := bufio.NewReader(file)
parser := parser.NewParser(reader, cfg)

// Writing output
writer, err := os.Create("output.pgn")
defer writer.Close()
```

**Performance Optimizations:**
- Streaming parser (low memory footprint)
- Buffered I/O for large files
- Parallel processing via worker pool (`--workers N`)

### Logging

**Log Destinations:**
- `stderr` - Error and diagnostic messages (default)
- Log file (`-l logfile.txt`) - Overwrite mode
- Log file (`-L logfile.txt`) - Append mode

**Log Modes:**
- Normal: Progress and error messages
- Silent (`-s`): Suppress game count messages
- Report-only (`-r`): Report errors without extracting games

**Implementation**: Uses `fmt.Fprintf(os.Stderr, ...)` and `log` package

## Data Processing Patterns

### 1. Streaming Input

**No Database**: Games are parsed from PGN files on-the-fly
**Memory Efficient**: Only current game held in memory
**Scalability**: Can process multi-gigabyte PGN files

**Parser Location**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/parser/`

### 2. In-Memory Caching

**ECO Classification:**
- ECO database loaded once from PGN file (`-e eco.pgn`)
- Stored in memory as trie/hash structure
- No persistent cache (reload on each run)

**Duplicate Detection:**
- Move sequence hashing (Zobrist hashing algorithm)
- Position hash tracking in memory
- Optional check file for cross-run deduplication

**Hashing Location**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/hashing/`

### 3. Worker Pool Pattern

**Concurrency**: Optional parallel processing (`--workers N`)
**Implementation**: Go channels + goroutines
**Thread Safety**: Synchronized hash tables for duplicate detection

**Worker Pool Location**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/worker/`

## Third-Party Data Formats

### ECO (Encyclopedia of Chess Openings)

**Format**: Standard PGN file with opening classifications
**Source**: User-provided (not bundled)
**Structure**:
```
[ECO "A00"]
[Opening "Polish Opening"]

1. b4
```

**Usage**: `-e eco.pgn` loads opening classifications

**Implementation**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/eco/eco.go`

### CQL (Chess Query Language)

**Format**: Domain-specific language for position queries
**Spec**: Created by Gady Costeff and Lewis Stiller
**Implementation**: Custom lexer/parser (no external library)

**Example Queries:**
```
mate                           # Find checkmate positions
piece K g1                     # Find king on g1
(and mate (piece Q h7))        # Queen on h7 giving mate
```

**CQL Location**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/internal/cql/`

## Platform Integration

### Operating System

**Supported Platforms:**
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

**OS-Specific Code**: None (pure Go, cross-platform)

**File Paths**: Uses `filepath.Join()` for cross-platform compatibility

### Command-Line Interface

**Flag Parsing**: Go `flag` package (stdlib)
**Exit Codes**:
- `0` - Success
- `1` - Error (file not found, parse error, etc.)

**Pipeline Support:**
```bash
# Read from stdin, write to stdout
cat games.pgn | pgn-extract -p Fischer > fischer.pgn

# Chain with other tools
pgn-extract -J games.pgn | jq '.tags.White'
```

## Build-Time Integrations

### GitHub Actions

**Purpose**: CI/CD automation
**Config**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/.github/workflows/ci.yml`

**Triggers:**
- Push to main/master/develop/feature/release branches
- Pull requests
- Version tags (`v*`)
- Manual workflow dispatch

**External Actions Used:**
- `actions/checkout@v4` - Git checkout
- `actions/setup-go@v5` - Go installation
- `actions/setup-python@v5` - Python for pre-commit
- `actions/cache@v4` - Dependency caching
- `actions/upload-artifact@v4` - Coverage artifact upload
- `securego/gosec@master` - Security scanning
- `github/codeql-action/upload-sarif@v3` - Security report upload
- `aquasecurity/trivy-action@master` - Vulnerability scanning
- `goreleaser/goreleaser-action@v6` - Release automation

**Secrets Required:**
- `GITHUB_TOKEN` (automatic, no setup needed)

**No other secrets** (no deployment keys, API tokens, etc.)

### GoReleaser

**Purpose**: Multi-platform binary distribution
**Integration**: GitHub Releases

**Release Artifacts:**
- Pre-compiled binaries (6 platform combinations)
- Archive files (`.tar.gz`, `.zip`)
- `checksums.txt` (SHA256)
- Auto-generated changelog (from git history)

**Download URLs:**
```
https://github.com/lgbarn/pgn-extract-go/releases/download/v1.0.0/pgn-extract_1.0.0_linux_amd64.tar.gz
https://github.com/lgbarn/pgn-extract-go/releases/download/v1.0.0/pgn-extract_1.0.0_darwin_arm64.tar.gz
```

**No package managers** (Homebrew tap commented out in config)

## Development Integrations

### Pre-commit Hooks

**Framework**: pre-commit.com (Python-based)
**Repository**: https://github.com/pre-commit/pre-commit-hooks

**External Hook Sources:**
- `pre-commit/pre-commit-hooks@v5.0.0` - Generic file checks
- `shellcheck-py/shellcheck-py@v0.10.0.1` - Shell linting

**Local Hooks**: All Go checks run via `bash -c` (no external dependencies)

### Linter Integrations

**golangci-lint Installation:**
```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s
```

**External Tools Installed:**
- `honnef.co/go/tools/cmd/staticcheck@latest`
- `golang.org/x/tools/cmd/goimports@latest`

**No persistent connections** - All tools run locally

## Security Considerations

### No External Network Calls

✅ **Attack Surface**: Minimal (filesystem only)
✅ **Data Privacy**: All processing local, no telemetry
✅ **Offline Usage**: Fully functional without internet
✅ **Supply Chain**: Zero runtime dependencies

### Input Validation

**PGN Parser**: Validates syntax, handles malformed input gracefully
**Strict Mode** (`--strict`): Rejects games with parse errors
**Validation Mode** (`--validate`): Verifies move legality

**Security Scanning:**
- gosec: Enabled for 30+ security checks (G101-G505)
- Trivy: Scans for known vulnerabilities
- No hardcoded credentials (G101 check enforced)
- No command injection vectors (G204 check enforced)

## Future Integration Possibilities

**Commented Out in GoReleaser Config:**
```yaml
# Homebrew formula (optional - uncomment if you want to publish to a tap)
# brews:
#   - repository:
#       owner: lgbarn
#       name: homebrew-tap
```

**Potential Future Integrations** (not currently implemented):
- Homebrew tap for `brew install pgn-extract`
- Docker image distribution
- Lichess/Chess.com API integration (for downloading games)
- Database export (PostgreSQL, SQLite)
- Web service wrapper (HTTP API)

**Current Status**: None of these are implemented; the tool is purely file-based.

## Summary

pgn-extract-go is an **isolated, self-contained tool** with:
- **Zero runtime dependencies** (pure Go stdlib)
- **No external service calls** (filesystem only)
- **No databases** (streaming PGN parser)
- **Build-time integrations** (GitHub Actions, GoReleaser)
- **Optional dev tools** (pre-commit, linters)

All "integrations" are for development/build automation, not runtime dependencies.
