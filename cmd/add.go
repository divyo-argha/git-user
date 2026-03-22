package cmd

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runAdd(args []string) error {
	var name, email, signingKey, method string
	posArgs := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--signing-key":
			if i+1 < len(args) {
				signingKey = args[i+1]
				i++
			}
		case "--method":
			if i+1 < len(args) {
				method = args[i+1]
				i++
			}
		default:
			if posArgs == 0 {
				name = args[i]
			} else if posArgs == 1 {
				email = args[i]
			}
			posArgs++
		}
	}

	if name == "" || email == "" {
		ui.Error("usage: git-user add <name> <email> [--signing-key <key>] [--method gpg|ssh]")
		return fmt.Errorf("missing arguments")
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if err := store.AddUser(name, email); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if signingKey != "" {
		if err := store.BindSigningKey(name, signingKey, method); err != nil {
			ui.Errorf("binding signing key: %v", err)
			return err
		}
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Added user %q (%s)", name, email))
	ui.Info(fmt.Sprintf("Run 'git-user switch %s' to activate", name))
	return nil
}
