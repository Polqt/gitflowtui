package tui

import "github.com/charmbracelet/lipgloss"

// Palette — all colours defined once, referenced everywhere.
const (
	colorGold    = lipgloss.Color("220") // main / HEAD
	colorCyan    = lipgloss.Color("81")  // develop / focused border
	colorBlue    = lipgloss.Color("39")  // feature
	colorGreen   = lipgloss.Color("42")  // release / staged / success
	colorRed     = lipgloss.Color("196") // hotfix / error / deleted
	colorOrange  = lipgloss.Color("208") // warning / ahead
	colorPurple  = lipgloss.Color("135") // AI indicator
	colorPink    = lipgloss.Color("205") // spinner / loading
	colorMuted   = lipgloss.Color("244") // secondary text
	colorDim     = lipgloss.Color("238") // inactive borders
	colorSubtle  = lipgloss.Color("240") // very faint backgrounds
	colorFg      = lipgloss.Color("252") // primary text
	colorFgBold  = lipgloss.Color("255") // emphatic text
	colorBg      = lipgloss.Color("235") // title bar background
	colorBgModal = lipgloss.Color("236") // modal background
)

type uiStyles struct {
	root              lipgloss.Style
	title             lipgloss.Style
	panel             lipgloss.Style
	panelFocused      lipgloss.Style
	panelTitle        lipgloss.Style
	panelTitleFocused lipgloss.Style
	statusNormal      lipgloss.Style
	statusError       lipgloss.Style
	help              lipgloss.Style
	loading           lipgloss.Style
	badge             badgeStyles
	diff              diffStyles
	modalBackdrop     lipgloss.Style
}

type badgeStyles struct {
	main      lipgloss.Style
	develop   lipgloss.Style
	feature   lipgloss.Style
	release   lipgloss.Style
	hotfix    lipgloss.Style
	unknown   lipgloss.Style
	staged    lipgloss.Style
	unstaged  lipgloss.Style
	untracked lipgloss.Style
	both      lipgloss.Style
	ahead     lipgloss.Style
	behind    lipgloss.Style
	ai        lipgloss.Style
}

type diffStyles struct {
	added   lipgloss.Style
	removed lipgloss.Style
	hunk    lipgloss.Style
	meta    lipgloss.Style
}

func defaultStyles() uiStyles {
	badge := func(fg, bg lipgloss.Color) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(colorBg).
			Background(bg).
			Bold(true).
			Padding(0, 1)
	}
	_ = badge // used below

	return uiStyles{
		root: lipgloss.NewStyle().Padding(0, 1),

		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorFgBold).
			Background(colorBg).
			Padding(0, 1),

		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDim),

		panelFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorCyan),

		panelTitle: lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(false),

		panelTitleFocused: lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true),

		statusNormal: lipgloss.NewStyle().
			Foreground(colorFg),

		statusError: lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true),

		help: lipgloss.NewStyle().
			Foreground(colorMuted),

		loading: lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true),

		modalBackdrop: lipgloss.NewStyle().
			Background(colorBgModal).
			Foreground(colorFgBold),

		badge: badgeStyles{
			main:      lipgloss.NewStyle().Foreground(colorGold).Bold(true),
			develop:   lipgloss.NewStyle().Foreground(colorCyan).Bold(true),
			feature:   lipgloss.NewStyle().Foreground(colorBlue).Bold(true),
			release:   lipgloss.NewStyle().Foreground(colorGreen).Bold(true),
			hotfix:    lipgloss.NewStyle().Foreground(colorRed).Bold(true),
			unknown:   lipgloss.NewStyle().Foreground(colorMuted),
			staged:    lipgloss.NewStyle().Foreground(colorGreen).Bold(true),
			unstaged:  lipgloss.NewStyle().Foreground(colorOrange).Bold(true),
			untracked: lipgloss.NewStyle().Foreground(colorMuted),
			both:      lipgloss.NewStyle().Foreground(colorOrange).Bold(true),
			ahead:     lipgloss.NewStyle().Foreground(colorGreen),
			behind:    lipgloss.NewStyle().Foreground(colorOrange),
			ai:        lipgloss.NewStyle().Foreground(colorPurple).Bold(true),
		},

		diff: diffStyles{
			added:   lipgloss.NewStyle().Foreground(colorGreen),
			removed: lipgloss.NewStyle().Foreground(colorRed),
			hunk:    lipgloss.NewStyle().Foreground(colorCyan),
			meta:    lipgloss.NewStyle().Bold(true).Foreground(colorFgBold),
		},
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
