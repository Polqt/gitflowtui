package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// FooterProps holds everything the footer needs to render.
type FooterProps struct {
	Loading      bool
	LoadingLbl   string
	Notification string
	NotifError   bool
	Width        int
	Spinner      spinner.Model
}

// RenderFooter returns a footer row inside a rounded border box.
// Layout: ● ready / notification        ↑/↓ navigate   enter select   q quit.
func RenderFooter(p FooterProps) string {
	// border takes 1 col each side → content box = Width-2; padding 2 each side
	hpad := 2
	boxW := max(1, p.Width-2)     // lipgloss Width() — border adds the 2 back
	innerW := max(1, boxW-hpad*2) // usable text width inside padding

	// ── status (left) ─────────────────────────────────────────────────────────
	sep := lipgloss.NewStyle().Foreground(TextDim).Render("   ")

	var status string
	switch {
	case p.Loading:
		status = p.Spinner.View() + " " +
			lipgloss.NewStyle().Foreground(AccentMagenta).Bold(true).Render(p.LoadingLbl)
	case p.NotifError:
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b6b")).Bold(true).Render("✕ ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b6b")).Render(truncate(p.Notification, 40))
	case p.Notification != "":
		status = lipgloss.NewStyle().Foreground(AccentGreen).Bold(true).Render("● ") +
			lipgloss.NewStyle().Foreground(TextPrimary).Render(truncate(p.Notification, 40))
	default:
		status = lipgloss.NewStyle().Foreground(AccentGreen).Bold(true).Render("● ") +
			lipgloss.NewStyle().Foreground(TextMuted).Render("ready")
	}

	// ── nav hints (right) — three tiers by width ──────────────────────────────
	key := func(k string) string {
		return lipgloss.NewStyle().Foreground(AccentCyan).Bold(true).Render(k)
	}
	lbl := func(l string) string {
		return lipgloss.NewStyle().Foreground(TextMuted).Render(l)
	}
	hint := func(k, l string) string { return key(k) + " " + lbl(l) }

	fullHints := strings.Join([]string{
		hint("↑/↓", "navigate"),
		hint("enter", "select"),
		hint("esc", "back"),
		hint("?", "help"),
		hint("q", "quit"),
	}, sep)
	medHints := strings.Join([]string{
		hint("↑/↓", "nav"),
		hint("enter", "select"),
		hint("q", "quit"),
	}, sep)
	shortHints := hint("q", "quit")

	statusW := lipgloss.Width(status)

	var hints string
	switch {
	case statusW+4+lipgloss.Width(fullHints) <= innerW:
		hints = fullHints
	case statusW+4+lipgloss.Width(medHints) <= innerW:
		hints = medHints
	default:
		hints = shortHints
	}

	gap := innerW - statusW - lipgloss.Width(hints)
	if gap < 1 {
		gap = 1
	}

	line := status + strings.Repeat(" ", gap) + hints

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(TextDim).
		Padding(0, hpad).
		Width(boxW).
		Render(line)
}
