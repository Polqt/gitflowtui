# Installation

## Quick Install (No Go Required)

### Windows

- Download the latest `gitflow-tui_windows_amd64.zip` from the GitHub Releases page at https://github.com/Polqt/gitflowtui/releases
- Unzip the file
- Move `gitflow-tui.exe` to a folder that is in your `PATH`, for example `C:\Users\YourName\bin\`
- Open a new terminal and type: `gitflow-tui`
- `PATH` is the list of folders your terminal searches automatically when you type a command name.

### macOS

- Download `gitflow-tui_darwin_arm64.tar.gz` (Apple Silicon M1/M2/M3) or `gitflow-tui_darwin_amd64.tar.gz` (older Intel Mac) from Releases
- Run:

```bash
tar -xzf gitflow-tui_darwin_*.tar.gz
```

- Run:

```bash
sudo mv gitflow-tui /usr/local/bin/
```

- Run:

```bash
gitflow-tui --help
```

- Note: if macOS says "cannot be opened because developer cannot be verified", go to System Settings > Privacy & Security > Open Anyway

### Linux

- Download `gitflow-tui_linux_amd64.tar.gz` from Releases
- Run:

```bash
tar -xzf gitflow-tui_linux_amd64.tar.gz
```

- Run:

```bash
chmod +x gitflow-tui && sudo mv gitflow-tui /usr/local/bin/
```

- Run:

```bash
gitflow-tui --help
```

## Install With Go (For Developers)

```bash
go install github.com/Polqt/gitflowtui/cmd/gitflow-tui@latest
```

## Verify Install

```bash
gitflow-tui --version
gitflow-tui /path/to/any/git/repo
```

If your current build does not expose `--version` yet, use `gitflow-tui --help` as a quick verification step.
