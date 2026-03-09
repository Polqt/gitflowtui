package git

import (
	"context"
	"fmt"
	"strings"
)

func (r *Repo) Diff(ctx context.Context, ref string) (string, error) {
	args := []string{"diff", "--no-color"}
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
	args := []string{"diff", "--no-color"}
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
