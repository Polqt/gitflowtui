package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type prTemplateForm struct {
	active bool
	action finishAction
	source string
	target string

	title     textinput.Model
	reviewers textinput.Model
	body      textarea.Model
	focus     int
}

func newPRTemplateForm() prTemplateForm {
	title := textinput.New()
	title.Placeholder = "feat: short summary"
	title.CharLimit = 120
	title.Prompt = "Title: "

	reviewers := textinput.New()
	reviewers.Placeholder = "@alice @bob"
	reviewers.CharLimit = 200
	reviewers.Prompt = "Reviewers: "

	body := textarea.New()
	body.Placeholder = "## What\nDescribe the change\n\n## Testing\n- [ ] Unit tests\n- [ ] Manual verification"
	body.SetHeight(10)
	body.ShowLineNumbers = false
	body.FocusedStyle.CursorLine = lipgloss.NewStyle()

	return prTemplateForm{
		title:     title,
		reviewers: reviewers,
		body:      body,
		focus:     0,
	}
}

func (f *prTemplateForm) open(action finishAction, source, target string) {
	f.active = true
	f.action = action
	f.source = source
	f.target = target
	f.focus = 0

	f.title.SetValue(defaultPRTitle(source))
	f.reviewers.SetValue("")
	f.body.SetValue(defaultPRBody())

	f.title.Focus()
	f.reviewers.Blur()
	f.body.Blur()
}

func (f *prTemplateForm) close() {
	f.active = false
	f.action = finishNone
	f.source = ""
	f.target = ""
	f.title.Blur()
	f.reviewers.Blur()
	f.body.Blur()
}

func (f *prTemplateForm) update(msg tea.Msg) (tea.Cmd, bool, bool, string) {
	if !f.active {
		return nil, false, false, ""
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			f.close()
			return nil, false, true, ""
		case "tab", "shift+tab":
			f.moveFocus(key.String() == "tab")
			return nil, false, false, ""
		case "ctrl+s":
			template := f.renderTemplate()
			f.close()
			return nil, true, false, template
		}
	}

	switch f.focus {
	case 0:
		var cmd tea.Cmd
		f.title, cmd = f.title.Update(msg)
		return cmd, false, false, ""
	case 1:
		var cmd tea.Cmd
		f.body, cmd = f.body.Update(msg)
		return cmd, false, false, ""
	case 2:
		var cmd tea.Cmd
		f.reviewers, cmd = f.reviewers.Update(msg)
		return cmd, false, false, ""
	default:
		return nil, false, false, ""
	}
}

func (f *prTemplateForm) moveFocus(next bool) {
	if next {
		f.focus = (f.focus + 1) % 3
	} else {
		f.focus--
		if f.focus < 0 {
			f.focus = 2
		}
	}

	f.title.Blur()
	f.body.Blur()
	f.reviewers.Blur()

	switch f.focus {
	case 0:
		f.title.Focus()
	case 1:
		f.body.Focus()
	case 2:
		f.reviewers.Focus()
	}
}

func (f *prTemplateForm) view(width, height int) string {
	if !f.active {
		return ""
	}
	if width < 40 {
		width = 40
	}
	if height < 16 {
		height = 16
	}

	inner := width - 4
	f.title.Width = inner - 8
	f.reviewers.Width = inner - 12
	f.body.SetWidth(inner - 2)
	f.body.SetHeight(max(6, height-14))

	header := lipgloss.NewStyle().Bold(true).Foreground(colorCyan).Render("New Pull Request")
	meta := lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%s -> %s", f.source, f.target))
	footer := lipgloss.NewStyle().Foreground(colorMuted).Render("[Tab] Next  [Ctrl+S] Submit  [Esc] Cancel")

	content := strings.Join([]string{
		header,
		meta,
		"",
		f.title.View(),
		"",
		"Description:",
		f.body.View(),
		"",
		f.reviewers.View(),
		"",
		footer,
	}, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorCyan).
		Background(colorBgModal).
		Width(width - 2).
		Render(content)

	return box
}

func (f *prTemplateForm) renderTemplate() string {
	title := strings.TrimSpace(f.title.Value())
	if title == "" {
		title = defaultPRTitle(f.source)
	}
	body := strings.TrimSpace(f.body.Value())
	reviewers := strings.TrimSpace(f.reviewers.Value())

	lines := []string{
		"Title: " + title,
		"",
		body,
	}
	if reviewers != "" {
		lines = append(lines, "", "Reviewers: "+reviewers)
	}
	return strings.Join(lines, "\n")
}

func persistPRTemplate(content string) (string, error) {
	_ = clipboard.WriteAll(content)

	name := fmt.Sprintf("gitflow-pr-%d.md", time.Now().Unix())
	path := filepath.Join(os.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func defaultPRBody() string {
	return "## What\nDescribe the change\n\n## Testing\n- [ ] Unit tests passing\n- [ ] Manual test in staging"
}

func defaultPRTitle(source string) string {
	s := strings.TrimSpace(source)
	if s == "" {
		return "feat: summary"
	}
	if strings.HasPrefix(s, "feature/") || strings.HasPrefix(s, "feat/") {
		return "feat: " + strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(s, "feature/"), "feat/"), "-", " ")
	}
	if strings.HasPrefix(s, "release/") {
		return "release: " + strings.TrimPrefix(s, "release/")
	}
	if strings.HasPrefix(s, "hotfix/") {
		return "hotfix: " + strings.TrimPrefix(s, "hotfix/")
	}
	return "feat: " + s
}
