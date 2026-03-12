# gitflow-tui

A GitFlow-aware terminal UI built with Bubble Tea for day-to-day branch, status, diff, stash, and release workflows.

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

## Realtime Websocket Stream

Enable websocket broadcasting to stream app state changes to external tools:

```bash
GITFLOW_TUI_WS_ADDR=127.0.0.1:7777 GITFLOW_TUI_WS_PATH=/ws gitflow-tui [path-to-repo]
```

The server sends JSON events for notifications and repository snapshots whenever the UI refreshes.

Minimal client example:

```js
const ws = new WebSocket("ws://127.0.0.1:7777/ws");
ws.onmessage = (event) => {
  const payload = JSON.parse(event.data);
  console.log(payload.type, payload);
};
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
- `GITFLOW_TUI_WS_ADDR` (example: `127.0.0.1:7777`; empty = disabled)
- `GITFLOW_TUI_WS_PATH` (default: `/ws`)

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, testing, linting, and PR guidance.

## Distribution

See [docs/RELEASE.md](docs/RELEASE.md).
