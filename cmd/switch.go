package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runSwitch(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user switch [-c] <name> [email]")
		return fmt.Errorf("missing arguments")
	}

	createMode := false
	name := ""
	rest := []string{}

	if args[0] == "-c" {
		if len(args) < 2 {
			ui.Error("usage: git-user switch -c <name> [email]")
			return fmt.Errorf("missing arguments")
		}
		createMode = true
		name = args[1]
		rest = args[2:]
	} else {
		name = args[0]
	}

	if !git.IsInstalled() {
		ui.Error("git is not installed or not on PATH")
		return fmt.Errorf("git not found")
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	// Clear credentials from previous user if not remembered
	if currentUser := store.CurrentUser(); currentUser != nil {
		if err := store.ClearCredentialsForUser(currentUser.Name); err != nil {
			ui.Warn(fmt.Sprintf("clearing credentials: %v", err))
		}
	}

	if createMode {
		if store.FindUser(name) != nil {
			ui.Errorf("user %q already exists", name)
			return fmt.Errorf("user exists")
		}
		// Create the user first
		if err := runAdd(append([]string{name}, rest...)); err != nil {
			return err
		}
		// Reload store to get the new user data
		store, _ = config.Load()
	}

	user := store.FindUser(name)
	if user == nil {
		ui.Errorf("user %q not found", name)
		return fmt.Errorf("user not found")
	}

	if err := ApplyIdentity(user, store); err != nil {
		return err
	}

	if err := store.SetCurrent(name); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Switched to %q (%s)", user.Name, user.Email))
	return nil
}

// ApplyIdentity synchronizes the global git and ssh configurations with the given user profile.
func ApplyIdentity(user *config.User, store *config.Store) error {
	if err := git.Apply(user.Name, user.Email); err != nil {
		ui.Errorf("applying git config: %v", err)
		return err
	}

	if user.SSHKey != "" {
		if err := git.ConfigureSSH(user.SSHKey); err != nil {
			ui.Warn(fmt.Sprintf("applying SSH config: %v", err))
		}
	} else {
		if err := git.RemoveSSHConfig(); err != nil {
			ui.Warn(fmt.Sprintf("removing SSH config: %v", err))
		}
	}

	if user.SigningKey != "" {
		keyExists := true
		// If it looks like a path, check if it exists
		if strings.Contains(user.SigningKey, "/") || strings.Contains(user.SigningKey, "~") {
			if _, err := os.Stat(user.SigningKey); os.IsNotExist(err) {
				keyExists = false
			}
		}

		if !keyExists {
			if !store.Strict {
				ui.Warn(fmt.Sprintf("Signing key %q not found. Operating in Flexible mode — disabling signing to prevent errors.", user.SigningKey))
				_ = git.RemoveSigningConfig()
			} else {
				ui.Warn(fmt.Sprintf("Signing key %q not found. Operating in Strict mode — signing will remain enabled (commits may fail).", user.SigningKey))
				_ = git.ApplySigning(user.SigningKey, user.SigningMethod)
			}
		} else {
			if err := git.ApplySigning(user.SigningKey, user.SigningMethod); err != nil {
				ui.Warn(fmt.Sprintf("applying signing config: %v", err))
			}
		}
	} else {
		if store.Strict {
			ui.Warn("No signing key bound. Operating in Strict mode — signing is still ENFORCED (commits will fail until you bind a key).")
			_ = git.ApplySigning("GIT_USER_STRICT_NO_KEY_BOUND", "ssh")
		} else {
			if err := git.RemoveSigningConfig(); err != nil {
				ui.Warn(fmt.Sprintf("removing signing config: %v", err))
			}
		}
	}

	return nil
}
