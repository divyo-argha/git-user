package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// Report is a scrollable read-only text screen used to display operation
// output (public keys, security audits, export/import summaries, doctor, …).
type Report struct {
	title   string
	lines   []string
	offset  int
	theme   theme.Theme
}

// NewReport creates a Report screen from a block of text.
func NewReport(title string, text string, th theme.Theme) *Report {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	return &Report{
		title: title,
		lines: lines,
		theme: th,
	}
}

func (r *Report) Init() tea.Cmd { return nil }

func (r *Report) Title() string { return r.title }

func (r *Report) ShortHelp() string {
	return "  ↑/↓/j/k scroll • Esc back • q quit"
}

func (r *Report) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if core.IsEscKey(msg) || msg.String() == "b" || msg.String() == "B" {
			return r, func() tea.Msg { return core.ScreenPopMsg{} }
		}
		switch msg.String() {
		case core.KeyCtrlC, core.KeyQuit:
			return r, tea.Quit
		case core.KeyUp, core.KeyK:
			if r.offset > 0 {
				r.offset--
			}
		case core.KeyDown, core.KeyJ:
			r.offset++
		}
	}
	return r, nil
}

func (r *Report) View(width, height int) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(r.theme.Bold().Render("  " + r.title))
	sb.WriteString("\n")
	sb.WriteString(r.theme.SeparatorLine(width - 6))
	sb.WriteString("\n\n")

	// Reserve 4 lines for the header/footer chrome.
	maxLines := height - 8
	if maxLines < 3 {
		maxLines = 3
	}

	start := r.offset
	if start > len(r.lines) {
		start = 0
	}
	end := start + maxLines
	if end > len(r.lines) {
		end = len(r.lines)
	}

	for i := start; i < end; i++ {
		line := r.lines[i]
		if line == "" {
			sb.WriteString("\n")
			continue
		}
		sb.WriteString("  ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	scrollInfo := ""
	if len(r.lines) > maxLines {
		scrollInfo = r.theme.Dim().Render("   (scroll ↑/↓)")
	}
	sb.WriteString("\n")
	sb.WriteString(r.theme.Dim().Render("  Press Esc to go back"))
	sb.WriteString(scrollInfo)
	sb.WriteString("\n")

	return sb.String()
}
