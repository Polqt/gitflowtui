package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// colorizeDiff applies ANSI colour to a unified diff string.
func colorizeDiff(raw string, maxWidth int) string {
	if strings.TrimSpace(raw) == "" {
		return lipgloss.NewStyle().Foreground(textSecondary).Render("No diff")
	}
	if maxWidth <= 0 {
		maxWidth = 120
	}

	added := lipgloss.NewStyle().Foreground(accentGreen)
	removed := lipgloss.NewStyle().Foreground(accentRed)
	hunk := lipgloss.NewStyle().Foreground(accentCyan)
	meta := lipgloss.NewStyle().Bold(true).Foreground(textPrimary)
	wordAdd := lipgloss.NewStyle().Foreground(accentGreen).Bold(true)
	wordDel := lipgloss.NewStyle().Foreground(accentRed).Bold(true)

	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		line = truncateString(line, maxWidth)
		line = highlightWordDiff(line, wordAdd, wordDel)
		switch {
		case strings.HasPrefix(line, "+++ "),
			strings.HasPrefix(line, "--- "),
			strings.HasPrefix(line, "diff --git"):
			lines[i] = meta.Render(line)
		case strings.HasPrefix(line, "@@"):
			lines[i] = hunk.Render(line)
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			lines[i] = added.Render(line)
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			lines[i] = removed.Render(line)
		default:
			lines[i] = line
		}
	}

	return strings.Join(lines, "\n")
}

func highlightWordDiff(line string, addStyle, removeStyle lipgloss.Style) string {
	var b strings.Builder
	i := 0
	for i < len(line) {
		nextAdd := strings.Index(line[i:], "{+")
		nextDel := strings.Index(line[i:], "[-")
		if nextAdd == -1 && nextDel == -1 {
			b.WriteString(line[i:])
			break
		}

		isAdd := false
		next := 0
		switch {
		case nextAdd == -1:
			next = nextDel
		case nextDel == -1:
			next = nextAdd
			isAdd = true
		case nextAdd < nextDel:
			next = nextAdd
			isAdd = true
		default:
			next = nextDel
		}

		start := i + next
		b.WriteString(line[i:start])

		if isAdd {
			endRel := strings.Index(line[start+2:], "+}")
			if endRel == -1 {
				b.WriteString(line[start:])
				break
			}
			end := start + 2 + endRel + 2
			b.WriteString(addStyle.Render(line[start:end]))
			i = end
			continue
		}

		endRel := strings.Index(line[start+2:], "-]")
		if endRel == -1 {
			b.WriteString(line[start:])
			break
		}
		end := start + 2 + endRel + 2
		b.WriteString(removeStyle.Render(line[start:end]))
		i = end
	}
	return b.String()
}

func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen == 1 {
		return "\u2026"
	}
	return string(runes[:maxLen-1]) + "\u2026"
}
