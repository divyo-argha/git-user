package cli

import (
	"fmt"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/ui"
	"golang.org/x/term"
)

// PassphrasePrompt is the only text shown when a profile's key passphrase is
// requested — no key path, nothing else.
const (
	PassphrasePrompt        = "Enter Passphrase: "
	ConfirmPassphrasePrompt = "Confirm Passphrase: "
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
		{"git@bitbucket.org", []string{"logged in as", "successfully authenticated", "authenticated via ssh key"}},
	}

	// A passphrase-protected key that is not loaded in the agent would make
	// ssh prompt on the terminal itself ("Enter passphrase for key '<path>'")
	// and leak the key path into the UI. Unlock it up front instead, so ssh
	// finds the identity in the agent and never prompts.
	if keyPath != "" {
		if err := ensureKeyUnlocked(keyPath); err != nil {
			return err
		}
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

// ensureKeyUnlocked loads a passphrase-protected key into the SSH agent using
// the keychain passphrase of its owning identity, or a clean terminal prompt.
// Keys that are unprotected or already loaded are left untouched.
func ensureKeyUnlocked(keyPath string) error {
	protected, err := isSSHKeyPassphraseProtected(keyPath)
	if err != nil || !protected {
		return nil
	}
	if ssh.IsSSHKeyLoaded(keyPath) {
		return nil
	}

	// Prefer the keychain passphrase of the identity that owns this key.
	if name := keychainNameForKey(keyPath); name != "" {
		if secret, kerr := keyring.GetKeychainPassphrase(name); kerr == nil && secret != "" {
			if ssh.VerifyPassphrase(keyPath, secret) {
				return ssh.AddSSHKeyWithPassphrase(keyPath, secret)
			}
			_ = keyring.DeleteKeychainPassphrase(name)
		}
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("SSH key %q requires a passphrase — run in a terminal to unlock it", keyPath)
	}
	pass, err := readPassphrase(PassphrasePrompt)
	if err != nil {
		return err
	}
	if !ssh.VerifyPassphrase(keyPath, pass) {
		return fmt.Errorf("incorrect passphrase")
	}
	return ssh.AddSSHKeyWithPassphrase(keyPath, pass)
}

// keychainNameForKey returns the identity name that owns the given key, if any.
func keychainNameForKey(keyPath string) string {
	store, err := config.Load()
	if err != nil {
		return ""
	}
	for _, u := range store.Users {
		if u.SSHKey == keyPath {
			return u.Name
		}
	}
	return ""
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
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath, "-N", "")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w", err)
	}
	ui.Success("SSH key generated!")

	if passphrase != "" {
		if err := changeSSHKeyPassphrase(keyPath, "", passphrase); err != nil {
			ui.Errorf("Could not add passphrase: %v", err)
		} else {
			ui.Success("Passphrase applied securely!")
			promptAndStoreKeychain(name, keyPath, passphrase)
		}
	} else {
		ui.Info("You will be prompted to set a passphrase for the key.")
		newPass, err := readPassphrase(PassphrasePrompt)
		if err != nil || newPass == "" {
			return keyPath, nil
		}
		confirm, err := readPassphrase(ConfirmPassphrasePrompt)
		if err != nil || newPass != confirm {
			ui.Error("Passphrases do not match.")
			return keyPath, nil
		}
		if err := changeSSHKeyPassphrase(keyPath, "", newPass); err != nil {
			ui.Errorf("Could not add passphrase: %v", err)
		} else {
			ui.Success("Passphrase applied securely!")
			promptAndStoreKeychain(name, keyPath, newPass)
		}
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
		ui.Success("SSH connection verified!")
	}

	return keyPath, nil
}

var readPassphraseFn func(prompt string) (string, error)

func readPassphrase(prompt string) (string, error) {
	if readPassphraseFn != nil {
		return readPassphraseFn(prompt)
	}
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

func checkAndPromptPassphrase(name string, keyPath string) {
	protected, err := isSSHKeyPassphraseProtected(keyPath)
	if err != nil {
		return
	}
	if !protected {
		fmt.Println()
		ui.Warn("⚠️  Your SSH key is not passphrase protected.")
		if ui.Confirm("Would you like to add a passphrase to protect this identity now?", true) {
			newPassphrase, err := promptRequiredPassphrase()
			if err == nil && newPassphrase != "" {
				if err := changeSSHKeyPassphrase(keyPath, "", newPassphrase); err != nil {
					ui.Errorf("Could not add passphrase: %v", err)
				} else {
					ui.Success("Passphrase added successfully!")
					promptAndStoreKeychain(name, keyPath, newPassphrase)
				}
			}
		}
	} else {
		promptAndStoreKeychain(name, keyPath, "")
	}
}

func promptAndStoreKeychain(name, keyPath, passphrase string) {
	protected, err := isSSHKeyPassphraseProtected(keyPath)
	if err != nil || !protected {
		return
	}

	if !ui.Confirm("Would you like to store the passphrase securely in your system keychain?", true) {
		return
	}

	if passphrase == "" {
		var err error
		passphrase, err = readPassphrase(PassphrasePrompt)
		if err != nil {
			ui.Errorf("Error reading passphrase: %v", err)
			return
		}
		if !ssh.VerifyPassphrase(keyPath, passphrase) {
			ui.Error("Incorrect passphrase. Not saved to keychain.")
			return
		}
	}

	if err := keyring.SetKeychainPassphrase(name, passphrase); err != nil {
		ui.Errorf("Could not save passphrase to keychain: %v", err)
	} else {
		ui.Success("Passphrase stored securely in system keychain.")
	}
}
