package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	
	if user.SSHKey != "" {
		if err := verifySSHConnection(); err != nil {
			ui.Warn("SSH verification failed. The key may not be added to your platform yet.")
			ui.Info("Test manually with: ssh -T git@github.com")
		} else {
			ui.Success("SSH verified: Connection successful!")
		}
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
			ui.Info("Convert to SSH for passwordless push? [Y/n]")
			
			response, err := ui.Prompt("")
			if err == nil {
				response = strings.ToLower(strings.TrimSpace(response))
				if response == "" || response == "y" || response == "yes" {
					_ = runFixRemote(nil)
				}
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
	fmt.Println("  1. Auto-generate (recommended)")
	fmt.Println("  2. Use existing key")
	fmt.Println("  3. Skip")
	fmt.Println()

	choice, err := ui.Prompt("Choice [1/2/3]:")
	if err != nil {
		choice = "1"
	}
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "1"
	}

	var sshKeyPath string

	switch choice {
	case "1":
		home, _ := os.UserHomeDir()
		sshDir := filepath.Join(home, ".ssh")
		keyPath := filepath.Join(sshDir, fmt.Sprintf("git_%s", name))

		if err := os.MkdirAll(sshDir, 0700); err != nil {
			ui.Warn("Could not create .ssh directory")
			break
		}

		if _, err := os.Stat(keyPath); err == nil {
			ui.Info(fmt.Sprintf("Using existing key: %s", keyPath))
			sshKeyPath = keyPath
			break
		}

		ui.Info("Generating SSH key...")
		cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath, "-N", "")
		if err := cmd.Run(); err != nil {
			ui.Warn("Key generation failed")
			break
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

	case "2":
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

	case "3":
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
