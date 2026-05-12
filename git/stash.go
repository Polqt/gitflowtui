package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// StashEntry is a parsed stash list row.
type StashEntry struct {
	Ref          string
	Message      string
	RelativeTime string
}

func (r *Repo) Stash(ctx context.Context, msg string) error {
	args := []string{"stash", "push", "--include-untracked"}
	if strings.TrimSpace(msg) != "" {
		args = append(args, "-m", strings.TrimSpace(msg))
	}
	if _, err := r.runGit(ctx, args...); err != nil {
		return fmt.Errorf("stash push: %w", err)
	}
	return nil
}

func (r *Repo) StashPop(ctx context.Context, index int) error {
	ref := fmt.Sprintf("stash@{%d}", index)
	if _, err := r.runGit(ctx, "stash", "pop", ref); err != nil {
		return fmt.Errorf("stash pop %s: %w", ref, err)
	}
	return nil
}

func (r *Repo) ListStashes(ctx context.Context) ([]StashEntry, error) {
	out, err := r.runGit(ctx, "stash", "list", "--format=%gd%x00%gs%x00%cr")
	if err != nil {
		return nil, fmt.Errorf("stash list: %w", err)
	}

	lines := strings.Split(strings.TrimRight(out, "\r\n"), "\n")
	entries := make([]StashEntry, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 3)
		if len(parts) != 3 {
			continue
		}
		entries = append(entries, StashEntry{
			Ref:          parts[0],
			Message:      parts[1],
			RelativeTime: parts[2],
		})
	}
	return entries, nil
}

func (r *Repo) StashDiff(ctx context.Context, ref string) (string, error) {
	return r.stashDiff(ctx, ref, false)
}

func (r *Repo) StashDiffWord(ctx context.Context, ref string) (string, error) {
	return r.stashDiff(ctx, ref, true)
}

func (r *Repo) stashDiff(ctx context.Context, ref string, wordDiff bool) (string, error) {
	if strings.TrimSpace(ref) == "" {
		ref = "stash@{0}"
	}
	args := []string{"stash", "show", "-p", gitNoColor}
	if wordDiff {
		args = append(args, "--word-diff=plain")
	}
	args = append(args, ref)
	out, err := r.runGit(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("stash diff %s: %w", ref, err)
	}
	return out, nil
}

func StashIndex(ref string) (int, error) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, "stash@{") || !strings.HasSuffix(ref, "}") {
		return 0, fmt.Errorf("invalid stash ref: %q", ref)
	}
	n := strings.TrimSuffix(strings.TrimPrefix(ref, "stash@{"), "}")
	idx, err := strconv.Atoi(n)
	if err != nil {
		return 0, fmt.Errorf("invalid stash index in %q: %w", ref, err)
	}
	return idx, nil
}
