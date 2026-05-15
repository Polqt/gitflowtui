package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderDiffSection(content string, outerW, outerH int) string {
	innerW := max(1, outerW-2)
	innerH := max(1, outerH-2)
	focused := a.activePanel == panelDiff

	var titleColor, borderColor lipgloss.Color
	if focused {
		titleColor  = accentCyan
		borderColor = accentCyan
	} else {
		titleColor  = textSecondary
		borderColor = bgBorder
	}

	titleTxt := lipgloss.NewStyle().Foreground(titleColor).Bold(true).Render("DIFF VIEWER")
	modePill := lipgloss.NewStyle().Foreground(textSecondary).Render("Line ▼")
	pipe     := lipgloss.NewStyle().Foreground(textDim).Render(" | ")

	// Show current file from selected status item.
	fileName := ""
	if item, ok := a.selectedFileItem(); ok {
		fileName = truncateString(item.file.Path, 30)
	}
	fileStr := ""
	if fileName != "" {
		fileStr = lipgloss.NewStyle().Foreground(textDim).Render("File: ") +
			lipgloss.NewStyle().Foreground(textPrimary).Render(fileName) + "  " +
			lipgloss.NewStyle().Foreground(accentRed).Render("→")
	}

	right    := modePill + pipe + fileStr
	gap      := max(1, innerW-lipgloss.Width(titleTxt)-lipgloss.Width(right))
	titleRow := titleTxt + strings.Repeat(" ", gap) + right

	bodyH := max(1, innerH-1)
	body  := renderPanelBody(content, bodyH)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor)

	return style.Width(innerW).Height(innerH).Render(titleRow + "\n" + body)
}
