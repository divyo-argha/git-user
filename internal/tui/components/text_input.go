package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// TextInputBlink is a command to blink the text input cursor.
func TextInputBlink() tea.Msg {
	return textinput.Blink()
}

// TextInput wraps the bubbles textinput with our theme.
type TextInput struct {
	model textinput.Model
	theme theme.Theme
}

// NewTextInput creates a new styled text input.
func NewTextInput(th theme.Theme, placeholder string, isPassword bool) TextInput {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = "  "
	ti.PromptStyle = th.Selected()
	ti.TextStyle = lipgloss.NewStyle().Foreground(th.Text)
	ti.PlaceholderStyle = th.Dim()
	ti.Cursor.Style = th.Selected()

	if isPassword {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
	}

	return TextInput{
		model: ti,
		theme: th,
	}
}

// Focus focuses the text input.
func (t *TextInput) Focus() tea.Cmd {
	return t.model.Focus()
}

// Blur blurs the text input.
func (t *TextInput) Blur() {
	t.model.Blur()
}

// Value returns the current value.
func (t *TextInput) Value() string {
	return t.model.Value()
}

// SetValue sets the value of the input.
func (t *TextInput) SetValue(v string) {
	t.model.SetValue(v)
}

// Update handles tea messages for the input.
func (t *TextInput) Update(msg tea.Msg) (TextInput, tea.Cmd) {
	var cmd tea.Cmd
	t.model, cmd = t.model.Update(msg)
	return *t, cmd
}

// View renders the text input.
func (t *TextInput) View(width int) string {
	t.model.Width = width - 4
	return t.model.View()
}
