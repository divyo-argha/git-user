package cmd

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runBind(args []string) error {
	var name, sshKeyPath string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ssh-key":
			if i+1 < len(args) {
				sshKeyPath = args[i+1]
				i++
			}
		default:
			name = args[i]
		}
	}

	if name == "" {
		ui.Error("usage: git-user bind <name> --ssh-key <path>")
		return fmt.Errorf("missing name")
	}

	if sshKeyPath == "" {
		ui.Error("usage: git-user bind <name> --ssh-key <path>")
		return fmt.Errorf("missing ssh-key")
	}

	// Validate SSH key exists
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		ui.Errorf("SSH key file %q does not exist", sshKeyPath)
		return err
	}

	store, err := config.Load()
	if err != nil {
		ui.Error("loading config")
		return err
	}

	if err := store.BindSSHKey(name, sshKeyPath); err != nil {
		ui.Error(err.Error())
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Error("saving config")
		return err
	}

	ui.Success(fmt.Sprintf("Associated SSH key %q with user %q", sshKeyPath, name))
	return nil
}
