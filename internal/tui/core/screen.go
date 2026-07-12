package core

import tea "github.com/charmbracelet/bubbletea"

// Screen is the interface that all TUI screens must implement.
// Each screen is a self-contained view that handles its own input and rendering.
type Screen interface {
	// Init returns an initial command for the screen (e.g., to load data).
	Init() tea.Cmd

	// Update handles messages and returns the updated screen and any commands.
	Update(msg tea.Msg) (Screen, tea.Cmd)

	// View renders the screen content (without status bar or help bar).
	View(width, height int) string

	// ShortHelp returns key binding hints for the help bar.
	ShortHelp() string

	// Title returns a display name for the screen (used in breadcrumbs if needed).
	Title() string
}
