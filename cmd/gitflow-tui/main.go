package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Polqt/gitflowtui/ai"
	"github.com/Polqt/gitflowtui/config"
	"github.com/Polqt/gitflowtui/git"
	"github.com/Polqt/gitflowtui/realtime"
	"github.com/Polqt/gitflowtui/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// Build-time variables injected by goreleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "TUI GitFlow Manager")
		fmt.Fprintln(os.Stderr, "Usage: gitflow-tui [flags] [path]")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Fprintf(os.Stdout, "gitflow-tui %s (%s) built %s\n", version, commit, date)
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
	advisor := ai.NewWithOptions(cfg.AIKey, cfg.OllamaModel)
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
		if err := wsServer.Start(); err != nil {
			return fmt.Errorf("error: %w", err)
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = wsServer.Shutdown(ctx)
		}()

		fmt.Fprintf(os.Stderr, "realtime websocket enabled at %s\n", wsServer.URL())
		opts = append(opts, tui.WithEventSink(wsServer))
	}

	app := tui.NewApp(repo, cfg, opts...)

	program := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("fatal: %w", err)
	}
	return nil
}
