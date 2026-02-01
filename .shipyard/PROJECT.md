# pgn-extract-go Quality Improvement

## Description

A comprehensive quality improvement pass for pgn-extract-go, a Go CLI tool for parsing, filtering, and converting PGN (chess game notation) files. This project addresses concurrency safety, memory management, test coverage gaps, and code cleanup — improving the tool's reliability and maintainability for production-scale workloads while maintaining full backward compatibility.

## Goals

1. Fix all concurrency safety issues in parallel processing paths
2. Address unbounded memory growth in duplicate detection and large file processing
3. Increase test coverage in under-tested packages (matching, processing)
4. Clean up technical debt (nolint suppressions, global config, Go version)
5. Bump minimum Go version to 1.23

## Non-Goals

- Adding new CLI features or flags
- Changing output formats
- Adding external dependencies
- Rewriting the parser or CQL engine
- Performance optimization beyond fixing memory issues

## Requirements

### Concurrency Safety
- DuplicateDetector must be safe for concurrent access from worker goroutines
- All shared state during parallel processing must be properly synchronized
- `go test -race ./...` must pass cleanly across the entire codebase
- Concurrent code paths must have dedicated test coverage

### Memory Management
- Duplicate detection hash tables must have configurable bounds
- Parser must not accumulate game objects unnecessarily during streaming
- Large file processing must not exhaust memory (bounded resource usage)

### Test Coverage
- Matching package coverage: increase from 34.6% to >70%
- Processing package coverage: increase from 36.1% to >70%
- Add concurrent/parallel test cases with race detector
- All new code must have accompanying tests

### Code Cleanup
- Bump go.mod minimum version to Go 1.23
- Review and reduce 50+ nolint suppressions (fix or document each)
- Refactor global config toward dependency injection where practical
- Adopt Go 1.23 features where they simplify existing code

## Non-Functional Requirements

- No breaking changes to CLI interface, flags, or output formats
- Zero external dependencies policy maintained
- All existing tests must continue to pass
- CI pipeline must remain green throughout

## Success Criteria

1. `go test -race ./...` passes with zero race conditions
2. No unbounded memory growth when processing 100K+ game files
3. Matching package test coverage >70%
4. Processing package test coverage >70%
5. Nolint suppressions reduced by at least 50%
6. go.mod specifies Go 1.23 as minimum version
7. All existing CLI behaviors preserved (backward compatible)

## Constraints

- **Backward compatibility**: All existing CLI flags, behavior, and output formats must remain stable
- **No external dependencies**: Pure Go standard library only
- **Go 1.23**: New minimum version target
