package tui

import (
	"fmt"
	"strings"

	"github.com/Polqt/gitflowtui/git"
	"github.com/Polqt/gitflowtui/gitflow"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// ── list.Item wrappers ────────────────────────────────────────────────────────

type branchItem struct {
	branch git.Branch
	label  string
}

func (i branchItem) Title() string       { return i.label }
func (i branchItem) Description() string { return "" }
func (i branchItem) FilterValue() string { return i.branch.Name }

type commitItem struct {
	commit git.Commit
	label  string
}

func (i commitItem) Title() string       { return i.label }
func (i commitItem) Description() string { return "" }
func (i commitItem) FilterValue() string { return i.commit.Hash + " " + i.commit.Subject }

type fileItem struct {
	file  git.FileStatus
	label string
}

func (i fileItem) Title() string       { return i.label }
func (i fileItem) Description() string { return "" }
func (i fileItem) FilterValue() string { return i.file.Path }

type stashItem struct {
	entry git.StashEntry
	label string
}

func (i stashItem) Title() string       { return i.label }
func (i stashItem) Description() string { return "" }
func (i stashItem) FilterValue() string { return i.entry.Ref + " " + i.entry.Message }

// ── snapshot application ──────────────────────────────────────────────────────

func (a *App) applySnapshot(s repoSnapshot) {
	a.currentBranch = s.CurrentBranch
	a.ahead = s.Ahead
	a.behind = s.Behind

	a.branches.SetItems(branchListItems(s.Branches, a.workflow.Config(), a.styles))
	a.log.SetItems(commitListItems(s.Commits))
	a.status.SetItems(fileListItems(s.Files, a.styles))
	a.stash.SetItems(stashListItems(s.Stashes))
}

// ── branch labels ─────────────────────────────────────────────────────────────

// branchKindTag returns a short coloured tag like MAIN, FEAT, REL, HOT, DEV.
func branchKindTag(kind gitflow.BranchKind, st uiStyles) string {
	switch kind {
	case gitflow.KindMain:
		return st.badge.main.Render("MAIN")
	case gitflow.KindDevelop:
		return st.badge.develop.Render("DEV")
	case gitflow.KindFeature:
		return st.badge.feature.Render("FEAT")
	case gitflow.KindRelease:
		return st.badge.release.Render("REL")
	case gitflow.KindHotfix:
		return st.badge.hotfix.Render("HOT")
	case gitflow.KindUnknown:
		return st.badge.unknown.Render("·")
	}
	return st.badge.unknown.Render("·")
}

func branchListItems(branches []git.Branch, cfg gitflow.Config, st uiStyles) []list.Item {
	items := make([]list.Item, 0, len(branches))
	for _, b := range branches {
		kind := gitflow.DetectKind(b.Name, cfg)

		// HEAD marker
		headMark := "  "
		if b.IsHead {
			headMark = lipgloss.NewStyle().Foreground(colorGold).Bold(true).Render("▸ ")
		}

		// kind tag
		tag := branchKindTag(kind, st)

		// branch name — coloured by kind
		nameStyle := kindNameStyle(kind, b.IsHead)
		name := nameStyle.Render(b.Name)

		// sync indicators
		sync := ""
		if b.Upstream != "" {
			parts := make([]string, 0, 2)
			if b.Ahead > 0 {
				parts = append(parts, st.badge.ahead.Render(fmt.Sprintf("↑%d", b.Ahead)))
			}
			if b.Behind > 0 {
				parts = append(parts, st.badge.behind.Render(fmt.Sprintf("↓%d", b.Behind)))
			}
			if len(parts) > 0 {
				sync = "  " + strings.Join(parts, " ")
			} else {
				sync = "  " + lipgloss.NewStyle().Foreground(colorGreen).Render("✓")
			}
		}

		label := headMark + tag + "  " + name + sync
		items = append(items, branchItem{branch: b, label: label})
	}
	return items
}

// kindNameStyle returns a text style appropriate for the branch kind and HEAD state.
func kindNameStyle(kind gitflow.BranchKind, isHead bool) lipgloss.Style {
	var fg lipgloss.Color
	switch kind {
	case gitflow.KindMain:
		fg = colorGold
	case gitflow.KindDevelop:
		fg = colorCyan
	case gitflow.KindFeature:
		fg = colorBlue
	case gitflow.KindRelease:
		fg = colorGreen
	case gitflow.KindHotfix:
		fg = colorRed
	case gitflow.KindUnknown:
		fg = colorFg
	}
	s := lipgloss.NewStyle().Foreground(fg)
	if isHead {
		s = s.Bold(true)
	}
	return s
}

// ── commit labels ─────────────────────────────────────────────────────────────

func commitListItems(commits []git.Commit) []list.Item {
	items := make([]list.Item, 0, len(commits))
	hashStyle := lipgloss.NewStyle().Foreground(colorBlue).Bold(true)
	dateStyle := lipgloss.NewStyle().Foreground(colorMuted)

	for _, c := range commits {
		hash := hashStyle.Render(c.Hash)
		date := dateStyle.Render(c.Date)
		subject := truncateString(c.Subject, 60)
		label := hash + "  " + date + "  " + subject
		items = append(items, commitItem{commit: c, label: label})
	}
	return items
}

// ── file status labels ────────────────────────────────────────────────────────

func fileListItems(files []git.FileStatus, st uiStyles) []list.Item {
	items := make([]list.Item, 0, len(files))
	for _, f := range files {
		badge, indicator := fileStatusBadge(f, st)
		path := truncateString(f.DisplayPath(), 50)
		label := badge + "  " + path + "  " + indicator
		items = append(items, fileItem{file: f, label: label})
	}
	return items
}

// fileStatusBadge returns a coloured [XY] badge and a plain-text state label.
func fileStatusBadge(f git.FileStatus, st uiStyles) (string, string) {
	xy := string(f.X) + string(f.Y)

	switch {
	case f.IsUntracked():
		return st.badge.untracked.Render("[??]"),
			lipgloss.NewStyle().Foreground(colorMuted).Render("untracked")
	case f.IsStaged() && f.IsUnstaged():
		return st.badge.both.Render("[" + xy + "]"),
			st.badge.both.Render("staged+unstaged")
	case f.IsStaged():
		return st.badge.staged.Render("[" + xy + "]"),
			st.badge.staged.Render("staged")
	case f.IsUnstaged():
		return st.badge.unstaged.Render("[" + xy + "]"),
			st.badge.unstaged.Render("unstaged")
	default:
		return lipgloss.NewStyle().Foreground(colorMuted).Render("[" + xy + "]"), ""
	}
}

// ── stash labels ──────────────────────────────────────────────────────────────

func stashListItems(entries []git.StashEntry) []list.Item {
	items := make([]list.Item, 0, len(entries))
	refStyle := lipgloss.NewStyle().Foreground(colorPurple).Bold(true)
	timeStyle := lipgloss.NewStyle().Foreground(colorMuted)

	for _, e := range entries {
		ref := refStyle.Render(e.Ref)
		age := timeStyle.Render("(" + e.RelativeTime + ")")
		msg := truncateString(e.Message, 40)
		label := ref + "  " + msg + "  " + age
		items = append(items, stashItem{entry: e, label: label})
	}
	return items
}

// ── selection helpers ─────────────────────────────────────────────────────────

func (a *App) selectedBranchItem() (branchItem, bool) {
	item := a.branches.SelectedItem()
	if item == nil {
		return branchItem{}, false
	}
	b, ok := item.(branchItem)
	return b, ok
}

func (a *App) selectedFileItem() (fileItem, bool) {
	item := a.status.SelectedItem()
	if item == nil {
		return fileItem{}, false
	}
	f, ok := item.(fileItem)
	return f, ok
}

func (a *App) selectedStashItem() (stashItem, bool) {
	item := a.stash.SelectedItem()
	if item == nil {
		return stashItem{}, false
	}
	s, ok := item.(stashItem)
	return s, ok
}
