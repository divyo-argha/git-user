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
	return "↑/↓•navigate  Tab•pane  Enter•select  /•filter  r•refresh  ?•keys  q•quit"
}

// DetailHelp returns the help text for the detail screen.
func DetailHelp() string {
	return "↑/↓•navigate  Enter•select  Esc•back  ?•keys  q•quit"
}

// FormHelp returns the help text for inline forms.
func FormHelp() string {
	return "Tab•next field  Shift+Tab•prev field  Enter•submit  Esc•cancel"
}

// ConfirmHelp returns the help text for confirmation dialogs.
func ConfirmHelp() string {
	return "←/→/↑/↓•select  y•yes  n•no  Enter•confirm  Esc•cancel  q•quit"
}

// FilterHelp returns the help text when filter mode is active.
func FilterHelp() string {
	return "Type to filter  Enter•select  Esc•clear filter"
}

// ImportExportHelp returns the help text for the Import/Export sub-screen.
func ImportExportHelp() string {
	return "↑/↓/j/k•navigate  Enter•select  Esc/b•back  q•quit"
}

// OptionsHelp returns the help text for generic option/choice screens.
func OptionsHelp() string {
	return "↑/↓/j/k•navigate  Enter•select  Esc•cancel  q•quit"
}

// HelpTitleDashboard and HelpTitleDetail name the full-keyboard-reference
// screen opened with "?" from each context.
const (
	HelpTitleDashboard = "Keyboard shortcuts — dashboard"
	HelpTitleDetail    = "Keyboard shortcuts — profile"
)

// DashboardHelpText is the full keyboard reference for the dashboard,
// including the shortcuts too situational to fit the one-line hint bar.
func DashboardHelpText() string {
	return `Navigation
  ↑ / ↓ / j / k          Move up / down
  Tab / ← / → / h / l    Switch between the identities and actions panes
  Enter                  Open the selected identity or run the selected action
  /                      Filter identities (Esc clears the filter)
  Mouse                  Click a pane to focus it, wheel to scroll

Shortcuts
  s                      Switch to the highlighted identity
  r                      Refresh identities and status
  u                      Install the update (only when one is available)
  f                      Re-apply the active identity (only when git config has drifted)

General
  ?                      Show this reference
  q                      Quit (asks for confirmation)
  Ctrl+C                 Quit immediately`
}

// DetailHelpText is the full keyboard reference for an identity's profile screen.
func DetailHelpText() string {
	return `Navigation
  ↑ / ↓ / j / k          Move up / down
  Enter                  Run the selected action
  Esc / b                Back to the dashboard

Shortcuts
  s                      Switch to this identity (when it is not already active)

General
  ?                      Show this reference
  q                      Quit (asks for confirmation)
  Ctrl+C                 Quit immediately`
}
