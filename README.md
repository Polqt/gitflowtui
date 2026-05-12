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

## AI Features

AI features work with two backends — pick whichever is free for you:

**Option A — Ollama (completely free, runs on your machine)**
1. Install from https://ollama.com (Windows, Mac, Linux)
2. Run: `ollama pull qwen2.5-coder:1.5b`
3. Start gitflow-tui — AI activates automatically, no config needed

**Option B — Anthropic API (free tier available)**
1. Sign up at https://console.anthropic.com
2. Get your free API key
3. `export ANTHROPIC_API_KEY=sk-ant-...`

If neither is configured, the tool works exactly as before —
AI panels show a hint but nothing breaks.

**AI keybindings**
- `X`        — predict merge conflicts before you merge (works on feature/release/hotfix)
- `E`        — explain current diff or stash in plain English (streams word by word)
- `ctrl+a`   — suggest a commit message from your staged changes (in commit prompt)

## Realtime Websocket Stream

Enable websocket broadcasting to stream app state changes to external tools:

```bash
GITFLOW_TUI_WS_ADDR=127.0.0.1:7777 GITFLOW_TUI_WS_PATH=/ws gitflow-tui [path-to-repo]
```

The server sends JSON events for notifications and repository snapshots whenever the UI refreshes.

For a deployable headless server that keeps running without the TUI:

```bash
gitflow-tui serve --repo /path/to/repo --addr 127.0.0.1:7373 --path ws
```

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

## Documentation

- [Installation (no Go required)](docs/INSTALL.md)
- [Setup & Configuration](docs/SETUP.md)
- [AI Features](docs/AI.md)
- [WebSocket Integration](docs/WEBSOCKET.md)
- [Release Guide](docs/RELEASE.md)
- [Contributing](CONTRIBUTING.md)

## Distribution

See [docs/RELEASE.md](docs/RELEASE.md).
