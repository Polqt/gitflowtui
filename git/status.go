package git

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

type FileState string

const (
	StateUntracked  FileState = "?"
	StateModified   FileState = "M"
	StateAdded      FileState = "A"
	StateDeleted    FileState = "D"
	StateRenamed    FileState = "R"
	StateCopied     FileState = "C"
	StateUnmerged   FileState = "U"
	StateUnmodified FileState = "."
)

type FileStatus struct {
	X    FileState
	Y    FileState
	Path string
	Orig string
}

func (f FileStatus) IsStaged() bool {
	return f.X != StateUnmodified && f.X != StateUntracked
}

func (f FileStatus) IsUnstaged() bool {
	return f.Y != StateUnmodified && f.Y != StateUntracked
}

func (f FileStatus) IsUntracked() bool {
	return f.X == StateUntracked && f.Y == StateUntracked
}

func (f FileStatus) DisplayPath() string {
	if f.Orig != "" {
		return f.Orig + " -> " + f.Path
	}
	return f.Path
}

func (r *Repo) Status(ctx context.Context) ([]FileStatus, error) {
	out, err := r.runGit(ctx, "status", "--porcelain=v2", "-z")
	if err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}

	records := bytes.Split([]byte(out), []byte{0})
	files := make([]FileStatus, 0, len(records))

	for i := 0; i < len(records); i++ {
		rec := string(records[i])
		if rec == "" {
			continue
		}

		switch rec[0] {
		case '1':
			f, err := parseOrdinaryEntry(rec)
			if err == nil {
				files = append(files, f)
			}
		case '2':
			if i+1 >= len(records) {
				continue
			}
			f, err := parseRenameEntry(rec, string(records[i+1]))
			if err == nil {
				files = append(files, f)
			}
			i++
		case 'u':
			f, err := parseUnmergedEntry(rec)
			if err == nil {
				files = append(files, f)
			}
		case '?':
			path := strings.TrimSpace(strings.TrimPrefix(rec, "?"))
			if path != "" {
				files = append(files, FileStatus{X: StateUntracked, Y: StateUntracked, Path: path})
			}
		}
	}

	return files, nil
}

func parseOrdinaryEntry(rec string) (FileStatus, error) {
	parts := strings.Fields(rec)
	if len(parts) < 9 {
		return FileStatus{}, fmt.Errorf("malformed ordinary entry: %q", rec)
	}
	xy := parts[1]
	if len(xy) != 2 {
		return FileStatus{}, fmt.Errorf("invalid XY code: %q", xy)
	}

	return FileStatus{
		X:    FileState(string(xy[0])),
		Y:    FileState(string(xy[1])),
		Path: parts[8],
	}, nil
}

func parseRenameEntry(rec, orig string) (FileStatus, error) {
	parts := strings.Fields(rec)
	if len(parts) < 10 {
		return FileStatus{}, fmt.Errorf("malformed rename entry: %q", rec)
	}
	xy := parts[1]
	if len(xy) != 2 {
		return FileStatus{}, fmt.Errorf("invalid XY code: %q", xy)
	}

	return FileStatus{
		X:    FileState(string(xy[0])),
		Y:    FileState(string(xy[1])),
		Path: parts[9],
		Orig: orig,
	}, nil
}

func parseUnmergedEntry(rec string) (FileStatus, error) {
	parts := strings.Fields(rec)
	if len(parts) < 10 {
		return FileStatus{}, fmt.Errorf("malformed unmerged entry: %q", rec)
	}
	xy := parts[1]
	if len(xy) != 2 {
		return FileStatus{}, fmt.Errorf("invalid XY code: %q", xy)
	}
	return FileStatus{
		X:    FileState(string(xy[0])),
		Y:    FileState(string(xy[1])),
		Path: parts[len(parts)-1],
	}, nil
}

func (r *Repo) Stage(ctx context.Context, path string) error {
	if _, err := r.runGit(ctx, "add", "--", path); err != nil {
		return fmt.Errorf("stage %q: %w", path, err)
	}
	return nil
}

func (r *Repo) Unstage(ctx context.Context, path string) error {
	if _, err := r.runGit(ctx, "restore", "--staged", "--", path); err == nil {
		return nil
	}
	if _, err := r.runGit(ctx, "reset", "HEAD", "--", path); err != nil {
		return fmt.Errorf("unstage %q: %w", path, err)
	}
	return nil
}
