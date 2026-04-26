// Package yaml implements a subset YAML parser for scrutineer test files.
// It supports block mappings, block sequences, flow collections, multi-line
// strings, comments, and scalar types (string, int, float, bool, null).
package yaml

import "fmt"

// ParseError represents an error that occurred during YAML parsing,
// including the location in the input where the error was detected.
type ParseError struct {
	Line    int
	Column  int
	Message string
}

// Error returns a human-friendly error message including line and column.
func (e *ParseError) Error() string {
	return fmt.Sprintf("yaml: line %d, column %d: %s", e.Line, e.Column, e.Message)
}

// newParseError creates a new ParseError with the given location and message.
func newParseError(line, col int, msg string) *ParseError {
	return &ParseError{Line: line, Column: col, Message: msg}
}

// newParseErrorf creates a new ParseError with a formatted message.
func newParseErrorf(line, col int, format string, args ...any) *ParseError {
	return &ParseError{Line: line, Column: col, Message: fmt.Sprintf(format, args...)}
}
