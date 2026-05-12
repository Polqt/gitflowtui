package tui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Polqt/gitflowtui/ai"
	"github.com/Polqt/gitflowtui/git"
	"github.com/Polqt/gitflowtui/gitflow"
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

	totalW := a.width

	// ── height budget ─────────────────────────────────────────────────────────
	// titleBar=1, breadcrumb=1, cardRow=5(3content+2border), statusLine=1, navBar=3(1content+2border)
	usedRows := 11
	if a.showHelp {
		usedRows++
	}
	bodyH := max(8, a.height-usedRows)

	// ── column widths (outer, including borders) ──────────────────────────────
	// LEFT=22%, CENTER=48%, RIGHT=30% — right gets remainder to fill exactly totalW
	leftW := max(22, (totalW*22)/100)
	rightW := max(26, (totalW*30)/100)
	centerW := totalW - leftW - rightW // exact remainder, no gaps

	// ── title bar ─────────────────────────────────────────────────────────────
	titleBar := a.renderTitleBar(totalW)

	// ── breadcrumb bar ────────────────────────────────────────────────────────
	breadcrumb := a.renderBreadcrumb(totalW)

	// ── stats cards row ───────────────────────────────────────────────────────
	cardRow := a.renderStatsCards(totalW)

	// ── LEFT column: BRANCHES + COMMANDS ─────────────────────────────────────
	// cmdPalette is a fixed-height box: title+3rows+prompt = 5 content lines + 2 border = 7 outer rows
	cmdPaletteOuterH := 7
	branchOuterH := bodyH - cmdPaletteOuterH
	branchInnerW := max(1, leftW-2)
	branchInnerH := max(1, branchOuterH-2)

	a.branches.SetSize(branchInnerW, branchInnerH)
	branchContent := a.branches.View()
	if len(a.branches.Items()) == 0 {
		branchContent = emptyState("No branches found", "run: git init")
	}

	leftCol := lipgloss.JoinVertical(lipgloss.Left,
		a.renderSection("BRANCHES", branchContent, leftW, branchOuterH, a.activePanel == panelBranches),
		a.renderCmdPalette(leftW, cmdPaletteOuterH),
	)

	// ── CENTER column: COMMIT LOG + STASH ────────────────────────────────────
	stashOuterH := max(5, bodyH/5)
	logOuterH := bodyH - stashOuterH
	centerInnerW := max(1, centerW-2)

	a.log.SetSize(centerInnerW, max(1, logOuterH-2))
	a.stash.SetSize(centerInnerW, max(1, stashOuterH-2))

	logContent := a.log.View()
	if len(a.log.Items()) == 0 {
		logContent = emptyState("No commits yet", "c: create first commit")
	}
	stashContent := a.stash.View()
	if len(a.stash.Items()) == 0 {
		stashContent = emptyState("No stashes", "a: stash changes")
	}

	centerCol := lipgloss.JoinVertical(lipgloss.Left,
		a.renderSection("COMMIT LOG", logContent, centerW, logOuterH, a.activePanel == panelLog),
		a.renderSection("STASH", stashContent, centerW, stashOuterH, a.activePanel == panelStash),
	)

	// ── RIGHT column: ACTIVITY + STATUS + DIFF ───────────────────────────────
	activityOuterH := max(5, bodyH/3)
	statusOuterH := max(5, bodyH/4)
	diffOuterH := bodyH - activityOuterH - statusOuterH
	rightInnerW := max(1, rightW-2)

	a.status.SetSize(rightInnerW, max(1, statusOuterH-2))
	a.diff.Width = rightInnerW
	a.diff.Height = max(1, diffOuterH-2)
	if a.aiExplainText != "" {
		a.diff.SetContent(a.aiExplainText)
	} else if a.rawDiff != "" {
		a.diff.SetContent(colorizeDiff(a.rawDiff, rightInnerW))
	}

	statusContent := a.status.View()
	if len(a.status.Items()) == 0 {
		statusContent = emptyState("Working tree clean", "nothing to commit")
	}
	diffContent := a.diff.View()
	if strings.TrimSpace(a.rawDiff) == "" && a.aiExplainText == "" {
		diffContent = emptyState("No diff", "select a file in Status")
	}
	activityContent := a.renderActivityLog(rightInnerW, max(1, activityOuterH-2))

	rightCol := lipgloss.JoinVertical(lipgloss.Left,
		a.renderSection("ACTIVITY", activityContent, rightW, activityOuterH, false),
		a.renderSection("STATUS", statusContent, rightW, statusOuterH, a.activePanel == panelStatus),
		a.renderSection("DIFF", diffContent, rightW, diffOuterH, a.activePanel == panelDiff),
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, centerCol, rightCol)

	// ── navbar (spans full width, no extra border overhead wrapping) ──────────
	navBar := a.renderNavbar(totalW)

	// ── compose ───────────────────────────────────────────────────────────────
	var parts []string
	parts = append(parts, titleBar, breadcrumb, cardRow, body)
	if a.showHelp {
		parts = append(parts, a.styles.help.Render(helpLine()))
	}
	parts = append(parts, a.statusLine(), navBar)

	ui := strings.Join(parts, "\n")

	// ── overlays ──────────────────────────────────────────────────────────────
	if a.prompt.active {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, a.renderPrompt())
	}
	if a.prForm.active {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center,
			a.prForm.view(min(a.width-4, 90), min(a.height-4, 32)))
	}
	if a.aiView.active {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, a.renderAIOverlay())
	}

	return ui
}

// ── title bar ─────────────────────────────────────────────────────────────────

func (a *App) renderTitleBar(width int) string {
	icon := lipgloss.NewStyle().Foreground(accentCyan).Bold(true).Render("◈")
	appName := lipgloss.NewStyle().Foreground(accentCyan).Bold(true).Render("GIT TERMINAL")
	sep := lipgloss.NewStyle().Foreground(textDim).Render("  /  ")
	subtitle := lipgloss.NewStyle().Foreground(textSecondary).Render("gitflow-tui")

	right := ""
	if a.loading {
		right = a.spinner.View() + " " +
			lipgloss.NewStyle().Foreground(accentMagenta).Bold(true).Render(a.loadingLabel)
	}

	left := icon + "  " + appName + sep + subtitle
	content := left
	if right != "" {
		gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(right)-2)
		content = left + strings.Repeat(" ", gap) + right
	}

	return lipgloss.NewStyle().
		Foreground(textPrimary).
		Width(width).
		Padding(0, 1).
		Render(content)
}

// ── breadcrumb bar ────────────────────────────────────────────────────────────

func (a *App) renderBreadcrumb(width int) string {
	sep := lipgloss.NewStyle().Foreground(textDim).Render("  ·  ")

	branch := a.currentBranch
	if branch == "" {
		branch = "DETACHED"
	}
	branchIcon := lipgloss.NewStyle().Foreground(accentCyan).Render("⎇")
	branchText := lipgloss.NewStyle().Foreground(textPrimary).Bold(true).Render(branch)

	var syncLabel string
	var syncStyle lipgloss.Style
	switch {
	case a.behind > 0:
		syncLabel = fmt.Sprintf("behind %d", a.behind)
		syncStyle = lipgloss.NewStyle().Foreground(accentOrange)
	case a.ahead > 0:
		syncLabel = fmt.Sprintf("ahead %d", a.ahead)
		syncStyle = lipgloss.NewStyle().Foreground(accentCyan)
	default:
		syncLabel = "synced"
		syncStyle = lipgloss.NewStyle().Foreground(accentGreen)
	}
	syncPill := syncStyle.Render(syncLabel)

	left := branchIcon + " " + branchText + sep + syncPill

	// Right-aligned indicators.
	rightParts := make([]string, 0, 2)
	if a.advisor != nil && a.advisor.Available() {
		rightParts = append(rightParts, lipgloss.NewStyle().Foreground(accentMagenta).Bold(true).Render("✦ AI"))
	}
	rightParts = append(rightParts, lipgloss.NewStyle().Foreground(textDim).Render("⚡ v1"))

	right := strings.Join(rightParts, "  ")
	gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(right)-4)
	content := left + strings.Repeat(" ", gap) + right

	return lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Render(content)
}

// ── stats cards ───────────────────────────────────────────────────────────────

func (a *App) renderStatsCards(totalW int) string {
	// Divide totalW into 3 outer card widths. Last card gets remainder.
	// Each card: border(2) + padding(2) = 4 overhead → innerW = outerW - 4.
	outerW1 := totalW / 3
	outerW2 := totalW / 3
	outerW3 := totalW - outerW1 - outerW2

	headVal := a.currentBranch
	if headVal == "" {
		headVal = "detached"
	}
	trackingInfo := ""
	for _, item := range a.branches.Items() {
		if bi, ok := item.(branchItem); ok && bi.branch.IsHead && bi.branch.Upstream != "" {
			trackingInfo = "● " + bi.branch.Upstream
			break
		}
	}
	card1 := renderStatsCard("LOCAL HEAD", headVal, trackingInfo, outerW1, accentCyan)

	commitCount := len(a.log.Items())
	syncSub := ""
	if a.ahead > 0 || a.behind > 0 {
		syncSub = fmt.Sprintf("↑%d  ↓%d", a.ahead, a.behind)
	}
	card2 := renderStatsCard("COMMITS", strconv.Itoa(commitCount), syncSub, outerW2, textPrimary)

	stashItems := a.stash.Items()
	stashLabel := "empty"
	stashColor := textDim
	stashSub := ""
	if len(stashItems) > 0 {
		stashLabel = fmt.Sprintf("%d stashed", len(stashItems))
		stashColor = accentYellow
		if si, ok := stashItems[0].(stashItem); ok {
			stashSub = truncateString(si.entry.Message, outerW3-6)
		}
	}
	card3 := renderStatsCard("STASH", stashLabel, stashSub, outerW3, stashColor)

	return lipgloss.JoinHorizontal(lipgloss.Top, card1, card2, card3)
}

// renderStatsCard renders a card with outerW = total width including border+padding.
func renderStatsCard(label, value, sub string, outerW int, valueColor lipgloss.Color) string {
	// border=2, padding left+right=2 → content width = outerW-4
	innerW := max(1, outerW-4)
	labelLine := lipgloss.NewStyle().Foreground(textDim).Render(truncateString(label, innerW))
	valueLine := lipgloss.NewStyle().Foreground(valueColor).Bold(true).Render(truncateString(value, innerW))
	// Always 3 content lines for equal height.
	subLine := lipgloss.NewStyle().Foreground(textSecondary).Render(truncateString(sub, innerW))
	content := labelLine + "\n" + valueLine + "\n" + subLine
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bgBorder).
		Padding(0, 1).
		Width(innerW).
		Render(content)
}

// ── activity log panel ────────────────────────────────────────────────────────

func (a *App) renderActivityLog(width, height int) string {
	if len(a.activityLog) == 0 {
		return emptyState("No activity yet", "actions will appear here")
	}

	dotColors := map[string]lipgloss.Color{
		"↑": accentCyan,
		"↓": accentMagenta,
		"⟳": accentCyan,
		"⎇": accentCyan,
		"●": accentGreen,
		"+": accentGreen,
		"-": accentOrange,
		"⊡": accentYellow,
		"✕": accentRed,
		"◆": accentCyan,
		"✗": accentRed,
		"■": textSecondary,
	}

	lines := make([]string, 0, min(height, len(a.activityLog)))

	start := len(a.activityLog) - height
	if start < 0 {
		start = 0
	}
	for i := len(a.activityLog) - 1; i >= start; i-- {
		entry := a.activityLog[i]

		dotColor := textSecondary
		if c, ok := dotColors[entry.icon]; ok {
			dotColor = c
		}
		dot := lipgloss.NewStyle().Foreground(dotColor).Bold(true).Render("●")

		textStyle := lipgloss.NewStyle().Foreground(textPrimary)
		if entry.isError {
			textStyle = lipgloss.NewStyle().Foreground(accentRed)
		}
		text := textStyle.Render(truncateString(entry.text, max(1, width-4)))

		line := dot + " " + text
		lines = append(lines, line)

		if len(lines) >= height {
			break
		}
	}

	return strings.Join(lines, "\n")
}

// ── command palette ───────────────────────────────────────────────────────────
// outerW/outerH include the border (2 chars each axis).

func (a *App) renderCmdPalette(outerW, outerH int) string {
	innerW := max(1, outerW-2)
	innerH := max(1, outerH-2)

	titleStyle := lipgloss.NewStyle().Foreground(textSecondary).Bold(true)

	kStyle := lipgloss.NewStyle().
		Foreground(bgBase).
		Background(accentCyan).
		Bold(true).
		Padding(0, 1)
	lStyle := lipgloss.NewStyle().Foreground(textPrimary)
	hintStyle := lipgloss.NewStyle().Foreground(textDim)

	keyRow := func(k, label, hint string) string {
		kRendered := kStyle.Render(k)
		lRendered := lStyle.Render(label)
		hRendered := hintStyle.Render(hint)
		space := max(1, innerW-lipgloss.Width(kRendered)-lipgloss.Width(lRendered)-lipgloss.Width(hRendered)-2)
		return kRendered + " " + lRendered + strings.Repeat(" ", space) + hRendered
	}

	rows := []string{
		titleStyle.Render("COMMANDS"),
		keyRow("N", "New feature", "TAB"),
		keyRow("S", "Sync remote", "S"),
		keyRow("P", "Push commit", "P"),
		lipgloss.NewStyle().Foreground(accentCyan).Bold(true).Render("›") +
			lipgloss.NewStyle().Foreground(textDim).Render(" Enter git command..."),
	}

	return a.styles.panel.Width(innerW).Height(innerH).Render(strings.Join(rows, "\n"))
}

// ── bottom navbar ─────────────────────────────────────────────────────────────

func (a *App) renderNavbar(width int) string {
	items := []struct {
		key    string
		label  string
		active bool
	}{
		{"F", "Feature", strings.HasPrefix(a.currentBranch, branchPrefixFeature) || strings.HasPrefix(a.currentBranch, branchPrefixFeat)},
		{"R", "Release", strings.HasPrefix(a.currentBranch, branchPrefixRelease)},
		{"H", "Hotfix", strings.HasPrefix(a.currentBranch, branchPrefixHotfix)},
		{"S", "Sync", false},
		{"Q", "Quit", false},
	}

	kStyle := lipgloss.NewStyle().Foreground(textDim)
	activeKStyle := lipgloss.NewStyle().Foreground(accentCyan).Bold(true)

	tabs := make([]string, 0, len(items))
	for _, item := range items {
		ks := kStyle
		ls := lipgloss.NewStyle().Foreground(textSecondary)
		if item.active {
			ks = activeKStyle
			ls = lipgloss.NewStyle().Foreground(textPrimary).Bold(true)
		}
		tab := ks.Render("["+item.key+"]") + " " + ls.Render(item.label)
		tabs = append(tabs, tab)
	}

	inner := strings.Join(tabs, "   ")

	// Right side indicators.
	diffMode := "line"
	diffColor := textDim
	if a.wordDiff {
		diffMode = "word"
		diffColor = accentCyan
	}
	rightParts := lipgloss.NewStyle().Foreground(textDim).Render("diff:") +
		lipgloss.NewStyle().Foreground(diffColor).Render(diffMode)

	// Width is content width inside a rounded border (border = 2 chars).
	innerW := max(1, width-4) // -2 border left/right, -2 padding left/right
	gap := max(1, innerW-lipgloss.Width(inner)-lipgloss.Width(rightParts))
	content := inner + strings.Repeat(" ", gap) + rightParts

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bgBorder).
		Padding(0, 1).
		Width(innerW).
		Render(content)
}

// ── sections ──────────────────────────────────────────────────────────────────
// outerW = total width including border (2 chars). outerH = total height including border (2 rows).

func (a *App) renderSection(name, content string, outerW, outerH int, focused bool) string {
	innerW := max(1, outerW-2)
	innerH := max(1, outerH-2)

	label := strings.ReplaceAll(name, "_", " ")
	var titleStyle lipgloss.Style
	if focused {
		titleStyle = lipgloss.NewStyle().Foreground(accentCyan).Bold(true)
	} else {
		titleStyle = lipgloss.NewStyle().Foreground(textSecondary).Bold(true)
	}
	title := titleStyle.Render(label)

	// Reserve 1 line for title, rest for content.
	body := renderPanelBody(content, max(1, innerH-1))
	panelContent := title + "\n" + body

	style := a.styles.panel
	if focused {
		style = a.styles.panelFocused
	}
	// Width sets content width; border adds 2 → total = outerW. Height sets content height; border adds 2 → total = outerH.
	return style.Width(innerW).Height(innerH).Render(panelContent)
}

// ── overlays ──────────────────────────────────────────────────────────────────

func (a *App) renderPrompt() string {
	w := min(max(52, a.width*2/3), a.width-4)
	innerW := max(1, w-4)

	a.prompt.input.Width = innerW - 2

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accentCyan)
	hintStyle := lipgloss.NewStyle().Foreground(textSecondary)
	keyStyle := lipgloss.NewStyle().Foreground(accentCyan)

	keys := keyStyle.Render("[Enter]") + hintStyle.Render(" confirm  ") +
		keyStyle.Render("[Esc]") + hintStyle.Render(" cancel")

	example := promptExample(a.prompt.mode)

	rows := []string{
		titleStyle.Render(a.prompt.title),
		"",
		hintStyle.Render(a.prompt.hint),
	}
	if example != "" {
		rows = append(rows, hintStyle.Render(example))
	}
	rows = append(rows, "", a.prompt.input.View(), "", keys)

	content := strings.Join(rows, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(accentCyan).
		Background(bgElevated).
		Width(w - 2).
		Render(content)
}

func promptExample(mode promptMode) string {
	switch mode {
	case promptFeatureName:
		return "e.g. login-redesign  ->  feature/login-redesign"
	case promptReleaseVersion:
		return "e.g. 1.2.0  ->  release/1.2.0"
	case promptHotfixVersion:
		return "e.g. 1.2.1  ->  hotfix/1.2.1"
	case promptCommit:
		return "e.g. feat(auth): add JWT refresh"
	case promptDeleteBranch:
		return "Type the branch name exactly to confirm"
	default:
		return ""
	}
}

func (a *App) renderAIOverlay() string {
	w := min(max(70, a.width*3/4), a.width-4)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accentMagenta)
	hintStyle := lipgloss.NewStyle().Foreground(textSecondary)

	hint := hintStyle.Render("[Esc] close")

	content := strings.Join([]string{
		titleStyle.Render("✦ " + a.aiView.title),
		"",
		a.aiView.content,
		"",
		hint,
	}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(accentMagenta).
		Background(bgElevated).
		Width(w - 2).
		Render(content)
}

// ── status bar ────────────────────────────────────────────────────────────────

func (a *App) statusLine() string {
	branch := a.currentBranch
	if branch == "" {
		branch = "(detached)"
	}

	branchStyle := lipgloss.NewStyle().Foreground(accentCyan).Bold(true)
	left := branchStyle.Render(branch)

	if a.ahead > 0 {
		left += " " + lipgloss.NewStyle().Foreground(accentGreen).Render(fmt.Sprintf("↑%d", a.ahead))
	}
	if a.behind > 0 {
		left += " " + lipgloss.NewStyle().Foreground(accentOrange).Render(fmt.Sprintf("↓%d", a.behind))
	}

	diffMode := lipgloss.NewStyle().Foreground(textDim).Render("line")
	if a.wordDiff {
		diffMode = lipgloss.NewStyle().Foreground(accentCyan).Render("word")
	}
	left += "  " + lipgloss.NewStyle().Foreground(textDim).Render("diff:") + diffMode

	if a.loading {
		left = a.spinner.View() + " " +
			lipgloss.NewStyle().Foreground(accentMagenta).Bold(true).Render(a.loadingLabel) +
			"  " + left
	}

	right := a.notification
	if right == "" {
		muted := lipgloss.NewStyle().Foreground(textDim)
		key := lipgloss.NewStyle().Foreground(accentCyan)
		right = key.Render("n") + muted.Render(":new  ") +
			key.Render("c") + muted.Render(":commit  ") +
			key.Render("s") + muted.Render(":stage  ") +
			key.Render("p") + muted.Render(":push  ") +
			key.Render("g") + muted.Render(":fetch  ") +
			key.Render("?") + muted.Render(":help")
	}

	sep := lipgloss.NewStyle().Foreground(textDim).Render("  │  ")
	combined := left + sep + right
	combined = truncateString(combined, max(1, a.width-4))

	if a.notifError {
		return a.styles.statusError.Render(combined)
	}
	return a.styles.statusNormal.Render(combined)
}

// ── helpers ───────────────────────────────────────────────────────────────────

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

func emptyState(primary, hint string) string {
	line1 := lipgloss.NewStyle().Foreground(textSecondary).Render(primary)
	line2 := lipgloss.NewStyle().Foreground(textDim).Render(hint)
	return line1 + "\n" + line2
}

func helpLine() string {
	key := func(k string) string {
		return lipgloss.NewStyle().Foreground(accentCyan).Bold(true).Render(k)
	}
	sep := lipgloss.NewStyle().Foreground(textDim).Render("  ")
	return strings.Join([]string{
		key("tab") + " focus",
		key("enter") + " action",
		key("s") + " stage",
		key("u") + " unstage",
		key("c") + " commit",
		key("ctrl+a") + " AI commit",
		key("a") + " stash",
		key("E") + " explain",
		key("X") + " merge risk",
		key("B") + " branch health",
		key("w") + " word diff",
		key("p") + " push",
		key("P") + " pull --rebase",
		key("g") + " fetch",
		key("D") + " delete branch",
		key("n") + " then " + key("f/r/h") + " new branch",
		key("F/R/H") + " finish",
		key("r") + " refresh",
		key("q") + " quit",
	}, sep)
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
	case strings.HasPrefix(a.currentBranch, branchPrefixFeature), strings.HasPrefix(a.currentBranch, branchPrefixFeat):
		return a.cfg.DevelopBranch, true
	case strings.HasPrefix(a.currentBranch, branchPrefixRelease), strings.HasPrefix(a.currentBranch, branchPrefixHotfix):
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
