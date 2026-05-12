package git

import (
	"context"
	"fmt"
	"strings"
)

const gitNoColor = "--no-color"

func (r *Repo) Diff(ctx context.Context, ref string) (string, error) {
	return r.diff(ctx, ref, false)
}

func (r *Repo) DiffWord(ctx context.Context, ref string) (string, error) {
	return r.diff(ctx, ref, true)
}

func (r *Repo) diff(ctx context.Context, ref string, wordDiff bool) (string, error) {
	args := []string{"diff", gitNoColor}
	if wordDiff {
		args = append(args, "--word-diff=plain")
	}
	if strings.TrimSpace(ref) != "" {
		args = append(args, ref)
	}
	out, err := r.runGit(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("diff: %w", err)
	}
	return out, nil
}

func (r *Repo) DiffFile(ctx context.Context, path string, staged bool) (string, error) {
	return r.diffFile(ctx, path, staged, false)
}

func (r *Repo) DiffFileWord(ctx context.Context, path string, staged bool) (string, error) {
	return r.diffFile(ctx, path, staged, true)
}

func (r *Repo) diffFile(ctx context.Context, path string, staged bool, wordDiff bool) (string, error) {
	args := []string{"diff", gitNoColor}
	if wordDiff {
		args = append(args, "--word-diff=plain")
	}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	out, err := r.runGit(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("diff file %q: %w", path, err)
	}
	return out, nil
}
