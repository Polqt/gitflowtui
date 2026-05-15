package tui

import (
	"fmt"
	"strings"

	"github.com/Polqt/gitflowtui/ai"
	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderPrompt() string {
	w := min(max(52, a.width*2/3), a.width-4)
	innerW := max(1, w-4)
	a.prompt.input.Width = innerW - 2

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accentCyan)
	hintStyle := lipgloss.NewStyle().Foreground(textSecondary)
	keyStyle := lipgloss.NewStyle().Foreground(accentCyan)

	keys := keyStyle.Render("[Enter]") + hintStyle.Render(" confirm  ") +
		keyStyle.Render("[Esc]") + hintStyle.Render(" cancel")
	example := promptExample(a.prompt.mode)

	rows := []string{titleStyle.Render(a.prompt.title), "", hintStyle.Render(a.prompt.hint)}
	if example != "" {
		rows = append(rows, hintStyle.Render(example))
	}
	rows = append(rows, "", a.prompt.input.View(), "", keys)

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(accentCyan).
		Background(bgElevated).
		Width(w - 2).
		Render(strings.Join(rows, "\n"))
}

func promptExample(mode promptMode) string {
	switch mode {
	case promptNone:
		return ""
	case promptStash:
		return ""
	case promptFeatureName:
		return "e.g. login-redesign  ->  feature/login-redesign"
	case promptReleaseVersion:
		return "e.g. 1.2.0  ->  release/1.2.0"
	case promptHotfixVersion:
		return "e.g. 1.2.1  ->  hotfix/1.2.1"
	case promptCommit:
		return "e.g. feat(auth): add JWT refresh"
	case promptDeleteBranch:
		return "Type the branch name exactly to confirm"
	}
	return ""
}

func (a *App) renderAIOverlay() string {
	w := min(max(70, a.width*3/4), a.width-4)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accentMagenta)
	hintStyle := lipgloss.NewStyle().Foreground(textSecondary)

	content := strings.Join([]string{
		titleStyle.Render("✦ " + a.aiView.title),
		"",
		a.aiView.content,
		"",
		hintStyle.Render("[Esc] close"),
	}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(accentMagenta).
		Background(bgElevated).
		Width(w - 2).
		Render(content)
}

func renderBranchHealth(report *ai.BranchHealthReport) string {
	var lines []string
	lines = append(lines, report.Summary)
	if len(report.StaleBranches) > 0 {
		lines = append(lines, "", "Stale branches:")
		for _, b := range report.StaleBranches {
			lines = append(lines, "  - "+b)
		}
	}
	if len(report.RiskyBranches) > 0 {
		lines = append(lines, "", "Risky branches:")
		for _, b := range report.RiskyBranches {
			lines = append(lines, "  - "+b)
		}
	}
	if len(report.Recommendations) > 0 {
		lines = append(lines, "", "Recommendations:")
		for _, rec := range report.Recommendations {
			priority := rec.PriorityLabel()
			lines = append(lines, fmt.Sprintf("  [%s] %s: %s — %s", priority, rec.Branch, rec.Action, rec.Reason))
		}
	}
	return strings.Join(lines, "\n")
}

func renderMergeRisk(risk *ai.MergeRisk) string {
	var lines []string
	lines = append(lines, risk.Summary)
	if strings.TrimSpace(risk.Recommendation) != "" {
		lines = append(lines, "", "Recommendation:", risk.Recommendation)
	}
	if len(risk.ConflictFiles) == 0 {
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "", "Conflict files:")
	for _, file := range risk.ConflictFiles {
		line := "- " + file.Path
		if file.Explanation != "" {
			line += ": " + file.Explanation
		} else if file.HunkCount > 0 {
			line += fmt.Sprintf(": %d predicted conflict hunk(s)", file.HunkCount)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
