package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ssh"
	"github.com/divyo-argha/git-user/internal/ui"
	"github.com/divyo-argha/git-user/internal/validate"
)

func runRekey(args []string) error {
	var name string
	var force bool
	for _, arg := range args {
		if arg == "--force" || arg == "-f" || arg == "--yes" || arg == "-y" {
			force = true
		} else if !strings.HasPrefix(arg, "-") {
			name = arg
		}
	}

	if name == "" {
		ui.Error("usage: git-user rekey <name> [--force]")
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

	if !force {
		ui.Warn(fmt.Sprintf("⚠️ WARNING: Rotating SSH key for identity %q will generate a new key pair.", user.Name))
		ui.Warn("Your current SSH key will be backed up, but it will NO LONGER WORK until updated on platforms.")
		ui.Warn("You MUST update your public key on GitHub/GitLab/Bitbucket after rotation.")
		if !ui.Confirm(fmt.Sprintf("Are you sure you want to rotate the SSH key for %q?", user.Name), false) {
			ui.Info("SSH key rotation cancelled.")
			return nil
		}
	}

	ui.Info(fmt.Sprintf("Rotating SSH key for identity: %s (%s)", user.Name, user.Email))

	// Rotate whichever key is actually bound to the identity, not
	// unconditionally git_<name> — an identity bound to a custom-named key
	// (via "use existing key") used to have rekey silently regenerate an
	// unrelated git_<name> file instead of the key actually in use.
	oldKeyPath := user.SSHKey
	if oldKeyPath == "" {
		var err error
		oldKeyPath, err = config.DefaultSSHKeyPath(name)
		if err != nil {
			ui.Errorf("%v", err)
			return err
		}
	}

	newKeyPath := oldKeyPath
	suggestion := filepath.Base(oldKeyPath)
	input, promptErr := ui.Prompt(fmt.Sprintf("New key filename [%s]:", suggestion))
	if promptErr == nil {
		filename := strings.TrimSpace(input)
		if filename != "" && filename != suggestion {
			if err := validate.SSHKeyFilename(filename); err != nil {
				ui.Warn(fmt.Sprintf("%v — keeping %q", err, suggestion))
			} else if candidate, cerr := config.SSHKeyPathForFilename(filename); cerr != nil {
				ui.Warn(fmt.Sprintf("%v — keeping %q", cerr, suggestion))
			} else if _, statErr := os.Stat(candidate); statErr == nil {
				ui.Warn(fmt.Sprintf("a key already exists at %s — keeping %q", candidate, suggestion))
			} else {
				newKeyPath = candidate
			}
		}
	} else if !errors.Is(promptErr, ui.ErrNotInteractive) {
		return promptErr
	}

	sshDir := filepath.Dir(newKeyPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		ui.Errorf("creating .ssh directory: %v", err)
		return err
	}
	if newKeyPath != oldKeyPath {
		if _, err := os.Stat(newKeyPath); err == nil {
			ui.Errorf("a key already exists at %s", newKeyPath)
			return fmt.Errorf("key already exists")
		}
	}

	backupPath := oldKeyPath + ".backup"
	hasOldKey := false
	if _, err := os.Stat(oldKeyPath); err == nil {
		hasOldKey = true
		// Unload the old key from the agent before it's rotated out from under
		// it, so a stale/orphaned identity doesn't linger there indefinitely.
		if ssh.IsSSHKeyLoaded(oldKeyPath) {
			_ = ssh.RemoveSSHKey(oldKeyPath)
		}
		ui.Warn(fmt.Sprintf("Backing up existing key to %s", backupPath))
		if err := os.Rename(oldKeyPath, backupPath); err != nil {
			ui.Errorf("backing up key: %v", err)
			return err
		}
		pubKeyPath := oldKeyPath + ".pub"
		if _, err := os.Stat(pubKeyPath); err == nil {
			os.Rename(pubKeyPath, backupPath+".pub")
		}
	}

	ui.Info(fmt.Sprintf("Generating new SSH key at %s...", newKeyPath))
	ui.Info("You will be prompted to set a passphrase for the key.")
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", user.Email, "-f", newKeyPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if hasOldKey {
			os.Rename(backupPath, oldKeyPath)
			os.Rename(backupPath+".pub", oldKeyPath+".pub")
			ui.Warn("Restored old key — nothing changed")
		}
		ui.Errorf("generating SSH key: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("New SSH key created at %s", newKeyPath))
	checkAndPromptPassphrase(name, newKeyPath)

	pubKeyPath := newKeyPath + ".pub"
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

	if err := verifySSHConnectionWithKey(newKeyPath); err != nil {
		ui.Warn("SSH verification failed. Please check that you've added the new key correctly.")
		ui.Info(fmt.Sprintf("You can test manually with: ssh -i %s -o IdentitiesOnly=yes -T git@github.com", newKeyPath))
	} else {
		ui.Success("SSH connection verified with new key!")
	}

	if err := store.BindSSHKey(name, newKeyPath); err != nil {
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
