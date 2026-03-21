package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type CommitSuggestion struct {
	// Message is the suggested commit message for the merge commit, if applicable.
	// "feat(auth): add JWT refresh token rotation"
	Message  string `json:"message"`
	Type     string `json:"type"`     // Conventional commit type.
	Scope    string `json:"scope"`    // Optional scope, for example "auth".
	Body     string `json:"body"`     // Optional explanation body.
	Breaking bool   `json:"breaking"` // Whether the change is breaking.
	//nolint:tagliatelle // prompt contract uses snake_case keys.
	MixedConcerns bool `json:"mixed_concerns"` // Whether the diff should be split.
	//nolint:tagliatelle // prompt contract uses snake_case keys.
	SplitSuggestions []string `json:"split_suggestions,omitempty"`
}

// maxDiffBytes is the maximum diff length sent to the API.
// Large diffs are truncated to avoid hitting token limits.
// 12 KB covers most realistic staged changes.
const maxDiffBytes = 12_000

func (a *Advisor) SuggestCommit(ctx context.Context, stagedDiff string) (*CommitSuggestion, error) {
	if !a.Available() {
		return nil, ErrNotAvailable
	}
	stagedDiff = strings.TrimSpace(stagedDiff)
	if stagedDiff == "" {
		return nil, ErrEmptyInput
	}
	if len(stagedDiff) > maxDiffBytes {
		stagedDiff = stagedDiff[:maxDiffBytes] + "\n\n... (diff truncated at 12 KB)"
	}

	cacheKey := a.cache.key("commit_v1", stagedDiff)
	if cached, ok := a.cache.get(cacheKey); ok {
		var s CommitSuggestion
		if err := json.Unmarshal([]byte(cached), &s); err == nil {
			return &s, nil
		}
	}

	promptSet := loadPrompts()

	userPrompt, err := render(promptSet.CommitUser, CommitPromptData{Diff: stagedDiff})
	if err != nil {
		return nil, fmt.Errorf("SuggestCommit: %w", err)
	}

	raw, err := a.client.complete(ctx, promptSet.CommitSystem, userPrompt, 600)
	if err != nil {
		return nil, fmt.Errorf("SuggestCommit: %w", err)
	}

	raw = stripJSONFences(raw)
	var suggestion CommitSuggestion
	if err := json.Unmarshal([]byte(raw), &suggestion); err != nil {
		return nil, fmt.Errorf("SuggestCommit: parse JSON: %w (raw: %.200s)", err, raw)
	}

	a.cache.set(cacheKey, raw)
	return &suggestion, nil
}

func stripJSONFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) < 3 {
		return s
	}
	return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
}
