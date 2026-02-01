# Technology Stack

## Overview

pgn-extract-go is a pure Go CLI application for processing chess PGN (Portable Game Notation) files. It has zero runtime dependencies and compiles to a single static binary.

## Primary Language

### Go

- **Minimum Version**: Go 1.21
- **CI/CD Version**: Go 1.23 (defined in `.github/workflows/ci.yml`)
- **Installed Version**: Go 1.25.6 (developer machine)
- **Module Path**: `github.com/lgbarn/pgn-extract-go`
- **Dependencies**: Zero external dependencies (uses only Go standard library)

**Key Configuration:**
```go
// go.mod
module github.com/lgbarn/pgn-extract-go

go 1.21
```

**Standard Library Usage:**
- `bufio` - Buffered I/O for PGN file parsing
- `flag` - Command-line flag parsing
- `io` - Core I/O interfaces
- `encoding/json` - JSON output format
- `fmt`, `strings`, `strconv` - String manipulation
- `os`, `path/filepath` - File system operations
- `sync`, `sync/atomic` - Concurrency primitives for worker pool
- `testing`, `testing/quick` - Test framework

**Build Settings:**
- `CGO_ENABLED=0` - Pure Go, no C dependencies
- Cross-compilation support: `linux`, `darwin`, `windows` on `amd64`, `arm64`

## Build Tools

### Just Command Runner

**Purpose**: Task automation and build orchestration
**Config File**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/justfile`
**Version**: Not pinned

The project uses `just` as its primary task runner with recipes for:
- Building: `just build`, `just build-release`, `just install`
- Testing: `just test`, `just test-race`, `just test-coverage`, `just test-golden`, `just test-cql`
- Development: `just fmt`, `just lint`, `just check`, `just watch`
- Benchmarking: `just bench`, `just bench-pkg`
- Dependencies: `just deps`, `just update-deps`, `just tidy`

**Alternative**: All tasks can be run directly with `go` commands (no hard dependency on `just`).

### GoReleaser

**Purpose**: Release automation and cross-platform binary distribution
**Config File**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/.goreleaser.yml`
**Version**: v2 (latest)

**Configuration Highlights:**
- **Platforms**: Linux, macOS, Windows on amd64 and arm64
- **Archive Formats**: tar.gz (Unix), zip (Windows)
- **Build Flags**: `-s -w` (strip debug info), version injection via `-X main.programVersion={{.Version}}`
- **Release Assets**: Binary archives, checksums (SHA256), changelog
- **GitHub Integration**: Automated releases on version tags (`v*`)

**Release Flow:**
```yaml
# Triggered on git tags
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
# GitHub Actions runs GoReleaser → creates GitHub release
```

## Code Quality Tools

### Go Toolchain

1. **gofmt** - Standard Go formatter (built-in)
2. **go vet** - Official Go static analyzer (built-in)
3. **go test** - Built-in test runner with race detection support

### External Linters

All optional for basic development but enforced in CI:

1. **golangci-lint**
   - **Version**: v2.1.6 (CI), not pinned locally
   - **Config**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/.golangci.yml`
   - **Enabled Linters**: errcheck, govet, ineffassign, staticcheck, unused, bodyclose, durationcheck, nilerr, noctx, misspell, cyclop, gocognit, nakedret, errorlint, prealloc, gosec
   - **Timeout**: 5 minutes
   - **Complexity Thresholds**:
     - Max cyclomatic complexity: 35
     - Max cognitive complexity: 50
     - Naked returns: max 30 lines

2. **staticcheck**
   - **Version**: Latest (via `go install honnef.co/go/tools/cmd/staticcheck@latest`)
   - **Checks**: All (`all`)
   - **Purpose**: Advanced Go static analysis

3. **goimports**
   - **Version**: Latest (via `go install golang.org/x/tools/cmd/goimports@latest`)
   - **Purpose**: Auto-format imports, enforce local import prefix `github.com/lgbarn/pgn-extract-go`

4. **shellcheck**
   - **Version**: v0.10.0.1 (via shellcheck-py)
   - **Purpose**: Shell script linting for `.sh` files in `/scripts`

### Security Scanning

1. **gosec** (Go Security Checker)
   - **Integration**: golangci-lint + GitHub Actions
   - **Output Format**: SARIF (uploaded to GitHub Security)
   - **Enabled Checks**: G101-G110, G201-G203, G204, G301-G306, G401-G404, G501-G505

2. **Trivy** (Vulnerability Scanner)
   - **Version**: Latest (via `aquasecurity/trivy-action@master`)
   - **Scan Type**: Filesystem (`fs`)
   - **Severity**: CRITICAL, HIGH
   - **CI Mode**: Non-blocking (`exit-code: 0`)

## Development Workflow Tools

### Pre-commit Framework

**Config File**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/.pre-commit-config.yaml`
**Version**: Minimum 2.20.0
**Installation**: Optional but recommended

**Hooks Enabled:**
- General file checks (no large files, no merge conflicts, trailing whitespace, YAML/JSON validation)
- Go formatting (`go fmt`)
- Go vetting (`go vet`)
- `go.mod` tidy check
- Static analysis (staticcheck, golangci-lint)
- Build verification (`go build`)
- Quick tests (`go test -short`)
- Shell script linting (shellcheck)
- Prevent commits to main/master branches

**Alternative Setup:**
Native git hooks available in `/Users/lgbarn/Personal/Chess/pgn-extract-go/scripts/`:
- `setup-hooks.sh` - Interactive setup script
- `pre-commit` - Native pre-commit hook
- `pre-push` - Native pre-push hook

### CI/CD Platform

**Platform**: GitHub Actions
**Config**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/.github/workflows/ci.yml`

**Jobs:**
1. **Pre-commit Checks** - Runs all pre-commit hooks (Ubuntu)
2. **Test with Coverage** - Tests with race detector, generates coverage reports
3. **Build Matrix** - Multi-platform builds (Ubuntu, macOS, Windows) × Go versions (1.21, 1.22, 1.23)
4. **Security Scan** - gosec + Trivy vulnerability scanning
5. **Release** - GoReleaser on version tags

**Environment:**
- Python 3.12 (for pre-commit)
- Go caching enabled
- Concurrency groups (auto-cancel in-progress runs)
- Timeout: 15-20 minutes per job

## Version Management

**Application Version**: Hardcoded in `cmd/pgn-extract/main.go`
```go
const programVersion = "0.1.0"
```

**Version Injection**: GoReleaser overrides this at build time:
```yaml
ldflags:
  - -X main.programVersion={{.Version}}
```

**Go Version Compatibility:**
- Minimum: Go 1.21 (`go.mod`)
- CI/CD: Go 1.23 (GitHub Actions)
- Build matrix tests: Go 1.21, 1.22, 1.23

## Package Structure

All packages are internal (not intended for external use):

```
internal/
├── chess/      - Core types (Board, Game, Move, Piece)
├── config/     - Configuration management
├── cql/        - Chess Query Language parser/evaluator
├── eco/        - ECO classification system
├── engine/     - Move validation, FEN parsing, Chess960 support
├── errors/     - Custom error types
├── hashing/    - Zobrist hashing, duplicate detection
├── matching/   - Game filtering, position matching, Soundex
├── output/     - PGN/JSON/EPD/FEN output formatters
├── parser/     - PGN lexer and parser
├── processing/ - Game analysis utilities
├── testutil/   - Test helpers
└── worker/     - Parallel processing worker pool
```

**Main Package**: `cmd/pgn-extract/` - CLI entry point and flag handling

## Documentation Tools

### Markdown

**Linter**: markdownlint
**Config**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/.markdownlint.json`

**Rules:**
- Default rules enabled
- Line length (MD013): disabled
- Inline HTML (MD033): allowed
- First line H1 (MD041): not required
- Duplicate headings (MD024): allowed if not siblings

**Documentation Files:**
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/README.md` - Main project documentation
- `/Users/lgbarn/Personal/Chess/pgn-extract-go/docs/CQL.md` - Chess Query Language spec

## Testing Framework

**Framework**: Go built-in `testing` package
**No external test dependencies**

**Test Types:**
- Unit tests: `*_test.go` files (co-located with source)
- Benchmark tests: `func Benchmark*` in `benchmark_test.go`
- Golden tests: Reference output comparison in `cmd/pgn-extract/golden_test.go`
- Table-driven tests: Standard Go pattern used throughout

**Test Data**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/testdata/` - PGN test files and golden outputs

**Coverage Tooling:**
- `go test -coverprofile=coverage.out`
- `go tool cover -html=coverage.out`
- Coverage tracked per-package during development

## Build Artifacts

**Binary Name**: `pgn-extract`
**Output Formats**:
- Native executable (per platform)
- Release archives: `pgn-extract_<version>_<os>_<arch>.tar.gz` (or `.zip` for Windows)

**Build Modes:**
- Development: `go build -o pgn-extract ./cmd/pgn-extract/`
- Release: `go build -ldflags="-s -w" -o pgn-extract ./cmd/pgn-extract/`
- Cross-compile: Handled by GoReleaser

## Development Environment

**Recommended Setup:**
- Go 1.21+ installed
- `just` command runner (optional)
- `golangci-lint`, `staticcheck`, `goimports` (optional, installed via scripts)
- Pre-commit framework (optional, Python-based)

**Setup Script**: `/Users/lgbarn/Personal/Chess/pgn-extract-go/scripts/setup-hooks.sh`

**No IDE-specific configuration** - Works with any Go-compatible editor (VS Code, GoLand, Vim, etc.)
