package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type DryRunConflictFile struct {
	Path            string // The file path relative to the repo root
	HunkCount       int    // The number of hunks in this file that are predicted to conflict
	ConflictContent string // The content of the conflict hunks, with <<<<<<<, =======, >>>>>>> markers
}

type DryRunResult struct {
	HasConflicts  bool   // Whether the merge would have conflicts
	MergeBase     string // The merge base commit hash
	CommitsAhead  int    // Number of commits ahead of the merge base
	FilesChanged  int    // Total number of files changed in the merge
	ConflictFiles []DryRunConflictFile
}

// DryRunMerge performs an in-memory merge of source into target using git merge-tree
// and returns structural conflict data WITHOUT touching the working tree, index,
// or any refs.
// It is safe to call at any time, including when the working tree has
// uncommitted changes.
// Algorithm:
//  1. git merge-base source target  → SHA of common ancestor
//  2. git merge-tree <base> <source> <target>  → dry merge, surfaces conflict markers
//  3. git rev-list --count target..source  → ahead count
//  4. git diff --name-only <base> <source>  → files changed
//
// On git versions that do not support the three-argument merge-tree (pre-2.13),
// DryRunMerge falls back to intersection analysis of changed file sets.
func (r *Repo) DryRunMerge(ctx context.Context, source, target string) (*DryRunResult, error) {
	// 1:  merge base
	baseOut, err := r.runGit(ctx, "merge-base", source, target)
	if err != nil {
		return nil, fmt.Errorf("DryRunMerge: find merge-base(%s, %s): %w", source, target, err)
	}
	base := strings.TrimSpace(baseOut)

	// 2: ahead count
	aheadCount, _ := r.runGit(ctx, "rev-list", "--count", target+".."+source)
	ahead := parseIntOrZero(strings.TrimSpace(aheadCount))

	// 3: files changed by source since base
	changedOut, _ := r.runGit(ctx, "diff", "--name-only", base, source)
	filesChanged := countNonEmptyLines(changedOut)

	// 4: dry-run merge
	// git merge-tree <base> <source> <target> exits 0 regardless of conflicts.
	// The merged content of every changed file is written to stdout.
	// Files with conflicts contain <<<<<<< / ======= / >>>>>>> markers.
	mergeOut, mergeErr := r.runGit(ctx, "merge-tree", base, source, target)
	if mergeErr != nil {
		return r.dryRunMergeFallback(ctx, base, source, target, ahead, filesChanged)
	}

	conflictFiles := parseMergeTreeOutput(mergeOut)
	return &DryRunResult{
		HasConflicts:  len(conflictFiles) > 0,
		MergeBase:     base,
		CommitsAhead:  ahead,
		FilesChanged:  filesChanged,
		ConflictFiles: conflictFiles,
	}, nil
}

// dryRunMergeFallback identifies conflict candidates by finding files that
// both branches changed since the merge base. Less precise than merge-tree
// (no hunk-level detail) but works on all git versions ≥ 1.7.
func (r *Repo) dryRunMergeFallback(ctx context.Context, base, source, target string, ahead, filesChanged int) (*DryRunResult, error) {
	ourFiles, err := r.changedFileSet(ctx, base, source)
	if err != nil {
		return nil, fmt.Errorf("DryRunMerge fallback (ours): %w", err)
	}
	theirFiles, err := r.changedFileSet(ctx, base, target)
	if err != nil {
		return nil, fmt.Errorf("DryRunMerge fallback (theirs): %w", err)
	}

	var conflicts []DryRunConflictFile
	for path := range theirFiles {
		if ourFiles[path] {
			conflicts = append(conflicts, DryRunConflictFile{
				Path:      path,
				HunkCount: 1, // unknown without merge-tree
			})
		}
	}

	return &DryRunResult{
		MergeBase:     base,
		CommitsAhead:  ahead,
		FilesChanged:  filesChanged,
		HasConflicts:  len(conflicts) > 0,
		ConflictFiles: conflicts,
	}, nil
}

// changedFileSet returns a set of file paths changed between base and ref.
func (r *Repo) changedFileSet(ctx context.Context, base, ref string) (map[string]bool, error) {
	out, err := r.runGit(ctx, "diff", "--name-only", base, ref)
	if err != nil {
		return nil, fmt.Errorf("changed files %s..%s: %w", base, ref, err)
	}
	set := make(map[string]bool)
	for _, f := range strings.Split(out, "\n") {
		if f = strings.TrimSpace(f); f != "" {
			set[f] = true
		}
	}
	return set, nil
}

// DiffRange returns the unified diff between two refs (base..head).
// Used by the AI conflict analyser to provide per-file context.
func (r *Repo) DiffRange(ctx context.Context, base, head string) (string, error) {
	out, err := r.runGit(ctx, "diff", "--no-color", base, head)
	if err != nil {
		return "", fmt.Errorf("DiffRange %s..%s: %w", base, head, err)
	}
	return out, nil
}

// merge-tree output parser
// parseMergeTreeOutput scans the stdout of `git merge-tree <base> <a> <b>`.

// The old-format merge-tree output is a sequence of "file records" each
// containing the full merged content of one changed file. Files with conflicts
// contain inline <<<<<<< / ======= / >>>>>>> markers.

// File record format:

// changed in both          ← or "added in remote", "removed in local", etc.
//
//	base mode 100644
//	their mode 100644
//	...
//	filename: path/to/file
//
// <merged content with optional conflict markers>
// (empty line separates records)
func parseMergeTreeOutput(out string) []DryRunConflictFile {
	if out == "" {
		return nil
	}

	const maxContentPerFile = 4096 // 4 KB per file, to avoid overwhelming the API with huge diffs. merge-tree output is already truncated at 64 KB, so this covers most cases.

	var (
		results     []DryRunConflictFile
		currentPath string
		content     strings.Builder
		inRecord    bool
	)

	flush := func() {
		if currentPath == "" {
			return
		}
		raw := content.String()
		hunkCount := strings.Count(raw, "<<<<<<<")
		if hunkCount > 0 {
			c := raw
			if len(c) > maxContentPerFile {
				c = c[:maxContentPerFile] + "\n... (truncated)"
			}
			results = append(results, DryRunConflictFile{
				Path:            currentPath,
				HunkCount:       hunkCount,
				ConflictContent: c,
			})
		}
		currentPath = ""
		content.Reset()
		inRecord = false
	}

	for _, line := range strings.Split(out, "\n") {
		// file record header lines
		lower := strings.ToLower(strings.TrimSpace(line))
		if isRecordHeader(lower) {
			flush()
			inRecord = true
			continue
		}

		// "filename: path/to/file" line
		if inRecord && strings.Contains(line, "filename:") {
			if idx := strings.Index(line, "filename:"); idx != -1 {
				currentPath = strings.TrimSpace(line[idx+9:])
			}
			continue
		}

		if inRecord && currentPath == "" {
			continue
		}

		if currentPath != "" {
			content.WriteString(line)
			content.WriteByte('\n')
		}
	}
	flush()

	// Fallback: if the parser found no file headers but the output contains
	// conflict markers, record it as a single unattributed conflict.
	// This handles pathological or future git output formats gracefully.
	if len(results) == 0 && strings.Count(out, "<<<<<<<") > 0 {
		c := out
		if len(c) > maxContentPerFile {
			c = c[:maxContentPerFile] + "\n... (truncated)"
		}
		results = append(results, DryRunConflictFile{
			Path:            "(unknown)",
			HunkCount:       strings.Count(out, "<<<<<<<"),
			ConflictContent: c,
		})
	}

	return results
}

func isRecordHeader(line string) bool {
	prefixes := []string{
		"changed in both",
		"added in remote",
		"added in local",
		"removed in remote",
		"removed in local",
		"renamed in",
		"conflict",
		"merged",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
}

// Helpers
func parseIntOrZero(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

func countNonEmptyLines(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}
