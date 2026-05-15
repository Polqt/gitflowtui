package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderCards(totalW int) string {
	// Each card outer width = totalW/5. Last card absorbs remainder.
	cardW := totalW / 5
	w1, w2, w3, w4 := cardW, cardW, cardW, cardW
	w5 := totalW - cardW*4

	// Card 1 — LOCAL HEAD
	headVal := a.currentBranch
	if headVal == "" {
		headVal = "detached"
	}
	upstream := ""
	for _, item := range a.branches.Items() {
		if bi, ok := item.(branchItem); ok && bi.branch.IsHead && bi.branch.Upstream != "" {
			upstream = truncateString(bi.branch.Upstream, 16)
			break
		}
	}
	card1 := renderCard("HEAD", headVal, upstream, w1, accentCyan)

	// Card 2 — COMMITS
	commitCount := len(a.log.Items())
	syncSub := ""
	if a.ahead > 0 || a.behind > 0 {
		syncSub = fmt.Sprintf("↑%d ↓%d  %s", a.ahead, a.behind, truncateString(a.currentBranch, 10))
	}
	card2 := renderCard("COMMITS", strconv.Itoa(commitCount), syncSub, w2, textPrimary)

	// Card 3 — STASHES
	stashItems := a.stash.Items()
	stashVal := "empty"
	stashColor := textDim
	stashSub := ""
	if len(stashItems) > 0 {
		stashVal = strconv.Itoa(len(stashItems))
		stashColor = accentYellow
		stashSub = "view stashes"
	}
	card3 := renderCard("STASHES", stashVal, stashSub, w3, stashColor)

	// Card 4 — WORKING TREE
	staged, unstaged, untracked := 0, 0, 0
	for _, item := range a.status.Items() {
		if fi, ok := item.(fileItem); ok {
			f := fi.file
			switch {
			case f.IsUntracked():
				untracked++
			case f.IsStaged() && f.IsUnstaged():
				staged++
				unstaged++
			case f.IsStaged():
				staged++
			case f.IsUnstaged():
				unstaged++
			}
		}
	}
	total := staged + unstaged + untracked
	wtVal := "clean"
	wtColor := accentGreen
	wtSub := ""
	if total > 0 {
		wtVal = fmt.Sprintf("%d changed", total)
		wtColor = accentOrange
		wtSub = fmt.Sprintf("%d stgd  %d unstgd  %d new", staged, unstaged, untracked)
	}
	card4 := renderCard("TREE", wtVal, wtSub, w4, wtColor)

	// Card 5 — REMOTE
	remoteName := "origin"
	remoteColor := textDim
	remoteSub := "not connected"
	for _, item := range a.branches.Items() {
		if bi, ok := item.(branchItem); ok && bi.branch.IsHead && bi.branch.Upstream != "" {
			parts := strings.SplitN(bi.branch.Upstream, "/", 2)
			if len(parts) > 0 {
				remoteName = parts[0]
			}
			remoteColor = accentGreen
			remoteSub = "last fetch: 1m ago"
			break
		}
	}
	card5 := renderCard("REMOTE", remoteName, remoteSub, w5, remoteColor)

	return lipgloss.JoinHorizontal(lipgloss.Top, card1, card2, card3, card4, card5)
}

func renderCard(label, value, sub string, outerW int, valueColor lipgloss.Color) string {
	// Width(w) sets the content box width. border adds 1 col each side → outerW = w+2.
	// So content width = outerW-2. We manually pad 1 space each side inside content.
	contentW := max(1, outerW-2) // lipgloss Width() value
	textW := max(1, contentW-2)  // visible text width (1 space pad each side)
	pad := " "

	lbl := pad + lipgloss.NewStyle().Foreground(textDim).Render(truncateString(label, textW))
	val := pad + lipgloss.NewStyle().Foreground(valueColor).Bold(true).Render(truncateString(value, textW))
	sub2 := pad + lipgloss.NewStyle().Foreground(textDim).Render(truncateString(sub, textW))

	return lipgloss.NewStyle().
		Background(bgElevated).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bgBorder).
		Width(contentW).
		Height(3).
		Render(lbl + "\n" + val + "\n" + sub2)
}
