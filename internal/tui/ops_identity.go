package tui

import (
	"fmt"
	"os"
	"os/exec"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
)

// ── Switch / Logout / Remove ──────────────────────────────────────────────────

// opSwitch switches the active identity. Passphrase is optional; if the key is
// protected and not loaded it must be provided (or retrieved from the keychain).
func opSwitch(store *config.Store, name, passphrase string) (opResult, error) {
	if !git.IsInstalled() {
		return opResult{}, fmt.Errorf("git is not installed or not on PATH")
	}

	store.SnapshotOriginal(git.CurrentName(), git.CurrentEmail(), git.CurrentSSHCommand(), git.CurrentSigningKey(), git.CurrentSignFormat(), git.CurrentCommitGPGSign())

	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}

	var warnings []string

	// Auto-logout: unload the previous identity's key and clean up temporaries.
	if store.Current != "" && store.Current != name {
		if prev := store.CurrentUser(); prev != nil {
			if prev.SSHKey != "" && ssh.IsSSHKeyLoaded(prev.SSHKey) {
				_ = ssh.RemoveSSHKey(prev.SSHKey)
			}
			if prev.GetPassphraseMode() == "everytime" && prev.SSHKey != "" {
				_ = ssh.RemoveSSHKey(prev.SSHKey)
			}
			if prev.IsTemporary {
				store.RemoveUser(prev.Name, true)
				if prev.SSHKey != "" {
					_ = os.Remove(prev.SSHKey)
					_ = os.Remove(prev.SSHKey + ".pub")
				}
				_ = keyring.DeleteKeychainPassphrase(prev.Name)
			}
		}
	}

	// Warn if the bound SSH key file is missing.
	if user.SSHKey != "" {
		if _, statErr := os.Stat(user.SSHKey); statErr != nil {
			warnings = append(warnings, fmt.Sprintf("Bound SSH key not found: %s — fix it with bind using the new key path.", user.SSHKey))
		}
	}

	// Passphrase gate.
	if user.SSHKey != "" {
		mode := user.GetPassphraseMode()
		protected, perr := isSSHKeyPassphraseProtected(user.SSHKey)
		if perr == nil && protected && !ssh.IsSSHKeyLoaded(user.SSHKey) {
			p := passphrase
			if p == "" && mode == "persistent" {
				if secret, kerr := keyring.GetKeychainPassphrase(user.Name); kerr == nil && secret != "" {
					if ssh.VerifyPassphrase(user.SSHKey, secret) {
						p = secret
					} else {
						_ = keyring.DeleteKeychainPassphrase(user.Name)
					}
				}
			}
			if p == "" {
				return opResult{}, ErrNeedsPassphrase
			}
			if !ssh.VerifyPassphrase(user.SSHKey, p) {
				return opResult{}, fmt.Errorf("incorrect passphrase")
			}
			if passphrase != "" && mode == "persistent" {
				_ = keyring.SetKeychainPassphrase(user.Name, passphrase)
			}
			if ssh.EnsureSSHAgent() == nil {
				if err := ssh.AddSSHKeyWithPassphrase(user.SSHKey, p); err != nil {
					warnings = append(warnings, fmt.Sprintf("Could not load key into agent: %v", err))
				}
			}
		}
	}

	if err := git.Apply(user.Name, user.Email); err != nil {
		return opResult{}, fmt.Errorf("applying git config: %w", err)
	}
	if err := applyUserSSHConfig(user, false); err != nil {
		warnings = append(warnings, fmt.Sprintf("applying SSH config: %v", err))
	}
	if !user.SignDisabled && user.SignKey != "" {
		if err := git.ConfigureSigning(user.SignKey, user.SignFormat); err != nil {
			warnings = append(warnings, fmt.Sprintf("applying signing config: %v", err))
		}
	} else {
		git.RemoveSigningConfig()
	}
	if prev := store.CurrentUser(); prev != nil {
		for k := range prev.CustomConfig {
			_ = unsetActiveCustomConfig(k, false)
		}
	}
	for k, v := range user.CustomConfig {
		_ = applyActiveCustomConfig(k, v, false)
	}

	if err := store.SetCurrent(name); err != nil {
		return opResult{}, err
	}
	if err := config.Save(store); err != nil {
		return opResult{}, fmt.Errorf("saving config: %w", err)
	}

	report := fmt.Sprintf("Switched to %q (%s)\n", user.Name, user.Email)
	if !user.SignDisabled && user.SignKey != "" {
		report += fmt.Sprintf("Commit Signing: Enabled (%s)\n", user.SignFormat)
	}
	if user.SSHKey != "" && !ssh.IsSSHKeyLoaded(user.SSHKey) {
		report += "Note: use 'Check SSH connection' from the profile menu to verify the key works.\n"
	}
	for _, w := range warnings {
		report += "⚠ " + w + "\n"
	}

	return opResult{detail: report, showReport: len(warnings) > 0}, nil
}

// applyUserSSHConfig mirrors the CLI logic for core.sshCommand.
func applyUserSSHConfig(user *config.User, local bool) error {
	if user.SSHCommand != "" {
		return git.SetSSHCommandScope(user.SSHCommand, local)
	}
	if user.SSHKey != "" {
		return git.ConfigureSSHScope(user.SSHKey, local)
	}
	return git.RemoveSSHConfigScope(local)
}

func applyActiveCustomConfig(key, value string, local bool) error {
	scope := "--global"
	if local {
		scope = "--local"
	}
	return exec.Command("git", "config", scope, key, value).Run()
}

func unsetActiveCustomConfig(key string, local bool) error {
	scope := "--global"
	if local {
		scope = "--local"
	}
	return exec.Command("git", "config", scope, "--unset-all", key).Run()
}

// opLogout signs out of the current identity.
func opLogout(store *config.Store) (opResult, error) {
	user := store.CurrentUser()
	if user == nil {
		return opResult{detail: "Already signed out — no active identity."}, nil
	}
	if user.SSHKey != "" && ssh.IsSSHKeyLoaded(user.SSHKey) {
		_ = ssh.RemoveSSHKey(user.SSHKey)
	}
	git.ClearIdentity()
	if user.IsTemporary {
		store.RemoveUser(user.Name, true)
		if user.SSHKey != "" {
			_ = os.Remove(user.SSHKey)
			_ = os.Remove(user.SSHKey + ".pub")
		}
		_ = keyring.DeleteKeychainPassphrase(user.Name)
	}
	store.Current = ""
	if err := config.Save(store); err != nil {
		return opResult{}, fmt.Errorf("saving config: %w", err)
	}
	return opResult{detail: fmt.Sprintf("Signed out from %q. No active git identity.", user.Name)}, nil
}

// opRename renames an identity.
func opRename(store *config.Store, name, newName string) error {
	if store.FindUser(newName) != nil {
		return fmt.Errorf("identity %q already exists", newName)
	}
	u := store.FindUser(name)
	if u == nil {
		return fmt.Errorf("identity %q not found", name)
	}
	u.Name = newName
	if store.Current == name {
		store.Current = newName
	}
	return config.Save(store)
}

// opChangeEmail updates an identity's email, reapplying git config when active.
func opChangeEmail(store *config.Store, name, newEmail string) error {
	for _, u := range store.Users {
		if u.Name != name && u.Email == newEmail {
			return fmt.Errorf("email already in use — each identity must have a unique email")
		}
	}
	if err := store.UpdateUser(name, newEmail); err != nil {
		return err
	}
	if store.Current == name {
		u := store.FindUser(name)
		if err := git.Apply(u.Name, u.Email); err != nil {
			return fmt.Errorf("re-applying git config: %w", err)
		}
	}
	return config.Save(store)
}

// opRemove removes an identity and returns the bound SSH key path so the UI can
// offer to delete the key files. The confirmation dialog already covers the
// destructive nature, so the active identity may also be removed (its git
// config is cleared afterwards).
func opRemove(store *config.Store, name string) (string, error) {
	user := store.FindUser(name)
	if user == nil {
		return "", fmt.Errorf("identity %q not found", name)
	}
	sshKey := user.SSHKey
	wasActive := store.Current == name
	if err := store.RemoveUser(name, true); err != nil {
		return "", err
	}
	_ = keyring.DeleteKeychainPassphrase(name)
	if wasActive {
		git.ClearIdentity()
	}
	if err := config.Save(store); err != nil {
		return "", err
	}
	return sshKey, nil
}

// opDeleteKeyFiles removes an SSH key pair.
func opDeleteKeyFiles(keyPath string) {
	if keyPath == "" {
		return
	}
	_ = os.Remove(keyPath)
	_ = os.Remove(keyPath + ".pub")
}

