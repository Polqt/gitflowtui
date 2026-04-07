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

// branchKindTag returns a short muted type prefix like [F], [R], [H], [M], [D].
func branchKindTag(kind gitflow.BranchKind) string {
	style := lipgloss.NewStyle().Foreground(textSecondary)
	switch kind {
	case gitflow.KindMain:
		return style.Render("[M]")
	case gitflow.KindDevelop:
		return style.Render("[D]")
	case gitflow.KindFeature:
		return style.Render("[F]")
	case gitflow.KindRelease:
		return style.Render("[R]")
	case gitflow.KindHotfix:
		return style.Render("[H]")
	case gitflow.KindUnknown:
		return style.Render("[·]")
	}
	return style.Render("[·]")
}

// branchStatusBadge returns a pill-style badge based on branch state.
func branchStatusBadge(b git.Branch, kind gitflow.BranchKind, st uiStyles) string {
	switch {
	case kind == gitflow.KindHotfix:
		return st.badge.urgent.Render("URGENT")
	case b.IsHead && b.Ahead > 0:
		return st.badge.readyPR.Render("STAGED")
	case b.IsHead:
		return st.badge.inProg.Render("IN PROGRESS")
	case b.Ahead > 0:
		return st.badge.readyPR.Render("READY PR")
	default:
		return ""
	}
}

func branchListItems(branches []git.Branch, cfg gitflow.Config, st uiStyles) []list.Item {
	items := make([]list.Item, 0, len(branches))
	for _, b := range branches {
		kind := gitflow.DetectKind(b.Name, cfg)

		tag := branchKindTag(kind)
		name := kindNameStyle(kind, b.IsHead).Render(b.Name)
		badge := branchStatusBadge(b, kind, st)

		// Head indicator.
		headDot := ""
		if b.IsHead {
			headDot = lipgloss.NewStyle().Foreground(accentCyan).Bold(true).Render(" ●")
		}

		// Sync indicators.
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
				sync = " " + strings.Join(parts, " ")
			}
		}

		label := tag + " " + name + headDot + sync
		if badge != "" {
			label += "  " + badge
		}

		items = append(items, branchItem{branch: b, label: label})
	}
	return items
}

// kindNameStyle returns a text style for the branch kind.
func kindNameStyle(kind gitflow.BranchKind, isHead bool) lipgloss.Style {
	s := lipgloss.NewStyle().Foreground(textPrimary)
	switch kind {
	case gitflow.KindMain:
		s = s.Foreground(tagMainFg)
	case gitflow.KindDevelop:
		s = s.Foreground(accentCyan)
	case gitflow.KindFeature:
		s = s.Foreground(tagFeatureFg)
	case gitflow.KindRelease:
		s = s.Foreground(tagReleaseFg)
	case gitflow.KindHotfix:
		s = s.Foreground(tagHotfixFg)
	case gitflow.KindUnknown:
		s = s.Foreground(textPrimary)
	}
	if isHead {
		s = s.Bold(true)
	}
	return s
}

// ── commit labels ─────────────────────────────────────────────────────────────

func commitListItems(commits []git.Commit) []list.Item {
	items := make([]list.Item, 0, len(commits))
	hashStyle := lipgloss.NewStyle().Foreground(textSecondary)
	dateStyle := lipgloss.NewStyle().Foreground(textSecondary)
	authorStyle := lipgloss.NewStyle().Foreground(textSecondary)

	for _, c := range commits {
		hash := hashStyle.Render(c.Hash)
		date := dateStyle.Render(c.Date)
		author := ""
		if c.Author != "" {
			author = authorStyle.Render("@" + c.Author)
		}
		subject := truncateString(c.Subject, 50)

		label := hash + "  " + date + "  " + subject
		if author != "" {
			label += "  " + author
		}
		items = append(items, commitItem{commit: c, label: label})
	}
	return items
}

// ── file status labels ────────────────────────────────────────────────────────

func fileListItems(files []git.FileStatus, st uiStyles) []list.Item {
	items := make([]list.Item, 0, len(files))
	for _, f := range files {
		badge, indicator := fileStatusBadge(f, st)
		path := truncateString(f.DisplayPath(), 40)
		label := badge + " " + path + "  " + indicator
		items = append(items, fileItem{file: f, label: label})
	}
	return items
}

func fileStatusBadge(f git.FileStatus, st uiStyles) (string, string) {
	xy := string(f.X) + string(f.Y)

	switch {
	case f.IsUntracked():
		return st.badge.untracked.Render("[??]"),
			lipgloss.NewStyle().Foreground(textDim).Render("untracked")
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
		return lipgloss.NewStyle().Foreground(textSecondary).Render("[" + xy + "]"), ""
	}
}

// ── stash labels ──────────────────────────────────────────────────────────────

func stashListItems(entries []git.StashEntry) []list.Item {
	items := make([]list.Item, 0, len(entries))
	refStyle := lipgloss.NewStyle().Foreground(accentMagenta).Bold(true)
	timeStyle := lipgloss.NewStyle().Foreground(textSecondary)

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
