package tui

import "github.com/charmbracelet/lipgloss"

type uiStyles struct {
	root          lipgloss.Style
	title         lipgloss.Style
	panel         lipgloss.Style
	panelFocused  lipgloss.Style
	statusNormal  lipgloss.Style
	statusError   lipgloss.Style
	help          lipgloss.Style
	loading       lipgloss.Style
	modalBackdrop lipgloss.Style
}

func defaultStyles() uiStyles {
	return uiStyles{
		root: lipgloss.NewStyle().Padding(0, 1),
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62")).
			Padding(0, 1),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")),
		panelFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("81")),
		statusNormal: lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		statusError:  lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		help:         lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		loading:      lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		modalBackdrop: lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255")),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
