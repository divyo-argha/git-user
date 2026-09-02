package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/identity"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
)

// ── Bind / Rekey / Passphrase ─────────────────────────────────────────────────

// opGenerateKey creates an ed25519 key non-interactively. keychainWarning is
// non-empty if a passphrase was set but couldn't be stored in the system
// keychain — the identity would otherwise default to "persistent" passphrase
// mode (config.User.GetPassphraseMode's zero-value default) while nothing was
// actually persisted, silently falling back to prompting every time instead
// of the auto-unlock the user asked for.
func opGenerateKey(name, email, passphrase string) (keyPath string, keychainWarning string, err error) {
	keyPath, err = config.DefaultSSHKeyPath(name)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return "", "", fmt.Errorf("creating .ssh directory: %w", err)
	}
	if _, statErr := os.Stat(keyPath); statErr == nil {
		return keyPath, "", nil
	}
	if err := ssh.GenerateKey(keyPath, email, passphrase); err != nil {
		return "", "", fmt.Errorf("ssh-keygen failed: %w", err)
	}
	if passphrase != "" {
		if err := keyring.SetKeychainPassphrase(name, passphrase); err != nil {
			keychainWarning = fmt.Sprintf("Could not store the passphrase in the system keychain (%v) — you'll be asked for it again instead of it auto-unlocking.", err)
		}
	}
	return keyPath, keychainWarning, nil
}

// opRegisterFinish completes profile creation: adds the user, binds the key if
// generated, and configures commit signing.
func opRegisterFinish(store *config.Store, name, email string, isTemp bool, keyPath, passphrase string, signEnabled bool) (opResult, error) {
	// No active identity at all yet: this registration becomes the active one
	// automatically below, instead of creating a fully-configured identity
	// that git never actually uses until a separate switch is remembered.
	activateOnCreate := store.Current == ""

	if err := store.AddUser(name, email); err != nil {
		return opResult{}, err
	}
	if isTemp {
		if u := store.FindUser(name); u != nil {
			u.IsTemporary = true
		}
	}
	if keyPath != "" {
		if err := store.BindSSHKey(name, keyPath); err != nil {
			return opResult{}, fmt.Errorf("binding SSH key: %w", err)
		}
		if signEnabled {
			if err := store.SetSigningKey(name, keyPath, "ssh"); err != nil {
				return opResult{}, fmt.Errorf("failed to enable SSH commit signing: %w", err)
			}
		} else {
			store.ToggleSigning(name, true)
		}
	}

	activated := false
	var activateWarnings []string
	if activateOnCreate {
		if user := store.FindUser(name); user != nil {
			if err := git.Apply(user.Name, user.Email); err != nil {
				activateWarnings = append(activateWarnings, fmt.Sprintf("Could not apply git identity: %v", err))
			} else {
				store.Current = name
				activated = true
				if keyPath != "" {
					if err := git.ConfigureSSH(keyPath); err != nil {
						activateWarnings = append(activateWarnings, fmt.Sprintf("Could not apply SSH config: %v", err))
					}
				}
				if signEnabled {
					if err := git.ConfigureSigning(keyPath, "ssh"); err != nil {
						activateWarnings = append(activateWarnings, fmt.Sprintf("Could not apply commit signing config: %v", err))
					}
				}
			}
		}
	}

	if err := config.Save(store); err != nil {
		return opResult{}, err
	}

	report := fmt.Sprintf("Identity created: %s (%s)\n", name, email)
	if keyPath != "" {
		report += fmt.Sprintf("SSH key: %s\n", keyPath)
		if pub, err := os.ReadFile(keyPath + ".pub"); err == nil {
			report += "\nPublic key:\n" + strings.TrimSpace(string(pub)) + "\n"
		}
	} else {
		report += "No SSH key set — bind one later from the profile detail view.\n"
	}
	if activated {
		report += "\nThis is your first identity, so it's now active.\n"
	} else {
		report += "\nActivate it from the dashboard or the profile detail view.\n"
	}
	for _, w := range activateWarnings {
		report += "⚠ " + w + "\n"
	}
	return opResult{detail: report, showReport: true}, nil
}

// opBind associates an existing SSH key with an identity.
func opBind(store *config.Store, name, keyPath string, signEnabled bool) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	expanded := expandPath(keyPath)
	if _, err := os.Stat(expanded); err != nil {
		return opResult{}, fmt.Errorf("SSH key file %q does not exist", keyPath)
	}
	if err := store.BindSSHKey(name, expanded); err != nil {
		return opResult{}, err
	}
	if signEnabled {
		if err := store.SetSigningKey(name, expanded, "ssh"); err != nil {
			return opResult{}, fmt.Errorf("failed to enable SSH commit signing: %w", err)
		}
	} else {
		store.ToggleSigning(name, true)
	}
	if err := config.Save(store); err != nil {
		return opResult{}, err
	}
	report := fmt.Sprintf("SSH key configured for %q\nKey: %s\n", name, expanded)
	if signEnabled {
		report += "Commit Signing: Enabled\n"
	} else {
		report += "Commit Signing: Disabled\n"
	}
	if store.Current == name {
		if err := git.ConfigureSSH(expanded); err != nil {
			report += fmt.Sprintf("⚠ updating git SSH config: %v\n", err)
		}
		if signEnabled {
			if err := git.ConfigureSigning(expanded, "ssh"); err != nil {
				report += fmt.Sprintf("⚠ updating git signing config: %v\n", err)
			}
		} else {
			git.RemoveSigningConfig()
		}
		report += "Git config updated for the active identity.\n"
	}
	return opResult{detail: report, showReport: true}, nil
}

// opUnbind removes an identity's SSH key binding.
func opUnbind(store *config.Store, name string) error {
	u := store.FindUser(name)
	if u == nil {
		return fmt.Errorf("identity %q not found", name)
	}
	u.SSHKey = ""
	if err := config.Save(store); err != nil {
		return err
	}
	if store.Current == name {
		git.RemoveSSHConfig()
	}
	return nil
}

// opAttachKey resolves the SSH key choice from the in-TUI setup chain
// (generate / existing / skip) and either completes profile creation
// (register, register-temp) or binds the key to an existing identity (bind).
func opAttachKey(store *config.Store, name, email, mode, choice, passphrase, keyPath string, signEnabled bool) (opResult, error) {
	keychainWarning := ""
	switch choice {
	case "generate":
		kp, warning, err := opGenerateKey(name, email, passphrase)
		if err != nil {
			return opResult{}, err
		}
		keyPath = kp
		keychainWarning = warning
	case "existing":
		expanded := expandPath(keyPath)
		if _, err := os.Stat(expanded); err != nil {
			return opResult{}, fmt.Errorf("key file not found: %s", keyPath)
		}
		keyPath = expanded
	case "skip", "":
		keyPath = ""
	}

	var res opResult
	var err error
	if mode == "bind" {
		if keyPath == "" {
			return opResult{}, fmt.Errorf("no SSH key configured")
		}
		res, err = opBind(store, name, keyPath, signEnabled)
	} else {
		isTemp := mode == "register-temp"
		res, err = opRegisterFinish(store, name, email, isTemp, keyPath, passphrase, signEnabled)
		if err == nil && isTemp && keyPath != "" {
			// Best-effort: a failure here doesn't lose the key, it only means
			// the crash-safety orphan scan won't catch it if this process dies
			// before the normal switch/logout cleanup path runs.
			_ = identity.RecordTempKey(name, keyPath)
		}
	}
	if err != nil {
		return res, err
	}

	if keychainWarning != "" {
		// The keychain write failed, so "persistent" mode (config.User's
		// zero-value default) would silently never have auto-unlocked
		// anything — record the real, working fallback instead of leaving
		// the identity claiming a mode that was never actually set up.
		if u := store.FindUser(name); u != nil {
			u.PassphraseMode = "everytime"
			_ = config.Save(store)
		}
		res.detail += "⚠ " + keychainWarning + "\n"
		res.showReport = true
	}
	return res, nil
}

// opRekey rotates an identity's SSH key (non-interactive, forced).
func opRekey(store *config.Store, name, passphrase string) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	keyPath, err := config.DefaultSSHKeyPath(name)
	if err != nil {
		return opResult{}, err
	}
	sshDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return opResult{}, fmt.Errorf("creating .ssh directory: %w", err)
	}

	backupPath := keyPath + ".backup"
	hasOldKey := false
	if _, err := os.Stat(keyPath); err == nil {
		hasOldKey = true
		// Unload the old key from the agent so the rotated fingerprint doesn't linger.
		if ssh.IsSSHKeyLoaded(keyPath) {
			_ = ssh.RemoveSSHKey(keyPath)
		}
		if err := os.Rename(keyPath, backupPath); err != nil {
			return opResult{}, fmt.Errorf("backing up key: %w", err)
		}
		if _, err := os.Stat(keyPath + ".pub"); err == nil {
			_ = os.Rename(keyPath+".pub", backupPath+".pub")
		}
	}

	if err := ssh.GenerateKey(keyPath, user.Email, passphrase); err != nil {
		if hasOldKey {
			_ = os.Rename(backupPath, keyPath)
			_ = os.Rename(backupPath+".pub", keyPath+".pub")
		}
		return opResult{}, fmt.Errorf("generating SSH key: %w", err)
	}
	if passphrase != "" {
		_ = keyring.SetKeychainPassphrase(name, passphrase)
	} else {
		_ = keyring.DeleteKeychainPassphrase(name)
	}

	if err := store.BindSSHKey(name, keyPath); err != nil {
		return opResult{}, err
	}
	if err := config.Save(store); err != nil {
		return opResult{}, err
	}

	// If this is the active identity and the new key is passphrase-protected,
	// load it into the agent now. Otherwise the next git operation against
	// this identity would need to unlock it with no TUI-owned terminal free
	// to answer an ssh passphrase prompt — mirrors the passphrase gate in
	// `switch`. Unprotected keys need no agent entry.
	agentNote := ""
	if store.Current == name && passphrase != "" {
		// EnsureSSHAgent prints its own guidance on failure, but that's a raw
		// stdout write invisible under the TUI's alt-screen — without this
		// explicit note the report would say nothing at all about the new
		// key never having been loaded anywhere.
		if agentErr := ssh.EnsureSSHAgent(); agentErr != nil {
			agentNote = "⚠ New key was NOT loaded into any ssh-agent (no agent reachable) — the next push/pull may hang or fail asking for a passphrase.\n\n"
		} else if err := ssh.AddSSHKeyWithPassphrase(keyPath, passphrase); err != nil {
			agentNote = fmt.Sprintf("⚠ Could not load new key into ssh-agent: %v\n\n", err)
		} else {
			agentNote = "New key unlocked and loaded into ssh-agent.\n\n"
		}
	}

	report := fmt.Sprintf("SSH key rotated successfully for %s\nOld key backed up with .backup extension\n\n", name) + agentNote
	if pub, err := os.ReadFile(keyPath + ".pub"); err == nil {
		report += "REPLACE YOUR OLD KEY WITH THIS NEW PUBLIC KEY\n"
		report += strings.TrimSpace(string(pub)) + "\n\n"
	}
	report += "• GitHub: Settings → SSH and GPG keys → Delete old key → Add new key\n"
	report += "• GitLab: Preferences → SSH Keys → Remove old key → Add new key\n"
	report += "• Bitbucket: Personal settings → SSH keys → Delete old → Add new\n"
	return opResult{detail: report, showReport: true}, nil
}

// opPassphraseSet sets or changes an identity's key passphrase.
func opPassphraseSet(store *config.Store, name, oldPass, newPass string) error {
	user := store.FindUser(name)
	if user == nil {
		return fmt.Errorf("identity %q not found", name)
	}
	if newPass == "" {
		return fmt.Errorf("passphrase must not be empty")
	}
	if err := ssh.ChangeKeyPassphrase(user.SSHKey, oldPass, newPass); err != nil {
		return err
	}
	if user.GetPassphraseMode() == "persistent" {
		_ = keyring.SetKeychainPassphrase(name, newPass)
	}
	return nil
}

// opPassphraseVerify verifies a key passphrase.
func opPassphraseVerify(store *config.Store, name, pass string) error {
	user := store.FindUser(name)
	if user == nil {
		return fmt.Errorf("identity %q not found", name)
	}
	if !ssh.VerifyPassphrase(user.SSHKey, pass) {
		return fmt.Errorf("incorrect passphrase")
	}
	return nil
}

// opPassphraseRemove removes a key passphrase (only valid for the active identity).
func opPassphraseRemove(store *config.Store, name, currentPass string) error {
	user := store.FindUser(name)
	if user == nil {
		return fmt.Errorf("identity %q not found", name)
	}
	if store.Current != name {
		return fmt.Errorf("must switch to profile %q to remove its passphrase", name)
	}
	if err := ssh.ChangeKeyPassphrase(user.SSHKey, currentPass, ""); err != nil {
		return err
	}
	_ = keyring.DeleteKeychainPassphrase(name)
	return nil
}
