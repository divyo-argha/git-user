package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runPassphrase(args []string) error {
	var name string
	if len(args) > 1 {
		ui.Error("usage: git-user passphrase [name]")
		return fmt.Errorf("unexpected arguments")
	} else if len(args) == 1 {
		name = args[0]
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	var user *config.User
	if name != "" {
		user = store.FindUser(name)
		if user == nil {
			ui.Errorf("identity %q not found", name)
			return fmt.Errorf("user not found")
		}
	} else {
		user = store.CurrentUser()
		if user == nil {
			ui.Error("No active identity.")
			ui.Info("Switch first: git-user switch <name>")
			ui.Info("Or specify identity name: git-user passphrase <name>")
			return fmt.Errorf("no active identity")
		}
	}

	if user.SSHKey == "" {
		ui.Warn(fmt.Sprintf("Identity %q has no SSH key bound", user.Name))
		ui.Info(fmt.Sprintf("Run: git-user bind %s", user.Name))
		return fmt.Errorf("no ssh key")
	}

	if _, err := os.Stat(user.SSHKey); err != nil {
		ui.Errorf("SSH key file is not accessible: %s", user.SSHKey)
		ui.Info(fmt.Sprintf("Run 'git-user bind %s' to attach an existing key, or 'git-user rekey %s' to create a new one.", user.Name, user.Name))
		return err
	}

	protected, err := isSSHKeyPassphraseProtected(user.SSHKey)
	if err != nil {
		ui.Errorf("Could not inspect SSH key: %v", err)
		return err
	}

	ui.Banner("SSH KEY PASSPHRASE")
	fmt.Println()
	ui.Info(fmt.Sprintf("Identity: %s (%s)", user.Name, user.Email))
	ui.Info(fmt.Sprintf("Key: %s", user.SSHKey))
	fmt.Println()

	oldPassphrase := ""
	if protected {
		ui.Info("This key already has a passphrase. Enter the current passphrase to change it.")
		oldPassphrase, err = readPassphrase("Current passphrase: ")
		if err != nil {
			return err
		}
	} else {
		ui.Warn("This key is currently not passphrase protected.")
	}

	newPassphrase, err := promptRequiredPassphrase("New passphrase: ", "Confirm new passphrase: ")
	if err != nil {
		return err
	}

	if err := changeSSHKeyPassphrase(user.SSHKey, oldPassphrase, newPassphrase); err != nil {
		if protected {
			ui.Error("Could not change passphrase.")
			ui.Info("The current passphrase may be wrong, or the key may be inaccessible.")
			ui.Info("git-user cannot recover a forgotten SSH key passphrase.")
		} else {
			ui.Error("Could not add passphrase.")
			ui.Info("Check that the key file is readable and writable by your user.")
		}
		return err
	}

	if protected {
		ui.Success(fmt.Sprintf("Passphrase changed for %q", user.Name))
	} else {
		ui.Success(fmt.Sprintf("Passphrase added for %q", user.Name))
	}
	ui.Info("Use 'git-user session start' to unlock this key for your work session.")

	return nil
}

func promptRequiredPassphrase(prompt, confirmPrompt string) (string, error) {
	passphrase, err := readPassphrase(prompt)
	if err != nil {
		return "", err
	}
	if passphrase == "" {
		ui.Error("Passphrase must not be empty.")
		return "", fmt.Errorf("empty passphrase")
	}

	confirm, err := readPassphrase(confirmPrompt)
	if err != nil {
		return "", err
	}
	if passphrase != confirm {
		ui.Error("Passphrases do not match.")
		return "", fmt.Errorf("passphrase mismatch")
	}

	return passphrase, nil
}

func changeSSHKeyPassphrase(keyPath, oldPassphrase, newPassphrase string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	cmd := exec.Command("ssh-keygen", "-p", "-f", keyPath)
	env := os.Environ()

	env = append(env, "GIT_USER_ASKPASS_MODE=true")
	env = append(env, "GIT_USER_OLD_PASSPHRASE="+oldPassphrase)
	env = append(env, "GIT_USER_NEW_PASSPHRASE="+newPassphrase)
	env = append(env, "SSH_ASKPASS="+exe)
	env = append(env, "SSH_ASKPASS_REQUIRE=force")

	hasDisplay := false
	for _, e := range env {
		if strings.HasPrefix(e, "DISPLAY=") {
			hasDisplay = true
			break
		}
	}
	if !hasDisplay {
		env = append(env, "DISPLAY=dummy:0")
	}

	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
