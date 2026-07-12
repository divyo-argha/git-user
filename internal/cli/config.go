package cli

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runConfig(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user config <identity> [set|unset|list] [key] [value]")
		return fmt.Errorf("missing identity")
	}

	name := args[0]
	action := "list"
	if len(args) > 1 {
		action = args[1]
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := store.FindUser(name)
	if user == nil {
		ui.Errorf("identity %q not found", name)
		return fmt.Errorf("identity not found")
	}

	switch action {
	case "set":
		if len(args) < 4 {
			ui.Error("usage: git-user config <identity> set <key> <value>")
			return fmt.Errorf("missing key or value")
		}
		key := args[2]
		value := args[3]

		if user.CustomConfig == nil {
			user.CustomConfig = make(map[string]string)
		}
		user.CustomConfig[key] = value

		if err := config.Save(store); err != nil {
			ui.Errorf("saving config: %v", err)
			return err
		}

		// If this is the active user, we want to immediately apply it
		if store.Current == name {
			// Set the value in git config
			// Note: We need to determine if we should set it globally or locally.
			// However, in git-user, global vs local configuration applies during switch.
			// Let's also apply it to global/active git config if it is currently active.
			// For simplicity, we can tell the user to re-switch or apply it globally.
			// But to be extra friendly, we can set it globally since store.Current is global.
			// Let's set it to global git config if the active identity is current.
			// In switch.go, we apply user configs. Let's do that.
			_ = applyActiveCustomConfig(key, value, false)
			ui.Info("Active identity updated. Applied changes to git config.")
		}

		ui.Successf("Set config %q = %q for identity %q", key, value, name)

	case "unset":
		if len(args) < 3 {
			ui.Error("usage: git-user config <identity> unset <key>")
			return fmt.Errorf("missing key")
		}
		key := args[2]

		if user.CustomConfig != nil {
			delete(user.CustomConfig, key)
		}

		if err := config.Save(store); err != nil {
			ui.Errorf("saving config: %v", err)
			return err
		}

		if store.Current == name {
			_ = unsetActiveCustomConfig(key, false)
			ui.Info("Active identity updated. Removed key from git config.")
		}

		ui.Successf("Unset config %q for identity %q", key, name)

	case "list":
		ui.Banner(fmt.Sprintf("CUSTOM CONFIG FOR IDENTITY: %s", strings.ToUpper(name)))
		if len(user.CustomConfig) == 0 {
			ui.Info("No custom config keys set.")
			return nil
		}

		// Sort keys for consistent output
		var keys []string
		for k := range user.CustomConfig {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fmt.Printf("  %s = %s\n", k, user.CustomConfig[k])
		}

	default:
		ui.Errorf("unknown config action %q. Supported actions: set, unset, list", action)
		return fmt.Errorf("unknown action")
	}

	return nil
}

func applyActiveCustomConfig(key, value string, local bool) error {
	scope := "--global"
	if local {
		scope = "--local"
	}
	return exec.Command("git", "config", scope, key, value).Run()
}

func unsetActiveCustomConfig(key string, local bool) error {
	scope := "--global"
	if local {
		scope = "--local"
	}
	return exec.Command("git", "config", scope, "--unset-all", key).Run()
}
