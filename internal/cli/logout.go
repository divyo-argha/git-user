package cli

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/switchops"
	"github.com/divyo-argha/git-user/internal/ui"
)

// runLogout signs out the active identity (see switchops.Logout for exactly
// what that undoes). Signing out of a temporary identity permanently deletes
// it and its keys, so in a terminal that asks first; -y/--yes skips the
// question, and non-interactive use proceeds because the command was explicit.
func runLogout(args []string) error {
	assumeYes := false
	for _, a := range args {
		switch a {
		case "-y", "--yes":
			assumeYes = true
		case "-s", "--session":
			// Ending a per-terminal session is done by the shell integration
			// (it evals `git-user env --unset`); the binary cannot change its
			// parent shell's environment. Reaching here means there is no
			// wrapper, and silently signing out globally instead would be a
			// much bigger action than the one asked for.
			err := fmt.Errorf("--session needs the shell integration; run: eval \"$(git-user env --unset)\"")
			ui.Errorf("%v", err)
			return err
		}
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := store.CurrentUser()
	if user == nil {
		ui.Info("Already signed out — no active identity.")
		return nil
	}

	if user.IsTemporary && !assumeYes && ui.IsTTY() {
		question := fmt.Sprintf("%q is a temporary identity: signing out permanently deletes it and its SSH key. Continue?", user.Name)
		if !ui.Confirm(question, false) {
			ui.Info("Sign out cancelled.")
			return nil
		}
	}

	res, err := switchops.Logout(store, unsetActiveCustomConfig)
	if err != nil {
		ui.Errorf("%v", err)
		return err
	}
	for _, n := range res.Notices {
		ui.Info(n)
	}

	ui.Success(fmt.Sprintf("Signed out from %q", res.Name))
	ui.Info("No active git identity. Run 'git-user switch <name>' to log back in.")
	return nil
}
