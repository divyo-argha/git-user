package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/ui"
	"golang.org/x/term"
)


func verifySSHConnection() error {
	return verifySSHConnectionWithKey("")
}

func verifySSHConnectionWithKey(keyPath string) error {
	platforms := []struct {
		host    string
		success []string
	}{
		{"git@github.com", []string{"Hi ", "successfully authenticated"}},
		{"git@gitlab.com", []string{"Welcome to GitLab", "successfully authenticated"}},
		{"git@bitbucket.org", []string{"logged in as", "successfully authenticated"}},
	}

	for _, p := range platforms {
		args := []string{"-T", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=5"}
		if keyPath != "" {
			args = append(args, "-i", keyPath, "-o", "IdentitiesOnly=yes")
		}
		args = append(args, p.host)

		cmd := exec.Command("ssh", args...)
		output, _ := cmd.CombinedOutput()
		out := string(output)
		for _, marker := range p.success {
			if strings.Contains(out, marker) {
				return nil
			}
		}
	}

	return fmt.Errorf("connection failed on all platforms")
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// generateAndDisplayKey creates an ed25519 key at keyPath, prints the public key,
// waits for the user to add it, then verifies the connection.
// Returns the key path on success.
func generateAndDisplayKey(name, email, passphrase string) (string, error) {
	home, _ := os.UserHomeDir()
	keyPath := filepath.Join(home, ".ssh", fmt.Sprintf("git_%s", name))

	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return "", fmt.Errorf("creating .ssh directory: %w", err)
	}

	if _, err := os.Stat(keyPath); err == nil {
		ui.Warn(fmt.Sprintf("Key already exists at %s", keyPath))
		if !ui.Confirm("Use existing key?", true) {
			return "", fmt.Errorf("key already exists")
		}
		return keyPath, nil
	}

	ui.Info(fmt.Sprintf("Generating SSH key at %s...", keyPath))
	var cmd *exec.Cmd
	if passphrase != "" {
		cmd = exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath, "-N", "")
	} else {
		ui.Info("You will be prompted to set a passphrase for the key.")
		cmd = exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w", err)
	}
	ui.Success("SSH key generated!")

	if passphrase != "" {
		if err := changeSSHKeyPassphrase(keyPath, "", passphrase); err != nil {
			ui.Errorf("Could not add passphrase: %v", err)
		} else {
			ui.Success("✓ Passphrase applied securely!")
		}
	} else {
		checkAndPromptPassphrase(keyPath)
	}

	pubKeyBytes, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return keyPath, nil
	}

	fingerprintOut, _ := exec.Command("ssh-keygen", "-l", "-f", keyPath+".pub").Output()

	fmt.Println()
	ui.Divider()
	ui.Banner("📋 YOUR PUBLIC KEY")
	fmt.Println()
	fmt.Println(string(pubKeyBytes))
	if len(fingerprintOut) > 0 {
		ui.Info(fmt.Sprintf("Fingerprint: %s", strings.TrimSpace(string(fingerprintOut))))
	}
	ui.Divider()
	fmt.Println()
	ui.Info("Copy the key above and add it to your Git platform:")
	fmt.Println("  GitHub:    Settings → SSH and GPG keys → New SSH key")
	fmt.Println("  GitLab:    Preferences → SSH Keys → Add new key")
	fmt.Println("  Bitbucket: Personal settings → SSH keys → Add key")
	fmt.Println()
	ui.Divider()
	fmt.Println()

	_, _ = ui.Prompt("Press Enter once you've added the key...")

	fmt.Println()
	ui.Info("Testing SSH connection...")
	if err := verifySSHConnectionWithKey(keyPath); err != nil {
		ui.Warn("SSH verification failed")
		ui.Info("The key may not be added yet, or it needs a few seconds to propagate")
		ui.Info(fmt.Sprintf("Test manually with: ssh -i %s -o IdentitiesOnly=yes -T git@github.com", keyPath))
	} else {
		ui.Success("✓ SSH connection verified!")
	}

	return keyPath, nil
}

func readPassphrase(prompt string) (string, error) {
	fmt.Print(prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		var s string
		_, err2 := fmt.Scanln(&s)
		if err2 != nil {
			return "", fmt.Errorf("reading passphrase: %w", err)
		}
		return s, nil
	}
	return string(b), nil
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func checkAndPromptPassphrase(keyPath string) {
	protected, err := isSSHKeyPassphraseProtected(keyPath)
	if err == nil && !protected {
		fmt.Println()
		ui.Warn("⚠️  Your SSH key is not passphrase protected.")
		if ui.Confirm("Would you like to add a passphrase to protect this identity now?", true) {
			newPassphrase, err := promptRequiredPassphrase("New passphrase: ", "Confirm new passphrase: ")
			if err == nil && newPassphrase != "" {
				if err := changeSSHKeyPassphrase(keyPath, "", newPassphrase); err != nil {
					ui.Errorf("Could not add passphrase: %v", err)
				} else {
					ui.Success("✓ Passphrase added successfully!")
				}
			}
		}
	}
}