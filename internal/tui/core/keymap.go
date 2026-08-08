package core

import tea "github.com/charmbracelet/bubbletea"

// Keymap defines all keybindings for the TUI in a central place.
// Screens reference these constants for consistent behavior.

// ── Navigation Keys ───────────────────────────────────────────────────────────

const (
	KeyUp     = "up"
	KeyDown   = "down"
	KeyLeft   = "left"
	KeyRight  = "right"
	KeyK      = "k"
	KeyJ      = "j"
	KeyH      = "h"
	KeyL      = "l"
	KeyTab    = "tab"
	KeyEnter  = "enter"
	KeyEsc    = "esc"
	KeyQuit   = "q"
	KeyCtrlC  = "ctrl+c"
	KeyFilter = "/"
	KeyHelp   = "?"
)

// IsEscKey checks if a tea.KeyMsg is an Escape key event across all terminal types.
func IsEscKey(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyEsc || msg.Type == tea.KeyEscape || msg.String() == "esc" || msg.String() == "\x1b"
}

// ── Help Text Builders ────────────────────────────────────────────────────────

// DashboardHelp returns the help text for the main dashboard.
func DashboardHelp() string {
	return "  Tab/←/→ switch pane  ↑/↓ navigate  s switch  Enter select  / filter  q quit"
}

// DetailHelp returns the help text for the detail screen.
func DetailHelp() string {
	return "  ↑/↓ navigate  s switch profile  Enter select  Esc back  q quit"
}

// FormHelp returns the help text for inline forms.
func FormHelp() string {
	return "  Enter submit  Esc cancel/back  Tab next field"
}

// ConfirmHelp returns the help text for confirmation dialogs.
func ConfirmHelp() string {
	return "  ←/→ select  Enter confirm  Esc cancel"
}

// FilterHelp returns the help text when filter mode is active.
func FilterHelp() string {
	return "  Type to filter  Enter select  Esc clear filter"
}

// ImportExportHelp returns the help text for the Import/Export sub-screen.
func ImportExportHelp() string {
	return "  ↑/↓/j/k navigate  Enter select  Esc back"
}

// OptionsHelp returns the help text for generic option/choice screens.
func OptionsHelp() string {
	return "  ↑/↓/j/k navigate  Enter select  Esc cancel"
}
