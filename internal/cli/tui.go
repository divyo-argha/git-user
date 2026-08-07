package cli

import (
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui"
	"github.com/divyo-argha/git-user/internal/ui"
)

// runTui starts the interactive Bubble Tea UI at the dashboard.
func runTui() error {
	return launchTUI("")
}

// runTuiForIdentity starts the interactive Bubble Tea UI focusing on a specific identity.
func runTuiForIdentity(name string) error {
	return launchTUI(name)
}

// launchTUI runs the TUI once. Every operation (switch, register, logout, export,
// import, security audit, …) now completes inside the TUI itself, so the TUI only
// returns when the user explicitly quits — there is no relaunch loop.
func launchTUI(startDetail string) error {
	store, err := config.Load()
	if err != nil {
		return err
	}

	if _, _, _, err := tui.Run(store, startDetail); err != nil {
		return err
	}
	return nil
}

// handleUnknownArg checks if arg is an identity name and opens detail view,
// otherwise returns false so root.go can show "unknown command".
func handleUnknownArg(name string) bool {
	store, err := config.Load()
	if err != nil {
		return false
	}
	// Check exact match first
	if store.FindUser(name) != nil {
		if err := runTuiForIdentity(name); err != nil {
			ui.Errorf("TUI error: %v", err)
		}
		return true
	}
	// Suggest similar names
	var similar []string
	lower := strings.ToLower(name)
	for _, u := range store.Users {
		if strings.Contains(strings.ToLower(u.Name), lower) {
			similar = append(similar, u.Name)
		}
	}
	if len(similar) > 0 {
		ui.Errorf("identity %q not found — did you mean: %s", name, strings.Join(similar, ", "))
		return true
	}
	return false
}
