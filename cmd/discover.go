package cmd

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runDiscover(args []string) error {
	ui.Info("Scanning system for existing identities and keys...")

	results, err := config.Harvest()
	if err != nil {
		ui.Errorf("discovery failed: %v", err)
		return err
	}

	if len(results) == 0 {
		ui.Warn("No existing Git identities or SSH keys found.")
		return nil
	}

	store, err := config.Load()
	if err != nil {
		return err
	}

	addedCount := 0
	for _, d := range results {
		// Skip if user already exists
		if store.FindUser(d.Name) != nil {
			continue
		}

		user := config.User{
			Name:          d.Name,
			Email:         d.Email,
			SSHKey:        d.SSHKey,
			SigningKey:    d.SigningKey,
			SigningMethod: d.Method,
		}

		store.Users = append(store.Users, user)
		ui.Success(fmt.Sprintf("Imported identity: %s (%s)", d.Name, d.Email))
		addedCount++
	}

	if addedCount > 0 {
		if err := config.Save(store); err != nil {
			return err
		}
		ui.Success(fmt.Sprintf("Smart Discovery finished. Added %d new identities.", addedCount))
	} else {
		ui.Info("No new identities found that weren't already in your config.")
	}

	return nil
}
