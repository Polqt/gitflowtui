package tui

import (
	"fmt"
	"strings"

	"github.com/Polqt/gitflowtui/git"
	"github.com/Polqt/gitflowtui/gitflow"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

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

func (a *App) applySnapshot(s repoSnapshot) {
	a.currentBranch = s.CurrentBranch
	a.ahead = s.Ahead
	a.behind = s.Behind

	a.branches.SetItems(branchListItems(s.Branches, a.workflow.Config()))
	a.log.SetItems(commitListItems(s.Commits))
	a.status.SetItems(fileListItems(s.Files))
	a.stash.SetItems(stashListItems(s.Stashes))
}

func branchListItems(branches []git.Branch, cfg gitflow.Config) []list.Item {
	items := make([]list.Item, 0, len(branches))
	for _, b := range branches {
		kind := gitflow.DetectKind(b.Name, cfg)
		color := lipgloss.Color(gitflow.KindColor(kind))

		name := b.Name
		if b.IsHead {
			name = "* " + name
		}
		label := lipgloss.NewStyle().Foreground(color).Render(name)
		if b.Upstream != "" {
			label = fmt.Sprintf("%s  %s  +%d -%d", label, b.Upstream, b.Ahead, b.Behind)
		}

		items = append(items, branchItem{branch: b, label: label})
	}
	return items
}

func commitListItems(commits []git.Commit) []list.Item {
	items := make([]list.Item, 0, len(commits))
	for _, c := range commits {
		label := fmt.Sprintf("%s  %s  %s", c.Hash, c.Date, c.Subject)
		items = append(items, commitItem{commit: c, label: label})
	}
	return items
}

func fileListItems(files []git.FileStatus) []list.Item {
	items := make([]list.Item, 0, len(files))
	for _, f := range files {
		status := string(f.X) + string(f.Y)
		label := fmt.Sprintf("[%s] %s", status, f.DisplayPath())

		parts := make([]string, 0, 2)
		if f.IsStaged() {
			parts = append(parts, "staged")
		}
		if f.IsUnstaged() {
			parts = append(parts, "unstaged")
		}
		if f.IsUntracked() {
			parts = append(parts, "untracked")
		}
		if len(parts) > 0 {
			label += "  (" + strings.Join(parts, ", ") + ")"
		}

		items = append(items, fileItem{file: f, label: label})
	}
	return items
}

func stashListItems(entries []git.StashEntry) []list.Item {
	items := make([]list.Item, 0, len(entries))
	for _, e := range entries {
		label := fmt.Sprintf("%s  %s  (%s)", e.Ref, e.Message, e.RelativeTime)
		items = append(items, stashItem{entry: e, label: label})
	}
	return items
}


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
