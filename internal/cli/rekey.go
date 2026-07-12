package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runRekey(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user rekey <name>")
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
		return fmt.Errorf("user not found")
	}

	ui.Info(fmt.Sprintf("Rotating SSH key for identity: %s (%s)", user.Name, user.Email))

	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	keyPath := filepath.Join(sshDir, fmt.Sprintf("git_%s", name))

	if err := os.MkdirAll(sshDir, 0700); err != nil {
		ui.Errorf("creating .ssh directory: %v", err)
		return err
	}

	backupPath := keyPath + ".backup"
	hasOldKey := false
	if _, err := os.Stat(keyPath); err == nil {
		hasOldKey = true
		ui.Warn(fmt.Sprintf("Backing up existing key to %s", backupPath))
		if err := os.Rename(keyPath, backupPath); err != nil {
			ui.Errorf("backing up key: %v", err)
			return err
		}
		pubKeyPath := keyPath + ".pub"
		if _, err := os.Stat(pubKeyPath); err == nil {
			os.Rename(pubKeyPath, backupPath+".pub")
		}
	}

	ui.Info(fmt.Sprintf("Generating new SSH key at %s...", keyPath))
	ui.Info("You will be prompted to set a passphrase for the key.")
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", user.Email, "-f", keyPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if hasOldKey {
			os.Rename(backupPath, keyPath)
			os.Rename(backupPath+".pub", keyPath+".pub")
			ui.Warn("Restored old key — nothing changed")
		}
		ui.Errorf("generating SSH key: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("New SSH key created at %s", keyPath))
	checkAndPromptPassphrase(name, keyPath)

	pubKeyPath := keyPath + ".pub"
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		ui.Errorf("reading public key: %v", err)
		return err
	}

	ui.Divider()
	ui.Banner("REPLACE YOUR OLD KEY WITH THIS NEW PUBLIC KEY")
	fmt.Println()
	fmt.Println(string(pubKeyBytes))
	ui.Divider()
	ui.Info("GitHub: Settings → SSH and GPG keys → Delete old key → Add new key")
	ui.Info("GitLab: Preferences → SSH Keys → Remove old key → Add new key")
	ui.Info("Bitbucket: Personal settings → SSH keys → Delete old → Add new")
	fmt.Println()

	_, _ = ui.Prompt("Press Enter once you've replaced the key on your platform...")

	if err := verifySSHConnectionWithKey(keyPath); err != nil {
		ui.Warn("SSH verification failed. Please check that you've added the new key correctly.")
		ui.Info(fmt.Sprintf("You can test manually with: ssh -i %s -o IdentitiesOnly=yes -T git@github.com", keyPath))
	} else {
		ui.Success("SSH connection verified with new key!")
	}

	if err := store.BindSSHKey(name, keyPath); err != nil {
		ui.Errorf("binding new SSH key: %v", err)
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("SSH key rotated successfully for %s", name))
	ui.Info("Old key backed up with .backup extension")
	return nil
}
