package cmd

import (
	"fmt"

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

	if err := git.Apply(user.Name, user.Email); err != nil {
		ui.Errorf("applying git config: %v", err)
		return err
	}

	if err := store.SetCurrent(name); err != nil {
		ui.Errorf("%v", err)
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
		if err := git.ApplySigning(user.SigningKey, user.SigningMethod); err != nil {
			ui.Warn(fmt.Sprintf("applying signing config: %v", err))
		}
	} else {
		if err := git.RemoveSigningConfig(); err != nil {
			ui.Warn(fmt.Sprintf("removing signing config: %v", err))
		}
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Switched to %q (%s)", user.Name, user.Email))
	return nil
}
