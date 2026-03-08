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
	X    FileState // staged change
	Y    FileState // unstaged change
	Path string    // current path of the file
	Orig string    // Original path for renamed/copied files
}

// IsStaged returns true if the file has a staged change (added, modified, deleted, renamed, or copied).
func (f FileStatus) IsStaged() bool {
	return f.X != StateUnmodified && f.X != StateUntracked
}

// IsUnstaged returns true if the file has an unstaged change (added, modified, deleted, renamed, or copied).
func (f FileStatus) IsUnstaged() bool {
	return f.Y != StateUnmodified && f.Y != StateUntracked
}

func (f FileStatus) IsUntracked() bool {
	return f.X == StateUntracked && f.Y == StateUntracked
}

// We use --porcelain=v2 -z:
//   - v2 gives a more stable output format that is easier to parse, especially for renamed/copied files
//   - -z uses null bytes as separators, which allows us to handle file names with special characters or newlines
func (r *Repo) Status(ctx context.Context) ([]FileStatus, error) {
	out, err := r.runGit(ctx, "status", "--percelain=v2", "-z")
	if err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}

	// null split the output.
	records := bytes.Split(out, []byte{0})

	var files []FileStatus
	for i := 0; i < len(records); i++ {
		rec := string(records[i])
		if rec == "" {
			continue
		}

		switch rec[0] {
		case '1':
			// Ordinary changed entry.
			// Format: 1 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>
		}
	}

	return files, nil
}

func parseOrdinaryEntry(rec string) (FileStatus, error) {
	// Minimum valid length: "1 XY sub mH mI mW hH hI path"
	parts := strings.Fields(rec)
	if len(parts) < 9 {
		return FileStatus{}, fmt.Errorf("malformed ordinary entry: %q", rec)
	}
	xy := parts[1] // two-char status code
	if len(xy) != 2 {
		return FileStatus{}, fmt.Errorf("invalid status code: %q", xy)
	}
	return FileStatus{
		X:    FileState(string(xy[0])),
		Y:    FileState(string(xy[1])),
		Path: parts[8],
	}, nil
}

// parseRenameEntry parses a rename or copy entry, which has the format:
// R <XY> <score> <mH> <mI> <mW> <hH> <hI> <path> NUL <orig_path>
func parseRenameEntry(rec, orig string) (FileStatus, error) {
	parts := strings.Fields(rec)
	if len(parts) < 9 {
		return FileStatus{}, fmt.Errorf("malformed rename entry: %q", rec)
	}
	xy := parts[1]
	if len(xy) != 2 {
		return FileStatus{}, fmt.Errorf("invalid status code: %q", xy)
	}
	return FileStatus{
		X:    FileState(string(xy[0])),
		Y:    FileState(string(xy[1])),
		Path: parts[8],
		Orig: orig,
	}, nil
}

// parseUnmergedEntry parses an unmerged entry, which has the format:
// U <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>
// For unmerged entries, the <XY> code is always "UU", but we can still use
// the same parsing logic as ordinary entries.
func parseUnmergedPath(rec string) string {
	parts := strings.Fields(rec)
	if len(parts) >= 10 {
		return parts[9]
	}
	return ""
}
