package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// BranchType represents the type of a Git branch.
type BranchType int

const (
	BranchTypeUnknown BranchType = iota
	BranchTypeMain               // main / master
	BranchTypeDevelop            // develop
	BranchTypeFeature            // feature/*
	BranchTypeRelease            // release/*
	BranchTypeHotfix             // hotfix/*
)

// Branch holds metadata for a single local branch.
type Branch struct {
	Name     string
	IsHead   bool       // true if this is the currently checked-out branch
	Ahead    int        // commits ahead of upstream
	Behind   int        // commits behind upstream
	Upstream string     // upstream tracking ref, e.g. "origin/main"
	Type     BranchType // classified by the gitflow package; default Unknown
}

// Branches returns all local branches with upstream tracking info.
//
// `git branch -vv` format per line:
//
//	"* main  a1b2c3d [origin/main: ahead 2, behind 1] commit msg"
//	"  feat  d4e5f6a [origin/feat] commit msg"
//	"  local 7890abc commit msg"        ← no upstream
//
// The leading "* " marks the current branch.
func (r *Repo) Branches(ctx context.Context) ([]Branch, error) {
	lines, err := r.runGitLines(ctx, "branch", "-vv", "--no-color")
	if err != nil {
		return nil, fmt.Errorf("branches: %w", err)
	}

	branches := make([]Branch, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		branch, err := parseBranchLine(line)
		if err != nil {
			// skip unparseable lines, but log them for debugging
			continue
		}
		branches = append(branches, branch)
	}

	return branches, nil
}

// parseBranchLine parses a single line of `git branch -vv` output into a Branch struct.
func parseBranchLine(line string) (Branch, error) {
	b := Branch{}

	// The first 2 chars indicate if this is the current branch
	if len(line) < 3 {
		return b, fmt.Errorf("line too short: %q", line)
	}
	b.IsHead = line[0] == '*'

	// strip the leading "* " or "  "
	line = line[2:]

	// split on whitespace: [name, hash, ...rest]
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return b, fmt.Errorf("unexpected format: %q", line)
	}
	b.Name = parts[0]

	// Check for upstream info in the rest of the line
	// it always starts with a "[" and ends with a "]"
	bracketStart := strings.Index(line, "[")
	bracketEnd := strings.Index(line, "]")
	if bracketStart != -1 && bracketEnd > bracketStart {
		tracking := line[bracketStart+1 : bracketEnd]
		b.Upstream, b.Ahead, b.Behind = parseTracking(tracking)
	}

	return b, nil
}

// parseTracking parses the content inside [...] from `git branch -vv`.
// Examples:
//
//	"origin/main"                     → upstream, 0, 0
//	"origin/main: ahead 3"            → upstream, 3, 0
//	"origin/main: behind 2"           → upstream, 0, 2
//	"origin/main: ahead 1, behind 4"  → upstream, 1, 4
func parseTracking(s string) (upstream string, ahead int, behind int) {
	// split on ": " to separate upstream from the rest
	colonIdx := strings.Index(s, ": ")
	if colonIdx == -1 {
		// no colon means no ahead/behind info, just the upstream name
		return strings.TrimSpace(s), 0, 0
	}
	upstream = strings.TrimSpace(s[:colonIdx])
	metrics := s[colonIdx+2:]

	// each metric is ahead N or behind N, separated by ", "
	for _, part := range strings.Split(metrics, ", ") {
		fields := strings.Fields(part)
		if len(fields) != 2 {
			continue
		}
		n, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		switch fields[0] {
		case "ahead":
			ahead = n
		case "behind":
			behind = n
		}
	}
	return upstream, ahead, behind
}

// Checkout switches to an existing branch.
func (r *Repo) Checkout(ctx context.Context, branch string) error {
	if _, err := r.runGit(ctx, "checkout", branch); err != nil {
		return fmt.Errorf("checkout %q: %w", branch, err)
	}
	return nil
}

// DeleteBranch deletes a local branch. force=true maps to -D (ignore unmerged).
func (r *Repo) DeleteBranch(ctx context.Context, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	if _, err := r.runGit(ctx, "branch", flag, name); err != nil {
		return fmt.Errorf("delete branch %q: %w", name, err)
	}
	return nil
}

// MergeBranch merges name into the current branch.
// noFF=true forces a merge commit even for fast-forward cases — required by Gitflow.
func (r *Repo) MergeBranch(ctx context.Context, name string, noFF bool) error {
	args := []string{"merge"}
	if noFF {
		// --no-ff preserves branch topology in the log — critical for Gitflow.
		args = append(args, "--no-ff")
	}
	args = append(args, name)
	if _, err := r.runGit(ctx, args...); err != nil {
		return fmt.Errorf("merge %q: %w", name, err)
	}
	return nil
}
