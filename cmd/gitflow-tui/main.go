package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Polqt/gitflowtui/config"
	"github.com/Polqt/gitflowtui/git"
	"github.com/Polqt/gitflowtui/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "TUI GitFlow Manager")
		fmt.Fprintln(os.Stderr, "Usage: gitflow-tui [path]")
		flag.PrintDefaults()
	}
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	repo, err := git.NewRepo(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	cfg := config.Load()
	app := tui.NewApp(repo, cfg)

	program := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}
