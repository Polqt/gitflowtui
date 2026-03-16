package ai

import (
	"os"
	"time"
)

// Advisor is the single entry-point for all AI-powered features in gitflowtui.
// It is safe for concurrent use. Every method checks Available() first and
// returns ErrNotAvailable immediately if no API key is present, so callers
// do not need to gate on Available() themselves unless they want to hide
// AI-specific UI elements.
// Results are cached by content hash with a 30-minute TTL so rapid re-opens
// of the same diff or branch never re-call the API.
type Advisor struct {
	client *client
	cache  *cache
}

// New creates an Advisor. It checks apiKey first, then the ANTHROPIC_API_KEY
// environment variable. If neither is set, Available() returns false and all
// methods return ErrNotAvailable without making any network calls.
//
// This means users who never set ANTHROPIC_API_KEY get a fully functional
// tool with AI panels simply grayed out — there is no setup step required
// to use the non-AI features.
func New(apiKey string) *Advisor {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	return &Advisor{
		client: newClient(apiKey, DefaultModel),
		cache:  newCache(100, 30*time.Minute),
	}
}

// Available reports whether the Advisor has a valid API key configured.
// TUI panels should call this once on startup to decide whether to render
// AI-powered sections or show a "Set ANTHROPIC_API_KEY to enable AI" hint.
func (a *Advisor) Available() bool {
	return a.client != nil
}

// FlushCache invalidates all cached AI responses. 
// Useful after a large rebase or force-push where cached conflict/diff analysis would be stale.
func (a *Advisor) FlushCache() {
	a.cache.flush()
}
