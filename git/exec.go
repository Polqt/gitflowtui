package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func (r *Repo) runGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.Root

	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return string(out), nil
}

func (r *Repo) runGitLines(ctx context.Context, args ...string) ([]string, error) {
	out, err := r.runGit(ctx, args...)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimRight(out, "\r\n")
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}
