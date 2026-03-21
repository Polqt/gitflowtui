package ai

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

// promptFS embeds every .txt file under the prompts/ directory tree.
//
//go:embed prompts
var promptFS embed.FS

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

func loadPrompts() promptSet {
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

// CommitPromptData maps to prompts/commit/user.txt.
type CommitPromptData struct {
	Diff string
}

// ConflictPromptData maps to prompts/conflict/user.txt.
type ConflictPromptData struct {
	SourceBranch   string
	CommitsAhead   int
	TargetBranch   string
	FilesChanged   int
	ConflictCount  int
	ConflictDigest string
}

// ExplainPromptData maps to prompts/explain/*.txt.
type ExplainPromptData struct {
	Diff string
}

// HealthPromptData maps to prompts/health/user.txt.
type HealthPromptData struct {
	Today       string
	Head        string
	BranchLines string
}
