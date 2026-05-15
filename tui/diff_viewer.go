package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// colorizeDiff applies ANSI colour and line numbers to a unified diff string.
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
	lineNo := lipgloss.NewStyle().Foreground(textDim)
	wordAdd := lipgloss.NewStyle().Foreground(accentGreen).Bold(true)
	wordDel := lipgloss.NewStyle().Foreground(accentRed).Bold(true)

	lines := strings.Split(raw, "\n")
	result := make([]string, 0, len(lines))
	oldLine := 0
	newLine := 0

	for _, line := range lines {
		line = truncateString(line, maxWidth-6)
		line = highlightWordDiff(line, wordAdd, wordDel)

		switch {
		case strings.HasPrefix(line, "+++ "),
			strings.HasPrefix(line, "--- "),
			strings.HasPrefix(line, "diff --git"):
			result = append(result, meta.Render(line))

		case strings.HasPrefix(line, "@@"):
			// Parse hunk header to reset line counters. Errors are non-fatal.
			var o, n int
			_, _ = fmt.Sscanf(line, "@@ -%d", &o)
			_, _ = fmt.Sscanf(line, "@@ -%*d,%*d +%d", &n)
			if o > 0 {
				oldLine = o - 1
			}
			if n > 0 {
				newLine = n - 1
			}
			result = append(result, hunk.Render(line))

		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			newLine++
			no := lineNo.Render(fmt.Sprintf("%3d", newLine))
			mk := added.Render(" +")
			result = append(result, no+mk+"  "+added.Render(line[1:]))

		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			oldLine++
			no := lineNo.Render(fmt.Sprintf("%3d", oldLine))
			mk := removed.Render(" -")
			result = append(result, no+mk+"  "+removed.Render(line[1:]))

		default:
			oldLine++
			newLine++
			no := lineNo.Render(fmt.Sprintf("%3d", newLine))
			result = append(result, no+"    "+line)
		}
	}

	return strings.Join(result, "\n")
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
