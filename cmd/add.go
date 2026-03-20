package cmd

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runAdd(args []string) error {
	if len(args) < 2 {
		ui.Error("usage: git-user add <name> <email>")
		return fmt.Errorf("missing arguments")
	}

	name := args[0]
	email := args[1]

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if err := store.AddUser(name, email); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Added user %q (%s)", name, email))
	ui.Info(fmt.Sprintf("Run 'git-user switch %s' to activate", name))
	return nil
}
