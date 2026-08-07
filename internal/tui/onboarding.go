package tui

import (
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

// firstRunOriginalIdentity reports the pre-git-user git identity (if any) that
// should be offered for import during the very first TUI launch. It returns
// false when the prompt has already been shown, when identities already exist,
// or when there is nothing to import.
func firstRunOriginalIdentity(store *config.Store) (name, email string, ok bool) {
	if store.ImportPrompted || len(store.Users) > 0 {
		return "", "", false
	}

	// Prefer the stored original snapshot (pre-git-user state), falling back to
	// the current global git config.
	if store.Original != nil {
		name, email = store.Original.Name, store.Original.Email
	} else {
		name, email = git.CurrentName(), git.CurrentEmail()
	}

	if name == "" && email == "" {
		return "", "", false
	}
	return name, email, true
}
