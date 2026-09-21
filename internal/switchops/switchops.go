// Package switchops holds the mechanical, non-interactive parts of
// switching the active git identity, shared by internal/cli's `switch`
// (including `switch --local`) and internal/tui's opSwitch — mirroring how
// internal/rekeyops shares key-rotation mechanics between the same two
// callers.
//
// Deliberately NOT here: the passphrase gate (deciding whether to prompt for
// one, and whether to offer storing it in the keychain) and all interactive
// I/O. Those differ in real, product-level ways between the two surfaces —
// the CLI blocks on a terminal prompt and asks explicit consent before
// storing a newly-entered passphrase in the keychain, while the TUI signals
// the caller via a sentinel error to push a form, and stores automatically
// once persistent mode is already the identity's configured setting.
// Unifying that gate would either silently drop the CLI's consent prompt or
// silently add a TUI consent prompt that was never there — a real behavior
// decision, not a refactor. So each surface keeps its own passphrase gate;
// this package only covers the parts where both already did the exact same
// thing and had no reason not to.
package switchops

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/gitenv"
	"github.com/divyo-argha/git-user/internal/identity"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
)

// AlreadyActive reports whether user is already the active identity and the
// live git config already matches it, in which case a global switch (never
// a local one — --local always re-applies, since it targets a possibly
// different repo's config) has nothing to do.
func AlreadyActive(store *config.Store, user *config.User, local bool) bool {
	return !local && store.Current == user.Name && git.IsIdentityInSync(user.Name, user.Email)
}

// BoundKeyMissing reports whether user has an SSH key bound but the file no
// longer exists on disk — a switch should still proceed (the caller decides
// how to warn), rather than block, since fixing the binding is easier to do
// as a follow-up than to be locked out of switching at all.
func BoundKeyMissing(user *config.User) bool {
	if user.SSHKey == "" {
		return false
	}
	_, err := os.Stat(user.SSHKey)
	return err != nil
}

// LogoutPrevious unloads the previously-active identity's SSH key from the
// agent and, if it was a temporary identity, deletes its key files and
// config record. Only meaningful for a global switch: a `--local` switch
// never changes store.Current, so the previous identity is still active
// everywhere else and must not be logged out or have its temp key deleted
// out from under it — callers must not call this for a local switch.
//
// Returns human-readable notices for the caller to surface however it likes
// (ui.Info lines for the CLI, folded into a report for the TUI).
func LogoutPrevious(store *config.Store, newName string) []string {
	if store.Current == "" || store.Current == newName {
		return nil
	}
	prev := store.CurrentUser()
	if prev == nil {
		return nil
	}

	var notices []string
	if prev.SSHKey != "" && ssh.IsSSHKeyLoaded(prev.SSHKey) {
		_ = ssh.RemoveSSHKey(prev.SSHKey)
		notices = append(notices, fmt.Sprintf("Unloaded SSH key for previous identity %q", prev.Name))
	}
	if prev.GetPassphraseMode() == "everytime" && prev.SSHKey != "" {
		_ = ssh.RemoveSSHKey(prev.SSHKey)
	}
	if prev.IsTemporary {
		if err := store.RemoveUser(prev.Name, true); err != nil {
			notices = append(notices, fmt.Sprintf("Could not remove temporary identity record: %v", err))
		} else {
			notices = append(notices, fmt.Sprintf("Temporary identity %q deleted.", prev.Name))
			if prev.SSHKey != "" {
				_ = identity.SecureDeleteKeyPair(prev.SSHKey)
				// Clears the crash-safety orphan-scan registry entry now that
				// this key was cleaned up the normal way. The CLI's switch
				// path used to skip this call (only the TUI's made it),
				// leaving a stale entry for doctor's orphan scan to carry
				// indefinitely after a CLI-driven temp-identity logout.
				_ = identity.ForgetTempKey(prev.SSHKey)
				notices = append(notices, fmt.Sprintf("Temporary SSH key files deleted: %s", prev.SSHKey))
			}
			_ = keyring.DeleteKeychainPassphrase(prev.Name)
		}
	}
	return notices
}

// ApplyHTTPSCredentialConfig wires core.askpass to this identity's stored
// HTTPS token (if any), or removes it if not. Returns a warning string
// (empty on success) instead of printing directly, so each caller can report
// it its own way (ui.Warn for the CLI, folded into a warnings slice for the
// TUI).
func ApplyHTTPSCredentialConfig(user *config.User, local bool) string {
	if !keyring.HasHTTPSToken(user.Name) {
		git.RemoveAskpassConfigScope(local)
		return ""
	}
	cmd, err := gitenv.AskpassCommand(user.Name)
	if err != nil {
		return fmt.Sprintf("Could not resolve git-user's own path to wire up the HTTPS token: %v", err)
	}
	if err := git.ConfigureAskpassScope(cmd, local); err != nil {
		return fmt.Sprintf("Could not apply core.askpass: %v", err)
	}
	return ""
}

// ApplyIdentity applies user's name/email/SSH/signing/HTTPS-credential/
// custom-config settings to git config at the given scope (local = the
// current repository's .git/config, else the global ~/.gitconfig), and —
// only for a global switch — updates store.Current, persists the store, and
// appends a switch-log entry.
//
// setCustomConfig/unsetCustomConfig apply or remove one "git config <scope>
// <key> [value]" pair each. Callers supply them because how the underlying
// git process's output is handled differs by surface: the CLI runs it
// directly, the TUI captures it so nothing leaks onto its alt-screen (see
// internal/tui/ops.go's runCaptured).
//
// A failure applying user.Name/user.Email, or (for a global switch) setting
// the current identity or saving the store, aborts immediately and is
// returned as an error. Every other step — SSH config, signing, HTTPS
// credential, custom config — is best-effort and reported as a warning
// string instead, matching how both callers already treated fatal-vs-soft
// failures before this was shared.
func ApplyIdentity(
	store *config.Store,
	user *config.User,
	local bool,
	setCustomConfig func(key, value string, local bool) error,
	unsetCustomConfig func(key string, local bool) error,
) (warnings []string, err error) {
	if !local && git.IsInRepo() && git.HasLocalOverride() {
		git.ClearIdentityScope(true)
	}

	if err := git.ApplyScope(user.Name, user.Email, local); err != nil {
		return nil, fmt.Errorf("applying git config: %w", err)
	}

	if err := git.ApplyIdentitySSHConfig(user.SSHCommand, user.SSHKey, local); err != nil {
		warnings = append(warnings, fmt.Sprintf("applying SSH config: %v", err))
	}
	if !user.SignDisabled && user.SignKey != "" {
		if err := git.ConfigureSigningScope(user.SignKey, user.SignFormat, local); err != nil {
			warnings = append(warnings, fmt.Sprintf("applying signing config: %v", err))
		}
	} else {
		git.RemoveSigningConfigScope(local)
	}
	if w := ApplyHTTPSCredentialConfig(user, local); w != "" {
		warnings = append(warnings, w)
	}

	if prev := store.CurrentUser(); prev != nil {
		for k := range prev.CustomConfig {
			_ = unsetCustomConfig(k, local)
		}
	}
	for k, v := range user.CustomConfig {
		_ = setCustomConfig(k, v, local)
	}

	if !local {
		if err := store.SetCurrent(user.Name); err != nil {
			return warnings, err
		}
		if err := config.Save(store); err != nil {
			return warnings, fmt.Errorf("saving config: %w", err)
		}
	}

	wd, _ := os.Getwd()
	_ = config.AppendSwitchLog(user.Name, wd)

	return warnings, nil
}
