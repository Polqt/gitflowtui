package tui

import "github.com/charmbracelet/lipgloss"

// Cyberpunk terminal palette — dark bg, cyan/magenta accents, neon indicators.
const (
	colorCyan    = lipgloss.Color("#00e5ff") // focused border, primary accent
	colorMagenta = lipgloss.Color("#ff00ff") // AI, hotfix, accent 2
	colorGreen   = lipgloss.Color("#39ff14") // staged, success, sync OK
	colorRed     = lipgloss.Color("#ff1744") // error, hotfix, deleted
	colorOrange  = lipgloss.Color("#ff9100") // warning, behind, unstaged
	colorYellow  = lipgloss.Color("#ffd600") // main/HEAD, gold
	colorBlue    = lipgloss.Color("#448aff") // feature, hash
	colorPurple  = lipgloss.Color("#b388ff") // AI badge, stash ref

	colorFg       = lipgloss.Color("#e0e0e0") // primary text
	colorFgBold   = lipgloss.Color("#ffffff") // emphatic text
	colorFgDim    = lipgloss.Color("#757575") // very muted
	colorMuted    = lipgloss.Color("#9e9e9e") // secondary text
	colorDim      = lipgloss.Color("#424242") // inactive borders, separators
	colorBg       = lipgloss.Color("#0d1117") // deep bg
	colorBgPanel  = lipgloss.Color("#161b22") // panel bg
	colorBgHeader = lipgloss.Color("#1a1a2e") // header bar bg
	colorBgModal  = lipgloss.Color("#16213e") // modal/overlay bg
	colorBgCard   = lipgloss.Color("#1e293b") // metric card bg
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
	navbar            navbarStyles
	header            headerStyles
	card              cardStyles
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
	syncOK    lipgloss.Style
	urgent    lipgloss.Style
	inProg    lipgloss.Style
	readyPR   lipgloss.Style
}

type diffStyles struct {
	added   lipgloss.Style
	removed lipgloss.Style
	hunk    lipgloss.Style
	meta    lipgloss.Style
}

type navbarStyles struct {
	active   lipgloss.Style
	inactive lipgloss.Style
	bar      lipgloss.Style
}

type headerStyles struct {
	bar    lipgloss.Style
	label  lipgloss.Style
	branch lipgloss.Style
	sync   lipgloss.Style
}

type cardStyles struct {
	box   lipgloss.Style
	label lipgloss.Style
	value lipgloss.Style
}

func defaultStyles() uiStyles {
	return uiStyles{
		root: lipgloss.NewStyle().Padding(0, 0),

		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan).
			Background(colorBgHeader).
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
			Foreground(colorMagenta).
			Bold(true),

		modalBackdrop: lipgloss.NewStyle().
			Background(colorBgModal).
			Foreground(colorFgBold),

		badge: badgeStyles{
			main:      lipgloss.NewStyle().Foreground(colorYellow).Bold(true),
			develop:   lipgloss.NewStyle().Foreground(colorCyan).Bold(true),
			feature:   lipgloss.NewStyle().Foreground(colorBlue).Bold(true),
			release:   lipgloss.NewStyle().Foreground(colorGreen).Bold(true),
			hotfix:    lipgloss.NewStyle().Foreground(colorRed).Bold(true),
			unknown:   lipgloss.NewStyle().Foreground(colorMuted),
			staged:    lipgloss.NewStyle().Foreground(colorGreen).Bold(true),
			unstaged:  lipgloss.NewStyle().Foreground(colorOrange).Bold(true),
			untracked: lipgloss.NewStyle().Foreground(colorFgDim),
			both:      lipgloss.NewStyle().Foreground(colorOrange).Bold(true),
			ahead:     lipgloss.NewStyle().Foreground(colorGreen),
			behind:    lipgloss.NewStyle().Foreground(colorOrange),
			ai:        lipgloss.NewStyle().Foreground(colorPurple).Bold(true),

			syncOK: lipgloss.NewStyle().
				Foreground(colorBg).
				Background(colorGreen).
				Bold(true).
				Padding(0, 1),
			urgent: lipgloss.NewStyle().
				Foreground(colorBg).
				Background(colorRed).
				Bold(true).
				Padding(0, 1),
			inProg: lipgloss.NewStyle().
				Foreground(colorBg).
				Background(colorCyan).
				Bold(true).
				Padding(0, 1),
			readyPR: lipgloss.NewStyle().
				Foreground(colorBg).
				Background(colorMagenta).
				Bold(true).
				Padding(0, 1),
		},

		diff: diffStyles{
			added:   lipgloss.NewStyle().Foreground(colorGreen),
			removed: lipgloss.NewStyle().Foreground(colorRed),
			hunk:    lipgloss.NewStyle().Foreground(colorCyan),
			meta:    lipgloss.NewStyle().Bold(true).Foreground(colorFgBold),
		},

		navbar: navbarStyles{
			active: lipgloss.NewStyle().
				Foreground(colorBg).
				Background(colorCyan).
				Bold(true).
				Padding(0, 1),
			inactive: lipgloss.NewStyle().
				Foreground(colorMuted).
				Background(colorBgHeader).
				Padding(0, 1),
			bar: lipgloss.NewStyle().
				Background(colorBgHeader).
				Padding(0, 1),
		},

		header: headerStyles{
			bar: lipgloss.NewStyle().
				Background(colorBgHeader).
				Foreground(colorFg).
				Padding(0, 1),
			label: lipgloss.NewStyle().
				Foreground(colorCyan).
				Bold(true),
			branch: lipgloss.NewStyle().
				Foreground(colorFgBold).
				Bold(true),
			sync: lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true),
		},

		card: cardStyles{
			box: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorDim).
				Padding(0, 1),
			label: lipgloss.NewStyle().
				Foreground(colorFgDim).
				Bold(false),
			value: lipgloss.NewStyle().
				Foreground(colorFgBold).
				Bold(true),
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
