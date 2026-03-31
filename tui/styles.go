package tui

import "github.com/charmbracelet/lipgloss"

// ── Design System: Deep-dark terminal with muted surfaces & accent pops ──────

// Backgrounds.
const (
	bgBase     = lipgloss.Color("#090D12")
	bgSurface  = lipgloss.Color("#0D1117")
	bgElevated = lipgloss.Color("#141B23")
	bgBorder   = lipgloss.Color("#1C2433")
)

// Accent colors.
const (
	accentCyan    = lipgloss.Color("#00D4D4")
	accentMagenta = lipgloss.Color("#C026D3")
	accentGreen   = lipgloss.Color("#22C55E")
	accentRed     = lipgloss.Color("#EF4444")
	accentYellow  = lipgloss.Color("#EAB308")
	accentOrange  = lipgloss.Color("#F97316")
)

// Text.
const (
	textPrimary   = lipgloss.Color("#E2E8F0")
	textSecondary = lipgloss.Color("#64748B")
	textDim       = lipgloss.Color("#334155")
	textAccent    = lipgloss.Color("#00D4D4")
)

// Branch tag pill backgrounds.
const (
	tagFeature = lipgloss.Color("#0E7490")
	tagRelease = lipgloss.Color("#14532D")
	tagHotfix  = lipgloss.Color("#7F1D1D")
	tagMain    = lipgloss.Color("#1E293B")
)

// Branch tag pill foregrounds.
const (
	tagFeatureFg = lipgloss.Color("#67E8F9")
	tagReleaseFg = lipgloss.Color("#86EFAC")
	tagHotfixFg  = lipgloss.Color("#FCA5A5")
	tagMainFg    = lipgloss.Color("#94A3B8")
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
		unknown:   lipgloss.NewStyle().Foreground(textSecondary),
		staged:    lipgloss.NewStyle().Foreground(accentGreen).Bold(true),
		unstaged:  lipgloss.NewStyle().Foreground(accentOrange).Bold(true),
		untracked: lipgloss.NewStyle().Foreground(textDim),
		both:      lipgloss.NewStyle().Foreground(accentOrange).Bold(true),
		ahead:     lipgloss.NewStyle().Foreground(accentGreen),
		behind:    lipgloss.NewStyle().Foreground(accentOrange),
		ai:        lipgloss.NewStyle().Foreground(accentMagenta).Bold(true),
		syncOK:    lipgloss.NewStyle().Background(accentGreen).Foreground(bgBase).Bold(true).Padding(0, 1),
		urgent:    lipgloss.NewStyle().Background(accentRed).Foreground(bgBase).Bold(true).Padding(0, 1),
		inProg:    lipgloss.NewStyle().Background(accentCyan).Foreground(bgBase).Bold(true).Padding(0, 1),
		readyPR:   lipgloss.NewStyle().Background(accentMagenta).Foreground(bgBase).Bold(true).Padding(0, 1),
	}
}

func defaultStyles() uiStyles {
	return uiStyles{
		root:         lipgloss.NewStyle().Background(bgBase).Foreground(textPrimary),
		panel:        lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(bgBorder),
		panelFocused: lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(accentCyan),
		sectionTitle: lipgloss.NewStyle().Foreground(textSecondary).Bold(true),
		statusNormal: lipgloss.NewStyle().Foreground(textPrimary),
		statusError:  lipgloss.NewStyle().Foreground(accentRed).Bold(true),
		help:         lipgloss.NewStyle().Foreground(textSecondary),
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
			inactive: lipgloss.NewStyle().Foreground(textSecondary).Padding(0, 1),
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
			box: lipgloss.NewStyle().Background(bgSurface).
				Border(lipgloss.NormalBorder()).
				BorderForeground(bgBorder).Padding(0, 1),
			label: lipgloss.NewStyle().Foreground(textSecondary),
			value: lipgloss.NewStyle().Foreground(textPrimary).Bold(true),
			sub:   lipgloss.NewStyle().Foreground(textSecondary),
		},

		row: rowStyles{
			normal: lipgloss.NewStyle().Foreground(textPrimary).PaddingLeft(1),
			selected: lipgloss.NewStyle().Background(bgElevated).
				Foreground(textPrimary).Bold(true).PaddingLeft(0).
				BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(accentCyan),
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
