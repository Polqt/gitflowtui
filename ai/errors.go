package ai

import "errors"

// ErrNotAvailable is returned when no AI backend is available.
var ErrNotAvailable = errors.New("AI advisor unavailable: set ANTHROPIC_API_KEY or install Ollama")

// ErrEmptyInput is returned when a required diff or branch name is blank.
var ErrEmptyInput = errors.New("input cannot be empty")

// APIError wraps upstream AI backend errors.
type APIError struct {
	Backend string
	Type    string
	Message string
}

func (e *APIError) Error() string {
	prefix := "AI backend"
	if e.Backend != "" {
		prefix = e.Backend
	}
	return prefix + " error (" + e.Type + "): " + e.Message
}
