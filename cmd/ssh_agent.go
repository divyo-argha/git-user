package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/divyo-argha/git-user/internal/ui"
)

func ensureSSHAgent() error {
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		return nil
	}
	// On Windows, OpenSSH agent uses a named pipe — SSH_AUTH_SOCK won't be set
	// but ssh-add may still work. Let it try rather than failing early.
	if runtime.GOOS == "windows" {
		return nil
	}
	ui.Warn("ssh-agent is not running in this shell")
	ui.Info("Start it with:")
	fmt.Println(`  eval "$(ssh-agent -s)"`)
	ui.Info("Then try again.")
	return fmt.Errorf("ssh-agent not running")
}

func isSSHKeyLoaded(keyPath string) bool {
	target, err := sshKeyFingerprint(keyPath)
	if err != nil {
		return false
	}

	loaded, err := loadedSSHKeyFingerprints()
	if err != nil {
		return false
	}

	for _, fingerprint := range loaded {
		if fingerprint == target {
			return true
		}
	}
	return false
}

func sshKeyFingerprint(keyPath string) (string, error) {
	pubKeyPath := keyPath + ".pub"
	if _, err := os.Stat(pubKeyPath); err != nil {
		return "", err
	}

	output, err := exec.Command("ssh-keygen", "-lf", pubKeyPath).Output()
	if err != nil {
		return "", err
	}
	return parseSSHKeyFingerprint(string(output))
}

func loadedSSHKeyFingerprints() ([]string, error) {
	output, err := exec.Command("ssh-add", "-l").Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	fingerprints := make([]string, 0, len(lines))
	for _, line := range lines {
		fingerprint, err := parseSSHKeyFingerprint(line)
		if err == nil {
			fingerprints = append(fingerprints, fingerprint)
		}
	}
	return fingerprints, nil
}

func parseSSHKeyFingerprint(line string) (string, error) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 {
		return "", fmt.Errorf("missing fingerprint")
	}
	return fields[1], nil
}

// addSSHKeyWithPassphrase adds the SSH key to the agent using the provided passphrase.
// It sets DISPLAY and SSH_ASKPASS so that ssh-add doesn't prompt interactively.
func addSSHKeyWithPassphrase(keyPath, passphrase string) error {
	cmd := exec.Command("ssh-add", keyPath)
	
	// Create a temporary script for SSH_ASKPASS
	askpassScript, err := os.CreateTemp("", "git-user-askpass-*")
	if err != nil {
		return fmt.Errorf("failed to create askpass script: %w", err)
	}
	defer os.Remove(askpassScript.Name())

	scriptContent := fmt.Sprintf("#!/bin/sh\necho '%s'\n", strings.ReplaceAll(passphrase, "'", "'\\''"))
	if err := os.WriteFile(askpassScript.Name(), []byte(scriptContent), 0700); err != nil {
		return fmt.Errorf("failed to write askpass script: %w", err)
	}

	env := os.Environ()
	hasDisplay := false
	for _, e := range env {
		if strings.HasPrefix(e, "DISPLAY=") {
			hasDisplay = true
			break
		}
	}

	if !hasDisplay {
		// SSH_ASKPASS requires DISPLAY to be set, even if it's a dummy value.
		env = append(env, "DISPLAY=dummy:0")
	}

	env = append(env, "SSH_ASKPASS="+askpassScript.Name())
	env = append(env, "SSH_ASKPASS_REQUIRE=force")

	cmd.Env = env
	
	// Suppress output
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-add failed: %v, output: %s", err, string(out))
	}
	return nil
}

func removeSSHKey(keyPath string) error {
	pubKeyPath := keyPath + ".pub"
	if _, err := os.Stat(pubKeyPath); err != nil {
		return fmt.Errorf("public key not found at %s", pubKeyPath)
	}
	cmd := exec.Command("ssh-add", "-d", pubKeyPath)
	return cmd.Run()
}

func verifyPassphrase(keyPath, passphrase string) bool {
	cmd := exec.Command("ssh-keygen", "-y", "-P", passphrase, "-f", keyPath)
	return cmd.Run() == nil
}
