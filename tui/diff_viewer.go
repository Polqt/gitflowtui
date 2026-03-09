package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func colorizeDiff(raw string, maxWidth int) string {
	if strings.TrimSpace(raw) == "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("No diff")
	}
	if maxWidth <= 0 {
		maxWidth = 120
	}

	added := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	removed := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	hunk := lipgloss.NewStyle().Foreground(lipgloss.Color("45"))
	meta := lipgloss.NewStyle().Bold(true)

	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		line = truncateString(line, maxWidth)
		switch {
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "diff --git"):
			lines[i] = meta.Render(line)
		case strings.HasPrefix(line, "@@"):
			lines[i] = hunk.Render(line)
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			lines[i] = added.Render(line)
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			lines[i] = removed.Render(line)
		default:
			lines[i] = line
		}
	}

	return strings.Join(lines, "\n")
}

func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen == 1 {
		return "\u2026"
	}
	return string(runes[:maxLen-1]) + "\u2026"
}
