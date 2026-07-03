package cmd

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runRemove(args []string) error {
	force := false
	filtered := args[:0]
	for _, a := range args {
		if a == "--force" || a == "-f" {
			force = true
		} else {
			filtered = append(filtered, a)
		}
	}
	args = filtered

	if len(args) < 1 {
		ui.Error("usage: git-user remove <n> [--force]")
		return fmt.Errorf("missing arguments")
	}

	name := args[0]

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := store.FindUser(name)
	if user == nil {
		ui.Errorf("identity %q not found", name)
		return fmt.Errorf("user not found")
	}
	sshKey := user.SSHKey

	if err := store.RemoveUser(name, force); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	_ = deleteKeychainPassphrase(name)

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Removed identity %q", name))

	if sshKey != "" {
		if ui.Confirm(fmt.Sprintf("Delete SSH key file %s?", sshKey), false) {
			os.Remove(sshKey)
			os.Remove(sshKey + ".pub")
			ui.Success("SSH key files deleted")
		}
	}

	if store.Current == "" {
		ui.Warn("No active identity — run 'git-user switch <name>'")
	}
	return nil
}
