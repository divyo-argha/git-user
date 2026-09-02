package cli

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
	"github.com/divyo-argha/git-user/internal/validate"
)

func runConfig(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user config <list|set|unset> ...")
		return fmt.Errorf("missing verb")
	}

	// Verb-first syntax: config list [identity], config set <identity> <key> <value>,
	// config unset <identity> <key>.
	verbs := map[string]bool{"list": true, "set": true, "unset": true}
	if verbs[args[0]] {
		return runConfigVerbFirst(args)
	}

	// Hidden backwards-compatible alias: config <identity> [list|set|unset] ...
	return runConfigIdentityFirst(args)
}

func runConfigVerbFirst(args []string) error {
	switch args[0] {
	case "set":
		if len(args) < 4 {
			ui.Error("usage: git-user config set <identity> <key> <value>")
			return fmt.Errorf("missing key or value")
		}
		return applyConfigAction(args[1], "set", args[2], args[3])

	case "unset":
		if len(args) < 3 {
			ui.Error("usage: git-user config unset <identity> <key>")
			return fmt.Errorf("missing key")
		}
		return applyConfigAction(args[1], "unset", args[2], "")

	case "list":
		name := ""
		if len(args) > 1 {
			name = args[1]
		}
		if name == "" {
			store, err := config.Load()
			if err != nil {
				ui.Errorf("loading config: %v", err)
				return err
			}
			name = store.Current
			if name == "" {
				ui.Error("no identity given and no active identity — usage: git-user config list <identity>")
				return fmt.Errorf("missing identity")
			}
		}
		return applyConfigAction(name, "list", "", "")
	}
	return nil
}

func runConfigIdentityFirst(args []string) error {
	name := args[0]
	action := "list"
	if len(args) > 1 {
		action = args[1]
	}

	switch action {
	case "set":
		if len(args) < 4 {
			ui.Error("usage: git-user config set <identity> <key> <value>")
			return fmt.Errorf("missing key or value")
		}
		return applyConfigAction(name, "set", args[2], args[3])
	case "unset":
		if len(args) < 3 {
			ui.Error("usage: git-user config unset <identity> <key>")
			return fmt.Errorf("missing key")
		}
		return applyConfigAction(name, "unset", args[2], "")
	case "list":
		return applyConfigAction(name, "list", "", "")
	default:
		ui.Errorf("unknown config action %q. Supported actions: list, set, unset", action)
		return fmt.Errorf("unknown action")
	}
}

func applyConfigAction(name, action, key, value string) error {
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
		if err := validate.GitConfigKey(key); err != nil {
			ui.Errorf("%v", err)
			return err
		}
		if err := validate.GitConfigValue(value); err != nil {
			ui.Errorf("%v", err)
			return err
		}

		if user.CustomConfig == nil {
			user.CustomConfig = make(map[string]string)
		}
		user.CustomConfig[key] = value

		if err := config.Save(store); err != nil {
			ui.Errorf("saving config: %v", err)
			return err
		}

		if store.Current == name {
			_ = applyActiveCustomConfig(key, value, false)
			ui.Info("Active identity updated. Applied changes to git config.")
		}

		ui.Successf("Set config %q = %q for identity %q", key, value, name)

	case "unset":
		if err := validate.GitConfigKey(key); err != nil {
			ui.Errorf("%v", err)
			return err
		}

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

		var keys []string
		for k := range user.CustomConfig {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fmt.Printf("  %s = %s\n", k, user.CustomConfig[k])
		}
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
