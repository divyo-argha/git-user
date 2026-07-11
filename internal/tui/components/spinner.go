package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// Spinner wraps the bubbles spinner.
type Spinner struct {
	model spinner.Model
}

// NewSpinner creates a new styled spinner.
func NewSpinner(th theme.Theme) Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = th.Selected()
	return Spinner{model: s}
}

// Init returns the command to start the spinner.
func (s Spinner) Init() tea.Cmd {
	return s.model.Tick
}

// Update handles tick messages.
func (s *Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
	var cmd tea.Cmd
	s.model, cmd = s.model.Update(msg)
	return *s, cmd
}

// View returns the spinner view.
func (s Spinner) View() string {
	return s.model.View()
}
