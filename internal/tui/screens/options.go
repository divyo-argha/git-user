package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// Option defines a single selectable choice in an Options screen.
type Option struct {
	Label string
	Key   string
}

// Options is a generic single-choice menu used for in-TUI decision points
// (SSH setup, platform selection, directory to unbind, …).
type Options struct {
	title   string
	help    string
	context string
	options []Option
	cursor  int
	theme   theme.Theme
}

// NewOptions creates an options menu with the given title and choices.
func NewOptions(title, help, context string, opts []Option, th theme.Theme) *Options {
	return &Options{
		title:   title,
		help:    help,
		context: context,
		options: opts,
		theme:   th,
	}
}

func (o *Options) Init() tea.Cmd { return nil }

func (o *Options) Title() string { return o.title }

func (o *Options) ShortHelp() string { return o.help }

func (o *Options) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if core.IsEscKey(msg) {
			return o, func() tea.Msg {
				return core.OptionResultMsg{Context: o.context, Choice: ""}
			}
		}
		switch msg.String() {
		case core.KeyCtrlC:
			return o, tea.Quit
		case core.KeyQuit:
			return o, func() tea.Msg { return core.ActionResultMsg{Kind: "quit-confirm"} }
		case core.KeyUp, core.KeyK:
			if o.cursor > 0 {
				o.cursor--
			} else {
				o.cursor = len(o.options) - 1 // wrap around
			}
		case core.KeyDown, core.KeyJ:
			if o.cursor < len(o.options)-1 {
				o.cursor++
			} else {
				o.cursor = 0 // wrap around
			}
		case "tab":
			// Tab: move forward, wrap.
			o.cursor = (o.cursor + 1) % len(o.options)
		case "shift+tab":
			// Shift+Tab: move backward, wrap.
			o.cursor = (o.cursor - 1 + len(o.options)) % len(o.options)
		case core.KeyEnter:
			choice := o.options[o.cursor].Key
			return o, func() tea.Msg {
				return core.OptionResultMsg{Context: o.context, Choice: choice}
			}
		}
	}
	return o, nil
}

func (o *Options) View(width, height int) string {
	var sb strings.Builder

	boxWidth := 58
	if width < 68 {
		boxWidth = width - 10
	}
	if boxWidth < 36 {
		boxWidth = 36
	}

	padTop := (height - 12 - len(o.options)) / 2
	if padTop < 0 {
		padTop = 0
	}
	for i := 0; i < padTop; i++ {
		sb.WriteString("\n")
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, o.theme.Bold().Render("  "+o.title))
	lines = append(lines, "")
	for i, opt := range o.options {
		label := opt.Label
		if i == o.cursor {
			label = o.theme.Selected().Render("▶ " + label)
		} else {
			label = "  " + label
		}
		lines = append(lines, "  "+label)
	}
	lines = append(lines, "")
	lines = append(lines, o.theme.Dim().Render("  ↑/↓/Tab: navigate • Enter: select • Esc: cancel"))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := o.theme.ActionPane(boxWidth, 0).Render(content)

	padLeft := (width - boxWidth - 6) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	padStr := strings.Repeat(" ", padLeft)
	for _, line := range strings.Split(box, "\n") {
		sb.WriteString(padStr)
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	return sb.String()
}
