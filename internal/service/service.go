// @ai-modified 2026-07-02 add shared service errors and validation type
package service

import "errors"

// ErrNotFound mirrors repository.ErrNotFound at the service boundary.
var ErrNotFound = errors.New("not found")

// ValidationError carries per-field messages for form re-rendering.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "validation failed" }

// NewValidation returns an empty ValidationError ready to be filled.
func NewValidation() *ValidationError {
	return &ValidationError{Fields: map[string]string{}}
}

// Any reports whether any field error was recorded.
func (e *ValidationError) Any() bool { return len(e.Fields) > 0 }
