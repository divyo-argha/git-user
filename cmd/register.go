package cmd

import (
	"fmt"
	"os"
	"regexp"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

func runRegister(args []string) error {
	var name, email, passphrase string
	var isTemp bool
	var err error

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name", "-n":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "--email", "-e":
			if i+1 < len(args) {
				email = args[i+1]
				i++
			}
		case "--passphrase", "-p":
			if i+1 < len(args) {
				passphrase = args[i+1]
				i++
			}
		case "--temp", "-t":
			isTemp = true
		}
	}

	ui.Banner("CREATE NEW IDENTITY")
	fmt.Println()

	if name == "" {
		name, err = ui.Prompt("Identity name (e.g., 'work', 'personal'):")
		if err != nil {
			return err
		}
		if name == "" {
			ui.Error("Name is required.")
			return fmt.Errorf("missing name")
		}
	}

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

	for !isValidEmail(email) {
		ui.Warn("Invalid email format")
		email, err = ui.Prompt("Enter a valid email address:")
		if err != nil {
			return err
		}
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if store.IsNameTaken(name) {
		ui.Errorf("identity %q already exists", name)
		return fmt.Errorf("user exists")
	}

	if store.IsEmailTaken(email) {
		ui.Errorf("Email already in use — each identity must have a unique email to prevent impersonation.")
		return fmt.Errorf("email exists")
	}

	if err := store.AddUser(name, email); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if isTemp {
		u := store.FindUser(name)
		if u != nil {
			u.IsTemporary = true
		}
	}

	fmt.Println()
	ui.Banner("SSH KEY SETUP")
	fmt.Println()
	
	idx, err := ui.Select("Choose how to set up your SSH key:", []string{
		"Generate new key automatically (recommended)",
		"Use existing key (provide path)",
		"Skip for now (set up later)",
	})
	if err != nil {
		idx = 0 // Default to generate
	}

	var sshKeyPath string

	switch idx {
	case 0: // Generate new key
		sshKeyPath, err = generateAndDisplayKey(name, email, passphrase)
		if err != nil {
			ui.Warn("Key generation failed. You can set up SSH later with: git-user bind")
		}

	case 1: // Use existing key
		keyPath, err := ui.Prompt("Enter path to your SSH private key:")
		if err == nil && keyPath != "" {
			expandedPath := expandPath(keyPath)
			if _, err := os.Stat(expandedPath); err == nil {
				sshKeyPath = expandedPath
				ui.Success(fmt.Sprintf("Using existing key: %s", sshKeyPath))
			} else {
				ui.Warn(fmt.Sprintf("Key file not found: %s", keyPath))
				ui.Info("You can bind a key later with: git-user bind " + name + " --ssh-key <path>")
			}
		}

	case 2: // Skip
		ui.Info("Skipping SSH key setup")
		ui.Info("You can set up SSH later with: git-user bind " + name + " --ssh-key <path>")

	default:
		ui.Warn("Invalid choice, skipping SSH setup")
		ui.Info("You can set up SSH later with: git-user bind " + name + " --ssh-key <path>")
	}

	if sshKeyPath != "" {
		if err := store.BindSSHKey(name, sshKeyPath); err != nil {
			ui.Errorf("binding SSH key: %v", err)
		}
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	fmt.Println()
	ui.Divider()
	ui.Success(fmt.Sprintf("Identity created: %s (%s)", name, email))
	if sshKeyPath != "" {
		ui.Success(fmt.Sprintf("SSH key configured: %s", sshKeyPath))
	}
	fmt.Println()
	ui.Info(fmt.Sprintf("Activate with: git-user switch %s", name))
	ui.Divider()

	return nil
}



