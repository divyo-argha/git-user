package cmd

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runRemove(args []string) error {
	force := false
	filtered := args[:0]
	for _, a := range args {
		if a == "--force" || a == "-f" {
			force = true
		} else {
			filtered = append(filtered, a)
		}
	}
	args = filtered

	if len(args) < 1 {
		ui.Error("usage: git-user remove <n> [--force]")
		return fmt.Errorf("missing arguments")
	}

	name := args[0]

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if err := store.RemoveUser(name, force); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Removed user %q", name))
	if store.Current == "" {
		ui.Warn("No active identity — run 'git-user switch <n>'")
	}
	return nil
}
