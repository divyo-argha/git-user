package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// HelpModal is a fullscreen guide overlay for beginner navigation & shortcut reference.
type HelpModal struct {
	theme theme.Theme
}

// NewHelpModal returns a new HelpModal screen.
func NewHelpModal(th theme.Theme) *HelpModal {
	return &HelpModal{theme: th}
}

func (h *HelpModal) Init() tea.Cmd { return nil }

func (h *HelpModal) Title() string { return "Help & Guide" }

func (h *HelpModal) ShortHelp() string {
	return "esc • close help  q • quit"
}

func (h *HelpModal) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case core.KeyEsc, "q", "?":
			return h, func() tea.Msg { return core.ScreenPopMsg{} }
		}
	}
	return h, nil
}

func (h *HelpModal) View(width, height int) string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(h.theme.Primary).
		Bold(true).
		MarginBottom(1)

	sectionHeader := lipgloss.NewStyle().
		Foreground(h.theme.Secondary).
		Bold(true)

	b.WriteString(titleStyle.Render("📖 Git-User Quick Navigation & Command Guide"))
	b.WriteString("\n\n")

	b.WriteString(sectionHeader.Render("⌨️  GLOBAL NAVIGATION KEYBINDINGS"))
	b.WriteString("\n")
	b.WriteString(h.renderRow("Tab", "Switch active pane (Identities ↔ Actions)"))
	b.WriteString(h.renderRow("↑ / k , ↓ / j", "Navigate items up / down"))
	b.WriteString(h.renderRow("Enter", "Select / activate profile or action"))
	b.WriteString(h.renderRow("/", "Filter identities by name or email"))
	b.WriteString(h.renderRow("Ctrl+P", "Open Fuzzy Command Palette"))
	b.WriteString(h.renderRow("?", "Toggle this interactive Help Modal"))
	b.WriteString(h.renderRow("Esc / q", "Go back / Quit TUI"))
	b.WriteString("\n\n")

	b.WriteString(sectionHeader.Render("🖱️  MOUSE GESTURES"))
	b.WriteString("\n")
	b.WriteString(h.renderRow("Left Click", "Focus pane or select item directly"))
	b.WriteString(h.renderRow("Scroll Wheel", "Scroll active list up or down"))
	b.WriteString("\n\n")

	b.WriteString(sectionHeader.Render("⚡ QUICK WORKFLOWS"))
	b.WriteString("\n")
	b.WriteString(h.renderRow("Switch Git User", "Select identity -> Press Enter -> Choose Switch"))
	b.WriteString(h.renderRow("Add SSH Key", "Select identity -> Detail view -> Add SSH key"))
	b.WriteString(h.renderRow("Auto-switch Path", "Bind local folder path to auto-switch identity"))
	b.WriteString("\n\n")

	b.WriteString(h.theme.Dim().Render("Press [Esc] or [?] to close this help window."))

	box := h.theme.ActivePane(width, height).Render(b.String())
	return box
}

func (h *HelpModal) renderRow(key, desc string) string {
	k := h.theme.Keycap().Render(key)
	d := h.theme.Bold().Render(desc)
	return "  " + k + strings.Repeat(" ", max(1, 16-len(stripAnsi(k)))) + " " + d + "\n"
}

func stripAnsi(s string) string {
	var result strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
