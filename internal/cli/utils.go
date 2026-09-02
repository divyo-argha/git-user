package cli

import (
	"errors"
	"fmt"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/ui"
	"github.com/divyo-argha/git-user/internal/validate"
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

// promptSSHKeyFilename asks for the filename of a new key to generate inside
// the managed SSH directory (~/.ssh), pre-filled with a suggestion that
// doesn't collide with anything already there, and re-prompts until the
// chosen name is both valid and free. Callers never need to fall back to
// silently reusing whatever file happens to already exist at the default
// git_<identityName> path — a leftover from a renamed or deleted identity
// used to get silently attached to any new identity that reused its name.
func promptSSHKeyFilename(identityName string) (string, error) {
	suggestion, err := config.SuggestSSHKeyFilename(identityName)
	if err != nil {
		return "", err
	}
	for {
		input, err := ui.Prompt(fmt.Sprintf("Key filename [%s]:", suggestion))
		if err != nil {
			if errors.Is(err, ui.ErrNotInteractive) {
				// No terminal to prompt on (scripted use, `switch -c`
				// quick-create, CI) — fall back to the collision-free
				// suggestion instead of blocking on input that can't arrive.
				return config.SSHKeyPathForFilename(suggestion)
			}
			return "", err
		}
		filename := strings.TrimSpace(input)
		if filename == "" {
			filename = suggestion
		}
		if err := validate.SSHKeyFilename(filename); err != nil {
			ui.Errorf("%v", err)
			continue
		}
		keyPath, err := config.SSHKeyPathForFilename(filename)
		if err != nil {
			ui.Errorf("%v", err)
			continue
		}
		if _, err := os.Stat(keyPath); err == nil {
			ui.Warn(fmt.Sprintf("A key already exists at %s — choose a different name", keyPath))
			continue
		}
		return keyPath, nil
	}
}

// promptExistingSSHKey offers an arrow-key list of the private key files
// found in the managed SSH directory, falling back to manual path entry when
// none are found (or the user asks to type one) — so binding an existing key
// doesn't require remembering or typing an exact path for a key that's
// already sitting in ~/.ssh.
func promptExistingSSHKey() (string, error) {
	keys, listErr := config.ListSSHKeyFiles()
	if listErr != nil {
		ui.Warn(fmt.Sprintf("Could not list keys in ~/.ssh: %v", listErr))
	}
	if len(keys) > 0 {
		labels := make([]string, 0, len(keys)+1)
		for _, k := range keys {
			label := k.Name
			if k.Comment != "" {
				label = fmt.Sprintf("%s  (%s)", k.Name, k.Comment)
			}
			labels = append(labels, label)
		}
		labels = append(labels, "Enter a path manually…")
		idx, err := ui.Select("Choose an existing SSH key:", labels)
		if err != nil {
			return "", err
		}
		if idx >= 0 && idx < len(keys) {
			return keys[idx].Path, nil
		}
		// "Enter a path manually…" was chosen — fall through below.
	}

	keyPath, err := ui.Prompt("Enter path to your SSH private key:")
	if err != nil {
		return "", err
	}
	keyPath = strings.TrimSpace(keyPath)
	if keyPath == "" {
		return "", fmt.Errorf("no path provided")
	}
	if err := validate.SSHKeyPath(keyPath, true); err != nil {
		return "", err
	}
	return validate.ExpandPath(keyPath), nil
}

// generateAndDisplayKey creates an ed25519 key at a filename the user picks
// (see promptSSHKeyFilename), prints the public key, waits for the user to
// add it, then verifies the connection. Returns the key path on success. The
// passphrase is always collected via an interactive prompt (never accepted
// as a CLI argument) so it never lands on argv, in /proc/<pid>/cmdline, or in
// shell history.
func generateAndDisplayKey(name, email string) (string, error) {
	keyPath, err := promptSSHKeyFilename(name)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return "", fmt.Errorf("creating .ssh directory: %w", err)
	}

	// promptSSHKeyFilename already re-prompts until the chosen filename is
	// free, so finding it occupied here would mean it appeared in the
	// meantime — refuse rather than silently reusing it.
	if _, err := os.Stat(keyPath); err == nil {
		return "", fmt.Errorf("a key already exists at %s", keyPath)
	}

	ui.Info(fmt.Sprintf("Generating SSH key at %s...", keyPath))
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath, "-N", "")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w", err)
	}
	ui.Success("SSH key generated!")

	ui.Info("You will be prompted to set a passphrase for the key. Press Enter to skip.")
	newPass, err := readPassphrase(PassphrasePrompt)
	if err != nil {
		ui.Warn("Skipping passphrase setup.")
	} else if newPass != "" {
		confirm, err := readPassphrase(ConfirmPassphrasePrompt)
		if err != nil || newPass != confirm {
			ui.Error("Passphrases do not match. Skipping passphrase setup.")
		} else if err := ssh.ChangeKeyPassphrase(keyPath, "", newPass); err != nil {
			ui.Errorf("Could not add passphrase: %v", err)
		} else {
			ui.Success("Passphrase applied securely!")
			promptAndStoreKeychain(name, keyPath, newPass)
		}
	}

	pubKeyBytes, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		out, kerr := exec.Command("ssh-keygen", "-y", "-f", keyPath).Output()
		if kerr == nil && len(out) > 0 {
			pubKeyBytes = out
			_ = os.WriteFile(keyPath+".pub", out, 0644)
		} else {
			return keyPath, nil
		}
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
				if err := ssh.ChangeKeyPassphrase(keyPath, "", newPassphrase); err != nil {
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
