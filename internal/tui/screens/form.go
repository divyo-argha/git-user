package screens

import (
	"github.com/divyo-argha/git-user/internal/tui/core"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// FormInput defines a single input field in a form.
type FormInput struct {
	Label       string
	Placeholder string
	IsPassword  bool
	Value       string
}

// Form is a generic form screen.
type Form struct {
	title   string
	help    string
	context string // to identify the form result
	inputs  []components.TextInput
	labels  []string
	cursor  int
	theme   theme.Theme
}

// NewForm creates a new generic form screen.
func NewForm(title, help, context string, fields []FormInput, th theme.Theme) *Form {
	var inputs []components.TextInput
	var labels []string

	for _, f := range fields {
		ti := components.NewTextInput(th, f.Placeholder, f.IsPassword)
		if f.Value != "" {
			ti.SetValue(f.Value)
		}
		inputs = append(inputs, ti)
		labels = append(labels, f.Label)
	}

	if len(inputs) > 0 {
		inputs[0].Focus()
	}

	return &Form{
		title:   title,
		help:    help,
		context: context,
		inputs:  inputs,
		labels:  labels,
		theme:   th,
	}
}

func (f *Form) Init() tea.Cmd {
	return components.TextInputBlink
}

func (f *Form) Title() string { return f.title }

func (f *Form) ShortHelp() string { return f.help }

func (f *Form) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case core.KeyCtrlC:
			return f, tea.Quit
		case core.KeyEsc:
			return f, func() tea.Msg { return core.ScreenPopMsg{} }
		case core.KeyEnter:
			if f.cursor == len(f.inputs)-1 {
				// Form complete
				values := make([]string, len(f.inputs))
				for i, input := range f.inputs {
					values[i] = input.Value()
				}
				return f, func() tea.Msg {
					return core.FormResultMsg{Context: f.context, Values: values}
				}
			}
			// Move to next input
			f.cursor++
			return f, f.focusActive()

		case core.KeyUp, "shift+tab":
			if f.cursor > 0 {
				f.cursor--
				return f, f.focusActive()
			}
		case core.KeyDown, core.KeyTab:
			if f.cursor < len(f.inputs)-1 {
				f.cursor++
				return f, f.focusActive()
			}
		}
	}

	if len(f.inputs) > 0 {
		var cmd tea.Cmd
		f.inputs[f.cursor], cmd = f.inputs[f.cursor].Update(msg)
		return f, cmd
	}

	return f, nil
}

func (f *Form) focusActive() tea.Cmd {
	var cmds []tea.Cmd
	for i := range f.inputs {
		if i == f.cursor {
			cmds = append(cmds, f.inputs[i].Focus())
		} else {
			f.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (f *Form) View(width, height int) string {
	var sb strings.Builder

	boxWidth := 50
	if width < 60 {
		boxWidth = width - 10
	}
	if boxWidth < 30 {
		boxWidth = 30
	}

	padTop := (height - 12) / 2
	if padTop < 0 {
		padTop = 0
	}
	for i := 0; i < padTop; i++ {
		sb.WriteString("\n")
	}

	var lines []string
	lines = append(lines, "")

	for i := range f.inputs {
		label := f.labels[i]
		if i == f.cursor {
			label = f.theme.Active().Render(label)
		} else {
			label = f.theme.Dim().Render(label)
		}
		lines = append(lines, "  "+label)
		lines = append(lines, f.inputs[i].View(boxWidth))
		lines = append(lines, "")
	}

	lines = append(lines, f.theme.Dim().Render("  Enter to submit, Esc to cancel"))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := f.theme.ActionPane(boxWidth, 0).Render(content)

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
