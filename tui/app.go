package tui

import (
	"github.com/Polqt/gitflowtui/git"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// panel represents a UI panel in the terminal interface.
type panel int

const (
	panelBranches panel = iota
	panelLog
	panelStatus
	panelCommit
	panelStash
	panelPR
	panelPrompt // modal for user input (e.g., creating a branch, entering a commit message)
)

// pendingAction represents an action that is awaiting user input or confirmation.
type pendingAction string

const (
	actionNone       pendingAction = ""
	actionNewFeature pendingAction = "feature"
	actionNewRelease pendingAction = "release"
	actionNewHotfix  pendingAction = "hotfix"
	actionStash      pendingAction = "stash"
)

// App is the main application struct that holds the state of the TUI.
// Fields are arrange by concern: dependencies, layout, size, sub-models
type App struct {
	// Dependencies

	// Terminal Layout
	width  int
	height int

	// focus + navigation
	activePanel   panel
	previousPanel panel // restore focus after closing a modal

	// Repository data

	// sub-models

	// Transient state for user interactions
	loading      bool
	loadingLabel string
	notification string
	notifError   bool
	showHelp     bool
	pending      pendingAction

	spinner spinner.Model
}

// promptOverlay is the modal for single-line input
type promptOverlay struct {
	input  textinput.Model
	title  string
	active bool
}

func NewApp(repo git.Repository) *App {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	ti := textinput.New()
	ti.Placeholder = "Please enter a value..."
	ti.CharLimit = 120
	ti.Width = 50

	return &App{
	}
} 