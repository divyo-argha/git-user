package screens

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// Report is a scrollable read-only text screen used to display operation
// output (public keys, security audits, export/import summaries, doctor, …).
type Report struct {
	title    string
	lines    []string
	offset   int
	maxLines int // updated each render, used to clamp scroll in Update
	theme    theme.Theme
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
	return "↑/↓/j/k•scroll  ctrl+d/u•page  c•copy  Enter/Esc•back  q•quit"
}

// maxScrollOffset returns the highest valid offset for the current maxLines.
func (r *Report) maxScrollOffset() int {
	max := len(r.lines) - r.maxLines
	if max < 0 {
		max = 0
	}
	return max
}

func (r *Report) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if core.IsEscKey(msg) || msg.String() == "b" || msg.String() == "B" || msg.String() == core.KeyEnter {
			return r, func() tea.Msg { return core.ScreenPopMsg{} }
		}
		switch msg.String() {
		case core.KeyCtrlC:
			return r, tea.Quit
		case core.KeyQuit:
			return r, func() tea.Msg { return core.ActionResultMsg{Kind: "quit-confirm"} }

		case core.KeyUp, core.KeyK:
			if r.offset > 0 {
				r.offset--
			}

		case core.KeyDown, core.KeyJ:
			// Clamp: never scroll past the last page of content.
			if r.offset < r.maxScrollOffset() {
				r.offset++
			}

		// Page down: ctrl+d or pgdown — move half a screen.
		case "ctrl+d", "pgdown":
			half := r.maxLines / 2
			if half < 1 {
				half = 1
			}
			r.offset += half
			if r.offset > r.maxScrollOffset() {
				r.offset = r.maxScrollOffset()
			}

		// Page up: ctrl+u or pgup — move half a screen.
		case "ctrl+u", "pgup":
			half := r.maxLines / 2
			if half < 1 {
				half = 1
			}
			r.offset -= half
			if r.offset < 0 {
				r.offset = 0
			}

		// Copy report content to clipboard.
		case "c", "C":
			return r, copyToClipboardCmd(r.lines)
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

	// Reserve lines for header (title + separator + 2 blank = 4) and footer (2).
	maxLines := height - 8
	if maxLines < 3 {
		maxLines = 3
	}
	// Store for use in Update's scroll clamp.
	r.maxLines = maxLines

	// Clamp offset defensively in case window was resized smaller.
	maxOff := len(r.lines) - maxLines
	if maxOff < 0 {
		maxOff = 0
	}
	if r.offset > maxOff {
		r.offset = maxOff
	}

	start := r.offset
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

	// Smart scroll indicators: only show directions that are still available.
	var hints []string
	if r.offset > 0 {
		hints = append(hints, "↑ more above")
	}
	if r.offset < maxOff {
		hints = append(hints, "↓ more below")
	}

	sb.WriteString("\n")
	sb.WriteString(r.theme.Dim().Render("  Press Enter or Esc to go back"))
	if len(hints) > 0 {
		sb.WriteString(r.theme.Dim().Render("   (" + strings.Join(hints, " • ") + ")"))
	}
	sb.WriteString("\n")

	return sb.String()
}

// copyToClipboardCmd copies the report lines to the system clipboard and
// shows a toast notification. It tries pbcopy (macOS), xclip, then xsel.
func copyToClipboardCmd(lines []string) tea.Cmd {
	return func() tea.Msg {
		text := strings.Join(lines, "\n")
		if err := ClipboardWrite(text); err != nil {
			return core.ToastMsg{
				Text:     "Copy failed: " + err.Error(),
				Style:    theme.ToastStyleError,
				Duration: 4 * time.Second,
			}
		}
		return core.ToastMsg{
			Text:     "Copied to clipboard!",
			Style:    theme.ToastStyleSuccess,
			Duration: 2 * time.Second,
		}
	}
}
