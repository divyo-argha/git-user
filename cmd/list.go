package cmd

import (
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runList(_ []string) error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if len(store.Users) == 0 {
		ui.Warn("No identities configured yet.")
		ui.Info("Run 'git-user add <n> <email>' to add one.")
		return nil
	}

	ui.Banner("Git Identities")
	for i, u := range store.Users {
		ui.UserRow(u.Name, u.Email, u.SSHKey, u.Name == store.Current)
		if i < len(store.Users)-1 {
			// Small padding if needed, but the cards have margins now.
		}
	}

	if store.Current == "" {
		ui.Warn("No active identity — run 'git-user switch <n>'")
	}
	return nil
}
