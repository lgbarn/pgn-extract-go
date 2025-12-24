# pgn-extract-go justfile
# Run 'just --list' to see available recipes

# Set shell for all recipes
set shell := ["bash", "-cu"]

# Enable .env file loading
set dotenv-load

# Default recipe - runs when you just type 'just'
default: build

# ─────────────────────────────────────────────────────────────
# Build Commands
# ─────────────────────────────────────────────────────────────

# Build the binary
build:
    GO111MODULE=on go build -o pgn-extract ./cmd/pgn-extract/

# Build optimized release binary
build-release:
    GO111MODULE=on go build -ldflags="-s -w" -o pgn-extract ./cmd/pgn-extract/

# Install to $GOPATH/bin
install:
    GO111MODULE=on go install ./cmd/pgn-extract/

# Remove build artifacts
clean:
    rm -f pgn-extract
    rm -f coverage.out coverage.html
    go clean -cache -testcache

# ─────────────────────────────────────────────────────────────
# Test Commands
# ─────────────────────────────────────────────────────────────

# Run all tests
test:
    GO111MODULE=on go test ./...

# Run tests with verbose output
test-verbose:
    GO111MODULE=on go test -v ./...

# Run tests with race detector
test-race:
    GO111MODULE=on go test -race ./...

# Run tests with coverage report
test-coverage:
    GO111MODULE=on go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report: coverage.html"

# Run only golden tests
test-golden:
    GO111MODULE=on go test -v ./cmd/pgn-extract/ -run Golden

# Run only CQL tests
test-cql:
    GO111MODULE=on go test -v ./internal/cql/...

# Run only parallel/worker tests
test-parallel:
    GO111MODULE=on go test -v -race ./internal/worker/... ./cmd/pgn-extract/ -run Parallel

# Run tests for a specific package (e.g., just test-pkg internal/parser)
test-pkg pkg:
    GO111MODULE=on go test -v ./{{pkg}}/...

# Run a specific test by name (e.g., just test-run TestGolden)
test-run name:
    GO111MODULE=on go test -v ./... -run {{name}}

# ─────────────────────────────────────────────────────────────
# Development Commands
# ─────────────────────────────────────────────────────────────

# Format all Go code
fmt:
    go fmt ./...

# Run go vet linter
lint:
    go vet ./...

# Run staticcheck if available
staticcheck:
    staticcheck ./... || echo "Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"

# Run all checks (fmt, lint, test)
check: fmt lint test

# Run all checks including race detector
check-full: fmt lint test-race

# Watch for changes and rebuild (requires watchexec)
watch:
    watchexec -e go -r -- just build

# Watch and run tests on change
watch-test:
    watchexec -e go -- just test

# ─────────────────────────────────────────────────────────────
# Run Commands
# ─────────────────────────────────────────────────────────────

# Build and run with arguments (e.g., just run -h)
run *args: build
    ./pgn-extract {{args}}

# Run without building (for quick iterations)
run-direct *args:
    GO111MODULE=on go run ./cmd/pgn-extract/ {{args}}

# ─────────────────────────────────────────────────────────────
# Utility Commands
# ─────────────────────────────────────────────────────────────

# Count lines of Go code
loc:
    @find . -name '*.go' -not -path './vendor/*' | xargs wc -l | tail -1

# Count lines by package
loc-detail:
    @echo "=== By Directory ==="
    @find . -name '*.go' -not -path './vendor/*' -print0 | xargs -0 wc -l | sort -n

# Download dependencies
deps:
    go mod download

# Update dependencies
update-deps:
    go get -u ./...
    go mod tidy

# Tidy go.mod
tidy:
    go mod tidy

# Show module dependency graph
deps-graph:
    go mod graph

# ─────────────────────────────────────────────────────────────
# Benchmarking
# ─────────────────────────────────────────────────────────────

# Run all benchmarks
bench:
    GO111MODULE=on go test -bench=. -benchmem ./...

# Run benchmarks for a specific package
bench-pkg pkg:
    GO111MODULE=on go test -bench=. -benchmem ./{{pkg}}/...

# ─────────────────────────────────────────────────────────────
# Help
# ─────────────────────────────────────────────────────────────

# Show all available recipes
help:
    @just --list

# Show recipe details
recipe name:
    @just --show {{name}}
