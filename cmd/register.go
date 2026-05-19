package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

func runRegister(args []string) error {
	ui.Banner("CREATE NEW IDENTITY")
	fmt.Println()

	var name, email string
	var err error

	name, err = ui.Prompt("Identity name (e.g., 'work', 'personal'):")
	if err != nil {
		return err
	}
	if name == "" {
		ui.Error("Name is required.")
		return fmt.Errorf("missing name")
	}

	email, err = ui.Prompt("Email address:")
	if err != nil {
		return err
	}
	if email == "" {
		ui.Error("Email is required.")
		return fmt.Errorf("missing email")
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

	if store.FindUser(name) != nil {
		ui.Errorf("identity %q already exists", name)
		return fmt.Errorf("user exists")
	}

	if err := store.AddUser(name, email); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	fmt.Println()
	ui.Banner("SSH KEY SETUP")
	fmt.Println()
	ui.Info("Choose how to set up your SSH key:")
	fmt.Println()
	fmt.Println("  1. Generate new key automatically (recommended)")
	fmt.Println("  2. Use existing key (provide path)")
	fmt.Println("  3. Skip for now (set up later)")
	fmt.Println()

	choice, err := ui.Prompt("Enter choice [1/2/3]:")
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
		sshKeyPath, err = generateSSHKey(name, email)
		if err != nil {
			ui.Warn("Key generation failed. You can set up SSH later with: git-user bind")
		}

	case "2":
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

	case "3":
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
	ui.Success(fmt.Sprintf("✓ Identity created: %s (%s)", name, email))
	if sshKeyPath != "" {
		ui.Success(fmt.Sprintf("✓ SSH key configured: %s", sshKeyPath))
	}
	fmt.Println()
	ui.Info(fmt.Sprintf("Activate with: git-user switch %s", name))
	ui.Divider()

	return nil
}

func generateSSHKey(name, email string) (string, error) {
	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	keyPath := filepath.Join(sshDir, fmt.Sprintf("git_%s", name))

	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return "", fmt.Errorf("creating .ssh directory: %w", err)
	}

	if _, err := os.Stat(keyPath); err == nil {
		ui.Warn(fmt.Sprintf("Key already exists at %s", keyPath))
		useExisting, _ := ui.Prompt("Use existing key? [Y/n]:")
		if useExisting == "" || strings.ToLower(useExisting) == "y" || strings.ToLower(useExisting) == "yes" {
			return keyPath, nil
		}
		return "", fmt.Errorf("key already exists")
	}

	ui.Info(fmt.Sprintf("Generating SSH key at %s...", keyPath))
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath, "-N", "")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w", err)
	}

	ui.Success("SSH key generated!")

	pubKeyPath := keyPath + ".pub"
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return keyPath, nil
	}

	cmd = exec.Command("ssh-keygen", "-l", "-f", pubKeyPath)
	fingerprintOutput, _ := cmd.Output()

	fmt.Println()
	ui.Divider()
	ui.Banner("📋 YOUR PUBLIC KEY")
	fmt.Println()
	fmt.Println(string(pubKeyBytes))
	if len(fingerprintOutput) > 0 {
		ui.Info(fmt.Sprintf("Fingerprint: %s", strings.TrimSpace(string(fingerprintOutput))))
	}
	ui.Divider()
	fmt.Println()
	ui.Info("Copy the key above and add it to your Git platform:")
	fmt.Println()
	fmt.Println("  GitHub:    Settings → SSH and GPG keys → New SSH key")
	fmt.Println("  GitLab:    Preferences → SSH Keys → Add new key")
	fmt.Println("  Bitbucket: Personal settings → SSH keys → Add key")
	fmt.Println()
	ui.Divider()
	fmt.Println()

	_, _ = ui.Prompt("Press Enter once you've added the key...")

	fmt.Println()
	ui.Info("Testing SSH connection...")
	if err := verifySSHConnection(); err != nil {
		ui.Warn("SSH verification failed")
		ui.Info("The key may not be added yet, or it needs a few seconds to propagate")
		ui.Info("Test manually with: ssh -T git@github.com")
	} else {
		ui.Success("✓ SSH connection verified!")
	}

	return keyPath, nil
}
