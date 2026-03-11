$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$demoRoot = Join-Path $root ".demo"
$origin = Join-Path $demoRoot "origin.git"
$worktree = Join-Path $demoRoot "worktree"

if (Test-Path $demoRoot) {
	Remove-Item -Recurse -Force $demoRoot
}

New-Item -ItemType Directory -Path $demoRoot | Out-Null

git init --bare $origin | Out-Null
New-Item -ItemType Directory -Path $worktree | Out-Null

Push-Location $worktree
git init | Out-Null
git config user.name "Codex Demo"
git config user.email "codex-demo@example.com"
git remote add origin $origin

@"
# Demo Repo

This repo exists to exercise gitflow-tui.
"@ | Set-Content README.md

New-Item -ItemType Directory -Path app | Out-Null
@"
package main

import "fmt"

func main() {
	fmt.Println("demo app")
}
"@ | Set-Content app\main.go

git add .
git commit -m "chore: initial commit" | Out-Null
git branch -M main
git push -u origin main | Out-Null

git checkout -b develop | Out-Null
@"
package main

func version() string {
	return "0.1.0-dev"
}
"@ | Set-Content app\version.go
git add .
git commit -m "feat: add develop version helper" | Out-Null
git push -u origin develop | Out-Null

git checkout -b feature/demo-panel develop | Out-Null
@"
feature work in progress
"@ | Set-Content notes.txt
git add notes.txt
git commit -m "feat: add demo notes" | Out-Null
git push -u origin feature/demo-panel | Out-Null

Add-Content notes.txt "`nmore local changes"
@"
package main

func pending() string {
	return "worktree change"
}
"@ | Set-Content app\pending.go
git add app\pending.go

@"
scratch change for stash
"@ | Set-Content scratch.txt
git stash push -m "demo stash" | Out-Null
Add-Content notes.txt "`nunstashed local edit"

Pop-Location

Write-Host "Demo repo created:"
Write-Host "  Origin:   $origin"
Write-Host "  Worktree: $worktree"
Write-Host ""
Write-Host "Next:"
Write-Host "  powershell -ExecutionPolicy Bypass -File scripts\\try-demo.ps1"
