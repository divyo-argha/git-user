package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/divyo-argha/git-user/internal/keyring"

	"github.com/divyo-argha/git-user/internal/bundle"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runSync(args []string) error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	home, _ := os.UserHomeDir()
	syncDir := filepath.Join(home, ".git-users", "sync")

	// 1. Setup sync config if not already configured
	if store.Sync == nil || store.Sync.RepoURL == "" {
		ui.Banner("SETUP GIT-USER SYNC")
		fmt.Println("Keep your identities synchronized securely across all your devices using your own private Git repository.")
		fmt.Println()

		if !ui.Confirm("Configure sync now?", true) {
			ui.Info("Sync setup cancelled.")
			return nil
		}

		repoURL, err := ui.Prompt("Private Git Repository URL (SSH recommended):")
		if err != nil || repoURL == "" {
			ui.Error("Repository URL is required.")
			return fmt.Errorf("missing repository URL")
		}

		passphrase, err := readPassphrase("Passphrase to encrypt/decrypt sync bundle: ")
		if err != nil || passphrase == "" {
			ui.Error("Encryption passphrase is required.")
			return fmt.Errorf("missing passphrase")
		}

		confirm, err := readPassphrase("Confirm passphrase: ")
		if err != nil || passphrase != confirm {
			ui.Error("Passphrases do not match.")
			return fmt.Errorf("passphrase mismatch")
		}

		deviceName, _ := os.Hostname()
		if deviceName == "" {
			deviceName = "device"
		}

		deviceNameInput, _ := ui.Prompt(fmt.Sprintf("Device Name [%s]:", deviceName))
		if deviceNameInput != "" {
			deviceName = deviceNameInput
		}

		// Store passphrase in keyring
		if err := keyring.KeyringSet("git-user-sync", "sync", passphrase); err != nil {
			ui.Warn(fmt.Sprintf("Could not store passphrase in system keychain securely: %v. Storing in session only.", err))
		}

		store.Sync = &config.SyncConfig{
			RepoURL:    repoURL,
			DeviceName: deviceName,
		}
		if err := config.Save(store); err != nil {
			ui.Errorf("saving sync config: %v", err)
			return err
		}

		// Initialize sync directory
		_ = os.RemoveAll(syncDir)
		if err := os.MkdirAll(syncDir, 0700); err != nil {
			ui.Errorf("creating sync directory: %v", err)
			return err
		}

		// Run git init, remote add, etc.
		if err := runCmdsInDir(syncDir,
			[]string{"init"},
			[]string{"remote", "add", "origin", repoURL},
			[]string{"branch", "-M", "main"},
		); err != nil {
			ui.Errorf("initializing git sync repo: %v", err)
			return err
		}

		ui.Success("Sync configured successfully!")
	}

	ui.Banner("SYNCHRONIZING CONFIGURATION")
	fmt.Println()

	// Get passphrase
	passphrase, err := keyring.KeyringGet("git-user-sync", "sync")
	if err != nil || passphrase == "" {
		// Prompt if not in keychain
		passphrase, err = readPassphrase("Enter sync decryption passphrase: ")
		if err != nil || passphrase == "" {
			ui.Error("Passphrase required to perform sync.")
			return fmt.Errorf("missing passphrase")
		}
	}

	// Pull from remote
	ui.Info("Fetching latest updates from sync remote...")
	// We run pull, ignoring errors if remote branch doesn't exist yet (e.g. first sync)
	_ = runGitInDir(syncDir, "pull", "origin", "main")

	bundlePath := filepath.Join(syncDir, "backup.bundle")
	var remoteIdentities []bundle.Identity

	if _, err := os.Stat(bundlePath); err == nil {
		// Bundle exists, decrypt it
		bundleData, err := os.ReadFile(bundlePath)
		if err == nil {
			remoteIdentities, err = bundle.Decrypt(bundleData, passphrase)
			if err != nil {
				ui.Errorf("failed to decrypt remote bundle: %v. Please verify your passphrase.", err)
				return err
			}
		}
	}

	// Merge remote identities into local config
	mergedCount := 0
	for _, rid := range remoteIdentities {
		existing := store.FindUser(rid.Name)
		if existing == nil {
			// Profile doesn't exist locally, import it
			var keyPath string
			if len(rid.PrivateKey) > 0 {
				keyPath = filepath.Join(home, ".ssh", fmt.Sprintf("git_%s", rid.Name))
				_ = os.WriteFile(keyPath, rid.PrivateKey, 0600)
				if len(rid.PublicKey) > 0 {
					_ = os.WriteFile(keyPath+".pub", rid.PublicKey, 0644)
				}
			}

			_ = store.AddUser(rid.Name, rid.Email)
			if keyPath != "" {
				_ = store.BindSSHKey(rid.Name, keyPath)
			}
			mergedCount++
			ui.Successf("Imported new identity: %s (%s)", rid.Name, rid.Email)
		} else {
			// Profile exists. If local profile does not have SSH key but remote does, import remote key
			if existing.SSHKey == "" && len(rid.PrivateKey) > 0 {
				keyPath := filepath.Join(home, ".ssh", fmt.Sprintf("git_%s", rid.Name))
				_ = os.WriteFile(keyPath, rid.PrivateKey, 0600)
				if len(rid.PublicKey) > 0 {
					_ = os.WriteFile(keyPath+".pub", rid.PublicKey, 0644)
				}
				_ = store.BindSSHKey(rid.Name, keyPath)
				mergedCount++
				ui.Successf("Imported SSH key for existing identity: %s", rid.Name)
			}
		}
	}

	if mergedCount > 0 {
		if err := config.Save(store); err != nil {
			ui.Errorf("saving merged configuration: %v", err)
			return err
		}
	}

	// Create and write the new encrypted bundle of all permanent users
	var localIdentities []bundle.Identity
	for _, u := range store.Users {
		if !u.IsTemporary {
			id := bundle.Identity{Name: u.Name, Email: u.Email}
			if u.SSHKey != "" {
				privKey, err := os.ReadFile(u.SSHKey)
				if err == nil {
					id.PrivateKey = privKey
					id.PublicKey, _ = os.ReadFile(u.SSHKey + ".pub")
				}
			}
			localIdentities = append(localIdentities, id)
		}
	}

	encryptedData, err := bundle.Encrypt(localIdentities, passphrase)
	if err != nil {
		ui.Errorf("failed to encrypt backup bundle: %v", err)
		return err
	}

	if err := os.WriteFile(bundlePath, encryptedData, 0600); err != nil {
		ui.Errorf("failed to save backup bundle file: %v", err)
		return err
	}

	// Commit and push changes
	ui.Info("Pushing synchronized changes to remote...")
	_ = runGitInDir(syncDir, "add", "backup.bundle")
	commitMsg := fmt.Sprintf("Sync: %s at %s", store.Sync.DeviceName, time.Now().Format(time.RFC3339))
	_ = runGitInDir(syncDir, "commit", "-m", commitMsg)

	if err := runGitInDir(syncDir, "push", "origin", "main"); err != nil {
		ui.Warn(fmt.Sprintf("Could not push changes to remote (it might be offline): %v", err))
	} else {
		ui.Success("Successfully pushed synchronized configurations to remote!")
	}

	return nil
}

func runCmdsInDir(dir string, cmds ...[]string) error {
	for _, c := range cmds {
		cmd := exec.Command("git", c...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed running git %v: %w", c, err)
		}
	}
	return nil
}

func runGitInDir(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}
