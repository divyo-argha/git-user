package cmd

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runConfig(args []string) error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if len(args) == 0 {
		fmt.Println("Global Configuration:")
		mode := "Flexible (Relaxed)"
		if store.Strict {
			mode = "Strict (Enforced)"
		}
		fmt.Printf("  %-10s: %s\n", ui.StyleDim().Render("Mode"), mode)
		return nil
	}

	sub := args[0]
	switch sub {
	case "--strict":
		if len(args) < 2 {
			return fmt.Errorf("usage: git-user config --strict <on|off>")
		}
		val := strings.ToLower(args[1])
		if val == "on" || val == "true" || val == "yes" {
			store.Strict = true
			ui.Success("Strict Mode enabled (Enforced)")
		} else {
			store.Strict = false
			ui.Success("Strict Mode disabled (Flexible/Relaxed)")
		}

		if err := config.Save(store); err != nil {
			return err
		}

		// Re-apply current identity to make the change instant
		if u := store.CurrentUser(); u != nil {
			if err := ApplyIdentity(u, store); err != nil {
				ui.Warn(fmt.Sprintf("Could not re-apply identity: %v", err))
			} else {
				ui.Success(fmt.Sprintf("Synchronized Git identity: %s", u.Name))
			}
		}
		return nil

	default:
		return fmt.Errorf("unknown config flag %q", sub)
	}
}
