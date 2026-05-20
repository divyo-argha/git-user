package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/ui"
)

// verifySSHConnection tries GitHub, GitLab, and Bitbucket and returns nil if any succeeds.
func verifySSHConnection() error {
	platforms := []struct {
		host    string
		success []string
	}{
		{"git@github.com", []string{"Hi ", "successfully authenticated"}},
		{"git@gitlab.com", []string{"Welcome to GitLab", "successfully authenticated"}},
		{"git@bitbucket.org", []string{"logged in as", "successfully authenticated"}},
	}

	for _, p := range platforms {
		cmd := exec.Command("ssh", "-T", p.host, "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=5")
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
func generateAndDisplayKey(name, email string) (string, error) {
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
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath, "-N", "")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w", err)
	}
	ui.Success("SSH key generated!")

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
	if err := verifySSHConnection(); err != nil {
		ui.Warn("SSH verification failed")
		ui.Info("The key may not be added yet, or it needs a few seconds to propagate")
		ui.Info("Test manually with: ssh -T git@github.com")
	} else {
		ui.Success("✓ SSH connection verified!")
	}

	return keyPath, nil
}
