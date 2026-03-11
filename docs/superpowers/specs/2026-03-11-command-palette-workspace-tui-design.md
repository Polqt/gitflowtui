# Command Palette Workspace TUI Design

Date: 2026-03-11
Status: Approved

## Summary

Redesign `gitflowtui` from a multi-panel dashboard into a keyboard-first workspace application with a global command palette, a repo home screen, and focused task workspaces for status/diff review, branch operations, commit/stash flows, and GitFlow actions.

The current backend split is worth preserving:

- `git` remains the Git CLI execution layer
- `gitflow` remains workflow orchestration for GitFlow operations
- `tui` is restructured into an app shell, command palette, and focused workspaces

The goal is to make the tool feel useful for day-to-day development work and visually closer to modern Charm CLI applications: intentional layout, clear hierarchy, one primary task at a time, and discoverable keyboard-driven actions.

## Context

The current application is a Bubble Tea TUI with five always-visible panels:

- branches
- log
- status
- stash
- diff

This structure works, but it pushes too much information onto the screen at once and makes navigation depend on remembering panel focus and single-letter shortcuts. The current `tui.App` also owns many unrelated concerns at the same time:

- refresh orchestration
- modal prompts
- PR form flow
- branch creation
- finish actions
- diff loading
- panel focus and rendering

That shape limits extensibility and makes it hard to produce a more polished, task-oriented experience.

## Goals

- Make the app useful as a daily Git/GitFlow companion for developers
- Replace the always-on multi-panel layout with a workspace-focused flow
- Add a global command palette as the primary discovery and navigation surface
- Launch into a repo overview/home workspace with immediate next-step actions
- Keep keyboard-first interaction with clear, low-friction shortcuts
- Preserve existing Git and GitFlow behavior where possible
- Support incremental migration without breaking the current backend model

## Non-Goals

- Rewriting the Git execution layer from CLI invocation to a native Git library
- Adding unrelated repository hosting integrations
- Building a fully generic command bus architecture beyond current repo needs
- Broad backend refactors that do not directly support the new user experience

## Recommended Approach

Three approaches were considered:

1. Workspace router with a global palette
2. Incremental retrofit of the existing panel app
3. Full command-bus architecture

The recommended approach is the workspace router with a global palette.

Rationale:

- It matches the desired Charm-style interaction model
- It improves boundaries inside `tui` without forcing a backend rewrite
- It supports a polished user experience without overshooting the size of the project
- It allows selective reuse of existing code such as diff rendering and form controls

## User Experience

### Launch Flow

On launch, the app opens a repo home workspace. The home screen should answer:

- what repository am I in
- what branch am I on
- what changed
- what needs attention
- what should I do next

The global command palette is available immediately and acts as the primary way to navigate or execute actions.

### Navigation Model

The new interaction model is intent-driven, not panel-driven.

Primary navigation:

- `ctrl+p` opens the global command palette
- `esc` closes the current overlay or returns to the prior workspace state
- each workspace defines a small local keymap for its own actions

The user should not need to memorize a large number of one-off keybindings. The palette is the fallback discovery mechanism for all major actions.

### Command Palette

The palette is hybrid:

- global actions appear as direct commands
- contextual entities appear as results when relevant

Examples:

- typing `sta` can surface `Status workspace`, `Stage all tracked`, and matching changed files
- typing `rel` can surface `Start release`, `Finish release`, and matching `release/*` branches
- typing a branch name can surface both `Checkout branch` and the matching branch entity

The palette should support fuzzy matching, ranked results, and intent handoff into workspaces. If the user invokes an action tied to a file, stash, or branch, the target workspace should open with that item selected.

### Workspaces

The first set of focused workspaces is:

- Home
- Status/Diff
- Branches
- Commit
- Stash
- GitFlow

#### Home Workspace

Shows high-signal repo state:

- repository name/path
- current branch and ahead/behind information
- changed files count and staged/unstaged breakdown
- recent commit summary
- stash count
- suggested next actions

#### Status/Diff Workspace

Primary workspace for the daily loop:

- browse changed files
- inspect diff
- stage and unstage files
- move into commit flow

The diff view is a first-class surface, not a side panel. Status selection and diff inspection belong to one focused task area.

#### Branches Workspace

Focuses on branch navigation and branch-aware actions:

- current branch
- local branches
- upstream state
- checkout
- create branch
- delete branch
- launch GitFlow actions when branch kind is recognized

#### Commit Workspace

A focused commit flow rather than a modal prompt:

- staged files summary
- commit message editor
- optional amend support if added
- validation feedback before submission

#### Stash Workspace

Supports:

- create stash
- inspect stashes
- pop selected stash
- review stash diff

#### GitFlow Workspace

Guided start and finish flows for:

- feature
- release
- hotfix

Finish flows should show preflight details before execution, especially when merge, delete, and tag operations are involved.

## Architecture

### High-Level Structure

The app becomes a shell with three persistent regions:

- top bar
- central workspace
- bottom status/hint area

The shell owns global behavior:

- command palette visibility
- app-wide shortcuts
- notifications
- background job state
- shared repo snapshot state
- routing between workspaces

Each workspace owns only its local interactions and rendering.

### TUI Package Restructure

The `tui` package should be reorganized around smaller focused units.

Planned components:

- `app_shell`
- `command_palette`
- `workspace_home`
- `workspace_status`
- `workspace_branches`
- `workspace_commit`
- `workspace_stash`
- `workspace_gitflow`
- shared helpers for styles, diff rendering, forms, and list items

This replaces the current model where one `App` type handles every screen, prompt, workflow, and panel transition.

### Shared State

A shared app state object should hold:

- active workspace
- current repo snapshot
- selected targets passed between workspaces
- active async operation
- notifications
- palette query and selection state

This state should be small and explicit. Workspaces should read what they need and return typed intents or messages to the shell rather than mutating unrelated global behavior directly.

## Data Flow

### Snapshot Refresh

The shell owns snapshot refresh. After any operation that mutates repository state, the app should:

1. show the action as running
2. execute it with a bounded context
3. surface success or failure
4. trigger a consistent refresh path

The refresh path should rebuild the shared repo snapshot used by the home workspace, palette context, and other task views.

### Action Dispatch

Instead of each workspace issuing ad hoc repo calls with its own timeout and error behavior, the shell should expose a small dispatcher for common actions such as:

- refresh
- load working diff
- load stash diff
- checkout branch
- stage file
- unstage file
- commit
- create stash
- pop stash
- start feature/release/hotfix
- finish feature/release/hotfix

This keeps timeouts, loading indicators, notifications, and post-action refresh behavior consistent.

### Intent Handoff

The command palette and workspaces should communicate through explicit intents. Examples:

- `OpenStatusForFile(path)`
- `OpenBranch(branchName)`
- `StartGitflowFinish(branchName, kind)`
- `OpenCommitWithStagedSummary`

This avoids overloading the root model with view-specific assumptions while still allowing smooth transitions between tasks.

## Backend Integration

### `git` Package

The `git` package remains the execution layer and should stay thin. The new UI may justify a few targeted additions if they are not already present:

- a lightweight repo summary helper or structured aggregation used by the home workspace
- staged and unstaged diff helpers if the status workspace needs clearer separation
- branch metadata shaped for palette result display
- optional commit template or amend support if the commit workspace grows

These changes should remain focused on supporting the UI, not redefining package boundaries.

### `gitflow` Package

`gitflow` should remain orchestration-first. The primary addition justified by the new UX is better finish preflight support.

Recommended addition:

- a validation or preview method that returns structured preflight information for finish actions

Example contents of a preview result:

- detected branch kind
- source branch
- target branches to merge into
- whether a tag will be created
- whether protected branches are behind remote
- whether branch deletion will be attempted

This lets the GitFlow workspace show users what will happen before running a destructive workflow.

## Error Handling

All blocking operations should use `context.Context` with explicit timeouts.

Errors should be surfaced at three levels:

- inline validation for form fields and guided flows
- non-blocking notifications for recoverable action failures
- explicit confirmation or blocking review for destructive GitFlow actions

The app should never leave action state ambiguous. Every operation should make it clear whether it is:

- idle
- running
- succeeded
- failed

Actions that fail must preserve the user's current workspace context where possible. For example, a failed stage or checkout should not eject the user from the current task view.

## Testing Strategy

Testing should target behavior and boundaries rather than fragile full-screen rendering snapshots.

Priority tests:

- command palette filtering and ranking
- workspace routing and intent handoff
- shell action dispatch for success and error flows
- repo snapshot refresh behavior after mutating actions
- GitFlow preview and finish preflight behavior
- focused workspace state transitions for status, branch checkout, commit, and stash flows

UI rendering tests should be narrow and reserved for stable string-producing helpers where they add value.

## Delivery Plan

Implement in phases to keep the app usable during migration.

### Phase 1

- introduce app shell
- introduce shared repo snapshot state
- add repo home workspace

### Phase 2

- add global command palette
- add workspace routing and intent handoff

### Phase 3

- migrate status/diff workspace
- migrate branches workspace

### Phase 4

- migrate commit workspace
- migrate stash workspace
- migrate GitFlow workspace

### Phase 5

- remove panel-centric code paths
- tighten styling, hints, and help content
- clean up obsolete prompt and panel abstractions

## Risks

- A partial migration could leave duplicate pathways active and increase complexity temporarily
- Over-designing the palette could delay the more important workspace restructuring
- Heavy rendering changes without clear state boundaries would recreate the current coupling problems

Mitigations:

- keep the shell as the single route owner
- keep backend changes small and UI-driven
- migrate one workspace at a time with tests around routing and dispatch

## Outcome

This design keeps the project grounded in its current strengths while changing the interaction model substantially. The result should feel more like a modern Charm-style developer tool:

- one primary task at a time
- fast keyboard-driven navigation
- a central command palette
- clearer visual hierarchy
- better separation between UI composition and Git/GitFlow execution
