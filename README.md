# GitFlow TUI Manager

A GitFlow-aware terminal UI built with Bubble Tea.

## Features
- Branch, log, status, stash, and diff panels
- Async git operations (no UI freeze)
- GitFlow start/finish flows for feature, release, and hotfix branches
- PR template form with clipboard + temp-file output

## Install

### Option 1: Build locally
```bash
go build -o gitflow-tui ./cmd/gitflow-tui
```

### Option 2: Go install
```bash
go install github.com/Polqt/gitflowtui/cmd/gitflow-tui@latest
```

## Run
```bash
gitflow-tui [path-to-repo]
```

If no path is provided, it uses the current directory.

## Keybindings
- `tab` / `shift+tab`: switch focused panel
- `r`: refresh
- `enter`: panel primary action
- `s`: stage selected file (Status panel)
- `u`: unstage selected file (Status panel)
- `c`: commit prompt
- `a`: stash prompt
- `n` then `f|r|h`: create feature/release/hotfix branch
- `F` / `R` / `H`: finish feature/release/hotfix (opens PR form)
- `p`: push current branch
- `P`: pull with rebase
- `?`: toggle help
- `q`: quit

## GitFlow Conventions
- Feature: `feature/*` or `feat/*`
- Release: `release/<semver>`
- Hotfix: `hotfix/<semver>`

## Environment Overrides
- `GITFLOW_TUI_MAIN` (default: `main`)
- `GITFLOW_TUI_DEVELOP` (default: `develop`)
- `GITFLOW_TUI_REMOTE` (default: `origin`)

## Distribution
See [docs/RELEASE.md](docs/RELEASE.md).
