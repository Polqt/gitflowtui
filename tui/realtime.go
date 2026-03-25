package tui

import (
	"time"

	"github.com/Polqt/gitflowtui/ai"
)

// AppOption configures optional App behavior.
type AppOption func(*App)

// EventSink receives realtime events emitted by the TUI.
type EventSink interface {
	Publish(event RealtimeEvent)
}

// RealtimeEvent is emitted when app state changes.
type RealtimeEvent struct {
	Type      string             `json:"type"`
	Timestamp time.Time          `json:"timestamp"`
	RepoRoot  string             `json:"repoRoot"`
	Branch    string             `json:"branch"`
	Loading   bool               `json:"loading"`
	Snapshot  *RealtimeSnapshot  `json:"snapshot,omitempty"`
	Status    *RealtimeStatusMsg `json:"status,omitempty"`
}

// RealtimeSnapshot contains a reduced repository state payload.
type RealtimeSnapshot struct {
	Ahead    int              `json:"ahead"`
	Behind   int              `json:"behind"`
	WordDiff bool             `json:"wordDiff"`
	Branches []RealtimeBranch `json:"branches"`
	Files    []RealtimeFile   `json:"files"`
	Stashes  []RealtimeStash  `json:"stashes"`
	Commits  []RealtimeCommit `json:"commits"`
}

// RealtimeStatusMsg describes UI notifications.
type RealtimeStatusMsg struct {
	Message string `json:"message"`
	Error   bool   `json:"error"`
}

// RealtimeBranch is a websocket-safe branch payload.
type RealtimeBranch struct {
	Name     string `json:"name"`
	IsHead   bool   `json:"isHead"`
	Upstream string `json:"upstream"`
	Ahead    int    `json:"ahead"`
	Behind   int    `json:"behind"`
}

// RealtimeFile is a websocket-safe file status payload.
type RealtimeFile struct {
	Path  string `json:"path"`
	Orig  string `json:"orig,omitempty"`
	X     string `json:"x"`
	Y     string `json:"y"`
	State string `json:"state"`
}

// RealtimeStash is a websocket-safe stash payload.
type RealtimeStash struct {
	Ref          string `json:"ref"`
	Message      string `json:"message"`
	RelativeTime string `json:"relativeTime"`
}

// RealtimeCommit is a websocket-safe commit payload.
type RealtimeCommit struct {
	Hash    string `json:"hash"`
	Date    string `json:"date"`
	Author  string `json:"author"`
	Subject string `json:"subject"`
}

// WithEventSink configures the app to emit realtime events.
func WithEventSink(sink EventSink) AppOption {
	return func(a *App) {
		a.eventSink = sink
	}
}

// WithAdvisor injects an AI advisor into the app.
func WithAdvisor(advisor *ai.Advisor) AppOption {
	return func(app *App) {
		app.advisor = advisor
	}
}

func (a *App) publishSnapshot(s repoSnapshot) {
	if a.eventSink == nil {
		return
	}

	branches := make([]RealtimeBranch, 0, len(s.Branches))
	for _, b := range s.Branches {
		branches = append(branches, RealtimeBranch{
			Name:     b.Name,
			IsHead:   b.IsHead,
			Upstream: b.Upstream,
			Ahead:    b.Ahead,
			Behind:   b.Behind,
		})
	}

	files := make([]RealtimeFile, 0, len(s.Files))
	for _, f := range s.Files {
		state := "clean"
		switch {
		case f.IsUntracked():
			state = "untracked"
		case f.IsStaged() && f.IsUnstaged():
			state = "staged+unstaged"
		case f.IsStaged():
			state = "staged"
		case f.IsUnstaged():
			state = "unstaged"
		}
		files = append(files, RealtimeFile{
			Path:  f.Path,
			Orig:  f.Orig,
			X:     string(f.X),
			Y:     string(f.Y),
			State: state,
		})
	}

	stashes := make([]RealtimeStash, 0, len(s.Stashes))
	for _, entry := range s.Stashes {
		stashes = append(stashes, RealtimeStash{
			Ref:          entry.Ref,
			Message:      entry.Message,
			RelativeTime: entry.RelativeTime,
		})
	}

	commits := make([]RealtimeCommit, 0, len(s.Commits))
	for _, c := range s.Commits {
		commits = append(commits, RealtimeCommit{
			Hash:    c.Hash,
			Date:    c.Date,
			Author:  c.Author,
			Subject: c.Subject,
		})
	}

	a.eventSink.Publish(RealtimeEvent{
		Type:      "snapshot",
		Timestamp: time.Now().UTC(),
		RepoRoot:  a.repo.Root,
		Branch:    s.CurrentBranch,
		Loading:   a.loading,
		Snapshot: &RealtimeSnapshot{
			Ahead:    s.Ahead,
			Behind:   s.Behind,
			WordDiff: a.wordDiff,
			Branches: branches,
			Files:    files,
			Stashes:  stashes,
			Commits:  commits,
		},
	})
}

func (a *App) publishStatus(message string, isError bool) {
	if a.eventSink == nil {
		return
	}
	a.eventSink.Publish(RealtimeEvent{
		Type:      "status",
		Timestamp: time.Now().UTC(),
		RepoRoot:  a.repo.Root,
		Branch:    a.currentBranch,
		Loading:   a.loading,
		Status: &RealtimeStatusMsg{
			Message: message,
			Error:   isError,
		},
	})
}
