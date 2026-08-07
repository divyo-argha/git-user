package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// Run launches the TUI. Every operation completes inside the TUI itself, so it
// only returns when the user quits (nil error) or when the program fails.
func Run(store *config.Store, startDetail string) error {
	th := theme.DefaultTheme()

	// The dashboard is always the root of the navigation stack so that Esc / the
	// "← Back" option reliably returns to it from any sub-screen.
	initialScreen := screens.NewDashboard(store, th)

	app := NewApp(store, initialScreen)
	if startDetail != "" {
		if user := store.FindUser(startDetail); user != nil {
			// Push the detail screen on top of the dashboard; its Init runs via
			// App.Init on the active screen.
			app.pushScreen(screens.NewDetail(store, startDetail, th))
		}
	} else {
		if name, email, ok := firstRunOriginalIdentity(store); ok {
			// First-run onboarding: offer to import the existing git identity
			// inside the TUI instead of prompting on the plain terminal.
			app.pushScreen(screens.NewFirstRun(store, name, email, th))
		}
	}

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	_, err := p.Run()
	return err
}
