package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "gitflow-tui — Gitflow-aware terminal UI for git")
		fmt.Fprintln(os.Stderr, "\nUsage: gitflow-tui [path]")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Default to current directory if no path is provided
	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	cfg := config.Load()
	_ = cfg // Placeholder to avoid unused variable error; replace with actual config usage

	repo, err := git.NewRepo(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

}