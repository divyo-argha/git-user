package cmd

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runPlatform(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: git-user platform <add|remove> ...")
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	sub := args[0]
	switch sub {
	case "add":
		if len(args) < 4 {
			return fmt.Errorf("usage: git-user platform add <name> <platform> <username>")
		}
		name := args[1]
		platform := strings.ToLower(args[2])
		user := args[3]

		if err := store.BindPlatform(name, platform, user); err != nil {
			ui.Errorf("binding platform: %v", err)
			return err
		}
		if err := config.Save(store); err != nil {
			ui.Errorf("saving config: %v", err)
			return err
		}
		ui.Successf("Bound %q to %s as %q", name, platform, user)

	case "remove":
		if len(args) < 3 {
			return fmt.Errorf("usage: git-user platform remove <name> <platform>")
		}
		name := args[1]
		platform := strings.ToLower(args[2])

		if err := store.UnbindPlatform(name, platform); err != nil {
			ui.Errorf("unbinding platform: %v", err)
			return err
		}
		if err := config.Save(store); err != nil {
			ui.Errorf("saving config: %v", err)
			return err
		}
		ui.Successf("Removed %s mapping from %q", platform, name)

	default:
		return fmt.Errorf("unknown platform subcommand %q", sub)
	}

	return nil
}
