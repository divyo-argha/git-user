package cmd

import (
	"fmt"

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

	ui.Header("Git Identities")
	ui.Divider()
	for _, u := range store.Users {
		ui.UserRow(u.Name, u.Email, u.SSHKey, u.Name == store.Current)
	}
	ui.Divider()

	if store.Current == "" {
		ui.Warn("No active identity — run 'git-user switch <n>'")
	} else {
		fmt.Printf("  Active: %s\n", store.Current)
	}
	return nil
}
