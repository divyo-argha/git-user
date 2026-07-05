package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func TestRunSync_SetupAndSync(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Create a mock remote git repo to serve as the backup target
	remoteRepoDir := filepath.Join(tmpDir, "remote-backup-repo")
	err := os.Mkdir(remoteRepoDir, 0755)
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Initialize git repo in the remote directory
	runGitCmd(t, remoteRepoDir, "init", "--bare")

	// Set up interactive mock prompts
	ui.PromptFn = func(label string) (string, error) {
		if label == "Private Git Repository URL (SSH recommended):" {
			return remoteRepoDir, nil
		}
		if label == "Device Name [device]:" {
			return "test-device", nil
		}
		return "", nil
	}

	readPassphraseFn = func(prompt string) (string, error) {
		return "secretpass", nil
	}

	// Set up initial user profile
	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = config.Save(store)

	// Run sync (triggers setup workflow and initial backup creation + push)
	err = runSync([]string{})
	if err != nil {
		t.Fatalf("unexpected error during initial sync setup: %v", err)
	}

	// Verify sync configurations in store
	store, _ = config.Load()
	if store.Sync == nil || store.Sync.RepoURL != remoteRepoDir {
		t.Fatalf("sync config mismatch: %v", store.Sync)
	}

	// Verify the backup.bundle exists on sync directory
	home, _ := os.UserHomeDir()
	bundlePath := filepath.Join(home, ".git-users", "sync", "backup.bundle")
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		t.Fatal("expected backup.bundle to exist in sync directory")
	}

	// Modify keychain settings to retrieve passphrase
	keyringGet = func(service, user string) (string, error) {
		return "secretpass", nil
	}

	// Now simulate another device syncing from the same remote repo!
	// We create a new clean local environment targeting the same remote
	tmpDir2 := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir2)
	configFilePath2 := filepath.Join(tmpDir2, ".git-users", "config.json")
	config.SetConfigPath(configFilePath2)

	// Clean sync dir for second environment
	syncDir2 := filepath.Join(tmpDir2, ".git-users", "sync")
	_ = os.MkdirAll(syncDir2, 0700)
	runGitCmd(t, syncDir2, "init")
	runGitCmd(t, syncDir2, "remote", "add", "origin", remoteRepoDir)
	runGitCmd(t, syncDir2, "branch", "-M", "main")

	// Setup sync configs for second environment
	store2, _ := config.Load()
	store2.Sync = &config.SyncConfig{
		RepoURL:    remoteRepoDir,
		DeviceName: "device2",
	}
	_ = config.Save(store2)

	// Run sync on the second device (should fetch backup.bundle and import alice)
	err = runSync([]string{})
	if err != nil {
		t.Fatalf("unexpected error during sync on second device: %v", err)
	}

	// Verify alice is imported successfully
	store2, _ = config.Load()
	alice := store2.FindUser("alice")
	if alice == nil || alice.Email != "alice@example.com" {
		t.Fatal("failed to import alice profile on second device sync")
	}

	// Restore original home for cleanup
	os.Setenv("HOME", oldHome)
}
