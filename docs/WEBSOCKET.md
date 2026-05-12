# WebSocket Integration

## What Is the WebSocket and Why Does It Exist

When `gitflow-tui` is open, it automatically starts a tiny background server on your computer at `127.0.0.1:7373`. `127.0.0.1` means "only your own machine", so nothing goes to the internet and nothing is exposed to your local network. It is a local-only connection point that other tools on your computer can use while the TUI is running.

The reason it exists is simple: other programs on your machine can connect to this address and receive live updates about what is happening in your repository. That makes it possible to build browser dashboards, IDE integrations, small helper scripts, or automation that reacts instantly while you work. This is already running. You do not start it. You do not configure it. It is just there.

For deployment or IDE integration that should keep running after the TUI exits, use headless server mode:

```bash
gitflow-tui serve --repo /path/to/repository --addr 127.0.0.1:7373 --path ws
```

Headless mode polls the repository and broadcasts `snapshot` events every 2 seconds by default. Change the interval with:

```bash
gitflow-tui serve --repo /path/to/repository --interval 5s
```

In Git Bash on Windows, prefer `--path ws` instead of `--path /ws`. Git Bash can rewrite slash-prefixed arguments into Windows paths; `gitflow-tui` normalizes `ws` to `/ws` internally.

Do not bind this server to a public interface unless you add network-level authentication such as VPN, SSH tunnel, or an authenticated reverse proxy. The stream includes repository paths, branch names, file status, commit subjects, and stash metadata.

## What Events Are Sent

### `branch.changed`

```json
{
  "kind": "branch.changed",
  "at": "2025-03-20T14:32:01Z",
  "branch": "feature/payment-api",
  "message": "",
  "meta": {}
}
```

### `commit.created`

```json
{
  "kind": "commit.created",
  "at": "2025-03-20T14:33:12Z",
  "branch": "feature/payment-api",
  "message": "feat(payments): add payment API client",
  "meta": {}
}
```

### `branch.created`

```json
{
  "kind": "branch.created",
  "at": "2025-03-20T14:34:20Z",
  "branch": "feature/payment-api",
  "message": "",
  "meta": {}
}
```

### `branch.deleted`

```json
{
  "kind": "branch.deleted",
  "at": "2025-03-20T14:40:03Z",
  "branch": "feature/payment-api",
  "message": "",
  "meta": {}
}
```

### `branch.merged`

```json
{
  "kind": "branch.merged",
  "at": "2025-03-20T14:41:55Z",
  "branch": "feature/payment-api",
  "message": "merged into develop",
  "meta": {}
}
```

### `stash.pushed`

```json
{
  "kind": "stash.pushed",
  "at": "2025-03-20T14:45:10Z",
  "branch": "feature/payment-api",
  "message": "wip: payment UI",
  "meta": {}
}
```

### `stash.popped`

```json
{
  "kind": "stash.popped",
  "at": "2025-03-20T14:47:44Z",
  "branch": "feature/payment-api",
  "message": "stash@{0}",
  "meta": {}
}
```

### `status.changed`

```json
{
  "kind": "status.changed",
  "at": "2025-03-20T14:48:32Z",
  "branch": "feature/payment-api",
  "message": "",
  "meta": {}
}
```

### `ai.ready`

```json
{
  "kind": "ai.ready",
  "at": "2025-03-20T14:49:01Z",
  "branch": "feature/payment-api",
  "message": "ollama",
  "meta": {}
}
```

### `error`

```json
{
  "kind": "error",
  "at": "2025-03-20T14:50:26Z",
  "branch": "feature/payment-api",
  "message": "pull --rebase failed",
  "meta": {}
}
```

## How To Connect (Examples)

### Browser Console (test it right now)

Paste this into your browser DevTools console while `gitflow-tui` is open:

```javascript
const ws = new WebSocket("ws://127.0.0.1:7373/ws");
ws.onmessage = (e) => console.log(JSON.parse(e.data));
```

### Node.js Script

```javascript
import WebSocket from "ws";

const ws = new WebSocket("ws://127.0.0.1:7373/ws");

ws.on("open", () => {
  console.log("Connected to gitflow-tui");
});

ws.on("message", (data) => {
  const event = JSON.parse(data.toString());
  console.log(event);
});

ws.on("close", () => {
  console.log("Disconnected");
});

ws.on("error", (err) => {
  console.error(err);
});
```

### Python Script

```python
import asyncio
import json
import websockets


async def main():
    async with websockets.connect("ws://127.0.0.1:7373/ws") as ws:
        async for message in ws:
            print(json.loads(message))


asyncio.run(main())
```

### VS Code Extension / IDE Plugin

Any IDE plugin can subscribe to this local URL and react to events from `gitflow-tui`. A VS Code extension, for example, could show the current branch or merge activity in the status bar without needing to re-scan the repo itself.

### CI/CD Integration

```bash
#!/usr/bin/env bash
set -euo pipefail

python - <<'PY'
import asyncio
import json
import websockets
import subprocess


async def main():
    async with websockets.connect("ws://127.0.0.1:7373/ws") as ws:
        async for message in ws:
            event = json.loads(message)
            if event.get("kind") == "branch.merged":
                subprocess.run(["./deploy.sh"], check=True)
                return


asyncio.run(main())
PY
```

## How To Disable It

Set `GITFLOW_TUI_WS_DISABLE=1` before running the interactive TUI:

Windows:

```powershell
$env:GITFLOW_TUI_WS_DISABLE="1"; gitflow-tui
```

Mac/Linux:

```bash
GITFLOW_TUI_WS_DISABLE=1 gitflow-tui
```

This does not affect `gitflow-tui serve`; that command is explicitly the WebSocket server.

## How To Change The Port

If port `7373` is already in use on your machine:

Windows:

```powershell
$env:GITFLOW_TUI_WS_ADDR="127.0.0.1:8080"; gitflow-tui
```

Mac/Linux:

```bash
GITFLOW_TUI_WS_ADDR=127.0.0.1:8080 gitflow-tui
```
