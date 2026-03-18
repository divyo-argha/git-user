package cmd

import (
	"fmt"

	"github.com/local/git-user/internal/config"
	"github.com/local/git-user/internal/git"
	"github.com/local/git-user/internal/ui"
)

func runEdit(args []string) error {
	if len(args) < 2 {
		ui.Error("usage: git-user edit <n> <new-email>")
		return fmt.Errorf("missing arguments")
	}

	name := args[0]
	newEmail := args[1]

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if err := store.UpdateUser(name, newEmail); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	// If this is the active user, re-apply git config immediately.
	if store.Current == name {
		u := store.FindUser(name)
		if err := git.Apply(u.Name, u.Email); err != nil {
			ui.Errorf("re-applying git config: %v", err)
			return err
		}
		ui.Info("Active identity updated — git config re-applied automatically.")
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Updated %q → email is now %s", name, newEmail))
	return nil
}
