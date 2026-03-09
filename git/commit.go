package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Commit contains information rendered in the log panel.
type Commit struct {
	Hash    string
	Date    string
	Author  string
	Subject string
}

func (r *Repo) Log(ctx context.Context, limit int) ([]Commit, error) {
	return r.LogRef(ctx, "HEAD", limit)
}

func (r *Repo) LogRef(ctx context.Context, ref string, limit int) ([]Commit, error) {
	if strings.TrimSpace(ref) == "" {
		ref = "HEAD"
	}
	if limit <= 0 {
		limit = 50
	}

	format := "%h%x00%ad%x00%an%x00%s"
	out, err := r.runGit(ctx,
		"log",
		"--date=short",
		"--pretty=format:"+format,
		"-n", strconv.Itoa(limit),
		ref,
	)
	if err != nil {
		return nil, fmt.Errorf("log %q: %w", ref, err)
	}

	lines := strings.Split(strings.TrimRight(out, "\r\n"), "\n")
	commits := make([]Commit, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 4)
		if len(parts) != 4 {
			continue
		}
		commits = append(commits, Commit{
			Hash:    strings.TrimSpace(parts[0]),
			Date:    strings.TrimSpace(parts[1]),
			Author:  strings.TrimSpace(parts[2]),
			Subject: strings.TrimSpace(parts[3]),
		})
	}
	return commits, nil
}

func (r *Repo) Commit(ctx context.Context, msg string, amend bool) error {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return fmt.Errorf("commit message cannot be empty")
	}

	args := []string{"commit", "-m", msg}
	if amend {
		args = append(args, "--amend")
	}

	if _, err := r.runGit(ctx, args...); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r *Repo) TagAnnotated(ctx context.Context, tag, msg string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}
	if strings.TrimSpace(msg) == "" {
		msg = tag
	}

	if _, err := r.runGit(ctx, "tag", "-a", tag, "-m", msg); err != nil {
		return fmt.Errorf("tag %q: %w", tag, err)
	}
	return nil
}
