package tui

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/keyring"
)

// opSetHTTPSToken stores an HTTPS personal-access-token for an identity (the
// TUI counterpart of `git-user token <name> --set`), and — if the identity
// is the currently active one — wires up core.askpass immediately via
// applyHTTPSCredentialConfig, the same helper opSwitch uses. expiresAt is an
// optional validate.DateLayout date (already validated by the caller); a
// fresh token clears whatever expiry was on record for the last one unless
// expiresAt gives a new one, so doctor never warns about a stale date.
func opSetHTTPSToken(store *config.Store, name, token, username, expiresAt string) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	if err := keyring.SetHTTPSToken(name, token); err != nil {
		return opResult{}, fmt.Errorf("storing token: %w", err)
	}
	configChanged := false
	if username != "" && username != user.HTTPSUsername {
		if err := store.SetHTTPSUsername(name, username); err != nil {
			return opResult{}, err
		}
		configChanged = true
	}
	if expiresAt != user.HTTPSTokenExpiresAt {
		if err := store.SetHTTPSTokenExpiry(name, expiresAt); err != nil {
			return opResult{}, err
		}
		configChanged = true
	}
	if configChanged {
		if err := config.Save(store); err != nil {
			return opResult{}, fmt.Errorf("saving config: %w", err)
		}
	}

	detail := fmt.Sprintf("Token stored securely for %q.", name)
	if expiresAt != "" {
		detail += fmt.Sprintf(" Expiry recorded: %s.", expiresAt)
	}
	if store.Current == name {
		if w := applyHTTPSCredentialConfig(user, false); w != "" {
			detail += "\n" + w
		} else {
			detail += "\nHTTPS remotes will now authenticate as this identity automatically."
		}
	} else {
		detail += fmt.Sprintf("\nIt will take effect the next time you switch to %q.", name)
	}
	return opResult{detail: detail, showReport: true}, nil
}

// opRemoveHTTPSToken removes an identity's stored HTTPS token and, if it's
// the active identity, clears core.askpass so git stops trying to shell back
// into git-user for credentials it can no longer provide.
func opRemoveHTTPSToken(store *config.Store, name string) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	if err := keyring.DeleteHTTPSToken(name); err != nil {
		return opResult{}, fmt.Errorf("removing token: %w", err)
	}
	if user.HTTPSTokenExpiresAt != "" {
		_ = store.SetHTTPSTokenExpiry(name, "")
		_ = config.Save(store)
	}
	if store.Current == name {
		applyHTTPSCredentialConfig(user, false)
	}
	return opResult{detail: fmt.Sprintf("HTTPS token removed for %q.", name)}, nil
}
