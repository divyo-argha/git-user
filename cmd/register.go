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
	var name, email string

	// Interactive entry
	var err error
	name, err = ui.Prompt("Enter name for this identity (e.g., 'work'):")
	if err != nil {
		return err
	}
	if name == "" {
		ui.Error("Name is required.")
		return fmt.Errorf("missing name")
	}

	email, err = ui.Prompt("Enter email address:")
	if err != nil {
		return err
	}
	if email == "" {
		ui.Error("Email is required.")
		return fmt.Errorf("missing email")
	}

	// Validate email format
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

	// Check if user already exists
	if store.FindUser(name) != nil {
		ui.Errorf("identity %q already exists", name)
		return fmt.Errorf("user exists")
	}

	// Add user first
	if err := store.AddUser(name, email); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	// Ask about SSH key generation
	generateKey, err := ui.Prompt("Generate a new SSH key for this identity? [Y/n]:")
	if err != nil {
		return err
	}

	var sshKeyPath string
	var platform string = "github" // default
	
	if generateKey == "" || strings.ToLower(generateKey) == "y" || strings.ToLower(generateKey) == "yes" {
		// Ask which platform
		platformIdx, err := ui.Select("Which platform will you use?", []string{"GitHub", "GitLab", "Bitbucket"})
		if err == nil {
			platforms := []string{"github", "gitlab", "bitbucket"}
			platform = platforms[platformIdx]
		}
		// Generate SSH key
		home, _ := os.UserHomeDir()
		sshDir := filepath.Join(home, ".ssh")
		keyPath := filepath.Join(sshDir, fmt.Sprintf("git_%s", name))

		// Ensure .ssh directory exists
		if err := os.MkdirAll(sshDir, 0700); err != nil {
			ui.Errorf("creating .ssh directory: %v", err)
			return err
		}

		// Check if key already exists
		if _, err := os.Stat(keyPath); err == nil {
			ui.Warn(fmt.Sprintf("Key already exists at %s", keyPath))
			useExisting, _ := ui.Prompt("Use existing key? [Y/n]:")
			if useExisting == "" || strings.ToLower(useExisting) == "y" || strings.ToLower(useExisting) == "yes" {
				sshKeyPath = keyPath
			} else {
				ui.Info("Skipping SSH key setup. You can bind a key later with: git user bind")
			}
		} else {
			// Generate the key
			ui.Info(fmt.Sprintf("Generating SSH key at %s...", keyPath))
			cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath, "-N", "")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				ui.Errorf("generating SSH key: %v", err)
				ui.Warn("You can generate a key manually and bind it later with: git user bind")
			} else {
				ui.Success(fmt.Sprintf("SSH key created at %s", keyPath))
				sshKeyPath = keyPath

				// Display the public key and fingerprint
				pubKeyPath := keyPath + ".pub"
				pubKeyBytes, err := os.ReadFile(pubKeyPath)
				if err == nil {
					// Get fingerprint
					cmd := exec.Command("ssh-keygen", "-l", "-f", pubKeyPath)
					fingerprintOutput, _ := cmd.Output()
					
					ui.Divider()
					ui.Banner("ADD THIS PUBLIC KEY TO YOUR GIT PLATFORM")
					fmt.Println()
					fmt.Println(string(pubKeyBytes))
					fmt.Println()
					if len(fingerprintOutput) > 0 {
						ui.Info(fmt.Sprintf("Fingerprint: %s", strings.TrimSpace(string(fingerprintOutput))))
					}
					ui.Divider()
					ui.Info("GitHub: Settings → SSH and GPG keys → New SSH key")
					ui.Info("GitLab: Preferences → SSH Keys → Add new key")
					ui.Info("Bitbucket: Personal settings → SSH keys → Add key")
					fmt.Println()

					// Wait for user confirmation
					_, _ = ui.Prompt("Press Enter once you've added the key to your platform...")

					// Verify SSH connection with selected platform
					if err := verifySSHConnectionPlatform(platform); err != nil {
						ui.Warn("SSH verification failed. You may need to add the key to your platform.")
						ui.Info(fmt.Sprintf("You can test manually with: ssh -T git@%s", map[string]string{"github": "github.com", "gitlab": "gitlab.com", "bitbucket": "bitbucket.org"}[platform]))
					} else {
						ui.Success("SSH connection verified!")
					}
				}
			}
		}
	} else {
		// Ask if they want to bind an existing key
		bindExisting, _ := ui.Prompt("Bind an existing SSH key? [y/N]:")
		if strings.ToLower(bindExisting) == "y" || strings.ToLower(bindExisting) == "yes" {
			keyPath, err := ui.Prompt("Enter path to SSH private key:")
			if err == nil && keyPath != "" {
				if _, err := os.Stat(keyPath); err == nil {
					sshKeyPath = keyPath
				} else {
					ui.Warn(fmt.Sprintf("Key file not found: %s", keyPath))
				}
			}
		}
	}

	// Bind the SSH key if we have one
	if sshKeyPath != "" {
		if err := store.BindSSHKey(name, sshKeyPath); err != nil {
			ui.Errorf("binding SSH key: %v", err)
		} else {
			ui.Success(fmt.Sprintf("SSH key bound: %s", sshKeyPath))
		}
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Identity registered: %s (%s)", name, email))
	if sshKeyPath == "" {
		ui.Info("You can bind an SSH key later with: git user bind " + name + " --ssh-key <path>")
	}
	ui.Info(fmt.Sprintf("Run 'git user switch %s' to activate", name))
	return nil
}
