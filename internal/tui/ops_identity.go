package tui

import (
	"fmt"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/identity"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"os"
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

	if store.Current == name && git.IsIdentityInSync(user.Name, user.Email) {
		return opResult{detail: fmt.Sprintf("Already using identity %q (%s) — nothing to do.", user.Name, user.Email)}, nil
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
					_ = identity.SecureDeleteKeyPair(prev.SSHKey)
					_ = identity.ForgetTempKey(prev.SSHKey)
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
			// EnsureSSHAgent prints its own guidance on failure, but that goes
			// to raw stdout — invisible or corrupted under the TUI's
			// alt-screen rendering, unlike everything else here which surfaces
			// through `warnings` into the report/toast the user actually
			// sees. Without this, a switch with no reachable agent would
			// report success with no indication the key was never loaded.
			if agentErr := ssh.EnsureSSHAgent(); agentErr != nil {
				warnings = append(warnings, fmt.Sprintf("Key for %q was NOT loaded into any ssh-agent (no agent reachable) — the next push/pull may hang or fail asking for a passphrase.", user.Name))
			} else if err := ssh.AddSSHKeyWithPassphrase(user.SSHKey, p); err != nil {
				warnings = append(warnings, fmt.Sprintf("Could not load key into agent: %v", err))
			}
		}
	}

	if git.IsInRepo() && git.HasLocalOverride() {
		git.ClearIdentityScope(true)
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
	wd, _ := os.Getwd()
	_ = config.AppendSwitchLog(user.Name, wd)

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

// opSwitchSession copies the shell command that activates an identity for the
// current terminal session only (via GIT_AUTHOR_*/GIT_CONFIG_PARAMETERS env
// vars). It never touches the global gitconfig or the config store, so other
// terminals — and this one after the session ends — are unaffected.
func opSwitchSession(store *config.Store, name string) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}

	cmd := fmt.Sprintf(`eval "$(git-user env %s)"`, name)
	if err := screens.ClipboardWrite(cmd); err != nil {
		return opResult{}, fmt.Errorf("copy to clipboard: %w", err)
	}

	detail := fmt.Sprintf(
		"Copied to clipboard: %s\n\nPaste it into the terminal you want %q active in and press Enter. It only affects that terminal session — your global identity and other terminals are unaffected.",
		cmd, name,
	)
	return opResult{detail: detail}, nil
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
	_, err := runCaptured("", "git", "config", scope, key, value)
	return err
}

func unsetActiveCustomConfig(key string, local bool) error {
	scope := "--global"
	if local {
		scope = "--local"
	}
	_, err := runCaptured("", "git", "config", scope, "--unset-all", key)
	return err
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
			// SecureDeleteKeyPair (not plain os.Remove) to match the same
			// temporary-key cleanup in opSwitch above — this is the same
			// short-lived private key material, deleted for the same reason.
			_ = identity.SecureDeleteKeyPair(user.SSHKey)
			_ = identity.ForgetTempKey(user.SSHKey)
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
	oldUser := store.FindUser(name)
	localOverrideMatched := oldUser != nil && git.IsInRepo() && git.HasLocalOverride() &&
		git.CurrentLocalName() == oldUser.Name && git.CurrentLocalEmail() == oldUser.Email

	if err := store.RenameUser(name, newName); err != nil {
		return err
	}
	u := store.FindUser(newName)
	if store.Current == newName && u != nil {
		if err := git.Apply(u.Name, u.Email); err != nil {
			return fmt.Errorf("re-applying git config: %w", err)
		}
	}
	if localOverrideMatched && u != nil {
		_ = git.ApplyScope(u.Name, u.Email, true)
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
	oldUser := store.FindUser(name)
	localOverrideMatched := oldUser != nil && git.IsInRepo() && git.HasLocalOverride() &&
		git.CurrentLocalName() == oldUser.Name && git.CurrentLocalEmail() == oldUser.Email

	if err := store.UpdateUser(name, newEmail); err != nil {
		return err
	}
	u := store.FindUser(name)
	if store.Current == name {
		if err := git.Apply(u.Name, u.Email); err != nil {
			return fmt.Errorf("re-applying git config: %w", err)
		}
	}
	if localOverrideMatched {
		_ = git.ApplyScope(u.Name, u.Email, true)
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
