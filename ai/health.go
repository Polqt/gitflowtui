package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Polqt/gitflowtui/git"
)

type BranchHealthReport struct {
	GeneratedAt     time.Time
	Summary         string
	Recommendations []BranchRecommendation
	StaleBranches   []string
	RiskyBranches   []string
}

type BranchRecommendation struct {
	Branch   string `json:"branch"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
	Priority int    `json:"priority"`
}

func (r BranchRecommendation) PriorityLabel() string {
	switch r.Priority {
	case 1:
		return "High"
	case 2:
		return "Medium"
	default:
		return "Low"
	}
}

// AnalyzeBranchHealth generates a prioritized health report for all branches.
// Results are cached for 30 minutes.
func (a *Advisor) AnalyzeBranchHealth(
	ctx context.Context,
	branches []git.Branch,
	head string,
) (*BranchHealthReport, error) {
	if !a.Available() {
		return nil, ErrNotAvailable
	}
	if len(branches) == 0 {
		return &BranchHealthReport{
			GeneratedAt: time.Now(),
			Summary:     "No local branches found.",
		}, nil
	}

	branchLines := formatBranchLines(branches)
	cacheKey := a.cache.key("health_v1", head, branchLines)

	if cached, ok := a.cache.get(cacheKey); ok {
		if report, err := parseHealthResponse(cached); err == nil {
			report.GeneratedAt = time.Now()
			return report, nil
		}
	}

	promptSet := loadPrompts()
	userPrompt, err := render(promptSet.HealthUser, HealthPromptData{
		Today:       time.Now().Format("2006-01-02"),
		Head:        head,
		BranchLines: branchLines,
	})
	if err != nil {
		return nil, fmt.Errorf("AnalyzeBranchHealth: %w", err)
	}

	raw, err := a.client.complete(ctx, promptSet.HealthSystem, userPrompt, 1800)
	if err != nil {
		return nil, fmt.Errorf("AnalyzeBranchHealth: %w", err)
	}

	raw = stripJSONFences(raw)
	report, err := parseHealthResponse(raw)
	if err != nil {
		return nil, fmt.Errorf("AnalyzeBranchHealth: parse response: %w", err)
	}

	a.cache.set(cacheKey, raw)
	report.GeneratedAt = time.Now()
	return report, nil
}

func formatBranchLines(branches []git.Branch) string {
	var sb strings.Builder
	for _, b := range branches {
		head := " "
		if b.IsHead {
			head = "*"
		}
		upstream := b.Upstream
		if upstream == "" {
			upstream = "(no upstream)"
		}
		_, _ = fmt.Fprintf(
			&sb,
			"%s %-45s  %-35s  +%-3d -%-3d\n",
			head,
			b.Name,
			upstream,
			b.Ahead,
			b.Behind,
		)
	}
	return sb.String()
}

func parseHealthResponse(raw string) (*BranchHealthReport, error) {
	var result struct {
		Summary         string                 `json:"summary"`
		Recommendations []BranchRecommendation `json:"recommendations"`
		//nolint:tagliatelle // prompt contract uses snake_case keys.
		StaleBranches []string `json:"stale_branches"`
		//nolint:tagliatelle // prompt contract uses snake_case keys.
		RiskyBranches []string `json:"risky_branches"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("unmarshal health JSON: %w", err)
	}
	return &BranchHealthReport{
		Summary:         result.Summary,
		Recommendations: result.Recommendations,
		StaleBranches:   result.StaleBranches,
		RiskyBranches:   result.RiskyBranches,
	}, nil
}
