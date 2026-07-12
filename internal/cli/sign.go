package cli

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
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
		store.ToggleSigning(name, true)
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
		if key == "" {
			// Try to auto-detect SSH key
			if user.SSHKey != "" {
				key = user.SSHKey
				if format == "" {
					format = "ssh"
				}
				ui.Info(fmt.Sprintf("Using bound SSH key for signing: %s", key))
			} else {
				ui.Error("No SSH key bound to this profile. Please provide a key using --key.")
				return fmt.Errorf("no key provided")
			}
		}

		if format == "" {
			if strings.HasPrefix(key, "ssh-") || strings.Contains(key, "id_") || strings.HasSuffix(key, ".pub") {
				format = "ssh"
			} else {
				format = "gpg"
			}
		}

		if format == "ssh" {
			key = expandPath(key)
		}

		store.SetSigningKey(name, key, format)
		if err := config.Save(store); err != nil {
			ui.Errorf("saving config: %v", err)
			return err
		}

		ui.Success(fmt.Sprintf("Commit signing enabled for user %q", name))
		ui.Success(fmt.Sprintf("Key: %s (%s)", key, format))

		if store.Current == name {
			if err := git.ConfigureSigning(key, format); err != nil {
				ui.Warn(fmt.Sprintf("Failed to update git signing config: %v", err))
			} else {
				ui.Success("Active git config updated with signing keys.")
			}
		}
		return nil
	}

	// Just show status
	if user.SignDisabled || user.SignKey == "" {
		ui.Info(fmt.Sprintf("Commit signing for %q is currently DISABLED.", name))
	} else {
		ui.Success(fmt.Sprintf("Commit signing for %q is ENABLED.", name))
		ui.Info(fmt.Sprintf("Key: %s", user.SignKey))
		ui.Info(fmt.Sprintf("Format: %s", user.SignFormat))
	}

	return nil
}
