package components

import (
	"strings"

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

// View renders the help bar with styled keycaps.
func (h HelpBar) View(width int) string {
	if h.text == "" {
		return ""
	}
	parts := strings.Split(h.text, "  ")
	var formatted []string
	for _, p := range parts {
		if strings.Contains(p, "•") {
			sub := strings.SplitN(p, "•", 2)
			key := strings.TrimSpace(sub[0])
			desc := strings.TrimSpace(sub[1])
			formatted = append(formatted, h.theme.Keycap().Render(key)+" "+h.theme.Dim().Render(desc))
		} else {
			formatted = append(formatted, h.theme.ItalicStyle().Render(p))
		}
	}
	// Append permanent quick help indicators
	formatted = append(formatted, h.theme.Keycap().Render("Ctrl+P")+" "+h.theme.Dim().Render("Search"), h.theme.Keycap().Render("?")+" "+h.theme.Dim().Render("Help"))

	return strings.Join(formatted, "   ")
}
