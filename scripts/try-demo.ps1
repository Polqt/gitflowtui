$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root "dist"
$binary = Join-Path $dist "gitflow-tui.exe"
$demoRoot = Join-Path $root ".demo"
$origin = Join-Path $demoRoot "origin.git"
$worktree = Join-Path $demoRoot "worktree"

if (-not (Test-Path $dist)) {
	New-Item -ItemType Directory -Path $dist | Out-Null
}

if (-not (Test-Path $binary)) {
	Write-Host "Building gitflow-tui..."
	go build -o $binary ./cmd/gitflow-tui
}

if (-not (Test-Path $origin) -or -not (Test-Path $worktree)) {
	Write-Host "Demo repo not found. Run scripts\\setup-demo.ps1 first."
	exit 1
}

Write-Host "Launching gitflow-tui against demo repo: $worktree"
& $binary $worktree
