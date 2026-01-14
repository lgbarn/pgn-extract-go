// Package errors provides sentinel errors and error types for the pgn-extract tool.
// It defines common error conditions and structured error types that preserve
// context while allowing error inspection with errors.Is() and errors.As().
package errors

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for common failure conditions.
// Use these with errors.Is() to check for specific error types.
var (
	// ErrInvalidFEN indicates a malformed FEN string.
	ErrInvalidFEN = errors.New("invalid FEN string")

	// ErrIllegalMove indicates a move that violates chess rules.
	ErrIllegalMove = errors.New("illegal move")

	// ErrParseFailure indicates a general PGN parsing error.
	ErrParseFailure = errors.New("parse failure")

	// ErrCQLSyntax indicates a Chess Query Language syntax error.
	ErrCQLSyntax = errors.New("CQL syntax error")

	// ErrInvalidConfig indicates invalid configuration values.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrDuplicateGame indicates a duplicate game was detected.
	ErrDuplicateGame = errors.New("duplicate game")

	// ErrMissingTag indicates a required PGN tag is missing.
	ErrMissingTag = errors.New("missing required tag")

	// ErrMaterialMismatch indicates material pattern doesn't match.
	ErrMaterialMismatch = errors.New("material pattern mismatch")
)

// GameError wraps errors with game context, including game number,
// ply position, and move information. It implements the error interface
// and supports unwrapping via errors.Is() and errors.As().
type GameError struct {
	Err      error  // The underlying error
	GameNum  int    // 1-based game number in the file
	PlyNum   int    // Ply number where error occurred (0 if not applicable)
	MoveText string // The move text that caused the error (if applicable)
	File     string // Source file name (if known)
	Line     int    // Line number in source file (if known)
}

// Error returns a formatted error message including all available context.
func (e *GameError) Error() string {
	var parts []string

	// Add file/line context if available
	if e.File != "" {
		if e.Line > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", e.File, e.Line))
		} else {
			parts = append(parts, e.File)
		}
	}

	// Add game number
	parts = append(parts, fmt.Sprintf("game %d", e.GameNum))

	// Add ply number if available
	if e.PlyNum > 0 {
		parts = append(parts, fmt.Sprintf("ply %d", e.PlyNum))
	}

	// Add move text if available
	if e.MoveText != "" {
		parts = append(parts, fmt.Sprintf("move %q", e.MoveText))
	}

	// Build the final message
	context := strings.Join(parts, ", ")

	if e.Err != nil {
		return fmt.Sprintf("%s: %v", context, e.Err)
	}
	return context
}

// Unwrap returns the underlying error, enabling errors.Is() and errors.As()
// to work through the GameError wrapper.
func (e *GameError) Unwrap() error {
	return e.Err
}

// ParseError represents a parsing error with file location context.
// It's used for PGN and CQL parsing errors.
type ParseError struct {
	Err      error  // The underlying error
	File     string // Source file name
	Line     int    // Line number (1-based)
	Column   int    // Column number (1-based)
	Expected string // What was expected (for syntax errors)
	Got      string // What was found instead
}

// Error returns a formatted error message with location and context.
func (e *ParseError) Error() string {
	var parts []string

	// Add file location
	if e.File != "" {
		loc := e.File
		if e.Line > 0 {
			loc += fmt.Sprintf(":%d", e.Line)
			if e.Column > 0 {
				loc += fmt.Sprintf(":%d", e.Column)
			}
		}
		parts = append(parts, loc)
	}

	// Add expected/got context
	if e.Expected != "" && e.Got != "" {
		parts = append(parts, fmt.Sprintf("expected %s, got %s", e.Expected, e.Got))
	} else if e.Expected != "" {
		parts = append(parts, fmt.Sprintf("expected %s", e.Expected))
	} else if e.Got != "" {
		parts = append(parts, fmt.Sprintf("unexpected %s", e.Got))
	}

	// Add underlying error
	if e.Err != nil {
		if len(parts) > 0 {
			return fmt.Sprintf("%s: %v", strings.Join(parts, ": "), e.Err)
		}
		return e.Err.Error()
	}

	if len(parts) > 0 {
		return strings.Join(parts, ": ")
	}
	return "parse error"
}

// Unwrap returns the underlying error.
func (e *ParseError) Unwrap() error {
	return e.Err
}

// Wrap adds context to an error while preserving the underlying error
// for inspection with errors.Is() and errors.As().
func Wrap(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

// Wrapf adds formatted context to an error while preserving the underlying
// error for inspection with errors.Is() and errors.As().
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return Wrap(err, fmt.Sprintf(format, args...))
}
