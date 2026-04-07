package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runAdd(args []string) error {
	var name, email, signingKey, method, sshKey string
	posArgs := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--signing-key":
			if i+1 < len(args) {
				signingKey = args[i+1]
				i++
			}
		case "--method":
			if i+1 < len(args) {
				method = args[i+1]
				i++
			}
		case "--ssh-key":
			if i+1 < len(args) {
				sshKey = args[i+1]
				i++
			}
		default:
			if posArgs == 0 {
				name = args[i]
			} else if posArgs == 1 {
				email = args[i]
			}
			posArgs++
		}
	}

	// Interactive entry if missing
	if name == "" {
		var err error
		name, err = ui.Prompt("Enter name for this identity (e.g., 'work')")
		if err != nil {
			return err
		}
	}
	if email == "" {
		var err error
		email, err = ui.Prompt("Enter email address")
		if err != nil {
			return err
		}
	}

	if name == "" || email == "" {
		ui.Error("Name and email are required.")
		return fmt.Errorf("missing information")
	}

	// Optional SSH Key
	if sshKey == "" {
		ui.Info("You can associate an SSH key now, or skip by pressing Enter.")
		keyInput, _ := ui.Prompt("SSH Key (path or content)")
		sshKey = keyInput
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if err := store.AddUser(name, email); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	// Handle SSH Key binding/saving
	if sshKey != "" {
		if strings.Contains(sshKey, "-----BEGIN") {
			home, _ := os.UserHomeDir()
			sshDir := filepath.Join(home, ".ssh")
			keyPath := filepath.Join(sshDir, "git-user_"+name)

			if err := os.MkdirAll(sshDir, 0700); err != nil {
				ui.Errorf("creating .ssh directory: %v", err)
			} else {
				if err := os.WriteFile(keyPath, []byte(sshKey), 0600); err != nil {
					ui.Errorf("saving SSH key: %v", err)
				} else {
					ui.Success(fmt.Sprintf("Saved SSH key to %s", keyPath))
					sshKey = keyPath
				}
			}
		}
		if err := store.BindSSHKey(name, sshKey); err != nil {
			ui.Errorf("binding SSH key: %v", err)
		}
	}

	if signingKey != "" {
		if err := store.BindSigningKey(name, signingKey, method); err != nil {
			ui.Errorf("binding signing key: %v", err)
			return err
		}
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Added identity: %s (%s)", name, email))
	if sshKey != "" {
		ui.Info(fmt.Sprintf("SSH key bound: %s", sshKey))
	}
	ui.Info(fmt.Sprintf("Run 'git-user switch %s' to activate", name))
	return nil
}
