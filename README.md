# gitflow-tui

A GitFlow-aware terminal UI built with Bubble Tea for day-to-day branch, status, diff, stash, and release workflows.

## About

`gitflow-tui` is a keyboard-first CLI app for developers who use GitFlow-style branching and want fewer context switches than raw Git commands or GUI tools.

## Features

- Branch, log, status, stash, and diff workflows
- Async git operations to keep UI responsive
- GitFlow start/finish flows for `feature`, `release`, and `hotfix`
- PR template form with clipboard + temp-file output
- Toggleable line diff and word diff modes

## Install

### Go install (recommended)

```bash
go install github.com/Polqt/gitflowtui/cmd/gitflow-tui@latest
```

### Build locally

```bash
go build -o gitflow-tui ./cmd/gitflow-tui
```

## Run

```bash
gitflow-tui [path-to-repo]
```

If no path is provided, the current directory is used.

## Quick Test On Your Machine

### Windows demo flow (scripted)

```powershell
powershell -ExecutionPolicy Bypass -File scripts\setup-demo.ps1
powershell -ExecutionPolicy Bypass -File scripts\try-demo.ps1
```

### Any OS (manual)

```bash
git clone <your-repo-url> gitflowtui
cd gitflowtui
go run ./cmd/gitflow-tui /path/to/any/git/repository
```

## Keybindings

- `tab` / `shift+tab`: switch focused panel
- `r`: refresh
- `enter`: panel primary action
- `s`: stage selected file (Status panel)
- `u`: unstage selected file (Status panel)
- `c`: commit prompt
- `a`: stash prompt
- `w`: toggle diff mode (line/word)
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

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, testing, linting, and PR guidance.

## Suggested GitHub Topics

Add these topics in your repository settings to improve discoverability:

- `golang`
- `cli`
- `tui`
- `bubbletea`
- `lipgloss`
- `git`
- `gitflow`
- `developer-tools`
- `terminal-ui`
- `opensource`

## Distribution

See [docs/RELEASE.md](docs/RELEASE.md).
