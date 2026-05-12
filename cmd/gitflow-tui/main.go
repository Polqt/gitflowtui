package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Polqt/gitflowtui/ai"
	"github.com/Polqt/gitflowtui/config"
	"github.com/Polqt/gitflowtui/git"
	"github.com/Polqt/gitflowtui/realtime"
	"github.com/Polqt/gitflowtui/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// Build-time variables injected by goreleaser via -ldflags.
// Named with a "build" prefix to satisfy gochecknoglobals: these are
// intentionally package-level because ldflags can only target var declarations.
//
//nolint:gochecknoglobals // ldflags injection requires package-level vars.
var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		return runServe(os.Args[2:])
	}

	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "TUI GitFlow Manager")
		fmt.Fprintln(os.Stderr, "Usage: gitflow-tui [flags] [path]")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		_, _ = fmt.Fprintf(os.Stdout, "gitflow-tui %s (%s) built %s\n", buildVersion, buildCommit, buildDate)
		return nil
	}

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	repo, err := git.NewRepo(dir)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	cfg := config.Load()
	var opts []tui.AppOption
	advisor := ai.NewWithConfig(cfg.AIKey, cfg.OllamaBaseURL, cfg.OllamaModel)
	opts = append(opts, tui.WithAdvisor(advisor))
	if advisor.Available() {
		fmt.Fprintf(os.Stderr, "AI backend: %s\n", advisor.Backend)
	} else {
		fmt.Fprintln(os.Stderr,
			"AI disabled. Set ANTHROPIC_API_KEY or install Ollama (ollama.ai) for free AI features.")
	}

	var wsServer *realtime.Server
	if cfg.WSAddr != "" {
		wsServer, err = realtime.NewServer(cfg.WSAddr, cfg.WSPath)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
		wsServer.SetLogger(log.New(os.Stderr, "", log.LstdFlags))
		if err := wsServer.Start(); err != nil {
			return fmt.Errorf("error: %w", err)
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = wsServer.Shutdown(ctx)
		}()
		opts = append(opts, tui.WithEventSink(wsServer))
	}

	app := tui.NewApp(repo, cfg, opts...)

	program := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("fatal: %w", err)
	}
	return nil
}

func runServe(args []string) error {
	cfg := config.Load()
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	repoPath := fs.String("repo", ".", "Git repository path to stream")
	addr := fs.String("addr", cfg.WSAddr, "WebSocket listen address")
	path := fs.String("path", cfg.WSPath, "WebSocket URL path")
	interval := fs.Duration("interval", 2*time.Second, "Repository polling interval")
	logLimit := fs.Int("log-limit", cfg.LogLimit, "Number of commits to include in snapshots")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Headless gitflow-tui realtime server")
		fmt.Fprintln(os.Stderr, "Usage: gitflow-tui serve [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *addr == "" {
		return errors.New("serve: websocket address cannot be empty")
	}
	if *interval <= 0 {
		return errors.New("serve: interval must be greater than zero")
	}

	repo, err := git.NewRepo(*repoPath)
	if err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	server, err := realtime.NewServer(*addr, *path)
	if err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	logger := log.New(os.Stderr, "", log.LstdFlags)
	server.SetLogger(logger)
	if err := server.Start(); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	server.Publish(tui.RealtimeEvent{
		Type:      tui.RealtimeEventStatus,
		Timestamp: time.Now().UTC(),
		RepoRoot:  repo.Root,
		Status: &tui.RealtimeStatusMsg{
			Message: "headless realtime server started",
		},
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Printf("streaming repository %s every %s", repo.Root, interval.String())
	if err := publishRepoSnapshot(ctx, server, repo, *logLimit); err != nil {
		logger.Printf("initial snapshot failed: %v", err)
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Printf("headless realtime server shutting down")
			return nil
		case <-ticker.C:
			if err := publishRepoSnapshot(ctx, server, repo, *logLimit); err != nil {
				logger.Printf("snapshot failed: %v", err)
				server.Publish(tui.RealtimeEvent{
					Type:      tui.RealtimeEventStatus,
					Timestamp: time.Now().UTC(),
					RepoRoot:  repo.Root,
					Status: &tui.RealtimeStatusMsg{
						Message: err.Error(),
						Error:   true,
					},
				})
			}
		}
	}
}

func publishRepoSnapshot(parent context.Context, server *realtime.Server, repo *git.Repo, logLimit int) error {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()

	branches, err := repo.Branches(ctx)
	if err != nil {
		return fmt.Errorf("branches: %w", err)
	}
	current, err := repo.CurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("current branch: %w", err)
	}
	commits, err := repo.LogRef(ctx, current, logLimit)
	if err != nil {
		return fmt.Errorf("commits: %w", err)
	}
	files, err := repo.Status(ctx)
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}
	stashes, err := repo.ListStashes(ctx)
	if err != nil {
		return fmt.Errorf("stashes: %w", err)
	}

	ahead, behind := 0, 0
	for _, branch := range branches {
		if branch.Name == current {
			ahead = branch.Ahead
			behind = branch.Behind
			break
		}
	}

	server.Publish(tui.RealtimeEvent{
		Type:      tui.RealtimeEventSnapshot,
		Timestamp: time.Now().UTC(),
		RepoRoot:  repo.Root,
		Branch:    current,
		Snapshot: &tui.RealtimeSnapshot{
			Ahead:    ahead,
			Behind:   behind,
			Branches: realtimeBranches(branches),
			Files:    realtimeFiles(files),
			Stashes:  realtimeStashes(stashes),
			Commits:  realtimeCommits(commits),
		},
	})
	return nil
}

func realtimeBranches(branches []git.Branch) []tui.RealtimeBranch {
	out := make([]tui.RealtimeBranch, 0, len(branches))
	for _, branch := range branches {
		out = append(out, tui.RealtimeBranch{
			Name:     branch.Name,
			IsHead:   branch.IsHead,
			Upstream: branch.Upstream,
			Ahead:    branch.Ahead,
			Behind:   branch.Behind,
		})
	}
	return out
}

func realtimeFiles(files []git.FileStatus) []tui.RealtimeFile {
	out := make([]tui.RealtimeFile, 0, len(files))
	for _, file := range files {
		state := "clean"
		switch {
		case file.IsUntracked():
			state = "untracked"
		case file.IsStaged() && file.IsUnstaged():
			state = "staged+unstaged"
		case file.IsStaged():
			state = "staged"
		case file.IsUnstaged():
			state = "unstaged"
		}
		out = append(out, tui.RealtimeFile{
			Path:  file.Path,
			Orig:  file.Orig,
			X:     string(file.X),
			Y:     string(file.Y),
			State: state,
		})
	}
	return out
}

func realtimeStashes(stashes []git.StashEntry) []tui.RealtimeStash {
	out := make([]tui.RealtimeStash, 0, len(stashes))
	for _, stash := range stashes {
		out = append(out, tui.RealtimeStash{
			Ref:          stash.Ref,
			Message:      stash.Message,
			RelativeTime: stash.RelativeTime,
		})
	}
	return out
}

func realtimeCommits(commits []git.Commit) []tui.RealtimeCommit {
	out := make([]tui.RealtimeCommit, 0, len(commits))
	for _, commit := range commits {
		out = append(out, tui.RealtimeCommit{
			Hash:    commit.Hash,
			Date:    commit.Date,
			Author:  commit.Author,
			Subject: commit.Subject,
		})
	}
	return out
}
