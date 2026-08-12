package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/divyo-argha/git-user/internal/bundle"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/keyring"
)

// ── Sync ──────────────────────────────────────────────────────────────────────

// opSync synchronizes identities across devices using a private git repository.
// If repoURL is non-empty the sync is configured for the first time.
func opSync(store *config.Store, repoURL, passphrase string) (opResult, error) {
	home, _ := os.UserHomeDir()
	syncDir := filepath.Join(home, ".git-users", "sync")

	// 1. Setup sync config if not already configured.
	if repoURL != "" {
		deviceName, _ := os.Hostname()
		if deviceName == "" {
			deviceName = "device"
		}
		if err := keyring.KeyringSet("git-user-sync", "sync", passphrase); err != nil {
			_ = err
		}
		store.Sync = &config.SyncConfig{
			RepoURL:    repoURL,
			DeviceName: deviceName,
		}
		if err := config.Save(store); err != nil {
			return opResult{}, fmt.Errorf("saving sync config: %v", err)
		}
		_ = os.RemoveAll(syncDir)
		if err := os.MkdirAll(syncDir, 0700); err != nil {
			return opResult{}, fmt.Errorf("creating sync directory: %v", err)
		}
		if err := runCmdsInDir(syncDir,
			[]string{"init"},
			[]string{"config", "user.name", "git-user-sync"},
			[]string{"config", "user.email", "sync@git-user.local"},
			[]string{"config", "commit.gpgsign", "false"},
			[]string{"remote", "add", "origin", repoURL},
			[]string{"branch", "-M", "main"},
		); err != nil {
			return opResult{}, fmt.Errorf("initializing git sync repo: %v", err)
		}
	} else if store.Sync == nil || store.Sync.RepoURL == "" {
		return opResult{}, fmt.Errorf("sync is not configured — provide the private repository URL to configure it")
	}

	// 2. Resolve the passphrase.
	if passphrase == "" {
		passphrase, _ = keyring.KeyringGet("git-user-sync", "sync")
	}
	if passphrase == "" {
		return opResult{}, fmt.Errorf("passphrase required to sync")
	}

	// 3. Pull from remote — auto-recover if syncDir was deleted.
	if _, statErr := os.Stat(syncDir); os.IsNotExist(statErr) {
		if err := os.MkdirAll(filepath.Dir(syncDir), 0700); err != nil {
			return opResult{}, fmt.Errorf("creating parent directory: %v", err)
		}
		cloneCmd := exec.Command("git", "clone", store.Sync.RepoURL, syncDir)
		cloneCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if out, err := cloneCmd.CombinedOutput(); err != nil {
			return opResult{}, fmt.Errorf("re-cloning sync repo failed: %v\n%s", err, string(out))
		}
		_ = runCmdsInDir(syncDir,
			[]string{"config", "user.name", "git-user-sync"},
			[]string{"config", "user.email", "sync@git-user.local"},
			[]string{"config", "commit.gpgsign", "false"},
		)
	} else {
		_ = runGitInDir(syncDir, "pull", "origin", "main")
	}

	bundlePath := filepath.Join(syncDir, "backup.bundle")
	var remoteIdentities []bundle.Identity
	if data, err := os.ReadFile(bundlePath); err == nil {
		remoteIdentities, err = bundle.Decrypt(data, passphrase)
		if err != nil {
			return opResult{}, fmt.Errorf("failed to decrypt remote bundle: %v — check your passphrase", err)
		}
	}

	// 4. Merge remote identities into the local config.
	mergedCount := 0
	for _, rid := range remoteIdentities {
		existing := store.FindUser(rid.Name)
		if existing == nil {
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
		} else if existing.SSHKey == "" && len(rid.PrivateKey) > 0 {
			keyPath := filepath.Join(home, ".ssh", fmt.Sprintf("git_%s", rid.Name))
			_ = os.WriteFile(keyPath, rid.PrivateKey, 0600)
			if len(rid.PublicKey) > 0 {
				_ = os.WriteFile(keyPath+".pub", rid.PublicKey, 0644)
			}
			_ = store.BindSSHKey(rid.Name, keyPath)
			mergedCount++
		}
	}

	if mergedCount > 0 {
		if err := config.Save(store); err != nil {
			return opResult{}, fmt.Errorf("saving merged configuration: %v", err)
		}
	}

	// 5. Build and write the new encrypted bundle of all permanent users.
	var localIdentities []bundle.Identity
	for _, u := range store.Users {
		if u.IsTemporary {
			continue
		}
		id := bundle.Identity{Name: u.Name, Email: u.Email}
		if u.SSHKey != "" {
			if priv, err := os.ReadFile(u.SSHKey); err == nil {
				id.PrivateKey = priv
				id.PublicKey, _ = os.ReadFile(u.SSHKey + ".pub")
			}
		}
		localIdentities = append(localIdentities, id)
	}

	encrypted, err := bundle.Encrypt(localIdentities, passphrase)
	if err != nil {
		return opResult{}, fmt.Errorf("failed to encrypt backup bundle: %v", err)
	}
	if err := os.WriteFile(bundlePath, encrypted, 0600); err != nil {
		return opResult{}, fmt.Errorf("failed to save backup bundle file: %v", err)
	}

	// 6. Commit and push.
	_ = runGitInDir(syncDir, "add", "backup.bundle")
	commitMsg := fmt.Sprintf("Sync: %s at %s", store.Sync.DeviceName, time.Now().Format(time.RFC3339))
	_ = runGitInDir(syncDir, "commit", "-m", commitMsg)

	report := fmt.Sprintf("Synced %d identities to %s\n", len(localIdentities), store.Sync.DeviceName)
	if mergedCount > 0 {
		report += fmt.Sprintf("Imported %d new identity/identities from remote\n", mergedCount)
	}
	if err := runGitInDir(syncDir, "push", "origin", "main"); err != nil {
		report += "⚠ Could not push changes to remote (it might be offline)\n"
	} else {
		report += "Pushed synchronized configurations to remote\n"
	}
	return opResult{detail: report, showReport: true}, nil
}

func runCmdsInDir(dir string, cmds ...[]string) error {
	for _, c := range cmds {
		if out, err := runCaptured(dir, "git", c...); err != nil {
			return fmt.Errorf("failed running git %v: %w\n%s", c, err, strings.TrimSpace(out))
		}
	}
	return nil
}

func runGitInDir(dir string, args ...string) error {
	_, err := runCaptured(dir, "git", args...)
	return err
}
