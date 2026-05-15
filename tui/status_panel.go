package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderStatusSection renders FILE STATUS with staged/modified/untracked counts in the title.
func (a *App) renderStatusSection(content string, outerW, outerH int) string {
	innerW := max(1, outerW-2)
	innerH := max(1, outerH-2)
	focused := a.activePanel == panelStatus

	var titleColor, borderColor lipgloss.Color
	if focused {
		titleColor = accentCyan
		borderColor = accentCyan
	} else {
		titleColor = textSecondary
		borderColor = bgBorder
	}

	staged, modified, untracked := 0, 0, 0
	for _, item := range a.status.Items() {
		if fi, ok := item.(fileItem); ok {
			f := fi.file
			switch {
			case f.IsUntracked():
				untracked++
			case f.IsStaged() && f.IsUnstaged():
				staged++
				modified++
			case f.IsStaged():
				staged++
			case f.IsUnstaged():
				modified++
			}
		}
	}

	titleTxt := lipgloss.NewStyle().Foreground(titleColor).Bold(true).Render("FILE STATUS")
	stagedLbl := lipgloss.NewStyle().Foreground(accentGreen).Bold(true).Render(strconv.Itoa(staged))
	modifiedLbl := lipgloss.NewStyle().Foreground(accentYellow).Bold(true).Render(strconv.Itoa(modified))
	untrkLbl := lipgloss.NewStyle().Foreground(textDim).Bold(true).Render(strconv.Itoa(untracked))
	countsStr := stagedLbl + "  " + modifiedLbl + "  " + untrkLbl

	gap := max(1, innerW-lipgloss.Width(titleTxt)-lipgloss.Width(countsStr))
	titleRow := titleTxt + strings.Repeat(" ", gap) + countsStr

	bodyH := max(1, innerH-1)
	body := renderPanelBody(content, bodyH)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(innerW).
		Height(innerH).
		Render(titleRow + "\n" + body)
}
