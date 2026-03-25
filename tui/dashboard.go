package tui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Polqt/gitflowtui/ai"
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

type aiCommitMsg struct {
	suggestion *ai.CommitSuggestion
	err        error
}

type aiRiskMsg struct {
	risk *ai.MergeRisk
	err  error
}

type aiExplainStartMsg struct {
	stream ai.StreamResult
	title  string
	cancel context.CancelFunc
	err    error
}

type aiExplainTokenMsg struct {
	token string
}

type aiExplainDoneMsg struct {
	err error
}

type aiBranchHealthMsg struct {
	report *ai.BranchHealthReport
	err    error
}

//nolint:gocognit,gocyclo,cyclop,funlen,maintidx // Central Bubble Tea event router; split further would obscure message flow.
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
		a.stopAIExplain()
		a.loading = false
		if msg.err != nil {
			a.setNotification(msg.err.Error(), true)
			return a, nil
		}
		a.applySnapshot(msg.snapshot)
		a.publishSnapshot(msg.snapshot)
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
		a.stopAIExplain()
		if msg.err != nil {
			a.setNotification(msg.err.Error(), true)
			return a, nil
		}
		a.aiExplainText = ""
		a.diffFromStash = msg.fromStash
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

	case aiCommitMsg:
		a.loading = false
		if msg.err != nil {
			a.setNotification(msg.err.Error(), true)
			return a, nil
		}
		if msg.suggestion == nil {
			a.setNotification("AI commit suggestion was empty", true)
			return a, nil
		}
		a.prompt.input.SetValue(msg.suggestion.Message)
		notice := "AI commit suggestion applied"
		if msg.suggestion.MixedConcerns {
			notice += "  [mixed: consider splitting]"
		}
		if msg.suggestion.Breaking {
			notice += "  [BREAKING CHANGE]"
		}
		a.setNotification(notice, false)
		return a, nil

	case aiRiskMsg:
		a.loading = false
		if msg.err != nil {
			a.setNotification(msg.err.Error(), true)
			return a, nil
		}
		if msg.risk == nil {
			a.setNotification("AI merge risk returned no data", true)
			return a, nil
		}
		a.aiView = aiOverlay{
			active:  true,
			title:   "AI Merge Risk",
			content: renderMergeRisk(msg.risk),
		}
		a.setNotification(msg.risk.Summary, false)
		return a, nil

	case aiBranchHealthMsg:
		a.loading = false
		if msg.err != nil {
			a.setNotification(msg.err.Error(), true)
			return a, nil
		}
		if msg.report == nil {
			a.setNotification("AI branch health returned no data", true)
			return a, nil
		}
		a.aiView = aiOverlay{
			active:  true,
			title:   "AI Branch Health",
			content: renderBranchHealth(msg.report),
		}
		a.setNotification(msg.report.Summary, false)
		return a, nil

	case aiExplainStartMsg:
		if msg.err != nil {
			a.setNotification(msg.err.Error(), true)
			return a, nil
		}
		a.stopAIExplain()
		a.aiExplainTokens = msg.stream.Tokens
		a.aiExplainErrs = msg.stream.Err
		a.aiExplainStop = msg.cancel
		a.aiExplainText = msg.title + "\n\n"
		a.diff.SetContent(a.aiExplainText)
		a.diff.GotoTop()
		a.setNotification("AI explanation streaming...", false)
		return a, waitAIExplainTokenCmd(ai.StreamResult{Tokens: a.aiExplainTokens, Err: a.aiExplainErrs})

	case aiExplainTokenMsg:
		if a.aiExplainTokens == nil || a.aiExplainErrs == nil {
			return a, nil
		}
		a.aiExplainText += msg.token
		a.diff.SetContent(a.aiExplainText)
		a.diff.GotoBottom()
		return a, waitAIExplainTokenCmd(ai.StreamResult{Tokens: a.aiExplainTokens, Err: a.aiExplainErrs})

	case aiExplainDoneMsg:
		if msg.err != nil && !errors.Is(msg.err, context.Canceled) {
			a.setNotification(msg.err.Error(), true)
		} else if a.aiExplainText != "" {
			a.setNotification("AI explanation complete", false)
		}
		a.stopAIExplain()
		return a, nil

	case tea.KeyMsg:
		if a.aiView.active && msg.String() == "esc" {
			a.aiView.active = false
			return a, nil
		}
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
			if msg.String() == "ctrl+a" && a.prompt.mode == promptCommit {
				quit, cmd := a.handleKey(msg)
				if quit || cmd != nil {
					return a, cmd
				}
			}
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

//nolint:gocognit,gocyclo,cyclop,funlen,maintidx // Keyboard routing stays centralized so keybindings remain easy to audit.
func (a *App) handleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	key := msg.String()

	if a.aiView.active && key == "esc" {
		a.aiView.active = false
		return false, nil
	}

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

	case "ctrl+a":
		if a.prompt.mode != promptCommit {
			return false, nil
		}
		if a.advisor == nil || !a.advisor.Available() {
			a.setNotification("AI commit: install Ollama free at ollama.ai", true)
			return false, nil
		}
		a.loading = true
		a.loadingLabel = "AI commit"
		return false, a.aiCommitCmd()

	case "w":
		a.wordDiff = !a.wordDiff
		mode := "line"
		if a.wordDiff {
			mode = "word"
		}
		a.setNotification("Diff mode: "+mode, false)
		if a.diffFromStash || a.activePanel == panelStash {
			if cmd := a.loadStashDiffCmd(); cmd != nil {
				return false, cmd
			}
		}
		return false, a.loadWorkingDiffCmd()

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

	case "g":
		a.loading = true
		return false, a.gitOpCmd("fetch", true, func(ctx context.Context) error {
			return a.repo.Fetch(ctx)
		})

	case "D":
		if a.activePanel != panelBranches {
			return false, nil
		}
		item, ok := a.selectedBranchItem()
		if !ok {
			return false, nil
		}
		if item.branch.IsHead {
			a.setNotification("Cannot delete the current branch", true)
			return false, nil
		}
		a.openPrompt(promptDeleteBranch, "Delete Branch", fmt.Sprintf("Type branch name to confirm deletion: %s", item.branch.Name), item.branch.Name)
		return false, nil

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

	case "X":
		if a.advisor == nil || !a.advisor.Available() {
			a.setNotification("AI unavailable. Install Ollama free at ollama.ai", true)
			return false, nil
		}
		target, ok := a.mergeRiskTarget()
		if !ok {
			a.setNotification("X works on feature/release/hotfix branches", true)
			return false, nil
		}
		a.loading = true
		a.loadingLabel = "AI merge risk"
		return false, a.aiMergeRiskCmd(target)

	case "B":
		if a.advisor == nil || !a.advisor.Available() {
			a.setNotification("AI unavailable. Install Ollama free at ollama.ai", true)
			return false, nil
		}
		a.loading = true
		a.loadingLabel = "AI branch health"
		return false, a.aiBranchHealthCmd()

	case "E":
		if a.activePanel != panelDiff && a.activePanel != panelStash {
			return false, nil
		}
		if a.advisor == nil || !a.advisor.Available() {
			a.setNotification("E: install Ollama free at ollama.ai", true)
			return false, nil
		}
		return false, a.aiExplainCmd()

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
	case promptDeleteBranch:
		item, ok := a.selectedBranchItem()
		if !ok {
			return nil
		}
		if value != item.branch.Name {
			a.setNotification("Delete cancelled: branch name did not match", true)
			return nil
		}
		a.loading = true
		name := item.branch.Name
		return a.gitOpCmd("delete "+name, true, func(ctx context.Context) error {
			return a.repo.DeleteBranch(ctx, name, false)
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
	if a.aiExplainText != "" {
		a.diff.SetContent(a.aiExplainText)
	} else if a.rawDiff != "" {
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
	if a.aiView.active {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, a.renderAIOverlay())
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

func (a *App) renderAIOverlay() string {
	w := min(max(70, a.width*3/4), a.width-4)
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("[Esc] Close")
	content := strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Render(a.aiView.title),
		"",
		a.aiView.content,
		"",
		hint,
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
	mode := "line"
	if a.wordDiff {
		mode = "word"
	}

	left := fmt.Sprintf("%s  diff:%s  +%d -%d", branch, mode, a.ahead, a.behind)
	if a.advisor != nil && a.advisor.Available() {
		left += "  ✦ AI:" + a.advisor.Backend
	}
	if a.loading {
		left = a.spinner.View() + " " + a.loadingLabel + "  " + left
	}

	right := a.notification
	if right == "" {
		right = "tab:focus  c:commit  ctrl+a:AI  E:explain  X:risk  B:health  g:fetch  ?:help"
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
	return "Keys: tab/shift+tab focus, enter action, s stage, u unstage, c commit, ctrl+a AI commit, a stash, E explain, X merge risk, B branch health, w toggle word diff, p push, P pull --rebase, g fetch, D delete branch, n then f/r/h create branches, F/R/H finish, r refresh, q quit"
}

func (a *App) aiCommitCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		diff, err := a.repo.Diff(ctx, "--cached")
		if err != nil {
			return aiCommitMsg{err: err}
		}
		suggestion, err := a.advisor.SuggestCommit(ctx, diff)
		return aiCommitMsg{suggestion: suggestion, err: err}
	}
}

func (a *App) aiMergeRiskCmd(targetBranch string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		risk, err := a.advisor.PredictMergeRisk(ctx, a.repo, a.currentBranch, targetBranch)
		return aiRiskMsg{risk: risk, err: err}
	}
}

func (a *App) aiExplainCmd() tea.Cmd {
	currentPanel := a.activePanel
	currentDiff := a.rawDiff
	wordDiff := a.wordDiff

	var stashRef string
	if currentPanel == panelStash {
		if item, ok := a.selectedStashItem(); ok {
			stashRef = item.entry.Ref
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

		switch currentPanel {
		case panelDiff:
			stream, err := a.advisor.ExplainDiff(ctx, currentDiff)
			if err != nil {
				cancel()
				return aiExplainStartMsg{err: err}
			}
			return aiExplainStartMsg{
				stream: stream,
				title:  "AI Diff Explanation",
				cancel: cancel,
			}
		case panelStash:
			stashDiff := currentDiff
			if strings.TrimSpace(stashDiff) == "" && stashRef != "" {
				var err error
				if wordDiff {
					stashDiff, err = a.repo.StashDiffWord(ctx, stashRef)
				} else {
					stashDiff, err = a.repo.StashDiff(ctx, stashRef)
				}
				if err != nil {
					cancel()
					return aiExplainStartMsg{err: err}
				}
			}
			stream, err := a.advisor.ExplainStash(ctx, stashDiff)
			if err != nil {
				cancel()
				return aiExplainStartMsg{err: err}
			}
			return aiExplainStartMsg{
				stream: stream,
				title:  "AI Stash Explanation",
				cancel: cancel,
			}
		default:
			cancel()
			return aiExplainStartMsg{err: errors.New("AI explanation is only available on diff or stash panels")}
		}
	}
}

func waitAIExplainTokenCmd(stream ai.StreamResult) tea.Cmd {
	return func() tea.Msg {
		token, ok := <-stream.Tokens
		if ok {
			return aiExplainTokenMsg{token: token}
		}

		var err error
		if streamErr, ok := <-stream.Err; ok {
			err = streamErr
		}
		return aiExplainDoneMsg{err: err}
	}
}

func (a *App) stopAIExplain() {
	if a.aiExplainStop != nil {
		a.aiExplainStop()
		a.aiExplainStop = nil
	}
	a.aiExplainTokens = nil
	a.aiExplainErrs = nil
}

func (a *App) aiBranchHealthCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		branches, err := a.repo.Branches(ctx)
		if err != nil {
			return aiBranchHealthMsg{err: err}
		}
		report, err := a.advisor.AnalyzeBranchHealth(ctx, branches, a.currentBranch)
		return aiBranchHealthMsg{report: report, err: err}
	}
}

func (a *App) mergeRiskTarget() (string, bool) {
	switch {
	case strings.HasPrefix(a.currentBranch, "feature/"), strings.HasPrefix(a.currentBranch, "feat/"):
		return a.cfg.DevelopBranch, true
	case strings.HasPrefix(a.currentBranch, "release/"), strings.HasPrefix(a.currentBranch, "hotfix/"):
		return a.cfg.MainBranch, true
	default:
		return "", false
	}
}

func renderBranchHealth(report *ai.BranchHealthReport) string {
	var lines []string
	lines = append(lines, report.Summary)

	if len(report.StaleBranches) > 0 {
		lines = append(lines, "", "Stale branches:")
		for _, b := range report.StaleBranches {
			lines = append(lines, "  - "+b)
		}
	}
	if len(report.RiskyBranches) > 0 {
		lines = append(lines, "", "Risky branches:")
		for _, b := range report.RiskyBranches {
			lines = append(lines, "  - "+b)
		}
	}
	if len(report.Recommendations) > 0 {
		lines = append(lines, "", "Recommendations:")
		for _, rec := range report.Recommendations {
			priority := rec.PriorityLabel()
			lines = append(lines, fmt.Sprintf("  [%s] %s: %s — %s", priority, rec.Branch, rec.Action, rec.Reason))
		}
	}
	return strings.Join(lines, "\n")
}

func renderMergeRisk(risk *ai.MergeRisk) string {
	var lines []string
	lines = append(lines, risk.Summary)
	if strings.TrimSpace(risk.Recommendation) != "" {
		lines = append(lines, "", "Recommendation:", risk.Recommendation)
	}
	if len(risk.ConflictFiles) == 0 {
		return strings.Join(lines, "\n")
	}

	lines = append(lines, "", "Conflict files:")
	for _, file := range risk.ConflictFiles {
		line := "- " + file.Path
		if file.Explanation != "" {
			line += ": " + file.Explanation
		} else if file.HunkCount > 0 {
			line += fmt.Sprintf(": %d predicted conflict hunk(s)", file.HunkCount)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
