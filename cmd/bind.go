package cmd

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runBind(args []string) error {
	var name, sshKeyPath, signingKey, method string
	unsetSigning := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ssh-key":
			if i+1 < len(args) {
				sshKeyPath = args[i+1]
				i++
			}
		case "--signing-key":
			if i+1 < len(args) {
				signingKey = args[i+1]
				i++
			}
		case "--method":
			if i+1 < len(args) {
				method = args[i+1]
				i++
			}
		case "--unset-signing":
			unsetSigning = true
		default:
			name = args[i]
		}
	}

	if name == "" {
		ui.Error("usage: git-user bind <name> [--ssh-key <path>] [--signing-key <key>] [--method gpg|ssh] [--unset-signing]")
		return fmt.Errorf("missing name")
	}

	if sshKeyPath == "" && signingKey == "" && !unsetSigning {
		ui.Error("nothing to bind. Use --ssh-key, --signing-key, or --unset-signing.")
		return fmt.Errorf("missing arguments")
	}

	// Basic validation for SSH key if provided.
	if sshKeyPath != "" {
		if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
			ui.Errorf("SSH key file %q does not exist", sshKeyPath)
			return err
		}
	}

	store, err := config.Load()
	if err != nil {
		ui.Error("loading config")
		return err
	}

	if sshKeyPath != "" {
		if err := store.BindSSHKey(name, sshKeyPath); err != nil {
			ui.Error(err.Error())
			return err
		}
		ui.Success(fmt.Sprintf("Associated SSH key %q with user %q", sshKeyPath, name))
	}

	if signingKey != "" {
		if err := store.BindSigningKey(name, signingKey, method); err != nil {
			ui.Error(err.Error())
			return err
		}
		ui.Success(fmt.Sprintf("Associated signing key %q (method: %s) with user %q", signingKey, method, name))
	}

	if unsetSigning {
		if err := store.BindSigningKey(name, "", ""); err != nil {
			ui.Error(err.Error())
			return err
		}
		ui.Success(fmt.Sprintf("Removed signing key from user %q", name))
	}

	if err := config.Save(store); err != nil {
		ui.Error("saving config")
		return err
	}

	return nil
	return nil
}
