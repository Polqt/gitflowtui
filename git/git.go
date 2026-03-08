package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Repo struct {
	Root string // The root directory of the repository
}

func NewRepo(root string) *Repo {
	return &Repo{Root: root}
}

// ErrNotARepo is returned by FindRoot when the directory is not inside a git repo.
var ErrNotARepo = fmt.Errorf("not a git repository")

func FindRoot(dir string) (string, error) {
	abs, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	if dir != "." {
		abs = dir
	}

	// ask git itself where the roots is
	cmd := exec.Command("git", "-C", abs, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", ErrNotARepo
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *Repo) Head(ctx context.Context) (string, error) {
	// --short strips the "refs/heads/" prefix
	out, err := r.runGit(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		// detached HEAD state, try to get the commit hash instead
		hash, err := r.runGit(ctx, "rev-parse", "--short", "HEAD")
		if err != nil {
			return "", fmt.Errorf("resolve HEAD: %w", err)
		}
		return strings.TrimSpace(string(hash)), nil
	}
	return strings.TrimSpace(string(out)), nil
}
