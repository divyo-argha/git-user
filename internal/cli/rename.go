package cli

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

// runRename renames an identity, keeping the active git config in sync when the
// active identity is renamed.
func runRename(args []string) error {
	if len(args) < 2 {
		ui.Error("usage: git-user rename <old-name> <new-name>")
		return fmt.Errorf("missing arguments")
	}

	oldName := args[0]
	newName := args[1]
	if newName == "" {
		ui.Error("new name must not be empty")
		return fmt.Errorf("empty name")
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if store.FindUser(oldName) == nil {
		ui.Errorf("identity %q not found", oldName)
		return fmt.Errorf("user not found")
	}

	if oldName != newName && store.FindUser(newName) != nil {
		ui.Errorf("identity %q already exists — choose a different name", newName)
		return fmt.Errorf("name exists")
	}

	if err := store.RenameUser(oldName, newName); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	// If this is the active user, re-apply git config so the global user.name
	// matches the new profile name.
	if store.Current == newName {
		u := store.FindUser(newName)
		if err := git.Apply(u.Name, u.Email); err != nil {
			ui.Errorf("re-applying git config: %v", err)
		}
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Renamed %q → %q", oldName, newName))
	return nil
}
