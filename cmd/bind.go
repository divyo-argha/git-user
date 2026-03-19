package cmd

import (
	"fmt"
	"os"

	"github.com/local/git-user/internal/config"
	"github.com/local/git-user/internal/ui"
)

func runBind(args []string) error {
	var name, keyPath string
	for i := 0; i < len(args); i++ {
		if args[i] == "--ssh-key" {
			if i+1 < len(args) {
				keyPath = args[i+1]
				i++
			}
		} else {
			name = args[i]
		}
	}

	if name == "" || keyPath == "" {
		ui.Error("usage: git-user bind <name> --ssh-key <path>")
		return fmt.Errorf("missing arguments")
	}

	// Basic validation: does the key file exist?
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		ui.Errorf("SSH key file %q does not exist", keyPath)
		return err
	}

	store, err := config.Load()
	if err != nil {
		ui.Error("loading config")
		return err
	}

	if err := store.BindSSHKey(name, keyPath); err != nil {
		ui.Error(err.Error())
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Error("saving config")
		return err
	}

	ui.Success(fmt.Sprintf("Associated SSH key %q with user %q", keyPath, name))
	return nil
}
