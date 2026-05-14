package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Polqt/gitflowtui/ai"
	"github.com/Polqt/gitflowtui/config"
	"github.com/Polqt/gitflowtui/git"
	"github.com/Polqt/gitflowtui/gitflow"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type panel int

const (
	panelBranches panel = iota
	panelLog
	panelStatus
	panelStash
	panelDiff
)

type promptMode int

const (
	promptNone promptMode = iota
	promptCommit
	promptStash
	promptFeatureName
	promptReleaseVersion
	promptHotfixVersion
	promptDeleteBranch
)

type finishAction int

const (
	finishNone finishAction = iota
	finishFeature
	finishRelease
	finishHotfix
)

// ── activity log ──────────────────────────────────────────────────────────────

const activityLogMax = 50

type activityEntry struct {
	timestamp time.Time
	icon      string
	text      string
	isError   bool
}

// App is the Bubble Tea root model.
type App struct {
	repo      *git.Repo
	workflow  *gitflow.Workflow
	cfg       config.Config
	eventSink EventSink
	advisor   *ai.Advisor
	repoName  string  // last segment of repo.Root

	width  int
	height int

	activePanel panel

	branches list.Model
	log      list.Model
	status   list.Model
	stash    list.Model
	diff     viewport.Model

	prompt promptOverlay
	prForm prTemplateForm
	aiView aiOverlay

	spinner spinner.Model
	styles  uiStyles

	loading      bool
	loadingLabel string
	notification string
	notifError   bool
	showHelp     bool

	currentBranch string
	ahead         int
	behind        int
	rawDiff       string
	wordDiff      bool
	diffFromStash bool

	newBranchArmed  bool
	pendingFinish   finishAction
	aiExplainText   string
	aiExplainErrs   <-chan error
	aiExplainStop   context.CancelFunc
	aiExplainTokens <-chan string

	// Activity log — ring buffer of recent operations.
	activityLog []activityEntry
}

type promptOverlay struct {
	active bool
	mode   promptMode
	title  string
	hint   string
	input  textinput.Model
}

type aiOverlay struct {
	active  bool
	title   string
	content string
}

type repoSnapshot struct {
	Branches      []git.Branch
	Commits       []git.Commit
	Files         []git.FileStatus
	Stashes       []git.StashEntry
	CurrentBranch string
	Ahead         int
	Behind        int
}

type refreshMsg struct {
	snapshot repoSnapshot
	err      error
}

type opMsg struct {
	label   string
	err     error
	refresh bool
}

type diffMsg struct {
	text      string
	err       error
	fromStash bool
}

func NewApp(repo *git.Repo, cfg config.Config, opts ...AppOption) *App {
	workflowCfg := gitflow.DefaultConfig()
	workflowCfg.MainBranch = cfg.MainBranch
	workflowCfg.DevelopBranch = cfg.DevelopBranch
	workflowCfg.RemoteName = cfg.RemoteName

	branchList := newPanelList()
	logList := newPanelList()
	statusList := newPanelList()
	stashList := newPanelList()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(accentMagenta)

	p := textinput.New()
	p.Prompt = "> "
	p.CharLimit = 160

	app := &App{
		repo:         repo,
		workflow:     gitflow.New(repo, workflowCfg),
		cfg:          cfg,
		activePanel:  panelBranches,
		branches:     branchList,
		log:          logList,
		status:       statusList,
		stash:        stashList,
		diff:         viewport.New(0, 0),
		prompt:       promptOverlay{input: p},
		prForm:       newPRTemplateForm(),
		spinner:      sp,
		styles:       defaultStyles(),
		loadingLabel: "Refreshing",
		activityLog:  make([]activityEntry, 0, activityLogMax),
	}

	// Derive repo name from the root directory path.
	app.repoName = filepath.Base(repo.Root)
	if app.repoName == "" || app.repoName == "." {
		app.repoName = "repo"
	}

	for _, opt := range opts {
		opt(app)
	}
	if app.advisor == nil {
		app.advisor = ai.New("")
	}

	return app
}

func newPanelList() list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	delegate.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(textSecondary).
		PaddingLeft(1)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Background(bgElevated).
		Foreground(accentCyan).
		Bold(true).
		PaddingLeft(0).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(accentCyan)

	m := list.New([]list.Item{}, delegate, 0, 0)
	m.SetShowTitle(false)
	m.SetShowFilter(false)
	m.SetShowStatusBar(false)
	m.SetShowPagination(false)
	m.SetShowHelp(false)
	m.SetFilteringEnabled(false)
	return m
}

func (a *App) Init() tea.Cmd {
	a.loading = true
	return tea.Batch(a.spinner.Tick, a.refreshCmd())
}

func (a *App) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		branches, err := a.repo.Branches(ctx)
		if err != nil {
			return refreshMsg{err: err}
		}

		current, err := a.repo.CurrentBranch(ctx)
		if err != nil {
			return refreshMsg{err: err}
		}

		commits, err := a.repo.LogRef(ctx, current, a.cfg.LogLimit)
		if err != nil {
			return refreshMsg{err: err}
		}

		files, err := a.repo.Status(ctx)
		if err != nil {
			return refreshMsg{err: err}
		}

		stashes, err := a.repo.ListStashes(ctx)
		if err != nil {
			return refreshMsg{err: err}
		}

		ahead, behind := 0, 0
		for _, b := range branches {
			if b.Name == current {
				ahead, behind = b.Ahead, b.Behind
				break
			}
		}

		return refreshMsg{snapshot: repoSnapshot{
			Branches:      branches,
			Commits:       commits,
			Files:         files,
			Stashes:       stashes,
			CurrentBranch: current,
			Ahead:         ahead,
			Behind:        behind,
		}}
	}
}

func (a *App) gitOpCmd(label string, refresh bool, fn func(context.Context) error) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		err := fn(ctx)
		return opMsg{label: label, err: err, refresh: refresh}
	}
}

func (a *App) loadWorkingDiffCmd() tea.Cmd {
	item, ok := a.selectedFileItem()
	if !ok {
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			var (
				out string
				err error
			)
			if a.wordDiff {
				out, err = a.repo.DiffWord(ctx, "")
			} else {
				out, err = a.repo.Diff(ctx, "")
			}
			return diffMsg{text: out, err: err, fromStash: false}
		}
	}

	path := item.file.Path
	staged := item.file.IsStaged() && !item.file.IsUnstaged()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		var (
			out string
			err error
		)
		if a.wordDiff {
			out, err = a.repo.DiffFileWord(ctx, path, staged)
		} else {
			out, err = a.repo.DiffFile(ctx, path, staged)
		}
		if err != nil {
			return diffMsg{err: err}
		}
		return diffMsg{text: out, fromStash: false}
	}
}

func (a *App) loadStashDiffCmd() tea.Cmd {
	item, ok := a.selectedStashItem()
	if !ok {
		return nil
	}
	ref := item.entry.Ref
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		var (
			out string
			err error
		)
		if a.wordDiff {
			out, err = a.repo.StashDiffWord(ctx, ref)
		} else {
			out, err = a.repo.StashDiff(ctx, ref)
		}
		return diffMsg{text: out, err: err, fromStash: true}
	}
}

func (a *App) openPrompt(mode promptMode, title, hint, placeholder string) {
	a.prompt.active = true
	a.prompt.mode = mode
	a.prompt.title = title
	a.prompt.hint = hint
	a.prompt.input.SetValue("")
	a.prompt.input.Placeholder = placeholder
	a.prompt.input.Focus()
}

func (a *App) setNotification(msg string, isError bool) {
	a.notification = strings.TrimSpace(msg)
	a.notifError = isError
	a.publishStatus(a.notification, isError)
}

func (a *App) opMessage(msg opMsg) string {
	if msg.err != nil {
		return fmt.Sprintf("%s failed: %v", msg.label, msg.err)
	}
	return msg.label + " done"
}

// ── activity log helpers ──────────────────────────────────────────────────────

func (a *App) addActivity(icon, text string, isError bool) {
	entry := activityEntry{
		timestamp: time.Now(),
		icon:      icon,
		text:      text,
		isError:   isError,
	}
	if len(a.activityLog) >= activityLogMax {
		a.activityLog = a.activityLog[1:]
	}
	a.activityLog = append(a.activityLog, entry)
}

// logOpResult logs a completed operation to the activity log.
func (a *App) logOpResult(msg opMsg) {
	if msg.err != nil {
		a.addActivity("✗", msg.label+" failed", true)
		return
	}

	icon := "■"
	switch {
	case strings.HasPrefix(msg.label, "push"):
		icon = "↑"
	case strings.HasPrefix(msg.label, "pull"):
		icon = "↓"
	case strings.HasPrefix(msg.label, "fetch"):
		icon = "⟳"
	case strings.HasPrefix(msg.label, "checkout"):
		icon = "⎇"
	case strings.HasPrefix(msg.label, "commit"):
		icon = "●"
	case strings.HasPrefix(msg.label, "stage"):
		icon = "+"
	case strings.HasPrefix(msg.label, "unstage"):
		icon = "-"
	case strings.HasPrefix(msg.label, "stash"):
		icon = "⊡"
	case strings.HasPrefix(msg.label, "delete"):
		icon = "✕"
	case strings.HasPrefix(msg.label, "new"):
		icon = "◆"
	}
	a.addActivity(icon, msg.label, false)
}
