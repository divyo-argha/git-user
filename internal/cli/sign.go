package cli

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/signing"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runSign(args []string) error {
	var name, key, format string
	var on, off bool

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--key":
			if i+1 < len(args) {
				key = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case "--on":
			on = true
		case "--off":
			off = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				name = args[i]
			}
		}
	}

	if name == "" {
		ui.Error("usage: git-user sign <name> [--on|--off] [--key <key>] [--format ssh|gpg]")
		return fmt.Errorf("missing name")
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

	if off {
		signing.Disable(store, name)
		if err := config.Save(store); err != nil {
			ui.Errorf("saving config: %v", err)
			return err
		}
		ui.Success(fmt.Sprintf("Commit signing disabled for user %q", name))
		if store.Current == name {
			git.RemoveSigningConfig()
			ui.Info("Removed signing configuration from active git profile.")
		}
		return nil
	}

	if on || key != "" {
		resolvedKey, resolvedFormat, autoDetected, err := signing.Enable(store, name, key, format)
		if err != nil {
			if err == signing.ErrNoKeyBound {
				ui.Error("No SSH key bound to this profile. Please provide a key using --key.")
			} else {
				ui.Errorf("%v", err)
			}
			return err
		}
		if autoDetected {
			ui.Info(fmt.Sprintf("Using bound SSH key for signing: %s", resolvedKey))
		}

		if err := config.Save(store); err != nil {
			ui.Errorf("saving config: %v", err)
			return err
		}

		ui.Success(fmt.Sprintf("Commit signing enabled for user %q", name))
		ui.Success(fmt.Sprintf("Key: %s (%s)", resolvedKey, resolvedFormat))

		if store.Current == name {
			if err := git.ConfigureSigning(resolvedKey, resolvedFormat); err != nil {
				ui.Warn(fmt.Sprintf("Failed to update git signing config: %v", err))
			} else {
				ui.Success("Active git config updated with signing keys.")
			}
		}
		return nil
	}

	status := signing.CurrentStatus(user)
	if !status.Enabled {
		ui.Info(fmt.Sprintf("Commit signing for %q is currently DISABLED.", name))
	} else {
		ui.Success(fmt.Sprintf("Commit signing for %q is ENABLED.", name))
		ui.Info(fmt.Sprintf("Key: %s", status.Key))
		ui.Info(fmt.Sprintf("Format: %s", status.Format))
	}

	return nil
}
