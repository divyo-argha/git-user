package tui

import (
	"fmt"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/identity"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/shellinit"
	"github.com/divyo-argha/git-user/internal/ssh"
	"github.com/divyo-argha/git-user/internal/switchops"
	"github.com/divyo-argha/git-user/internal/tui/screens"
)

// ── Switch / Logout / Remove ──────────────────────────────────────────────────

// opSwitch switches the active identity. Passphrase is optional; if the key is
// protected and not loaded it must be provided (or retrieved from the keychain).
func opSwitch(store *config.Store, name, passphrase string) (opResult, error) {
	var warnings []string
	if !git.IsInstalled() {
		warnings = append(warnings, "Git binary not found on PATH. Saved identity directly to ~/.gitconfig.")
	}

	store.SnapshotOriginal(git.CurrentName(), git.CurrentEmail(), git.CurrentSSHCommand(), git.CurrentSigningKey(), git.CurrentSignFormat(), git.CurrentCommitGPGSign())

	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}

	if switchops.AlreadyActive(store, user, false) {
		return opResult{detail: fmt.Sprintf("Already using identity %q (%s) — nothing to do.", user.Name, user.Email)}, nil
	}

	// Auto-logout: unload the previous identity's key and clean up temporaries.
	// Notices are discarded (not surfaced in `warnings`) to keep this
	// unchanged from opSwitch's previous behavior, which never reported them.
	_ = switchops.LogoutPrevious(store, name)

	// Warn if the bound SSH key file is missing.
	if switchops.BoundKeyMissing(user) {
		warnings = append(warnings, fmt.Sprintf("Bound SSH key not found: %s — fix it with bind using the new key path.", user.SSHKey))
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

	applyWarnings, err := switchops.ApplyIdentity(store, user, false, applyActiveCustomConfig, unsetActiveCustomConfig)
	if err != nil {
		return opResult{}, err
	}
	warnings = append(warnings, applyWarnings...)

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

	sh := shellinit.Detect("")
	var cmd string
	switch sh {
	case shellinit.PowerShell:
		cmd = fmt.Sprintf(`Invoke-Expression (& git-user env %s --pwsh)`, name)
	case shellinit.Cmd:
		cmd = fmt.Sprintf(`gu %s`, name)
	case shellinit.Fish:
		cmd = fmt.Sprintf(`git-user env %s --fish | source`, name)
	default:
		cmd = fmt.Sprintf(`eval "$(git-user env %s)"`, name)
	}

	if err := screens.ClipboardWrite(cmd); err != nil {
		return opResult{}, fmt.Errorf("copy to clipboard: %w", err)
	}

	detail := fmt.Sprintf(
		"Copied to clipboard: %s\n\nPaste it into the terminal you want %q active in and press Enter. It only affects that terminal session — your global identity and other terminals are unaffected.",
		cmd, name,
	)
	return opResult{detail: detail}, nil
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
	migrateKeyringOnRename(name, newName)
	u := store.FindUser(newName)
	if store.Current == newName && u != nil {
		if err := git.Apply(u.Name, u.Email); err != nil {
			return fmt.Errorf("re-applying git config: %w", err)
		}
		switchops.ApplyHTTPSCredentialConfig(u, false)
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
	_ = keyring.DeleteHTTPSToken(name)
	if wasActive {
		git.ClearIdentity()
		git.RemoveAskpassConfig()
	}
	if err := config.Save(store); err != nil {
		return "", err
	}
	return sshKey, nil
}

// opDeleteKeyFiles securely deletes an SSH key pair (3-pass overwrite before
// unlink), matching the CLI's remove path (internal/cli/remove.go) so a key
// deleted from the TUI isn't forensically recoverable from disk any more
// than one deleted from the CLI.
func opDeleteKeyFiles(keyPath string) error {
	if keyPath == "" {
		return nil
	}
	return identity.SecureDeleteKeyPair(keyPath)
}
