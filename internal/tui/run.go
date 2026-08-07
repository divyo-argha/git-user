package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// Run launches the TUI and returns the pending action (if any) when the TUI exits.
// The returned values are: kind (action key), name (identity name), arg (extra argument).
// If kind is empty, the user quit without selecting an action.
func Run(store *config.Store, startDetail string) (kind, name, arg string, err error) {
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
	}

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	finalRaw, err := p.Run()
	if err != nil {
		return "", "", "", err
	}

	final := finalRaw.(*App)
	if final.Quit() || final.action == nil {
		return "", "", "", nil
	}

	k, n, a := final.PendingAction()
	return k, n, a, nil
}
