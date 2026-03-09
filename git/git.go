package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repo wraps git CLI operations for a repository root.
type Repo struct {
	Root string
}

// Repository exposes operations used by the TUI and workflow layers.
type Repository interface {
	Status(ctx context.Context) ([]FileStatus, error)
	Branches(ctx context.Context) ([]Branch, error)
	Log(ctx context.Context, limit int) ([]Commit, error)
	LogRef(ctx context.Context, ref string, limit int) ([]Commit, error)
	Diff(ctx context.Context, ref string) (string, error)
	DiffFile(ctx context.Context, path string, staged bool) (string, error)
	Stage(ctx context.Context, path string) error
	Unstage(ctx context.Context, path string) error
	Commit(ctx context.Context, msg string, amend bool) error
	Stash(ctx context.Context, msg string) error
	StashPop(ctx context.Context, index int) error
	ListStashes(ctx context.Context) ([]StashEntry, error)
	StashDiff(ctx context.Context, ref string) (string, error)
	Checkout(ctx context.Context, branch string) error
	CreateBranch(ctx context.Context, name, base string) error
	DeleteBranch(ctx context.Context, name string, force bool) error
	MergeBranch(ctx context.Context, name string, noFF bool) error
	Push(ctx context.Context) error
	PullRebase(ctx context.Context) error
	Fetch(ctx context.Context) error
	CurrentBranch(ctx context.Context) (string, error)
	RevParse(ctx context.Context, ref string) (string, error)
	AheadBehind(ctx context.Context, left, right string) (int, int, error)
	TagAnnotated(ctx context.Context, tag, msg string) error
}

var ErrNotARepo = errors.New("not a git repository")

func NewRepo(path string) (*Repo, error) {
	root, err := FindRoot(path)
	if err != nil {
		return nil, err
	}
	return &Repo{Root: root}, nil
}

func FindRoot(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	cmd := exec.Command("git", "-C", abs, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", ErrNotARepo
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *Repo) CurrentBranch(ctx context.Context) (string, error) {
	out, err := r.runGit(ctx, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err == nil {
		return strings.TrimSpace(out), nil
	}

	// detached HEAD fallback
	hash, hashErr := r.runGit(ctx, "rev-parse", "--short", "HEAD")
	if hashErr != nil {
		return "", fmt.Errorf("resolve HEAD: %w", hashErr)
	}
	return strings.TrimSpace(hash), nil
}

func (r *Repo) RevParse(ctx context.Context, ref string) (string, error) {
	out, err := r.runGit(ctx, "rev-parse", "--verify", ref)
	if err != nil {
		return "", fmt.Errorf("rev-parse %q: %w", ref, err)
	}
	return strings.TrimSpace(out), nil
}
