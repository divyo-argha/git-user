package cmd

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runLogout(args []string) error {
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

	// Unload SSH key
	if user.SSHKey != "" && isSSHKeyLoaded(user.SSHKey) {
		_ = removeSSHKey(user.SSHKey)
	}

	// Clear gitconfig
	git.ClearIdentity()

	if user.IsTemporary {
		store.RemoveUser(user.Name, true)
		ui.Info(fmt.Sprintf("Temporary identity %q deleted.", user.Name))
		if user.SSHKey != "" {
			_ = os.Remove(user.SSHKey)
			_ = os.Remove(user.SSHKey + ".pub")
			ui.Info(fmt.Sprintf("Temporary SSH key files deleted: %s", user.SSHKey))
		}
		_ = deleteKeychainPassphrase(user.Name)
	}

	// Clear store.Current
	store.Current = ""
	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Signed out from %q", user.Name))
	ui.Info("No active git identity. Run 'git-user switch <name>' to log back in.")
	return nil
}
