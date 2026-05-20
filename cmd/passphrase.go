package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runPassphrase(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user passphrase <name>")
		return fmt.Errorf("missing name")
	}

	name := args[0]

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := store.FindUser(name)
	if user == nil {
		ui.Errorf("identity %q not found", name)
		ui.Info("Run 'git-user list' to see available identities.")
		return fmt.Errorf("user not found")
	}

	if err := requireActivePassphraseIdentity(store, user); err != nil {
		return err
	}

	if user.SSHKey == "" {
		ui.Warn(fmt.Sprintf("Identity %q has no SSH key bound", name))
		ui.Info(fmt.Sprintf("Run: git-user bind %s", name))
		return fmt.Errorf("no ssh key")
	}

	if _, err := os.Stat(user.SSHKey); err != nil {
		ui.Errorf("SSH key file is not accessible: %s", user.SSHKey)
		ui.Info(fmt.Sprintf("Run 'git-user bind %s' to attach an existing key, or 'git-user rekey %s' to create a new one.", name, name))
		return err
	}

	if err := requireActivePassphraseSession(user); err != nil {
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

func requireActivePassphraseIdentity(store *config.Store, user *config.User) error {
	if store.Current == "" {
		ui.Error("No active identity.")
		ui.Info(fmt.Sprintf("Switch first: git-user switch %s", user.Name))
		return fmt.Errorf("no active identity")
	}
	if store.Current != user.Name {
		ui.Errorf("Cannot change passphrase for %q while %q is active.", user.Name, store.Current)
		ui.Info(fmt.Sprintf("Switch first: git-user switch %s", user.Name))
		return fmt.Errorf("identity not active")
	}
	return nil
}

func requireActivePassphraseSession(user *config.User) error {
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		ui.Error("No active SSH session.")
		ui.Info("Start ssh-agent, then run: git-user session start")
		return fmt.Errorf("ssh-agent not running")
	}
	if !isSSHKeyLoaded(user.SSHKey) {
		ui.Errorf("SSH key for %q is not loaded in this session.", user.Name)
		ui.Info("Unlock it first: git-user session start")
		return fmt.Errorf("identity session not active")
	}
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
	cmd := exec.Command("ssh-keygen", "-p", "-f", keyPath, "-P", oldPassphrase, "-N", newPassphrase)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
