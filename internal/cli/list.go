package cli

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
		ui.Info("Run 'git-user register' to add one.")
		return nil
	}

	ui.Banner("Git Identities")
	for _, u := range store.Users {
		ui.UserRow(u.Name, u.Email, u.SSHKey, u.Name == store.Current, u.Source == "original")
	}

	if store.Current == "" {
		ui.Warn("No active identity — run 'git-user switch <n>'")
	}
	return nil
}
