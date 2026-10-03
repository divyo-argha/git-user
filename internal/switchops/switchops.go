// Package switchops implements shared identity switching and logout logic
// for the CLI and TUI.
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

// AlreadyActive reports whether user is already active and in sync with git config.
func AlreadyActive(store *config.Store, user *config.User, local bool) bool {
	return !local && store.Current == user.Name && git.IsIdentityInSync(user.Name, user.Email)
}

// BoundKeyMissing reports whether the identity's bound SSH key file does not exist on disk.
func BoundKeyMissing(user *config.User) bool {
	if user.SSHKey == "" {
		return false
	}
	_, err := os.Stat(user.SSHKey)
	return err != nil
}

// LogoutPrevious unloads the previous identity's SSH key from the agent and
// removes temporary identities. Only applied during global switches.
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
				_ = identity.ForgetTempKey(prev.SSHKey)
				notices = append(notices, fmt.Sprintf("Temporary SSH key files deleted: %s", prev.SSHKey))
			}
			_ = keyring.DeleteKeychainPassphrase(prev.Name)
		}
	}
	return notices
}

// ApplyHTTPSCredentialConfig configures or removes core.askpass for the identity's HTTPS token.
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

// ApplyIdentity applies user settings to git config at the given scope.
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

// LogoutResult describes an identity signed out by Logout.
type LogoutResult struct {
	Name         string
	WasTemporary bool
	Notices      []string
}

// Logout signs out the active identity and clears global git config.
func Logout(store *config.Store, unsetCustomConfig func(key string, local bool) error) (*LogoutResult, error) {
	user := store.CurrentUser()
	if user == nil {
		return nil, nil
	}
	res := &LogoutResult{Name: user.Name, WasTemporary: user.IsTemporary}
	customKeys := make([]string, 0, len(user.CustomConfig))
	for k := range user.CustomConfig {
		customKeys = append(customKeys, k)
	}

	res.Notices = LogoutPrevious(store, "")

	git.ClearIdentity()
	git.RemoveAskpassConfigScope(false)
	for _, k := range customKeys {
		_ = unsetCustomConfig(k, false)
	}

	store.Current = ""
	if err := config.Save(store); err != nil {
		return res, fmt.Errorf("saving config: %w", err)
	}

	wd, _ := os.Getwd()
	_ = config.AppendSwitchLog(res.Name+" (signed out)", wd)
	return res, nil
}
