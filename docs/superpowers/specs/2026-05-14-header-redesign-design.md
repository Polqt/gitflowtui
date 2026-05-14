# Header Redesign Design

**Date:** 2026-05-14
**Status:** Approved

## Goal

Replace the current two-row header (title bar + breadcrumb) with a single prompt-style title bar that shows the app identity, detected repo name, current branch, sync status, AI availability, and version — all on one line. Remove the breadcrumb row entirely to reclaim one vertical row for content.

## Current State

The header occupies two rows:

1. **Title bar** — `◈  GIT TERMINAL  /  gitflow-tui` with a loading spinner on the right
2. **Breadcrumb** — `⎇ <branch>  ·  synced  ✦ AI  ⚡ v1`

Both rows are hardcoded and do not show the repo name.

## New Design

### Title Bar (single row)

```
>_ gitflowy  |  repo: gitflowtui  ·  branch: feature/ui-redesign  ·  synced ✓          ✦ AI  ⚡ v1.2.0
```

**Left side:**
- `>_` — terminal prompt icon, accent cyan, bold
- `gitflowy` — app name, accent cyan, bold
- `|` — separator, dim
- `repo:` — label, dim
- `<repoName>` — accent cyan, bold; derived from `filepath.Base(repo.Root)`
- `·  branch:` — separator + label, dim
- `<currentBranch>` — primary text, bold
- `·` — separator, dim
- sync status: `synced ✓` (green) / `ahead N` (cyan) / `behind N` (orange)

**Right side (right-aligned):**
- When loading: spinner + loading label in magenta, bold — replaces AI badge
- When AI available (and not loading): `✦ AI` in magenta, bold
- `⚡ v1.2.0` — dim (version injected at build time; hard-coded to `v1` if build var unavailable)

**AI badge states:**
- Available + idle: `✦ AI` magenta bold
- Available + loading: spinner + label (e.g. `⣾ AI commit`) magenta bold — existing spinner mechanism
- Unavailable: omitted entirely

### Breadcrumb Row

Removed. The `renderBreadcrumb()` function is deleted. The `View()` call to it is removed, and `usedRows` is decremented from 11 to 10.

## Repo Name Detection

```go
repoName = filepath.Base(repo.Root)
```

- `filepath.Base` returns the last path segment (e.g. `C:\Users\poyhi\gitflowtui` → `gitflowtui`)
- If `repo.Root` is empty or `.`, use `"repo"` as fallback
- No additional git calls required
- Stored as `repoName string` on the `App` struct, set once in `NewApp`

## Code Changes

| File | Change |
|------|--------|
| `tui/app.go` | Add `repoName string` to `App` struct; set via `filepath.Base(repo.Root)` in `NewApp` |
| `tui/dashboard.go` | Rewrite `renderTitleBar()` with new prompt-style format |
| `tui/dashboard.go` | Remove `renderBreadcrumb()` call from `View()`; decrement `usedRows` from 11 to 10 |
| `tui/dashboard.go` | Delete `renderBreadcrumb()` function |

## Non-Goals

- Config override for repo name
- Remote-based repo name detection
- Any changes to the stats cards, body panels, navbar, or status line
