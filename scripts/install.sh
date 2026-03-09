#!/usr/bin/env bash
set -euo pipefail

REPO="github.com/Polqt/gitflowtui/cmd/gitflow-tui"

echo "Installing gitflow-tui..."
go install "${REPO}@latest"

echo "Installed. Ensure \$GOBIN (or \$GOPATH/bin) is in your PATH."
