package ai

import (
	"context"
	"os"
	"time"
)

// Advisor is the single entry-point for all AI-powered features in gitflowtui.
// It is safe for concurrent use. Every method checks Available() first and
// returns ErrNotAvailable immediately if no backend is configured, so callers
// do not need to gate on Available() themselves unless they want to hide
// AI-specific UI elements.
//
// Results are cached by content hash with a 30-minute TTL so rapid re-opens
// of the same diff or branch never re-call the backend.
type Advisor struct {
	client  completionClient
	cache   *cache
	Backend string
}

// New creates an Advisor. It prefers Anthropic when ANTHROPIC_API_KEY is set.
// Otherwise it auto-detects a local Ollama instance with a fast reachability
// check before falling back to a disabled advisor.
func New(apiKey string) *Advisor {
	return NewWithOptions(apiKey, "")
}

// NewWithOptions creates an Advisor with explicit Ollama model override.
// Pass an empty ollamaModel to use the GITFLOW_TUI_OLLAMA_MODEL env var or the
// default ("llama3").
func NewWithOptions(apiKey, ollamaModel string) *Advisor {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	advisor := &Advisor{
		cache:   newCache(100, 30*time.Minute),
		Backend: "none",
	}

	if client := newAnthropicClient(apiKey, DefaultModel); client != nil {
		advisor.client = client
		advisor.Backend = "anthropic"
		return advisor
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if ollamaReachable(ctx) {
		advisor.client = newOllamaClient(ollamaModel)
		advisor.Backend = "ollama"
	}

	return advisor
}

// Available reports whether the Advisor has a configured backend.
func (a *Advisor) Available() bool {
	return a.client != nil
}

// FlushCache invalidates all cached AI responses.
// Useful after a large rebase or force-push where cached conflict/diff analysis would be stale.
func (a *Advisor) FlushCache() {
	a.cache.flush()
}
