# Contributing

Thanks for your interest in contributing to `gitflow-tui`.

This guide explains how to run, test, and submit changes from your machine.

## Development Prerequisites

- Go `1.25+`
- Git
- A terminal that supports TUI apps
- Optional: `golangci-lint` for local lint checks

## Clone And Run

```bash
git clone <your-fork-or-repo-url> gitflowtui
cd gitflowtui
go run ./cmd/gitflow-tui [path-to-git-repo]
```

## Build

```bash
go build -o gitflow-tui ./cmd/gitflow-tui
```

## Quick Local Demo (Windows)

This repo includes scripts to create a demo Git repo and launch the app against it:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\setup-demo.ps1
powershell -ExecutionPolicy Bypass -File scripts\try-demo.ps1
```

## Quality Checks

Run these before opening a PR:

```bash
go test ./...
go test -race ./...
go vet ./...
golangci-lint run
```

If you use `make`:

```bash
make lint
```

## Branching And Commits

- Create a feature branch from `main`
- Keep commits focused and readable
- Use clear commit messages (`feat:`, `fix:`, `chore:`, `docs:`)
- Rebase onto the latest `main` before opening a PR

## Pull Request Checklist

- Change is scoped and explained in the PR description
- Tests pass locally
- Lint and vet pass locally
- Docs updated when behavior or workflow changes
- Screenshots or terminal recording attached for TUI behavior changes

## Reporting Issues

When filing an issue, include:

- OS and terminal
- Go version (`go version`)
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or screenshots
