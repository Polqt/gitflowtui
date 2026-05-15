package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	keyTab   = "tab"
	keyEnter = "enter"
	keyEsc   = "esc"
	keyCtrlA = "ctrl+a"
)

func (a *App) renderCmdPalette(outerW, outerH int) string {
	innerW := max(1, outerW-2)
	innerH := max(1, outerH-2)
	pad := " "

	titleStyle := lipgloss.NewStyle().Foreground(textSecondary).Bold(true)
	lStyle := lipgloss.NewStyle().Foreground(textPrimary)
	dimStyle := lipgloss.NewStyle().Foreground(textDim)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bgBorder).
		Width(innerW).
		Height(innerH)

	// ── help mode: scrollable keybindings ─────────────────────────────────────
	if a.showHelp {
		vpH := max(1, innerH-3) // 1 title + 1 sep + 1 blank
		a.helpVP.Width = innerW
		a.helpVP.Height = vpH

		scrollIndicator := ""
		if !a.helpVP.AtBottom() {
			scrollIndicator = dimStyle.Render(" ▼")
		}
		title := titleStyle.Render("KEYBINDINGS") + scrollIndicator
		sep := dimStyle.Render(strings.Repeat("─", max(1, innerW)))
		content := title + "\n" + sep + "\n\n" + a.helpVP.View()
		return border.Render(content)
	}

	// ── normal command list ────────────────────────────────────────────────────
	bgGreen := lipgloss.Color("#0d2e1a")
	bgCyan := lipgloss.Color("#0a2535")
	bgYellow := lipgloss.Color("#2e1f00")
	bgMagenta := lipgloss.Color("#1e0a2e")
	bgRed := lipgloss.Color("#2e0a0a")
	bgDim := lipgloss.Color("#1a1a1a")

	keyPill := func(k string, fg, bg lipgloss.Color) string {
		return lipgloss.NewStyle().
			Foreground(fg).
			Background(bg).
			Bold(true).
			Padding(0, 1).
			Render(k)
	}

	keyRow := func(k string, fg, bg lipgloss.Color, label string) string {
		pill := keyPill(k, fg, bg)
		lr := lStyle.Render(label)
		used := len(pad) + lipgloss.Width(pill) + 1 + lipgloss.Width(lr)
		space := max(0, innerW-used)
		return pad + pill + " " + lr + strings.Repeat(" ", space)
	}

	rows := []string{
		pad + titleStyle.Render("COMMANDS"),
		"",
		keyRow("n", accentGreen, bgGreen, "New branch"),
		"",
		keyRow("c", accentCyan, bgCyan, "Commit"),
		keyRow("s", accentYellow, bgYellow, "Stage file"),
		keyRow("u", accentYellow, bgYellow, "Unstage file"),
		keyRow("a", accentMagenta, bgMagenta, "Stash"),
		"",
		keyRow("p", accentCyan, bgCyan, "Push"),
		keyRow("P", accentCyan, bgCyan, "Pull (rebase)"),
		keyRow("g", accentCyan, bgCyan, "Fetch"),
		"",
		keyRow("D", accentRed, bgRed, "Delete branch"),
		keyRow("F", accentGreen, bgGreen, "Finish feature"),
		keyRow("R", accentMagenta, bgMagenta, "Finish release"),
		keyRow("H", accentYellow, bgYellow, "Finish hotfix"),
		"",
		keyRow("?", textDim, bgDim, "Help"),
		keyRow("r", textDim, bgDim, "Refresh"),
		keyRow("q", textDim, bgDim, "Quit"),
	}

	if len(rows) > innerH {
		rows = rows[:innerH]
	}

	return border.Render(strings.Join(rows, "\n"))
}

// helpContent returns the full scrollable keybinding reference text.
func helpContent() string {
	titleStyle := lipgloss.NewStyle().Foreground(accentCyan).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(accentCyan).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(textDim)
	descStyle := lipgloss.NewStyle().Foreground(textSecondary)

	type entry struct{ key, desc string }
	groups := []struct {
		title   string
		entries []entry
	}{
		{"Navigation", []entry{
			{keyTab, "cycle panels"},
			{"↑ / ↓", "move selection"},
			{keyEnter, "select or action"},
			{keyEsc, "cancel or back"},
		}},
		{"Working Tree", []entry{
			{"s", "stage file"},
			{"u", "unstage file"},
			{"c", "commit"},
			{keyCtrlA, "AI commit sugg."},
			{"a", "stash changes"},
			{"w", "toggle word diff"},
			{"E", "AI explain diff"},
		}},
		{"Remote", []entry{
			{"p", "push to remote"},
			{"P", "pull with rebase"},
			{"g", "fetch from remote"},
		}},
		{"Branches", []entry{
			{"n", "new feature / release / hotfix"},
			{"F", "finish feature"},
			{"R", "finish release"},
			{"H", "finish hotfix"},
			{"D", "delete branch"},
			{"X", "AI merge risk"},
			{"B", "AI branch health"},
		}},
		{"App", []entry{
			{"r", "refresh"},
			{"?", "toggle help"},
			{"q", "quit"},
		}},
	}

	sep := sepStyle.Render(" - ")
	var lines []string
	for _, g := range groups {
		lines = append(lines, titleStyle.Render(g.title))
		for _, e := range g.entries {
			lines = append(lines, keyStyle.Render(e.key)+sep+descStyle.Render(e.desc))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
