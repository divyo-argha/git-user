package cmd

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runTui() error {
	ui.Header("Git-User Interactive Mode")

	for {
		options := []string{
			"Add New User (Simple)",
			"Register New User (Unified + SSH)",
			"Switch Active User",
			"Show Current User",
			"List All Users",
			"Bind Keys (SSH/Signing)",
			"Remove User",
			"Exit",
		}

		idx, err := ui.Select("Choose an action:", options)
		if err != nil {
			return err
		}

		switch idx {
		case 0: // Add
			if err := runAdd(nil); err != nil {
				ui.Errorf("add failed: %v", err)
			}
		case 1: // Register
			if err := runRegister(nil); err != nil {
				ui.Errorf("register failed: %v", err)
			}
		case 2: // Switch
			if err := handleTuiSwitch(); err != nil {
				ui.Errorf("switch failed: %v", err)
			}
		case 3: // Current
			if err := runCurrent(nil); err != nil {
				ui.Errorf("current failed: %v", err)
			}
		case 4: // List
			if err := runList(nil); err != nil {
				ui.Errorf("list failed: %v", err)
			}
		case 5: // Bind
			if err := handleTuiBind(); err != nil {
				ui.Errorf("bind failed: %v", err)
			}
		case 6: // Remove
			if err := handleTuiRemove(); err != nil {
				ui.Errorf("remove failed: %v", err)
			}
		case 7: // Exit
			ui.Info("Goodbye!")
			return nil
		}
		fmt.Println()
	}
}

func handleTuiSwitch() error {
	store, err := config.Load()
	if err != nil {
		return err
	}
	if len(store.Users) == 0 {
		ui.Warn("No users found.")
		return nil
	}

	names := make([]string, len(store.Users))
	for i, u := range store.Users {
		prefix := ""
		if u.Name == store.Current {
			prefix = "* "
		}
		names[i] = prefix + u.Name
	}

	idx, err := ui.Select("Select user to switch to:", names)
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
		ui.Warn("No users found.")
		return nil
	}

	names := make([]string, len(store.Users))
	for i, u := range store.Users {
		names[i] = u.Name
	}

	idx, err := ui.Select("Select user to bind keys for:", names)
	if err != nil {
		return err
	}

	name := store.Users[idx].Name
	bindType, err := ui.Select("What do you want to bind?", []string{"SSH Key", "Signing Key", "Cancel"})
	if err != nil {
		return err
	}

	switch bindType {
	case 0: // SSH
		path, err := ui.Prompt("Enter path to SSH key (e.g., ~/.ssh/id_rsa):")
		if err != nil || path == "" {
			return nil
		}
		return runBind([]string{name, "--ssh-key", path})
	case 1: // Signing
		key, err := ui.Prompt("Enter Signing Key (GPG ID or SSH Key path):")
		if err != nil || key == "" {
			return nil
		}
		method, err := ui.Select("Select signing method:", []string{"gpg", "ssh"})
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
		ui.Warn("No users found.")
		return nil
	}

	names := make([]string, len(store.Users))
	for i, u := range store.Users {
		names[i] = u.Name
	}

	idx, err := ui.Select("Select user to remove:", names)
	if err != nil {
		return err
	}

	confirm, err := ui.Prompt(fmt.Sprintf("Are you sure you want to remove %q? (y/N):", store.Users[idx].Name))
	if err != nil || (confirm != "y" && confirm != "Y") {
		ui.Info("Aborted.")
		return nil
	}

	return runRemove([]string{store.Users[idx].Name})
}
