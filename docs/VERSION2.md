# Version 2 Deployment And Distribution Guide

## Ollama Setup

Install Ollama:

```bash
# macOS
brew install --cask ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh
```

```powershell
# Windows
winget install Ollama.Ollama
```

Use a small laptop-friendly model:

```bash
ollama pull qwen2.5-coder:1.5b
```

`qwen2.5-coder:1.5b` is the recommended Version 2 default because it is small, code-oriented, fast enough for commit messages and diff explanations, and runs comfortably under 8 GB RAM.

Run Ollama on demand and unload idle models:

```bash
OLLAMA_KEEP_ALIVE=0 ollama serve
```

```powershell
$env:OLLAMA_KEEP_ALIVE="0"; ollama serve
```

Configure gitflow-tui:

```bash
export GITFLOW_TUI_OLLAMA_BASE_URL=http://127.0.0.1:11434
export GITFLOW_TUI_OLLAMA_MODEL=qwen2.5-coder:1.5b
gitflow-tui
```

```powershell
$env:GITFLOW_TUI_OLLAMA_BASE_URL="http://127.0.0.1:11434"
$env:GITFLOW_TUI_OLLAMA_MODEL="qwen2.5-coder:1.5b"
gitflow-tui
```

If Ollama is not running and `ANTHROPIC_API_KEY` is not set, gitflow-tui disables AI features and prints a clear message instead of blocking or crashing.

## Binary Distribution

Build all supported Version 2 binaries:

```bash
make dist
```

Outputs:

```text
dist/gitflow-tui-darwin-arm64
dist/gitflow-tui-darwin-amd64
dist/gitflow-tui-linux-amd64
dist/gitflow-tui-windows-amd64.exe
```

The build uses `CGO_ENABLED=0`, `-trimpath`, and `-ldflags "-s -w ..."` for clean static binaries with build-time version metadata.

## GitHub Releases

Tag and publish:

```bash
git tag v2.0.0
git push origin v2.0.0
goreleaser release --clean
```

The repository `.goreleaser.yaml` builds macOS arm64, macOS amd64, Linux amd64, and Windows amd64; packages Windows as zip and macOS/Linux as tar.gz; generates `checksums.txt`; and creates a GitHub Release with changelog notes.

## Homebrew Formula Template

Create `Formula/gitflow-tui.rb` in a Homebrew tap:

```ruby
class GitflowTui < Formula
  desc "GitFlow-aware terminal UI for branch, release, stash, diff, AI, and realtime workflows"
  homepage "https://github.com/Polqt/gitflowtui"
  version "2.0.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Polqt/gitflowtui/releases/download/v2.0.0/gitflow-tui_2.0.0_darwin_arm64.tar.gz"
      sha256 "REPLACE_WITH_DARWIN_ARM64_SHA256"
    else
      url "https://github.com/Polqt/gitflowtui/releases/download/v2.0.0/gitflow-tui_2.0.0_darwin_amd64.tar.gz"
      sha256 "REPLACE_WITH_DARWIN_AMD64_SHA256"
    end
  end

  def install
    bin.install "gitflow-tui"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/gitflow-tui --version")
  end
end
```

After each release, update `version`, both `url` values, and both `sha256` values from `checksums.txt`.

## Auto-Run WebSocket Server

The realtime WebSocket starts automatically while `gitflow-tui` is running. For deployment, use the headless server command and point it at a repository path:

```bash
gitflow-tui serve --repo /path/to/repository --addr 127.0.0.1:7373 --path ws
```

Keep stdout/stderr in standard log locations.

macOS launchd, save as `~/Library/LaunchAgents/com.gitflowtui.websocket.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "https://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.gitflowtui.websocket</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/gitflow-tui</string>
    <string>serve</string>
    <string>--repo</string>
    <string>/path/to/repository</string>
    <string>--addr</string>
    <string>127.0.0.1:7373</string>
    <string>--path</string>
    <string>ws</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>/tmp/gitflow-tui-websocket.log</string>
  <key>StandardErrorPath</key><string>/tmp/gitflow-tui-websocket.err</string>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/com.gitflowtui.websocket.plist
```

Linux systemd, save as `~/.config/systemd/user/gitflow-tui-websocket.service`:

```ini
[Unit]
Description=gitflow-tui realtime websocket
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gitflow-tui serve --repo /path/to/repository --addr 127.0.0.1:7373 --path ws
Restart=on-failure
RestartSec=5
Environment=GITFLOW_TUI_WS_ADDR=127.0.0.1:7373
Environment=GITFLOW_TUI_WS_PATH=/ws
StandardOutput=append:%h/.local/state/gitflow-tui/websocket.log
StandardError=append:%h/.local/state/gitflow-tui/websocket.err

[Install]
WantedBy=default.target
```

```bash
mkdir -p ~/.local/state/gitflow-tui ~/.config/systemd/user
systemctl --user daemon-reload
systemctl --user enable --now gitflow-tui-websocket.service
```

Windows Task Scheduler XML:

```xml
<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.4" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <Triggers><LogonTrigger><Enabled>true</Enabled></LogonTrigger></Triggers>
  <Principals><Principal id="Author"><LogonType>InteractiveToken</LogonType><RunLevel>LeastPrivilege</RunLevel></Principal></Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <RestartOnFailure><Interval>PT5M</Interval><Count>3</Count></RestartOnFailure>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Enabled>true</Enabled>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>powershell.exe</Command>
      <Arguments>-NoProfile -WindowStyle Hidden -Command "gitflow-tui.exe serve --repo C:\path\to\repository --addr 127.0.0.1:7373 --path ws *> $env:LOCALAPPDATA\gitflow-tui\websocket.log"</Arguments>
    </Exec>
  </Actions>
</Task>
```

```powershell
New-Item -ItemType Directory -Force "$env:LOCALAPPDATA\gitflow-tui"
schtasks /Create /TN GitflowTuiWebSocket /XML .\gitflow-tui-websocket.xml
```

## Version Command

```bash
gitflow-tui --version
```

The command prints the version, commit, and build date injected with ldflags by `make build`, `make dist`, or GoReleaser.
