package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Polqt/gitflowtui/git"
	"github.com/Polqt/gitflowtui/gitflow"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type logMsg struct {
	branch  string
	commits []git.Commit
	err     error
}

type finishMsg struct {
	action finishAction
	result gitflow.FinishResult
	err    error
}

type prArtifactMsg struct {
	path string
	err  error
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case spinner.TickMsg:
		if a.loading {
			var cmd tea.Cmd
			a.spinner, cmd = a.spinner.Update(msg)
			return a, cmd
		}

	case refreshMsg:
		a.loading = false
		if msg.err != nil {
			a.setNotification(msg.err.Error(), true)
			return a, nil
		}
		a.applySnapshot(msg.snapshot)
		cmds = append(cmds, a.loadWorkingDiffCmd())
		if len(msg.snapshot.Stashes) > 0 {
			cmds = append(cmds, a.loadStashDiffCmd())
		}
		return a, tea.Batch(cmds...)

	case opMsg:
		a.loading = false
		a.setNotification(a.opMessage(msg), msg.err != nil)
		if msg.refresh && msg.err == nil {
			a.loading = true
			return a, a.refreshCmd()
		}
		return a, nil

	case diffMsg:
		if msg.err != nil {
			a.setNotification(msg.err.Error(), true)
			return a, nil
		}
		a.rawDiff = msg.text
		a.diff.SetContent(colorizeDiff(msg.text, max(20, a.diff.Width)))
		a.diff.GotoTop()
		return a, nil

	case logMsg:
		if msg.err != nil {
			a.setNotification(msg.err.Error(), true)
			return a, nil
		}
		a.log.SetItems(commitListItems(msg.commits))
		return a, nil

	case finishMsg:
		a.loading = false
		if msg.err != nil {
			a.setNotification(msg.err.Error(), true)
			return a, nil
		}
		detail := ""
		if msg.result.Tag != "" {
			detail = " tag " + msg.result.Tag
		}
		a.setNotification("Gitflow finish complete"+detail, false)
		a.loading = true
		return a, a.refreshCmd()

	case prArtifactMsg:
		if msg.err != nil {
			a.setNotification("PR template save failed: "+msg.err.Error(), true)
			return a, nil
		}
		a.setNotification("PR template copied to clipboard and saved: "+msg.path, false)
		return a, nil

	case tea.KeyMsg:
		if a.prForm.active {
			cmd, submitted, _, template := a.prForm.update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			if submitted {
				cmds = append(cmds, a.persistPRCmd(template))
				if a.pendingFinish != finishNone {
					a.loading = true
					cmds = append(cmds, a.finishCmd(a.pendingFinish, a.currentBranch))
					a.pendingFinish = finishNone
				}
			}
			return a, tea.Batch(cmds...)
		}

		if a.prompt.active {
			switch msg.String() {
			case "esc":
				a.prompt.active = false
				a.prompt.mode = promptNone
				a.prompt.input.Blur()
				return a, nil
			case "enter":
				cmd := a.submitPrompt()
				return a, cmd
			}
			var cmd tea.Cmd
			a.prompt.input, cmd = a.prompt.input.Update(msg)
			return a, cmd
		}

		if quit, cmd := a.handleKey(msg); quit {
			return a, cmd
		} else if cmd != nil {
			return a, cmd
		}
	}

	cmd := a.updateFocusedPanel(msg)
	if cmd != nil {
		return a, cmd
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	key := msg.String()

	if a.newBranchArmed {
		a.newBranchArmed = false
		switch key {
		case "f":
			a.openPrompt(promptFeatureName, "New Feature Branch", "Name (without feature/)", "checkout-redesign")
			return false, nil
		case "r":
			a.openPrompt(promptReleaseVersion, "New Release Branch", "Semver version", "1.2.0")
			return false, nil
		case "h":
			a.openPrompt(promptHotfixVersion, "New Hotfix Branch", "Semver version", "1.2.1")
			return false, nil
		default:
			return false, nil
		}
	}

	switch key {
	case "ctrl+c", "q":
		return true, tea.Quit

	case "tab":
		a.activePanel = (a.activePanel + 1) % 5
		return false, nil
	case "shift+tab":
		a.activePanel--
		if a.activePanel < 0 {
			a.activePanel = panelDiff
		}
		return false, nil

	case "r":
		a.loading = true
		return false, a.refreshCmd()

	case "?":
		a.showHelp = !a.showHelp
		return false, nil

	case "n":
		a.newBranchArmed = true
		a.setNotification("Create branch: press f (feature), r (release), or h (hotfix)", false)
		return false, nil

	case "c":
		a.openPrompt(promptCommit, "Commit", "Message", "feat: describe change")
		return false, nil

	case "a":
		a.openPrompt(promptStash, "Stash", "Optional stash message", "wip")
		return false, nil

	case "p":
		a.loading = true
		return false, a.gitOpCmd("push", true, func(ctx context.Context) error {
			return a.repo.Push(ctx)
		})

	case "P":
		a.loading = true
		return false, a.gitOpCmd("pull --rebase", true, func(ctx context.Context) error {
			return a.repo.PullRebase(ctx)
		})

	case "F":
		a.pendingFinish = finishFeature
		a.prForm.open(finishFeature, a.currentBranch, a.cfg.DevelopBranch)
		return false, nil

	case "R":
		a.pendingFinish = finishRelease
		a.prForm.open(finishRelease, a.currentBranch, a.cfg.MainBranch)
		return false, nil

	case "H":
		a.pendingFinish = finishHotfix
		a.prForm.open(finishHotfix, a.currentBranch, a.cfg.MainBranch)
		return false, nil

	case "enter":
		switch a.activePanel {
		case panelBranches:
			item, ok := a.selectedBranchItem()
			if !ok {
				return false, nil
			}
			a.loading = true
			return false, a.gitOpCmd("checkout "+item.branch.Name, true, func(ctx context.Context) error {
				return a.repo.Checkout(ctx, item.branch.Name)
			})

		case panelStatus:
			return false, a.loadWorkingDiffCmd()

		case panelStash:
			item, ok := a.selectedStashItem()
			if !ok {
				return false, nil
			}
			idx, err := git.StashIndex(item.entry.Ref)
			if err != nil {
				a.setNotification(err.Error(), true)
				return false, nil
			}
			a.loading = true
			return false, a.gitOpCmd("stash pop "+item.entry.Ref, true, func(ctx context.Context) error {
				return a.repo.StashPop(ctx, idx)
			})
		}

	case "s":
		if a.activePanel != panelStatus {
			return false, nil
		}
		item, ok := a.selectedFileItem()
		if !ok {
			return false, nil
		}
		a.loading = true
		return false, a.gitOpCmd("stage "+item.file.Path, true, func(ctx context.Context) error {
			return a.repo.Stage(ctx, item.file.Path)
		})

	case "u":
		if a.activePanel != panelStatus {
			return false, nil
		}
		item, ok := a.selectedFileItem()
		if !ok {
			return false, nil
		}
		a.loading = true
		return false, a.gitOpCmd("unstage "+item.file.Path, true, func(ctx context.Context) error {
			return a.repo.Unstage(ctx, item.file.Path)
		})
	}

	return false, nil
}

func (a *App) submitPrompt() tea.Cmd {
	value := strings.TrimSpace(a.prompt.input.Value())
	mode := a.prompt.mode
	a.prompt.active = false
	a.prompt.mode = promptNone
	a.prompt.input.Blur()

	switch mode {
	case promptCommit:
		a.loading = true
		return a.gitOpCmd("commit", true, func(ctx context.Context) error {
			return a.repo.Commit(ctx, value, false)
		})
	case promptStash:
		a.loading = true
		return a.gitOpCmd("stash", true, func(ctx context.Context) error {
			return a.repo.Stash(ctx, value)
		})
	case promptFeatureName:
		a.loading = true
		return a.gitOpCmd("new feature", true, func(ctx context.Context) error {
			_, err := a.workflow.StartFeature(ctx, value)
			return err
		})
	case promptReleaseVersion:
		a.loading = true
		return a.gitOpCmd("new release", true, func(ctx context.Context) error {
			_, err := a.workflow.StartRelease(ctx, value)
			return err
		})
	case promptHotfixVersion:
		a.loading = true
		return a.gitOpCmd("new hotfix", true, func(ctx context.Context) error {
			_, err := a.workflow.StartHotfix(ctx, value)
			return err
		})
	default:
		return nil
	}
}

func (a *App) finishCmd(action finishAction, branch string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		var (
			res gitflow.FinishResult
			err error
		)
		switch action {
		case finishFeature:
			res, err = a.workflow.FinishFeature(ctx, branch)
		case finishRelease:
			res, err = a.workflow.FinishRelease(ctx, branch)
		case finishHotfix:
			res, err = a.workflow.FinishHotfix(ctx, branch)
		}
		return finishMsg{action: action, result: res, err: err}
	}
}

func (a *App) persistPRCmd(content string) tea.Cmd {
	return func() tea.Msg {
		path, err := persistPRTemplate(content)
		return prArtifactMsg{path: path, err: err}
	}
}

func (a *App) loadLogCmd(branch string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		commits, err := a.repo.LogRef(ctx, branch, a.cfg.LogLimit)
		return logMsg{branch: branch, commits: commits, err: err}
	}
}

func (a *App) updateFocusedPanel(msg tea.Msg) tea.Cmd {
	switch a.activePanel {
	case panelBranches:
		prev := a.branches.Index()
		var cmd tea.Cmd
		a.branches, cmd = a.branches.Update(msg)
		if a.branches.Index() != prev {
			if item, ok := a.selectedBranchItem(); ok {
				return tea.Batch(cmd, a.loadLogCmd(item.branch.Name))
			}
		}
		return cmd

	case panelLog:
		var cmd tea.Cmd
		a.log, cmd = a.log.Update(msg)
		return cmd

	case panelStatus:
		prev := a.status.Index()
		var cmd tea.Cmd
		a.status, cmd = a.status.Update(msg)
		if a.status.Index() != prev {
			return tea.Batch(cmd, a.loadWorkingDiffCmd())
		}
		return cmd

	case panelStash:
		prev := a.stash.Index()
		var cmd tea.Cmd
		a.stash, cmd = a.stash.Update(msg)
		if a.stash.Index() != prev {
			return tea.Batch(cmd, a.loadStashDiffCmd())
		}
		return cmd

	case panelDiff:
		var cmd tea.Cmd
		a.diff, cmd = a.diff.Update(msg)
		return cmd
	}
	return nil
}

func (a *App) View() string {
	if a.width <= 0 || a.height <= 0 {
		return "loading..."
	}

	title := a.styles.title.Render(fmt.Sprintf(" GitFlow TUI  %s ", filepath.Base(a.repo.Root)))

	contentHeight := a.height - 2 // title + status
	if a.showHelp {
		contentHeight--
	}
	contentHeight = max(contentHeight, 10)
	contentWidth := max(a.width-2, 40)

	leftWeight, centerWeight, rightWeight := 3, 3, 4
	totalWeight := leftWeight + centerWeight + rightWeight
	leftWidth := (contentWidth * leftWeight) / totalWeight
	centerWidth := (contentWidth * centerWeight) / totalWeight
	rightWidth := contentWidth - leftWidth - centerWidth

	leftTopHeight := max(6, (contentHeight*2)/3)
	leftBottomHeight := max(4, contentHeight-leftTopHeight)

	rightTopHeight := max(6, (contentHeight*2)/5)
	rightBottomHeight := max(4, contentHeight-rightTopHeight)

	branchBodyH := max(1, leftTopHeight-3)
	stashBodyH := max(1, leftBottomHeight-3)
	logBodyH := max(1, contentHeight-3)
	statusBodyH := max(1, rightTopHeight-3)
	diffBodyH := max(1, rightBottomHeight-3)

	branchInnerW := max(1, leftWidth-2)
	centerInnerW := max(1, centerWidth-2)
	rightInnerW := max(1, rightWidth-2)

	a.branches.SetSize(branchInnerW, branchBodyH)
	a.stash.SetSize(branchInnerW, stashBodyH)
	a.log.SetSize(centerInnerW, logBodyH)
	a.status.SetSize(rightInnerW, statusBodyH)
	a.diff.Width = rightInnerW
	a.diff.Height = diffBodyH
	if a.rawDiff != "" {
		a.diff.SetContent(colorizeDiff(a.rawDiff, rightInnerW))
	}

	leftCol := lipgloss.JoinVertical(
		lipgloss.Left,
		a.renderPanel("Branches", a.branches.View(), leftWidth, leftTopHeight, a.activePanel == panelBranches),
		a.renderPanel("Stash", a.stash.View(), leftWidth, leftBottomHeight, a.activePanel == panelStash),
	)

	centerCol := a.renderPanel("Log", a.log.View(), centerWidth, contentHeight, a.activePanel == panelLog)

	rightCol := lipgloss.JoinVertical(
		lipgloss.Left,
		a.renderPanel("Status", a.status.View(), rightWidth, rightTopHeight, a.activePanel == panelStatus),
		a.renderPanel("Diff", a.diff.View(), rightWidth, rightBottomHeight, a.activePanel == panelDiff),
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, centerCol, rightCol)

	var parts []string
	parts = append(parts, title, body)
	if a.showHelp {
		parts = append(parts, a.styles.help.Render(helpLine()))
	}
	parts = append(parts, a.statusLine())

	ui := a.styles.root.Render(strings.Join(parts, "\n"))

	if a.prompt.active {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, a.renderPrompt())
	}
	if a.prForm.active {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, a.prForm.view(min(a.width-4, 90), min(a.height-4, 32)))
	}

	return ui
}

func (a *App) renderPanel(title, content string, width, height int, focused bool) string {
	innerW := max(1, width-2)
	innerH := max(1, height-2)

	titleLine := truncateString(title, innerW)
	bodyH := max(0, innerH-1)
	body := renderPanelBody(content, bodyH)
	panelContent := titleLine
	if bodyH > 0 {
		panelContent += "\n" + body
	}

	style := a.styles.panel
	if focused {
		style = a.styles.panelFocused
	}
	return style.Width(innerW).Render(panelContent)
}

func (a *App) renderPrompt() string {
	w := min(max(50, a.width*2/3), a.width-4)
	innerW := max(1, w-4)

	a.prompt.input.Width = innerW - 2
	content := strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Render(a.prompt.title),
		"",
		a.prompt.hint,
		a.prompt.input.View(),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("[Enter] Submit  [Esc] Cancel"),
	}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("81")).
		Width(w - 2).
		Render(content)
}

func (a *App) statusLine() string {
	branch := a.currentBranch
	if branch == "" {
		branch = "(detached)"
	}

	left := fmt.Sprintf("%s  \u2191%d \u2193%d", branch, a.ahead, a.behind)
	if a.loading {
		left = a.spinner.View() + " " + a.loadingLabel + "  " + left
	}

	right := a.notification
	if right == "" {
		right = "tab:focus  n+f/r/h:new branch  F/R/H:finish  c:commit  a:stash"
	}

	combined := left + " | " + right
	combined = truncateString(combined, max(1, a.width-4))
	if a.notifError {
		return a.styles.statusError.Render(combined)
	}
	return a.styles.statusNormal.Render(combined)
}

func renderPanelBody(content string, height int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func helpLine() string {
	return "Keys: tab/shift+tab focus, enter action, s stage, u unstage, c commit, a stash, p push, P pull --rebase, n then f/r/h create branches, F/R/H finish, r refresh, q quit"
}
