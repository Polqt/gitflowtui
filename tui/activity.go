package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderActivityLog(width, height int) string {
	if len(a.activityLog) == 0 {
		return emptyState("No activity yet", "actions will appear here")
	}

	iconColors := map[string]lipgloss.Color{
		"↑": accentCyan, "↓": accentMagenta, "⟳": accentCyan,
		"⎇": accentCyan, "●": accentGreen, "+": accentGreen,
		"-": accentOrange, "⊡": accentYellow, "✕": accentRed,
		"◆": accentCyan, "✗": accentRed, "■": textSecondary,
	}

	lines := make([]string, 0, min(height, len(a.activityLog)))
	start := len(a.activityLog) - height
	if start < 0 {
		start = 0
	}
	for i := len(a.activityLog) - 1; i >= start; i-- {
		entry := a.activityLog[i]
		iconColor := textSecondary
		if c, ok := iconColors[entry.icon]; ok {
			iconColor = c
		}
		icon := lipgloss.NewStyle().Foreground(iconColor).Bold(true).Render(entry.icon)
		ts   := lipgloss.NewStyle().Foreground(textDim).Render(entry.timestamp.Format("15:04"))
		txtS := lipgloss.NewStyle().Foreground(textPrimary)
		if entry.isError {
			txtS = lipgloss.NewStyle().Foreground(accentRed)
		}
		text := txtS.Render(truncateString(entry.text, max(1, width-10)))
		lines = append(lines, ts+"  "+icon+"  "+text)
		if len(lines) >= height {
			break
		}
	}

	body := strings.Join(lines, "\n")
	if len(a.activityLog) > height {
		body += "\n" + lipgloss.NewStyle().Foreground(textDim).Render("View all activity...")
	}
	return body
}

// renderActivitySection renders the ACTIVITY panel with "~~ LIVE" right in the title.
func (a *App) renderActivitySection(content string, outerW, outerH int) string {
	innerW := max(1, outerW-2)
	innerH := max(1, outerH-2)

	titleTxt := lipgloss.NewStyle().Foreground(textSecondary).Bold(true).Render("~~ ACTIVITY")
	live     := lipgloss.NewStyle().Foreground(accentGreen).Bold(true).Render("+ LIVE")
	gap      := max(1, innerW-lipgloss.Width(titleTxt)-lipgloss.Width(live))
	titleRow := titleTxt + strings.Repeat(" ", gap) + live

	bodyH := max(1, innerH-1)
	body  := renderPanelBody(content, bodyH)

	return a.styles.panel.Width(innerW).Height(innerH).Render(titleRow + "\n" + body)
}
