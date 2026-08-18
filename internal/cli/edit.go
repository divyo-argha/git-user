package cli

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runEdit(args []string) error {
	if len(args) < 2 {
		ui.Error("usage: git-user edit <n> <new-email>")
		return fmt.Errorf("missing arguments")
	}

	name := args[0]
	newEmail := args[1]

	if !isValidEmail(newEmail) {
		ui.Error("invalid email format")
		return fmt.Errorf("invalid email")
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := store.FindUser(name)
	if user == nil {
		ui.Errorf("identity %q not found", name)
		return fmt.Errorf("user not found")
	}

	for _, u := range store.Users {
		if u.Name != name && u.Email == newEmail {
			ui.Errorf("Email already in use — each identity must have a unique email to prevent impersonation.")
			return fmt.Errorf("email exists")
		}
	}

	// A repo-local override pointing at this identity's old email needs the
	// same update applied at --local scope, or it goes stale.
	localOverrideMatched := git.IsInRepo() && git.HasLocalOverride() &&
		git.CurrentLocalName() == user.Name && git.CurrentLocalEmail() == user.Email

	if err := store.UpdateUser(name, newEmail); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	u := store.FindUser(name)

	// If this is the active user, re-apply git config immediately.
	if store.Current == name {
		if err := git.Apply(u.Name, u.Email); err != nil {
			ui.Errorf("re-applying git config: %v", err)
			return err
		}
		ui.Info("Active identity updated — git config re-applied automatically.")
	}

	if localOverrideMatched {
		if err := git.ApplyScope(u.Name, u.Email, true); err != nil {
			ui.Warn(fmt.Sprintf("could not update this repo's local override: %v", err))
		} else {
			ui.Info("Updated this repository's local override to match the new email.")
		}
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Updated %q → email is now %s", name, newEmail))
	return nil
}
