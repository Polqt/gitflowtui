package ai

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

// promptFS embeds every .txt file under the prompts/ directory tree.
// Using embed.FS instead of individual //go:embed directives means:
//   - A single directive covers all current and future prompt files.
//   - The binary is fully self-contained — no external assets required.
//   - Prompt files can be read, reviewed, and diffed as plain text in git
//     without ever opening a Go source file.
//
//go:embed prompts
var promptFS embed.FS

// promptSet
type promptSet struct {
	CommitSystem     string
	CommitUser       *template.Template
	ConflictSystem   string
	ConflictUser     *template.Template
	ExplainDiffSys   string
	ExplainDiffUser  *template.Template
	ExplainStashSys  string
	ExplainStashUser *template.Template
	HealthSystem     string
	HealthUser       *template.Template
}

var prompts = mustLoadPrompts()

func mustLoadPrompts() promptSet {
	load := func(path string) string {
		b, err := promptFS.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("ai: missing prompt file %q: %v", path, err))
		}
		return string(b)
	}
	parse := func(path string) *template.Template {
		src := load(path)
		t, err := template.New(path).Parse(src)
		if err != nil {
			panic(fmt.Sprintf("ai: invalid prompt template %q: %v", path, err))
		}
		return t
	}

	return promptSet{
		CommitSystem:     load("prompts/commit/system.txt"),
		CommitUser:       parse("prompts/commit/user.txt"),
		ConflictSystem:   load("prompts/conflict/system.txt"),
		ConflictUser:     parse("prompts/conflict/user.txt"),
		ExplainDiffSys:   load("prompts/explain/diff_system.txt"),
		ExplainDiffUser:  parse("prompts/explain/diff_user.txt"),
		ExplainStashSys:  load("prompts/explain/stash_system.txt"),
		ExplainStashUser: parse("prompts/explain/stash_user.txt"),
		HealthSystem:     load("prompts/health/system.txt"),
		HealthUser:       parse("prompts/health/user.txt"),
	}
}

func render(t *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render prompt %q: %w", t.Name(), err)
	}
	return buf.String(), nil
}

// ─── Template data structs ────────────────────────────────────────────────────
// Each struct maps 1-to-1 with the {{.Field}} placeholders in its .txt file.
// Keeping them here (not scattered across feature files) makes it trivial to
// see every variable a prompt template can access.

type CommitPromptData struct {
	Diff string
}

type ConflictPromptData struct {
	SourceBranch   string
	CommitsAhead   int
	TargetBranch   string
	FilesChanged   int
	ConflictCount  int
	ConflictDigest string
}

type ExplainPromptData struct {
	Diff string
}

type HealthPromptData struct {
	Today       string
	Head        string
	BranchLines string
}
