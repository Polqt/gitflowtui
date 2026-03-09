$ErrorActionPreference = "Stop"

$repo = "github.com/Polqt/gitflowtui/cmd/gitflow-tui"
Write-Host "Installing gitflow-tui..."
go install "$repo@latest"

Write-Host "Installed. Ensure your Go bin directory is in PATH."
