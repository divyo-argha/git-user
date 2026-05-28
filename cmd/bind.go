package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runBind(args []string) error {
	var name, sshKeyPath string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ssh-key":
			if i+1 < len(args) {
				sshKeyPath = args[i+1]
				i++
			}
		default:
			name = args[i]
		}
	}

	if name == "" {
		ui.Error("usage: git-user bind <name> [--ssh-key <path>]")
		return fmt.Errorf("missing name")
	}

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

	if sshKeyPath == "" {
		return interactiveSSHSetup(name, user.Email, store)
	}

	expanded := expandPath(sshKeyPath)
	if _, err := os.Stat(expanded); os.IsNotExist(err) {
		ui.Errorf("SSH key file %q does not exist", sshKeyPath)
		return err
	}

	if err := store.BindSSHKey(name, expanded); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Associated SSH key %q with user %q", expanded, name))

	if store.Current == name {
		ui.Info("This is your active identity. Updating git config...")
		if err := git.ConfigureSSH(expanded); err != nil {
			ui.Warn(fmt.Sprintf("Failed to update git SSH config: %v", err))
		} else {
			ui.Success("Git SSH config updated")
		}
	}

	return nil
}

func interactiveSSHSetup(name, email string, store *config.Store) error {
	ui.Banner("SSH KEY SETUP: " + name)
	fmt.Println()

	idx, err := ui.Select("Choose SSH key setup:", []string{
		"Auto-generate (recommended)",
		"Use existing key",
		"Cancel",
	})
	if err != nil || idx == 2 {
		ui.Info("Cancelled")
		return nil
	}

	var sshKeyPath string

	switch idx {
	case 0: // Auto-generate
		home, _ := os.UserHomeDir()
		sshDir := filepath.Join(home, ".ssh")
		keyPath := filepath.Join(sshDir, fmt.Sprintf("git_%s", name))

		if err := os.MkdirAll(sshDir, 0700); err != nil {
			ui.Error("Could not create .ssh directory")
			return err
		}

		if _, err := os.Stat(keyPath); err == nil {
			ui.Info(fmt.Sprintf("Using existing key: %s", keyPath))
			sshKeyPath = keyPath
		} else {
			ui.Info("Generating SSH key...")
			ui.Info("You will be prompted to set a passphrase for the key.")
			cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				ui.Error("Key generation failed")
				return err
			}

			ui.Success("Key generated!")
			sshKeyPath = keyPath

			pubKeyBytes, err := os.ReadFile(keyPath + ".pub")
			if err == nil {
				fmt.Println()
				ui.Divider()
				ui.Banner("📋 PUBLIC KEY")
				fmt.Println()
				fmt.Println(string(pubKeyBytes))
				ui.Divider()
				fmt.Println()
				ui.Info("Add this key to GitHub/GitLab/Bitbucket")
				fmt.Println()
				_, _ = ui.Prompt("Press Enter when done...")
			}
		}

	case 1: // Use existing key
		keyPath, err := ui.Prompt("Path to SSH key:")
		if err != nil {
			return err
		}
		if keyPath == "" {
			ui.Error("No path provided")
			return fmt.Errorf("no path")
		}
		expanded := expandPath(keyPath)
		if _, err := os.Stat(expanded); err != nil {
			ui.Error("Key not found")
			return err
		}
		sshKeyPath = expanded
		ui.Success("Using existing key")
	}

	if sshKeyPath == "" {
		ui.Error("No SSH key configured")
		return fmt.Errorf("no key")
	}

	if err := store.BindSSHKey(name, sshKeyPath); err != nil {
		ui.Errorf("binding SSH key: %v", err)
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	fmt.Println()
	ui.Success(fmt.Sprintf("✓ SSH key configured for %q", name))
	ui.Success(fmt.Sprintf("✓ Key: %s", sshKeyPath))
	fmt.Println()

	ui.Info("Testing SSH connection...")
	if err := verifySSHConnectionWithKey(sshKeyPath); err != nil {
		ui.Warn("SSH verification failed")
		ui.Info("The key may not be added to your platform yet")
		ui.Info(fmt.Sprintf("Test manually: ssh -i %s -o IdentitiesOnly=yes -T git@github.com", sshKeyPath))
	} else {
		ui.Success("SSH connection verified!")
	}

	return nil
}
