package ai

import (
	"context"
	"fmt"
	"strings"
)

const maxExplainDiffBytes = 8_000

// ExplainDiff streams a plain-English explanation of a unified diff.
// Tokens arrive on StreamResult.Tokens as they stream from the API.
// Returns (zero StreamResult, ErrNotAvailable) if no API key is set.
// Returns (zero StreamResult, ErrEmptyInput) if diff is blank.
func (a *Advisor) ExplainDiff(ctx context.Context, diff string) (StreamResult, error) {
	if !a.Available() {
		return StreamResult{}, ErrNotAvailable
	}
	diff = strings.TrimSpace(diff)
	if diff == "" {
		return StreamResult{}, ErrEmptyInput
	}
	if len(diff) > maxExplainDiffBytes {
		diff = diff[:maxExplainDiffBytes] + "\n\n... (diff truncated)"
	}

	userPrompt, err := render(prompts.ExplainDiffUser, ExplainPromptData{Diff: diff})
	if err != nil {
		return StreamResult{}, fmt.Errorf("ExplainDiff: %w", err)
	}

	return a.client.stream(ctx, prompts.ExplainDiffSys, userPrompt, 400), nil
}

// ExplainStash streams a plain-English summary of a stash entry's diff.
// Returns (zero StreamResult, ErrNotAvailable) if no API key is set.
func (a *Advisor) ExplainStash(ctx context.Context, stashDiff string) (StreamResult, error) {
	if !a.Available() {
		return StreamResult{}, ErrNotAvailable
	}
	stashDiff = strings.TrimSpace(stashDiff)
	if stashDiff == "" {
		return StreamResult{}, ErrEmptyInput
	}
	if len(stashDiff) > maxExplainDiffBytes {
		stashDiff = stashDiff[:maxExplainDiffBytes] + "\n\n... (truncated)"
	}

	userPrompt, err := render(prompts.ExplainStashUser, ExplainPromptData{Diff: stashDiff})
	if err != nil {
		return StreamResult{}, fmt.Errorf("ExplainStash: %w", err)
	}

	return a.client.stream(ctx, prompts.ExplainStashSys, userPrompt, 250), nil
}
