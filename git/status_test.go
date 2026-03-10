package git

import "testing"

func TestParseOrdinaryEntry_PreservesSpacesInPath(t *testing.T) {
	t.Parallel()

	rec := "1 M. N... 100644 100644 100644 abcdef1 abcdef2 dir/with spaces/file name.txt"
	got, err := parseOrdinaryEntry(rec)
	if err != nil {
		t.Fatalf("parseOrdinaryEntry returned error: %v", err)
	}

	if got.Path != "dir/with spaces/file name.txt" {
		t.Fatalf("unexpected path: %q", got.Path)
	}
	if got.X != StateModified || got.Y != StateUnmodified {
		t.Fatalf("unexpected XY: %s%s", got.X, got.Y)
	}
}

func TestParseRenameEntry_PreservesSpacesInPaths(t *testing.T) {
	t.Parallel()

	rec := "2 R. N... 100644 100644 100644 abcdef1 abcdef2 R100 new path/with spaces.txt"
	orig := "old path/with spaces.txt"
	got, err := parseRenameEntry(rec, orig)
	if err != nil {
		t.Fatalf("parseRenameEntry returned error: %v", err)
	}

	if got.Path != "new path/with spaces.txt" {
		t.Fatalf("unexpected new path: %q", got.Path)
	}
	if got.Orig != orig {
		t.Fatalf("unexpected orig path: %q", got.Orig)
	}
}

func TestParseUntrackedEntry_PreservesLeadingSpace(t *testing.T) {
	t.Parallel()

	rec := "? " + " leading-space.txt"
	got, err := parseUntrackedEntry(rec)
	if err != nil {
		t.Fatalf("parseUntrackedEntry returned error: %v", err)
	}
	if got != " leading-space.txt" {
		t.Fatalf("unexpected path: %q", got)
	}
}

func TestParseStatusOutput_MixedRecords(t *testing.T) {
	t.Parallel()

	ordinary := "1 M. N... 100644 100644 100644 abcdef1 abcdef2 src/main.go"
	rename := "2 R. N... 100644 100644 100644 abcdef1 abcdef2 R100 docs/new name.md"
	orig := "docs/old name.md"
	untracked := "? testdata/file with spaces.txt"
	out := ordinary + "\x00" + rename + "\x00" + orig + "\x00" + untracked + "\x00"

	got := parseStatusOutput(out)
	if len(got) != 3 {
		t.Fatalf("unexpected file count: got %d want 3", len(got))
	}

	if got[0].Path != "src/main.go" {
		t.Fatalf("unexpected ordinary path: %q", got[0].Path)
	}
	if got[1].Path != "docs/new name.md" || got[1].Orig != "docs/old name.md" {
		t.Fatalf("unexpected rename entry: %+v", got[1])
	}
	if got[2].Path != "testdata/file with spaces.txt" || !got[2].IsUntracked() {
		t.Fatalf("unexpected untracked entry: %+v", got[2])
	}
}
