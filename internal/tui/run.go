package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// Run launches the TUI and returns the pending action (if any) when the TUI exits.
// The returned values are: kind (action key), name (identity name), arg (extra argument).
// If kind is empty, the user quit without selecting an action.
func Run(store *config.Store, startDetail string) (kind, name, arg string, err error) {
	th := theme.DefaultTheme()
	var initialScreen Screen

	if startDetail != "" {
		user := store.FindUser(startDetail)
		if user != nil {
			initialScreen = NewDetail(store, startDetail, th)
		} else {
			initialScreen = NewDashboard(store, th)
		}
	} else {
		initialScreen = NewDashboard(store, th)
	}

	app := NewApp(store, initialScreen)
	p := tea.NewProgram(app, tea.WithAltScreen())

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
