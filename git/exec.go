package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// runGit is the single call-site for every git subprocess in the codebase.
// All other files in this package call runGit — they never call exec.Command
// directly. This makes it trivial to swap the executor in tests.
func (r *Repo) runGit(ctx context.Context, args ...string) ([]byte, error) {
	// exec.CommandContext autommatically sends SIGKILL when ctx is is cancelled.
	cmd := exec.CommandContext(ctx, "git", args...)

	// Run every command from the repo root so relative paths work as expected.
	cmd.Dir = r.Root

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Wrap stderr so callers get actionable context
			return nil, fmt.Errorf("git %s: %w\nstderr: %s", strings.Join(args, " "), err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

// runGitLines is a wrapper that splits stdout on newlines 
func (r *Repo) runGitLines(ctx context.Context, args ...string) ([]string, error) {
	out, err := r.runGit(ctx, args...)
	if err != nil {
		return nil, err
	}

	raw := strings.TrimRight(string(out), "\n")
	if raw == "" {
		// No output is valid 
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

