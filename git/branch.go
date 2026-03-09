package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// BranchType is a UI-friendly branch classification.
type BranchType int

const (
	BranchTypeUnknown BranchType = iota
	BranchTypeMain
	BranchTypeDevelop
	BranchTypeFeature
	BranchTypeRelease
	BranchTypeHotfix
)

// Branch contains local branch metadata.
type Branch struct {
	Name     string
	IsHead   bool
	Upstream string
	Ahead    int
	Behind   int
	Type     BranchType
}

func (r *Repo) Branches(ctx context.Context) ([]Branch, error) {
	lines, err := r.runGitLines(ctx, "branch", "-vv", "--no-color")
	if err != nil {
		return nil, fmt.Errorf("branches: %w", err)
	}

	branches := make([]Branch, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		b, err := parseBranchLine(line)
		if err != nil {
			continue
		}
		branches = append(branches, b)
	}
	return branches, nil
}

func parseBranchLine(line string) (Branch, error) {
	var b Branch
	if len(line) < 3 {
		return b, fmt.Errorf("malformed branch line: %q", line)
	}

	b.IsHead = line[0] == '*'
	line = strings.TrimSpace(line[1:])

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return b, fmt.Errorf("unexpected branch format: %q", line)
	}
	b.Name = parts[0]

	l := strings.Index(line, "[")
	r := strings.Index(line, "]")
	if l != -1 && r > l {
		tracking := line[l+1 : r]
		b.Upstream, b.Ahead, b.Behind = parseTracking(tracking)
	}

	b.Type = branchTypeFromName(b.Name)
	return b, nil
}

func parseTracking(s string) (string, int, int) {
	colon := strings.Index(s, ":")
	if colon == -1 {
		return strings.TrimSpace(s), 0, 0
	}

	upstream := strings.TrimSpace(s[:colon])
	metrics := strings.TrimSpace(s[colon+1:])

	ahead := 0
	behind := 0
	for _, part := range strings.Split(metrics, ",") {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) < 2 {
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

func branchTypeFromName(name string) BranchType {
	switch {
	case name == "main" || name == "master":
		return BranchTypeMain
	case name == "develop":
		return BranchTypeDevelop
	case strings.HasPrefix(name, "feature/") || strings.HasPrefix(name, "feat/"):
		return BranchTypeFeature
	case strings.HasPrefix(name, "release/"):
		return BranchTypeRelease
	case strings.HasPrefix(name, "hotfix/"):
		return BranchTypeHotfix
	default:
		return BranchTypeUnknown
	}
}

func (r *Repo) Checkout(ctx context.Context, branch string) error {
	if _, err := r.runGit(ctx, "checkout", branch); err != nil {
		return fmt.Errorf("checkout %q: %w", branch, err)
	}
	return nil
}

func (r *Repo) CreateBranch(ctx context.Context, name, base string) error {
	args := []string{"checkout", "-b", name}
	if strings.TrimSpace(base) != "" {
		args = append(args, base)
	}
	if _, err := r.runGit(ctx, args...); err != nil {
		return fmt.Errorf("create branch %q: %w", name, err)
	}
	return nil
}

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

func (r *Repo) MergeBranch(ctx context.Context, name string, noFF bool) error {
	args := []string{"merge"}
	if noFF {
		args = append(args, "--no-ff")
	}
	args = append(args, name)
	if _, err := r.runGit(ctx, args...); err != nil {
		return fmt.Errorf("merge %q: %w", name, err)
	}
	return nil
}

func (r *Repo) Push(ctx context.Context) error {
	if _, err := r.runGit(ctx, "push"); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	return nil
}

func (r *Repo) PullRebase(ctx context.Context) error {
	if _, err := r.runGit(ctx, "pull", "--rebase", "--autostash"); err != nil {
		return fmt.Errorf("pull --rebase: %w", err)
	}
	return nil
}

func (r *Repo) Fetch(ctx context.Context) error {
	if _, err := r.runGit(ctx, "fetch", "--all", "--prune"); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	return nil
}

// AheadBehind returns commits unique to left and right in left...right.
func (r *Repo) AheadBehind(ctx context.Context, left, right string) (int, int, error) {
	out, err := r.runGit(ctx, "rev-list", "--left-right", "--count", left+"..."+right)
	if err != nil {
		return 0, 0, fmt.Errorf("ahead/behind %s...%s: %w", left, right, err)
	}
	fields := strings.Fields(out)
	if len(fields) < 2 {
		return 0, 0, fmt.Errorf("unexpected ahead/behind output: %q", out)
	}

	ahead, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse ahead count: %w", err)
	}
	behind, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse behind count: %w", err)
	}
	return ahead, behind, nil
}
