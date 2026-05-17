package cmd

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runTui() error {
	ui.Banner("🎯 Git-User Manager")
	fmt.Println()

	for {
		store, _ := config.Load()
		activeStatus := ""
		if store != nil && store.Current != "" {
			activeStatus = fmt.Sprintf(" [✓ Active: %s]", store.Current)
		}

		userCount := 0
		if store != nil {
			userCount = len(store.Users)
		}
		countStatus := fmt.Sprintf(" [%d identities]", userCount)

		options := []string{
			"📝 Register New Identity (Guided Setup)",
			"🔄 Switch Active User",
			fmt.Sprintf("👤 Show Current User%s", activeStatus),
			fmt.Sprintf("📋 List All Users%s", countStatus),
			"🔑 Bind Keys (SSH/Signing)",
			"🗑️  Remove User",
			"🔧 Run Diagnostics",
			"👋 Exit",
		}

		idx, err := ui.Select("Choose an action:", options)
		if err != nil {
			return err
		}

		switch idx {
		case 0:
			if err := runRegister(nil); err != nil {
				ui.Errorf("register failed: %v", err)
			}
		case 1:
			if err := handleTuiSwitch(); err != nil {
				ui.Errorf("switch failed: %v", err)
			}
		case 2:
			if err := runCurrent(nil); err != nil {
				ui.Errorf("current failed: %v", err)
			}
		case 3:
			if err := runList(nil); err != nil {
				ui.Errorf("list failed: %v", err)
			}
		case 4:
			if err := handleTuiBind(); err != nil {
				ui.Errorf("bind failed: %v", err)
			}
		case 5:
			if err := handleTuiRemove(); err != nil {
				ui.Errorf("remove failed: %v", err)
			}
		case 6:
			if err := runDoctor(nil); err != nil {
				ui.Errorf("doctor failed: %v", err)
			}
		case 7:
			ui.Info("👋 Goodbye!")
			return nil
		}
		ui.Divider()
	}
}

func handleTuiSwitch() error {
	store, err := config.Load()
	if err != nil {
		return err
	}
	if len(store.Users) == 0 {
		ui.Warn("No identities found.")
		return nil
	}

	names := make([]string, len(store.Users))
	for i, u := range store.Users {
		status := "○"
		if u.Name == store.Current {
			status = "✓"
		}
		email := ""
		if u.Email != "" {
			email = fmt.Sprintf(" (%s)", u.Email)
		}
		names[i] = fmt.Sprintf("%s %s%s", status, u.Name, email)
	}

	idx, err := ui.Select("Select identity to switch to:", names)
	if err != nil {
		return err
	}

	return runSwitch([]string{store.Users[idx].Name})
}

func handleTuiBind() error {
	store, err := config.Load()
	if err != nil {
		return err
	}
	if len(store.Users) == 0 {
		ui.Warn("No identities found.")
		return nil
	}

	names := make([]string, len(store.Users))
	for i, u := range store.Users {
		names[i] = u.Name
	}

	idx, err := ui.Select("Select identity to bind keys for:", names)
	if err != nil {
		return err
	}

	name := store.Users[idx].Name
	bindType, err := ui.Select("What do you want to bind?", []string{"🔑 SSH Key", "✍️  Signing Key", "❌ Cancel"})
	if err != nil {
		return err
	}

	switch bindType {
	case 0:
		path, err := ui.Prompt("Enter path to SSH key (e.g., ~/.ssh/id_rsa):")
		if err != nil || path == "" {
			return nil
		}
		return runBind([]string{name, "--ssh-key", path})
	case 1:
		key, err := ui.Prompt("Enter Signing Key (GPG ID or SSH Key path):")
		if err != nil || key == "" {
			return nil
		}
		method, err := ui.Select("Select signing method:", []string{"🔐 GPG", "🔑 SSH"})
		if err != nil {
			return err
		}
		return runBind([]string{name, "--signing-key", key, "--method", []string{"gpg", "ssh"}[method]})
	default:
		return nil
	}
}

func handleTuiRemove() error {
	store, err := config.Load()
	if err != nil {
		return err
	}
	if len(store.Users) == 0 {
		ui.Warn("No identities found.")
		return nil
	}

	names := make([]string, len(store.Users))
	for i, u := range store.Users {
		names[i] = u.Name
	}

	idx, err := ui.Select("Select identity to remove:", names)
	if err != nil {
		return err
	}

	confirm, err := ui.Prompt(fmt.Sprintf("⚠️  Are you sure you want to remove %q? (y/N):", store.Users[idx].Name))
	if err != nil || (confirm != "y" && confirm != "Y") {
		ui.Info("Cancelled.")
		return nil
	}

	return runRemove([]string{store.Users[idx].Name})
}
