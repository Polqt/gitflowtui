# Setup & Configuration

## First Run

Open a terminal inside any Git repository and run:

```bash
gitflow-tui
```

You can also point it at any folder directly:

```bash
gitflow-tui /path/to/repo
```

## Environment Variables (full table)

| Variable | Default | What It Does |
| --- | --- | --- |
| `GITFLOW_TUI_MAIN` | `main` | Name of your production branch |
| `GITFLOW_TUI_DEVELOP` | `develop` | Name of your integration branch |
| `GITFLOW_TUI_REMOTE` | `origin` | Name of your git remote |
| `GITFLOW_TUI_WS_DISABLE` | `(unset)` | Set to `1` to turn off WebSocket |
| `GITFLOW_TUI_WS_ADDR` | `127.0.0.1:7373` | WebSocket address and port |
| `GITFLOW_TUI_WS_PATH` | `/ws` | WebSocket URL path |
| `ANTHROPIC_API_KEY` | `(unset)` | Anthropic API key for AI features |
| `GITFLOW_TUI_OLLAMA_BASE_URL` | `http://127.0.0.1:11434` | Ollama API base URL |
| `GITFLOW_TUI_OLLAMA_MODEL` | `qwen2.5-coder:1.5b` | Ollama model for free AI features |

## Setting Environment Variables

### Windows (PowerShell)

Set them for the current terminal session:

```powershell
$env:GITFLOW_TUI_MAIN="main"
$env:GITFLOW_TUI_DEVELOP="develop"
$env:GITFLOW_TUI_REMOTE="origin"
$env:ANTHROPIC_API_KEY="sk-ant-..."
$env:GITFLOW_TUI_OLLAMA_MODEL="qwen2.5-coder:1.5b"
gitflow-tui
```

To make them permanent on Windows:

- Open Start and search for `Environment Variables`
- Open `Edit the system environment variables`
- Click `Environment Variables...`
- Add your variables under `User variables`
- Open a new terminal window after saving

### macOS / Linux (bash or zsh)

Set them for the current shell session:

```bash
export GITFLOW_TUI_MAIN="main"
export GITFLOW_TUI_DEVELOP="develop"
export GITFLOW_TUI_REMOTE="origin"
export ANTHROPIC_API_KEY="sk-ant-..."
export GITFLOW_TUI_OLLAMA_MODEL="qwen2.5-coder:1.5b"
gitflow-tui
```

To make them permanent on macOS or Linux:

- Add the `export ...` lines to `~/.zshrc` if you use zsh
- Or add them to `~/.bashrc` if you use bash
- Restart your terminal or run `source ~/.zshrc` or `source ~/.bashrc`

## Gitflow Branch Conventions

`gitflow-tui` recognizes these branch naming patterns:

- `feature/your-feature-name` or `feat/your-feature-name`
- `release/1.2.0` and the suffix must be a version number like `1.0.0`
- `hotfix/1.2.1` and the suffix must be a version number

Examples:

- `feature/payment-api`
- `feat/login-form`
- `release/2.0.0`
- `hotfix/2.0.1`

`main` and `develop` are permanent branches. They are the long-lived branches in the workflow and should never be deleted.

## Keybindings Reference (full table)

| Key | Panel | What It Does |
| --- | --- | --- |
| `tab / shift+tab` | Any | Switch between panels |
| `enter` | Any | Primary action (checkout/pop/view diff) |
| `r` | Any | Refresh all panels |
| `q` | Any | Quit |
| `?` | Any | Toggle help |
| `n then f` | Any | Create new feature branch |
| `n then r` | Any | Create new release branch |
| `n then h` | Any | Create new hotfix branch |
| `F` | Branch | Finish feature (merge into develop, delete) |
| `R` | Branch | Finish release (merge into main+develop, tag) |
| `H` | Branch | Finish hotfix  (merge into main+develop, tag) |
| `c` | Any | Open commit prompt |
| `ctrl+a` | Commit | AI: suggest commit message from staged diff |
| `a` | Any | Stash current changes |
| `s` | Status | Stage selected file |
| `u` | Status | Unstage selected file |
| `p` | Any | Push current branch |
| `P` | Any | Pull with rebase (`--autostash`) |
| `w` | Any | Toggle line/word diff mode |
| `X` | Branch | AI: predict merge conflicts before merging |
| `E` | Diff | AI: explain current diff in plain English |
| `E` | Stash | AI: explain selected stash entry |
