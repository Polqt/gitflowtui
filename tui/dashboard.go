package tui

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Polqt/gitflowtui/ai"
	"github.com/Polqt/gitflowtui/git"
	"github.com/Polqt/gitflowtui/gitflow"
	"github.com/Polqt/gitflowtui/tui/components"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	branchPrefixFeature = "feature/"
	branchPrefixFeat    = "feat/"
	branchPrefixRelease = "release/"
	branchPrefixHotfix  = "hotfix/"
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

// ── Update ────────────────────────────────────────────────────────────────────

//nolint:gocognit,gocyclo,cyclop,funlen,maintidx
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
		a.logOpResult(msg)
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

// ── handleKey ─────────────────────────────────────────────────────────────────

//nolint:gocognit,gocyclo,cyclop,funlen,maintidx
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
		if a.showHelp {
			a.helpVP.SetContent(helpContent())
			a.helpVP.GotoTop()
		}
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
		a.openPrompt(promptDeleteBranch, "Delete Branch", "Type branch name to confirm deletion: "+item.branch.Name, item.branch.Name)
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

// ── submitPrompt ──────────────────────────────────────────────────────────────

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

// ── View ──────────────────────────────────────────────────────────────────────

func (a *App) View() string {
	if a.width <= 0 || a.height <= 0 {
		return "loading..."
	}

	totalW := a.width

	// ── height budget ─────────────────────────────────────────────────────────
	fixedRows := 12
	bodyH := max(3, a.height-fixedRows)

	// ── column widths ─────────────────────────────────────────────────────────
	leftW   := max(22, (totalW*20)/100)
	rightW  := max(28, (totalW*35)/100)
	centerW := totalW - leftW - rightW

	// ── header (component) ────────────────────────────────────────────────────
	header := components.Render(components.HeaderProps{
		AppName:    "gitflowy",
		RepoName:   a.repoName,
		Branch:     a.currentBranch,
		Ahead:      a.ahead,
		Behind:     a.behind,
		Loading:    a.loading,
		LoadingLbl: a.loadingLabel,
		AIEnabled:  a.advisor != nil && a.advisor.Available(),
		Version:    "v1.2.0",
		Spinner:    a.spinner,
		Width:      totalW,
	})

	// ── stat cards ────────────────────────────────────────────────────────────
	cards := a.renderCards(totalW)

	// ── LEFT: branches + commands ─────────────────────────────────────────────
	cmdH       := 20
	branchOutH := bodyH - cmdH
	branchInW  := max(1, leftW-2)
	branchInH  := max(1, branchOutH-2)

	a.branches.SetSize(branchInW, branchInH)
	branchContent := a.branches.View()
	if len(a.branches.Items()) == 0 {
		branchContent = emptyState("No branches found", "run: git init")
	}
	branchCount := len(a.branches.Items())
	branchSection := a.renderBranchSection(branchContent, branchCount, leftW, branchOutH)
	leftCol := lipgloss.JoinVertical(lipgloss.Left,
		branchSection,
		a.renderCmdPalette(leftW, cmdH),
	)

	// ── CENTER: commit log + stash ────────────────────────────────────────────
	stashOutH := max(5, bodyH/5)
	logOutH   := bodyH - stashOutH
	centerInW := max(1, centerW-2)

	a.log.SetSize(centerInW, max(1, logOutH-2))
	a.stash.SetSize(centerInW, max(1, stashOutH-2))

	logContent := a.log.View()
	if len(a.log.Items()) == 0 {
		logContent = emptyState("No commits yet", "c: create first commit")
	}
	stashContent := a.stash.View()
	if len(a.stash.Items()) == 0 {
		stashContent = emptyState("No stashes", "a: stash changes")
	}
	centerCol := lipgloss.JoinVertical(lipgloss.Left,
		a.renderSection("COMMIT LOG", logContent, centerW, logOutH, a.activePanel == panelLog),
		a.renderSection("STASH", stashContent, centerW, stashOutH, a.activePanel == panelStash),
	)

	// ── RIGHT: activity + status + diff ──────────────────────────────────────
	actH    := max(5, bodyH/3)
	statH   := max(5, bodyH/4)
	diffH   := bodyH - actH - statH
	rightInW := max(1, rightW-2)

	a.status.SetSize(rightInW, max(1, statH-2))
	a.diff.Width  = rightInW
	a.diff.Height = max(1, diffH-2)
	if a.aiExplainText != "" {
		a.diff.SetContent(a.aiExplainText)
	} else if a.rawDiff != "" {
		a.diff.SetContent(colorizeDiff(a.rawDiff, rightInW))
	}

	statusContent := a.status.View()
	if len(a.status.Items()) == 0 {
		statusContent = emptyState("Working tree clean", "nothing to commit")
	}
	diffContent := a.diff.View()
	if strings.TrimSpace(a.rawDiff) == "" && a.aiExplainText == "" {
		diffContent = emptyState("No diff", "select a file in Status")
	}
	actContent := a.renderActivityLog(rightInW, max(1, actH-2))

	rightCol := lipgloss.JoinVertical(lipgloss.Left,
		a.renderActivitySection(actContent, rightW, actH),
		a.renderStatusSection(statusContent, rightW, statH),
		a.renderDiffSection(diffContent, rightW, diffH),
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, centerCol, rightCol)

	// ── footer (component) ────────────────────────────────────────────────────
	footer := components.RenderFooter(components.FooterProps{
		Loading:      a.loading,
		LoadingLbl:   a.loadingLabel,
		Notification: a.notification,
		NotifError:   a.notifError,
		Width:        totalW,
		Spinner:      a.spinner,
	})

	// ── compose ───────────────────────────────────────────────────────────────
	blank := strings.Repeat(" ", totalW)
	var parts []string
	parts = append(parts, header, blank, cards, blank, body, blank, footer)

	// Clamp to terminal height — prevents any overflow pushing the header off screen.
	allLines := strings.Split(strings.Join(parts, "\n"), "\n")
	if len(allLines) > a.height {
		allLines = allLines[:a.height]
	}
	ui := strings.Join(allLines, "\n")

	// ── overlays ──────────────────────────────────────────────────────────────
	if a.prompt.active {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center,
			a.renderPrompt(), lipgloss.WithWhitespaceBackground(bgBase))
	}
	if a.prForm.active {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center,
			a.prForm.view(min(a.width-4, 90), min(a.height-4, 32)),
			lipgloss.WithWhitespaceBackground(bgBase))
	}
	if a.aiView.active {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center,
			a.renderAIOverlay(), lipgloss.WithWhitespaceBackground(bgBase))
	}

	return ui
}

// ── misc cmds ─────────────────────────────────────────────────────────────────

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
	if a.showHelp {
		var cmd tea.Cmd
		a.helpVP, cmd = a.helpVP.Update(msg)
		return cmd
	}
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

// ── AI cmds ───────────────────────────────────────────────────────────────────

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
	currentDiff  := a.rawDiff
	wordDiff     := a.wordDiff

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
			return aiExplainStartMsg{stream: stream, title: "AI Diff Explanation", cancel: cancel}

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
			return aiExplainStartMsg{stream: stream, title: "AI Stash Explanation", cancel: cancel}

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
	a.aiExplainErrs   = nil
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
	case strings.HasPrefix(a.currentBranch, branchPrefixFeature),
		strings.HasPrefix(a.currentBranch, branchPrefixFeat):
		return a.cfg.DevelopBranch, true
	case strings.HasPrefix(a.currentBranch, branchPrefixRelease),
		strings.HasPrefix(a.currentBranch, branchPrefixHotfix):
		return a.cfg.MainBranch, true
	default:
		return "", false
	}
}

