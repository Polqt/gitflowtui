package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Polqt/gitflowtui/git"
)

// RiskLevel represents the severity of a conflict.
type RiskLevel int

const (
	RiskClean RiskLevel = iota // git can fast-forward or auto-merge without conflicts
	RiskLow
	RiskMedium
	RiskHigh
	RiskCritical
)

// Label returns a display string suitable for the risk level.
func (r RiskLevel) Label() string {
	switch r {
	case RiskClean:
		return "Clean"
	case RiskLow:
		return "Low"
	case RiskMedium:
		return "Medium"
	case RiskHigh:
		return "High"
	case RiskCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// ANSIColor returns the 256-color ANSI code for lipgloss.
func (r RiskLevel) ANSIColor() string {
	switch r {
	case RiskClean:
		return "42" // Green
	case RiskLow:
		return "226" // Yellow
	case RiskMedium:
		return "208" // Orange
	case RiskHigh:
		return "196" // Red
	case RiskCritical:
		return "197"
	default:
		return "250"
	}
}

// ConflictFile is a single predicted conflict within a file.
type ConflictFile struct {
	Path      string // The file path relative to the repo root
	HunkCount int    // The number of hunks in this file that are predicted to conflict

	// ConflictContent is the raw conflicted file content from merge-tree
	// (contains <<<<<<<, =======, >>>>>>> markers). May be truncated.
	ConflictContent string
	Explanation     string
	Severity        RiskLevel
}

type MergeRisk struct {
	SourceBranch string // The branch we're merging from (e.g. feature/foo)
	TargetBranch string // The branch we're merging into (e.g. main)
	Level        RiskLevel
	// AutoResolvable reports whether git can auto-resolve the merge.
	AutoResolvable bool
	CommitsAhead   int // Number of commits in source branch that are not in target branch
	FilesChanged   int // Total number of files changed in the merge (including non-conflicting files)
	ConflictFiles  []ConflictFile
	Summary        string // A human-friendly summary of the merge risk, suitable for display in the UI
	Recommendation string // A human-friendly recommendation for how to proceed, suitable for display in the UI
	MergeBase      string // The common ancestor commit hash of the source and target branches
}

// PredictMergeRisk performs a dry-run merge and enriches conflict data with
// AI semantic explanations. Never touches the working tree or index.
func (a *Advisor) PredictMergeRisk(ctx context.Context, repo *git.Repo, source, target string) (*MergeRisk, error) {
	dryRun, err := repo.DryRunMerge(ctx, source, target)
	if err != nil {
		return nil, fmt.Errorf("predict merge risk: %w", err)
	}

	risk := &MergeRisk{
		SourceBranch:   source,
		TargetBranch:   target,
		AutoResolvable: !dryRun.HasConflicts,
		CommitsAhead:   dryRun.CommitsAhead,
		FilesChanged:   dryRun.FilesChanged,
		MergeBase:      dryRun.MergeBase,
	}

	if !dryRun.HasConflicts {
		risk.Level = RiskClean
		risk.Summary = fmt.Sprintf(
			"Clean merge predicted. %s is %d commit(s) ahead of %s with no conflicts across %d changed file(s).",
			source, dryRun.CommitsAhead, target, dryRun.FilesChanged,
		)
		risk.Recommendation = "Safe to merge. Run `git merge --no-ff` to preserve branch topology."
		return risk, nil
	}

	for _, conflict := range dryRun.ConflictFiles {
		risk.ConflictFiles = append(risk.ConflictFiles, ConflictFile{
			Path:            conflict.Path,
			HunkCount:       conflict.HunkCount,
			ConflictContent: conflict.ConflictContent,
		})
	}
	risk.Level = structuralRisk(risk.ConflictFiles)

	if !a.Available() {
		risk.Summary = fmt.Sprintf(
			"%d file(s) will conflict (%d total hunk(s)). Install Ollama or set ANTHROPIC_API_KEY for semantic analysis.",
			len(risk.ConflictFiles), totalHunks(risk.ConflictFiles),
		)
		risk.Recommendation = "Resolve conflicts starting with the highest-hunk-count files."
		return risk, nil
	}

	conflictDigest := buildConflictDigest(risk.ConflictFiles)
	cacheKey := a.cache.key("conflict_v2", source, target, dryRun.MergeBase, conflictDigest)

	if cached, ok := a.cache.get(cacheKey); ok {
		if aiResult, err := parseConflictAIResponse(cached); err == nil {
			return applyAIToRisk(risk, aiResult), nil
		}
	}

	promptSet := loadPrompts()
	userPrompt, ok := func() (string, bool) {
		rendered, err := render(promptSet.ConflictUser, ConflictPromptData{
			SourceBranch:   source,
			CommitsAhead:   dryRun.CommitsAhead,
			TargetBranch:   target,
			FilesChanged:   dryRun.FilesChanged,
			ConflictCount:  len(risk.ConflictFiles),
			ConflictDigest: conflictDigest,
		})
		if err != nil {
			return "", false
		}
		return rendered, true
	}()
	if !ok {
		return risk, nil
	}

	raw, completeErr := a.client.complete(ctx, promptSet.ConflictSystem, userPrompt, 1500)
	if completeErr != nil {
		risk.Summary = fmt.Sprintf("%d file(s) will conflict. AI analysis failed: %v", len(risk.ConflictFiles), completeErr)
		risk.Recommendation = "Review each conflicting file before merging."
		return risk, nil
	}

	raw = stripJSONFences(raw)
	aiResult, ok := func() (*conflictAIResponse, bool) {
		result, err := parseConflictAIResponse(raw)
		if err != nil {
			return nil, false
		}
		return result, true
	}()
	if !ok {
		risk.Summary = fmt.Sprintf("%d file(s) will conflict.", len(risk.ConflictFiles))
		return risk, nil
	}

	a.cache.set(cacheKey, raw)
	return applyAIToRisk(risk, aiResult), nil
}

// conflictAIResponse matches the prompt JSON contract.
type conflictAIResponse struct {
	Summary        string `json:"summary"`
	Recommendation string `json:"recommendation"`
	Level          int    `json:"level"`
	//nolint:tagliatelle // prompt contract uses snake_case keys.
	FileExplanations map[string]struct {
		Explanation string `json:"explanation"`
		Severity    int    `json:"severity"`
	} `json:"file_explanations"`
}

func parseConflictAIResponse(raw string) (*conflictAIResponse, error) {
	var result conflictAIResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("parse conflicts AI response: %w", err)
	}
	return &result, nil
}

func applyAIToRisk(risk *MergeRisk, ai *conflictAIResponse) *MergeRisk {
	risk.Summary = ai.Summary
	risk.Recommendation = ai.Recommendation
	if ai.Level >= 0 && ai.Level <= 4 {
		risk.Level = RiskLevel(ai.Level)
	}
	for i, cf := range risk.ConflictFiles {
		if exp, ok := ai.FileExplanations[cf.Path]; ok {
			risk.ConflictFiles[i].Explanation = exp.Explanation
			if exp.Severity >= 0 && exp.Severity <= 4 {
				risk.ConflictFiles[i].Severity = RiskLevel(exp.Severity)
			}
		}
	}
	return risk
}

// buildConflictDigest formats conflict context for the AI prompt.
func buildConflictDigest(files []ConflictFile) string {
	const maxContentPerFile = 600
	var sb strings.Builder
	for _, f := range files {
		_, _ = fmt.Fprintf(&sb, "FILE: %s (%d hunk(s))\n", f.Path, f.HunkCount)
		context := f.ConflictContent
		if len(context) > maxContentPerFile {
			context = context[:maxContentPerFile] + "\n... (truncated)"
		}
		if context != "" {
			sb.WriteString(context)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func structuralRisk(files []ConflictFile) RiskLevel {
	if len(files) == 0 {
		return RiskClean
	}

	hunks := totalHunks(files)
	switch {
	case len(files) >= 6 || hunks >= 25:
		return RiskCritical
	case len(files) >= 4 || hunks >= 12:
		return RiskHigh
	case len(files) >= 2 || hunks >= 5:
		return RiskMedium
	default:
		return RiskLow
	}
}

func totalHunks(files []ConflictFile) int {
	n := 0
	for _, f := range files {
		n += f.HunkCount
	}
	return n
}
