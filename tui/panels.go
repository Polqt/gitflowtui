package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderSection(name, content string, outerW, outerH int, focused bool) string {
	return a.renderSectionBordered(name, content, outerW, outerH, focused, true)
}

func (a *App) renderSectionBare(name, content string, outerW, outerH int, focused bool) string {
	return a.renderSectionBordered(name, content, outerW, outerH, focused, false)
}

func (a *App) renderSectionBordered(name, content string, outerW, outerH int, focused bool, bordered bool) string {
	var titleStyle lipgloss.Style
	if focused {
		titleStyle = lipgloss.NewStyle().Foreground(accentCyan).Bold(true)
	} else {
		titleStyle = lipgloss.NewStyle().Foreground(textSecondary).Bold(true)
	}
	title := titleStyle.Render(strings.ReplaceAll(name, "_", " "))

	if bordered {
		innerW := max(1, outerW-2)
		innerH := max(1, outerH-2)
		body := renderPanelBody(content, max(1, innerH-1))
		style := a.styles.panel
		if focused {
			style = a.styles.panelFocused
		}
		return style.Width(innerW).Height(innerH).Render(title + "\n" + body)
	}

	// Borderless: title + content, sized to outerW/outerH directly.
	body := renderPanelBody(content, max(1, outerH-1))
	return lipgloss.NewStyle().Width(outerW).Height(outerH).Render(title + "\n" + body)
}

func renderPanelBody(content string, height int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// renderBranchSection renders the BRANCHES panel with a rounded border box,
// Q ▽ icons in the title row, and a branch count line at the bottom.
func (a *App) renderBranchSection(content string, count, outerW, outerH int) string {
	focused := a.activePanel == panelBranches

	// border eats 2 cols and 2 rows
	innerW := max(1, outerW-2)
	innerH := max(1, outerH-2)

	var titleColor lipgloss.Color
	var borderColor lipgloss.Color
	if focused {
		titleColor = accentCyan
		borderColor = accentCyan
	} else {
		titleColor = textSecondary
		borderColor = bgBorder
	}

	titleTxt := lipgloss.NewStyle().Foreground(titleColor).Bold(true).Render("BRANCHES")
	icons    := lipgloss.NewStyle().Foreground(textDim).Render("Q  ▽")
	titleGap := max(1, innerW-lipgloss.Width(titleTxt)-lipgloss.Width(icons))
	titleRow := titleTxt + strings.Repeat(" ", titleGap) + icons

	// Reserve title row + count footer row inside the border.
	bodyH := max(1, innerH-2)
	body  := renderPanelBody(content, bodyH)

	countLine := lipgloss.NewStyle().Foreground(textDim).
		Render("• " + pluralize(count, "branch", "branches"))

	inner := titleRow + "\n" + body + "\n" + countLine

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(innerW).
		Height(innerH).
		Render(inner)
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return strings.Join([]string{itoa(n), plural}, " ")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := make([]byte, 0, 10)
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func emptyState(primary, hint string) string {
	line1 := lipgloss.NewStyle().Foreground(textSecondary).Render(primary)
	line2 := lipgloss.NewStyle().Foreground(textDim).Render(hint)
	return line1 + "\n" + line2
}
