package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runRegister(args []string) error {
	var name, email, sshKey string

	// Parse arguments if provided
	if len(args) >= 1 {
		name = args[0]
	}
	if len(args) >= 2 {
		email = args[1]
	}

	// Interactive prompts for missing data
	if name == "" {
		var err error
		name, err = ui.Prompt("Enter name for this identity (e.g., 'work'):")
		if err != nil {
			return err
		}
	}

	if email == "" {
		var err error
		email, err = ui.Prompt("Enter email address:")
		if err != nil {
			return err
		}
	}

	// Always prompt for SSH key in the register flow
	ui.Info("Paste your SSH key below (private key content or path):")
	sshKey, _ = ui.Prompt("SSH Key:")

	if name == "" || email == "" {
		ui.Error("Name and email are required.")
		return fmt.Errorf("missing arguments")
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	// Add the user
	if err := store.AddUser(name, email); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	// If SSH key is provided, bind it
	if sshKey != "" {
		// If it looks like a private key content, save it to a file
		if strings.Contains(sshKey, "-----BEGIN") {
			home, _ := os.UserHomeDir()
			sshDir := filepath.Join(home, ".ssh")
			keyPath := filepath.Join(sshDir, "git-user_"+name)

			if err := os.MkdirAll(sshDir, 0700); err != nil {
				ui.Errorf("creating .ssh directory: %v", err)
			} else {
				if err := os.WriteFile(keyPath, []byte(sshKey), 0600); err != nil {
					ui.Errorf("saving ssh key: %v", err)
				} else {
					ui.Success(fmt.Sprintf("Saved SSH key to %s", keyPath))
					sshKey = keyPath
				}
			}
		}

		if err := store.BindSSHKey(name, sshKey); err != nil {
			ui.Errorf("binding ssh key: %v", err)
		}
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Registered user %q (%s)", name, email))
	if sshKey != "" {
		ui.Info(fmt.Sprintf("SSH key bound: %s", sshKey))
	}
	ui.Info(fmt.Sprintf("Run 'git-user switch %s' to activate", name))

	return nil
}
