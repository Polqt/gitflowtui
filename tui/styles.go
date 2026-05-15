package tui

import "github.com/charmbracelet/lipgloss"

const (
	bgBase     = lipgloss.Color("#050816")
	bgSurface  = lipgloss.Color("#08111f")
	bgElevated = lipgloss.Color("#0b1627")
	bgBorder   = lipgloss.Color("#12324a")
)

// Accent colors.
const (
	accentCyan    = lipgloss.Color("#00d9ff")
	accentMagenta = lipgloss.Color("#c084fc")
	accentGreen   = lipgloss.Color("#00ff9c")
	accentRed     = lipgloss.Color("#ff6b6b")
	accentYellow  = lipgloss.Color("#ffb84d")
	accentOrange  = lipgloss.Color("#ffb84d")
)

// Text.
const (
	textPrimary   = lipgloss.Color("#e2e8f0")
	textSecondary = lipgloss.Color("#94a3b8")
	textDim       = lipgloss.Color("#6b7280")
	textAccent    = lipgloss.Color("#00d9ff")
)

// Branch tag pill backgrounds.
const (
	tagFeature = lipgloss.Color("#0a2540")
	tagRelease = lipgloss.Color("#062010")
	tagHotfix  = lipgloss.Color("#2d0a0a")
	tagMain    = lipgloss.Color("#0b1627")
)

// Branch tag pill foregrounds.
const (
	tagFeatureFg = lipgloss.Color("#00d9ff")
	tagReleaseFg = lipgloss.Color("#00ff9c")
	tagHotfixFg  = lipgloss.Color("#ff6b6b")
	tagMainFg    = lipgloss.Color("#94a3b8")
)

// ── Style structs ────────────────────────────────────────────────────────────

type uiStyles struct {
	root         lipgloss.Style
	panel        lipgloss.Style
	panelFocused lipgloss.Style
	sectionTitle lipgloss.Style
	statusNormal lipgloss.Style
	statusError  lipgloss.Style
	help         lipgloss.Style
	loading      lipgloss.Style

	badge  badgeStyles
	diff   diffStyles
	navbar navbarStyles
	header headerStyles
	card   cardStyles
	row    rowStyles
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
	sep    lipgloss.Style
}

type cardStyles struct {
	box   lipgloss.Style
	label lipgloss.Style
	value lipgloss.Style
	sub   lipgloss.Style
}

type rowStyles struct {
	normal   lipgloss.Style
	selected lipgloss.Style
}

func defaultBadgeStyles() badgeStyles {
	return badgeStyles{
		main: lipgloss.NewStyle().
			Background(tagMain).Foreground(tagMainFg).Padding(0, 1).Bold(true),
		develop:   lipgloss.NewStyle().Foreground(accentCyan).Bold(true),
		feature:   lipgloss.NewStyle().Background(tagFeature).Foreground(tagFeatureFg).Padding(0, 1).Bold(true),
		release:   lipgloss.NewStyle().Background(tagRelease).Foreground(tagReleaseFg).Padding(0, 1).Bold(true),
		hotfix:    lipgloss.NewStyle().Background(tagHotfix).Foreground(tagHotfixFg).Padding(0, 1).Bold(true),
		unknown:   lipgloss.NewStyle().Foreground(textDim),
		staged:    lipgloss.NewStyle().Foreground(accentGreen).Bold(true),
		unstaged:  lipgloss.NewStyle().Foreground(accentYellow).Bold(true),
		untracked: lipgloss.NewStyle().Foreground(textDim),
		both:      lipgloss.NewStyle().Foreground(accentYellow).Bold(true),
		ahead:     lipgloss.NewStyle().Foreground(accentGreen),
		behind:    lipgloss.NewStyle().Foreground(accentYellow),
		ai:        lipgloss.NewStyle().Foreground(accentMagenta).Bold(true),
		syncOK:    lipgloss.NewStyle().Foreground(accentGreen).Bold(true),
		urgent:    lipgloss.NewStyle().Foreground(accentRed).Bold(true),
		inProg:    lipgloss.NewStyle().Foreground(accentCyan).Bold(true),
		readyPR:   lipgloss.NewStyle().Foreground(accentMagenta).Bold(true),
	}
}

func defaultStyles() uiStyles {
	return uiStyles{
		root:         lipgloss.NewStyle().Background(bgBase).Foreground(textPrimary),
		panel:        lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(bgBorder),
		panelFocused: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentCyan),
		sectionTitle: lipgloss.NewStyle().Foreground(textDim).Bold(true),
		statusNormal: lipgloss.NewStyle().Foreground(textPrimary),
		statusError:  lipgloss.NewStyle().Foreground(accentRed).Bold(true),
		help:         lipgloss.NewStyle().Foreground(textDim),
		loading:      lipgloss.NewStyle().Foreground(accentMagenta).Bold(true),

		badge: defaultBadgeStyles(),

		diff: diffStyles{
			added:   lipgloss.NewStyle().Foreground(accentGreen),
			removed: lipgloss.NewStyle().Foreground(accentRed),
			hunk:    lipgloss.NewStyle().Foreground(accentCyan),
			meta:    lipgloss.NewStyle().Bold(true).Foreground(textPrimary),
		},

		navbar: navbarStyles{
			active:   lipgloss.NewStyle().Foreground(bgBase).Background(accentCyan).Bold(true).Padding(0, 1),
			inactive: lipgloss.NewStyle().Foreground(textDim).Padding(0, 1),
			bar:      lipgloss.NewStyle().Background(bgSurface),
		},

		header: headerStyles{
			bar:    lipgloss.NewStyle().Background(bgSurface).Foreground(textPrimary).Padding(0, 1),
			label:  lipgloss.NewStyle().Foreground(accentCyan).Bold(true),
			branch: lipgloss.NewStyle().Foreground(textPrimary).Bold(true),
			sync:   lipgloss.NewStyle().Foreground(accentGreen).Bold(true),
			sep:    lipgloss.NewStyle().Foreground(textDim),
		},

		card: cardStyles{
			box: lipgloss.NewStyle().Background(bgElevated).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(bgBorder).Padding(0, 1),
			label: lipgloss.NewStyle().Foreground(textDim),
			value: lipgloss.NewStyle().Foreground(textPrimary).Bold(true),
			sub:   lipgloss.NewStyle().Foreground(textDim),
		},

		row: rowStyles{
			normal: lipgloss.NewStyle().Foreground(textSecondary).PaddingLeft(1),
			selected: lipgloss.NewStyle().Background(bgElevated).
				Foreground(accentGreen).Bold(true).PaddingLeft(0).
				BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(accentGreen),
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
