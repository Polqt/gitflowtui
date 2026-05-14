# Header Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the two-row header (title bar + breadcrumb) with a single prompt-style line showing app name, repo name, branch, sync status, AI badge, and version.

**Architecture:** Add `repoName` to the `App` struct (derived once from `filepath.Base(repo.Root)` in `NewApp`). Rewrite `renderTitleBar()` in `dashboard.go` to use the new format. Delete `renderBreadcrumb()` and remove its call from `View()`, decrementing `usedRows` by 1.

**Tech Stack:** Go, Bubble Tea (`github.com/charmbracelet/bubbletea`), Lip Gloss (`github.com/charmbracelet/lipgloss`), `path/filepath`

---

### Task 1: Add `repoName` to `App` struct and populate in `NewApp`

**Files:**
- Modify: `tui/app.go`

- [ ] **Step 1: Add the `repoName` field to the `App` struct**

In [tui/app.go](tui/app.go), find the `App` struct (around line 64). Add `repoName` after `advisor`:

```go
type App struct {
	repo      *git.Repo
	workflow  *gitflow.Workflow
	cfg       config.Config
	eventSink EventSink
	advisor   *ai.Advisor
	repoName  string  // last segment of repo.Root

	width  int
	// ... rest unchanged
```

- [ ] **Step 2: Add `path/filepath` import to `tui/app.go`**

In [tui/app.go](tui/app.go), the import block currently starts at line 3. Add `"path/filepath"` to it:

```go
import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Polqt/gitflowtui/ai"
	// ... rest unchanged
)
```

- [ ] **Step 3: Populate `repoName` in `NewApp`**

In [tui/app.go](tui/app.go), find `NewApp` (around line 154). After the `app := &App{...}` literal, add the repo name derivation. The `&App{...}` block ends before the `for _, opt := range opts` loop. Insert immediately after the closing `}` of the struct literal:

```go
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
```

- [ ] **Step 4: Build to verify no compile errors**

```bash
go build ./...
```

Expected: no output (clean build).

- [ ] **Step 5: Commit**

```bash
git add tui/app.go
git commit -m "feat(tui): add repoName field derived from repo root path"
```

---

### Task 2: Rewrite `renderTitleBar()` with prompt-style format

**Files:**
- Modify: `tui/dashboard.go:803-827`

- [ ] **Step 1: Replace the `renderTitleBar` function body**

In [tui/dashboard.go](tui/dashboard.go), find `renderTitleBar` (lines 803–827). Replace the entire function with:

```go
func (a *App) renderTitleBar(width int) string {
	dimStyle := lipgloss.NewStyle().Foreground(textDim)
	cyanBold := lipgloss.NewStyle().Foreground(accentCyan).Bold(true)
	primaryBold := lipgloss.NewStyle().Foreground(textPrimary).Bold(true)

	// Left: >_ gitflowy  |  repo: <name>  ·  branch: <branch>  ·  <sync>
	prompt := cyanBold.Render(">_")
	appName := cyanBold.Render("gitflowy")
	sep := dimStyle.Render("  |  ")
	repoLabel := dimStyle.Render("repo: ")
	repoName := cyanBold.Render(a.repoName)
	branchSep := dimStyle.Render("  ·  branch: ")

	branch := a.currentBranch
	if branch == "" {
		branch = "DETACHED"
	}
	branchText := primaryBold.Render(branch)

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
		syncLabel = "synced ✓"
		syncStyle = lipgloss.NewStyle().Foreground(accentGreen)
	}
	syncPill := dimStyle.Render("  ·  ") + syncStyle.Bold(true).Render(syncLabel)

	left := prompt + " " + appName + sep + repoLabel + repoName + branchSep + branchText + syncPill

	// Right: spinner+label when loading, else ✦ AI (if available) + version
	var right string
	if a.loading {
		right = a.spinner.View() + " " +
			lipgloss.NewStyle().Foreground(accentMagenta).Bold(true).Render(a.loadingLabel)
	} else {
		var rightParts []string
		if a.advisor != nil && a.advisor.Available() {
			rightParts = append(rightParts, lipgloss.NewStyle().Foreground(accentMagenta).Bold(true).Render("✦ AI"))
		}
		rightParts = append(rightParts, dimStyle.Render("⚡ v1"))
		right = strings.Join(rightParts, "  ")
	}

	gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(right)-2)
	content := left + strings.Repeat(" ", gap) + right

	return lipgloss.NewStyle().
		Foreground(textPrimary).
		Width(width).
		Padding(0, 1).
		Render(content)
}
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
go build ./...
```

Expected: no output (clean build).

- [ ] **Step 3: Commit**

```bash
git add tui/dashboard.go
git commit -m "feat(tui): rewrite renderTitleBar with prompt-style format and repo name"
```

---

### Task 3: Remove breadcrumb — delete function, update `View()`, fix `usedRows`

**Files:**
- Modify: `tui/dashboard.go`

- [ ] **Step 1: Remove the `renderBreadcrumb()` call from `View()` and fix `usedRows`**

In [tui/dashboard.go](tui/dashboard.go), find the `View()` function (around line 670). The height budget comment and `usedRows` currently read:

```go
	// ── height budget ─────────────────────────────────────────────────────────
	// titleBar=1, breadcrumb=1, cardRow=5(3content+2border), statusLine=1, navBar=3(1content+2border)
	usedRows := 11
```

Change it to:

```go
	// ── height budget ─────────────────────────────────────────────────────────
	// titleBar=1, cardRow=5(3content+2border), statusLine=1, navBar=3(1content+2border)
	usedRows := 10
```

- [ ] **Step 2: Remove the `breadcrumb` variable and its use in `parts`**

Still in `View()`, find these two lines and remove them:

```go
	// ── breadcrumb bar ────────────────────────────────────────────────────────
	breadcrumb := a.renderBreadcrumb(totalW)
```

And find the `parts` assembly (around line 777) which currently reads:

```go
	parts = append(parts, titleBar, breadcrumb, cardRow, body)
```

Change it to:

```go
	parts = append(parts, titleBar, cardRow, body)
```

- [ ] **Step 3: Delete the `renderBreadcrumb` function**

In [tui/dashboard.go](tui/dashboard.go), find and delete the entire `renderBreadcrumb` function (lines 831–873):

```go
// ── breadcrumb bar ────────────────────────────────────────────────────────

func (a *App) renderBreadcrumb(width int) string {
	// ... entire function body ...
}
```

Delete from the `// ── breadcrumb bar` comment through the closing `}` of the function.

- [ ] **Step 4: Build to verify no compile errors**

```bash
go build ./...
```

Expected: no output (clean build).

- [ ] **Step 5: Run the app to visually verify the header**

```bash
go run ./cmd/gitflow-tui
```

Expected: single-line header showing `>_ gitflowy  |  repo: gitflowtui  ·  branch: <name>  ·  synced ✓` with version right-aligned. No breadcrumb row below it. All panels render correctly with one extra row of content space.

- [ ] **Step 6: Commit**

```bash
git add tui/dashboard.go
git commit -m "feat(tui): remove breadcrumb row, save one vertical content row"
```
