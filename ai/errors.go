package ai

import "errors"

// ErrNotAvailable is returned when ANTHROPIC_API_KEY is not set.
// The TUI uses this to hide AI panels rather than show error messages.
var ErrNotAvailable = errors.New("AI advisor unavailable: set ANTHROPIC_API_KEY")

// ErrEmptyInput is returned when a required diff or branch name is blank.
var ErrEmptyInput = errors.New("input cannot be empty")

// ErrAPIFailure wraps upstream Anthropic API errors.
// Callers can check errors.As(err, &APIError{}) to inspect the type.
type ErrAPIFailure struct {
	Type    string
	Message string
}

func (e *ErrAPIFailure) Error() string {
	return "anthropic API error (" + e.Type + "): " + e.Message
}
