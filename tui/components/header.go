package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// Colors — cyber-minimalism palette matching the design system.
const (
	BgBase        = lipgloss.Color("#050816")
	TextPrimary   = lipgloss.Color("#e2e8f0")
	TextMuted     = lipgloss.Color("#94a3b8")
	TextDim       = lipgloss.Color("#6b7280")
	AccentCyan    = lipgloss.Color("#00d9ff")
	AccentGreen   = lipgloss.Color("#00ff9c")
	AccentOrange  = lipgloss.Color("#ffb84d")
	AccentYellow  = lipgloss.Color("#ffb84d")
	AccentMagenta = lipgloss.Color("#c084fc")
)

// HeaderProps holds everything the header needs to render.
type HeaderProps struct {
	AppName    string
	RepoName   string
	Branch     string
	Ahead      int
	Behind     int
	Loading    bool
	LoadingLbl string
	AIEnabled  bool
	Version    string
	Spinner    spinner.Model
	Width      int
}

// Render returns a single clean header row.
// Layout: >_ gitflowy  |  repo: name  ·  branch: main  ·  synced ●      v1.2.0.
func Render(p HeaderProps) string {
	sep := lipgloss.NewStyle().Foreground(TextDim).Render("  ·  ")
	pipe := lipgloss.NewStyle().Foreground(TextDim).Render("  |  ")
	hpad := 2

	// ── left segments ─────────────────────────────────────────────────────────
	appSeg := lipgloss.NewStyle().Foreground(AccentCyan).Bold(true).Render(">_ " + p.AppName)

	repoSeg := lipgloss.NewStyle().Foreground(TextMuted).Render("repo: ") +
		lipgloss.NewStyle().Foreground(TextPrimary).Render(truncate(p.RepoName, 20))

	branch := p.Branch
	if branch == "" {
		branch = "DETACHED"
	}
	branchSeg := lipgloss.NewStyle().Foreground(TextMuted).Render("branch: ") +
		lipgloss.NewStyle().Foreground(AccentCyan).Bold(true).Render(truncate(branch, 28))

	var syncSeg string
	switch {
	case p.Loading:
		syncSeg = p.Spinner.View() + " " +
			lipgloss.NewStyle().Foreground(AccentMagenta).Render(p.LoadingLbl)
	case p.Behind > 0:
		syncSeg = lipgloss.NewStyle().Foreground(AccentOrange).
			Render(fmt.Sprintf("↓ behind %d", p.Behind))
	case p.Ahead > 0:
		syncSeg = lipgloss.NewStyle().Foreground(AccentYellow).
			Render(fmt.Sprintf("↑ ahead %d", p.Ahead))
	default:
		syncSeg = lipgloss.NewStyle().Foreground(AccentGreen).Render("synced ●")
	}

	// ── right segment ─────────────────────────────────────────────────────────
	var rightParts []string
	if p.AIEnabled {
		rightParts = append(rightParts,
			lipgloss.NewStyle().Foreground(AccentMagenta).Render("✦ AI"))
	}
	ver := p.Version
	if ver == "" {
		ver = "v1.2.0"
	}
	rightParts = append(rightParts,
		lipgloss.NewStyle().Foreground(AccentYellow).Render("⚡ "+ver))
	right := strings.Join(rightParts, "  ")

	// ── adaptive left — drop segments when terminal is narrow ─────────────────
	innerW := maxInt(1, p.Width-hpad*2)
	rightW := lipgloss.Width(right)

	left := appSeg + pipe + repoSeg + sep + branchSeg + sep + syncSeg
	if lipgloss.Width(left)+2+rightW > innerW {
		left = appSeg + pipe + branchSeg + sep + syncSeg
	}
	if lipgloss.Width(left)+2+rightW > innerW {
		left = appSeg + pipe + branchSeg
	}
	if lipgloss.Width(left)+2+rightW > innerW {
		left = appSeg
	}

	leftW := lipgloss.Width(left)
	gap := innerW - leftW - rightW
	if gap < 1 {
		gap = 1
	}

	// Plain text row — no background rectangle, floats on canvas.
	line := strings.Repeat(" ", hpad) + left + strings.Repeat(" ", gap) + right + strings.Repeat(" ", hpad)
	return line
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
