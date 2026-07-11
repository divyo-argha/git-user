package components

import (
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// HelpBar renders the context-sensitive help footer.
type HelpBar struct {
	text  string
	theme theme.Theme
}

// NewHelpBar creates a new help bar component.
func NewHelpBar(th theme.Theme) HelpBar {
	return HelpBar{theme: th}
}

// SetText updates the help text.
func (h *HelpBar) SetText(text string) { h.text = text }

// View renders the help bar.
func (h HelpBar) View(width int) string {
	return h.theme.ItalicStyle().Render(h.text)
}
