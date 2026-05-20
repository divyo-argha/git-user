package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runSwitch(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user switch [-c] <name> [email]")
		return fmt.Errorf("missing arguments")
	}

	createMode := false
	name := ""
	email := ""

	if args[0] == "-c" {
		if len(args) < 2 {
			ui.Error("usage: git-user switch -c <name> [email]")
			return fmt.Errorf("missing name")
		}
		createMode = true
		name = args[1]
		if len(args) > 2 {
			email = args[2]
		}
	} else {
		name = args[0]
	}

	if !git.IsInstalled() {
		ui.Error("git is not installed or not on PATH")
		return fmt.Errorf("git not found")
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if createMode {
		if store.FindUser(name) != nil {
			ui.Errorf("identity %q already exists", name)
			return fmt.Errorf("user exists")
		}

		if err := quickRegister(name, email, store); err != nil {
			return err
		}

		store, _ = config.Load()
	}

	user := store.FindUser(name)
	if user == nil {
		ui.Errorf("identity %q not found", name)
		ui.Info("Create it with: git-user register")
		ui.Info("Or create and switch: git-user switch -c " + name)
		return fmt.Errorf("user not found")
	}

	if err := git.Apply(user.Name, user.Email); err != nil {
		ui.Errorf("applying git config: %v", err)
		return err
	}

	if user.SSHKey != "" {
		if err := git.ConfigureSSH(user.SSHKey); err != nil {
			ui.Warn(fmt.Sprintf("applying SSH config: %v", err))
		}

		if os.Getenv("SSH_AUTH_SOCK") == "" {
			ui.Info("ssh-agent is not running; start it with: eval \"$(ssh-agent -s)\"")
			ui.Info("Then run: git-user session start")
		} else if !isSSHKeyLoaded(user.SSHKey) {
			ui.Warn(fmt.Sprintf("SSH key for %q is not loaded", user.Name))
			if ui.Confirm("Start session now?", true) {
				if err := addSSHKey(user.SSHKey, ""); err != nil {
					ui.Warn("Could not start authenticated session")
				} else {
					ui.Success("Session started")
				}
			} else {
				ui.Info("You can start it later with: git-user session start")
			}
		}
	} else {
		if err := git.RemoveSSHConfig(); err != nil {
			ui.Warn(fmt.Sprintf("removing SSH config: %v", err))
		}
	}

	if err := store.SetCurrent(name); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Switched to %q (%s)", user.Name, user.Email))

	if user.SSHKey != "" && isSSHKeyLoaded(user.SSHKey) {
		if err := verifySSHConnection(); err != nil {
			ui.Warn("SSH verification failed. The key may not be added to your platform yet.")
			ui.Info("Test manually with: ssh -T git@github.com")
		} else {
			ui.Success("SSH verified: Connection successful!")
		}
	} else if user.SSHKey != "" {
		ui.Info("Skipping SSH verification until the key is loaded")
	}

	if git.IsInRepo() {
		remotes, _ := git.ListRemotes()
		hasHTTPS := false
		for _, remote := range remotes {
			url, err := git.GetRemoteURL(remote)
			if err == nil && strings.HasPrefix(url, "https://") {
				hasHTTPS = true
				break
			}
		}

		if hasHTTPS {
			fmt.Println()
			ui.Warn("This repo uses HTTPS remotes")

			if ui.Confirm("Convert to SSH for passwordless push?", true) {
				_ = runFixRemote(nil)
			}
		}
	}

	return nil
}

func quickRegister(name, email string, store *config.Store) error {
	ui.Banner("QUICK SETUP: " + name)
	fmt.Println()

	var err error

	if email == "" {
		email, err = ui.Prompt("Email address:")
		if err != nil {
			return err
		}
		if email == "" {
			ui.Error("Email is required.")
			return fmt.Errorf("missing email")
		}
	}

	if err := store.AddUser(name, email); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	fmt.Println()
	ui.Info("SSH Key Setup:")

	idx, err := ui.Select("Choose SSH key setup:", []string{
		"Auto-generate (recommended)",
		"Use existing key",
		"Skip",
	})
	if err != nil {
		idx = 0 // Default to auto-generate
	}

	var sshKeyPath string

	switch idx {
	case 0: // Auto-generate
		path, err := generateAndDisplayKey(name, email)
		if err != nil {
			ui.Warn("Key generation failed")
			break
		}
		sshKeyPath = path

	case 1: // Use existing key
		keyPath, err := ui.Prompt("Path to SSH key:")
		if err == nil && keyPath != "" {
			expanded := expandPath(keyPath)
			if _, err := os.Stat(expanded); err == nil {
				sshKeyPath = expanded
				ui.Success("Using existing key")
			} else {
				ui.Warn("Key not found")
			}
		}

	case 2: // Skip
		ui.Info("Skipping SSH setup")
	}

	if sshKeyPath != "" {
		if err := store.BindSSHKey(name, sshKeyPath); err != nil {
			ui.Warn("Could not bind SSH key")
		}
	}

	if err := config.Save(store); err != nil {
		return err
	}

	fmt.Println()
	ui.Success(fmt.Sprintf("✓ Identity created: %s (%s)", name, email))
	if sshKeyPath != "" {
		ui.Success(fmt.Sprintf("✓ SSH key: %s", sshKeyPath))
	}
	fmt.Println()

	return nil
}
