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
	Validate    func(string) error
}

// Form is a generic form screen.
type Form struct {
	title      string
	help       string
	context    string // to identify the form result
	inputs     []components.TextInput
	labels     []string
	validators []func(string) error
	cursor     int
	theme      theme.Theme
	skippable  bool
	errMessage string
}

// NewForm creates a new generic form screen.
func NewForm(title, help, context string, fields []FormInput, th theme.Theme) *Form {
	var inputs []components.TextInput
	var labels []string
	var validators []func(string) error

	for _, f := range fields {
		ti := components.NewTextInput(th, f.Placeholder, f.IsPassword)
		if f.Value != "" {
			ti.SetValue(f.Value)
		}
		inputs = append(inputs, ti)
		labels = append(labels, f.Label)
		validators = append(validators, f.Validate)
	}

	if len(inputs) > 0 {
		inputs[0].Focus()
	}

	return &Form{
		title:      title,
		help:       help,
		context:    context,
		inputs:     inputs,
		labels:     labels,
		validators: validators,
		theme:      th,
	}
}

// Skippable marks the form so Esc submits it with empty values instead of
// cancelling back to the previous screen — for optional steps (like setting a
// passphrase) where declining should still complete the flow, not abandon it.
func (f *Form) Skippable() *Form {
	f.skippable = true
	return f
}

func (f *Form) Init() tea.Cmd {
	return components.TextInputBlink
}

func (f *Form) Title() string { return f.title }

func (f *Form) ShortHelp() string { return f.help }

func (f *Form) validateCurrent() error {
	if f.cursor >= 0 && f.cursor < len(f.validators) && f.validators[f.cursor] != nil {
		return f.validators[f.cursor](f.inputs[f.cursor].Value())
	}
	return nil
}

func (f *Form) validateAll() (int, error) {
	for i := range f.inputs {
		if f.validators[i] != nil {
			if err := f.validators[i](f.inputs[i].Value()); err != nil {
				return i, err
			}
		}
	}
	return -1, nil
}

func (f *Form) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if core.IsEscKey(msg) {
			if f.skippable {
				values := make([]string, len(f.inputs))
				return f, func() tea.Msg {
					return core.FormResultMsg{Context: f.context, Values: values}
				}
			}
			return f, func() tea.Msg { return core.ScreenPopMsg{} }
		}
		switch msg.String() {
		case core.KeyCtrlC:
			return f, tea.Quit
		case core.KeyEnter:
			// Check validation before moving or submitting
			if err := f.validateCurrent(); err != nil {
				f.errMessage = err.Error()
				return f, nil
			}
			f.errMessage = ""

			if f.cursor == len(f.inputs)-1 {
				// Final submit check across all inputs
				if idx, err := f.validateAll(); err != nil {
					f.cursor = idx
					f.errMessage = err.Error()
					return f, f.focusActive()
				}
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
				f.errMessage = ""
				return f, f.focusActive()
			}
		case core.KeyDown, core.KeyTab:
			if f.cursor < len(f.inputs)-1 {
				f.cursor++
				f.errMessage = ""
				return f, f.focusActive()
			}
		}
	}

	if len(f.inputs) > 0 {
		var cmd tea.Cmd
		f.inputs[f.cursor], cmd = f.inputs[f.cursor].Update(msg)
		if f.errMessage != "" && f.validateCurrent() == nil {
			f.errMessage = ""
		}
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

	padTop := (height - 14) / 2
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

		// Show field validation message if invalid
		if f.validators[i] != nil {
			val := f.inputs[i].Value()
			if val != "" {
				if err := f.validators[i](val); err != nil {
					lines = append(lines, "  "+f.theme.ErrorStyle().Render("⚠ "+err.Error()))
				}
			}
		}
		lines = append(lines, "")
	}

	if f.errMessage != "" {
		lines = append(lines, "  "+f.theme.ErrorStyle().Render("✖ "+f.errMessage))
		lines = append(lines, "")
	}

	footer := "  Enter to submit, Esc to cancel"
	if f.skippable {
		footer = "  Enter to submit, Esc to skip"
	}
	lines = append(lines, f.theme.Dim().Render(footer))
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
