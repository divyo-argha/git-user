package cli

import (
	"github.com/divyo-argha/git-user/internal/keyring"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func TestRunSync_SetupAndSync(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Configure git user.name and user.email globally for the test environment to allow commits during sync
	_ = exec.Command("git", "config", "--global", "user.name", "Test User").Run()
	_ = exec.Command("git", "config", "--global", "user.email", "test@example.com").Run()

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
	_ = store.AddUser("dev", "dev@example.com")
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
	private, _ := os.UserHomeDir()
	bundlePath := filepath.Join(private, ".git-users", "sync", "backup.bundle")
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		t.Fatal("expected backup.bundle to exist in sync directory")
	}

	// Modify keychain settings to retrieve passphrase
	keyring.KeyringGet = func(service, user string) (string, error) {
		return "secretpass", nil
	}

	// Now simulate another device syncing from the same remote repo!
	// We create a new clean local environment targeting the same remote
	tmpDir2 := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir2)
	_ = exec.Command("git", "config", "--global", "--add", "safe.directory", "*").Run()
	_ = exec.Command("git", "config", "--global", "user.name", "Test User").Run()
	_ = exec.Command("git", "config", "--global", "user.email", "test@example.com").Run()
	configFilePath2 := filepath.Join(tmpDir2, ".git-users", "config.json")
	t.Setenv("GIT_USER_CONFIG", configFilePath2)

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

	// Run sync on the second device (should fetch backup.bundle and import dev)
	err = runSync([]string{})
	if err != nil {
		t.Fatalf("unexpected error during sync on second device: %v", err)
	}

	// Verify dev is imported successfully
	store2, _ = config.Load()
	dev := store2.FindUser("dev")
	if dev == nil || dev.Email != "dev@example.com" {
		t.Fatal("failed to import dev profile on second device sync")
	}

	// The second device had no active identity at all before this sync —
	// the freshly-synced "dev" identity should have been activated instead
	// of sitting inert until a separate manual switch.
	if store2.Current != "dev" {
		t.Errorf("expected \"dev\" to be activated on a device with no prior active identity, got current=%q", store2.Current)
	}

	// Restore original private for cleanup
	os.Setenv("HOME", oldHome)
}

// TestRunSync_AppliesKeyToActiveIdentity guards against a regression where
// sync could merge a freshly-synced SSH key into an existing, keyless, and
// *currently active* local identity, updating the config store but never
// re-pointing core.sshCommand at the new key — leaving the active identity
// looking fully configured while git still couldn't use it.
func TestRunSync_AppliesKeyToActiveIdentity(t *testing.T) {
	tmpDir := setupTestEnv(t)

	_ = exec.Command("git", "config", "--global", "user.name", "Test User").Run()
	_ = exec.Command("git", "config", "--global", "user.email", "test@example.com").Run()

	remoteRepoDir := filepath.Join(tmpDir, "remote-backup-repo")
	if err := os.Mkdir(remoteRepoDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	runGitCmd(t, remoteRepoDir, "init", "--bare")

	// First device: "dev" has a real SSH key and pushes a backup containing it.
	sshDir := filepath.Join(tmpDir, ".ssh")
	_ = os.MkdirAll(sshDir, 0700)
	keyPath := filepath.Join(sshDir, "id_ed25519")
	_ = os.WriteFile(keyPath, []byte("private key data"), 0600)
	_ = os.WriteFile(keyPath+".pub", []byte("public key data"), 0644)

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

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.BindSSHKey("dev", keyPath)
	_ = config.Save(store)

	if err := runSync([]string{}); err != nil {
		t.Fatalf("unexpected error during initial sync setup: %v", err)
	}

	keyring.KeyringGet = func(service, user string) (string, error) {
		return "secretpass", nil
	}

	// Second device: "dev" already exists locally (e.g. imported by name
	// earlier) but has no SSH key yet, and it's already the active identity.
	tmpDir2 := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir2)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })
	_ = exec.Command("git", "config", "--global", "--add", "safe.directory", "*").Run()
	_ = exec.Command("git", "config", "--global", "user.name", "Test User").Run()
	_ = exec.Command("git", "config", "--global", "user.email", "test@example.com").Run()
	configFilePath2 := filepath.Join(tmpDir2, ".git-users", "config.json")
	t.Setenv("GIT_USER_CONFIG", configFilePath2)

	syncDir2 := filepath.Join(tmpDir2, ".git-users", "sync")
	_ = os.MkdirAll(syncDir2, 0700)
	runGitCmd(t, syncDir2, "init")
	runGitCmd(t, syncDir2, "remote", "add", "origin", remoteRepoDir)
	runGitCmd(t, syncDir2, "branch", "-M", "main")

	store2, _ := config.Load()
	_ = store2.AddUser("dev", "dev@example.com") // keyless, matches by name
	_ = store2.SetCurrent("dev")
	store2.Sync = &config.SyncConfig{RepoURL: remoteRepoDir, DeviceName: "device2"}
	_ = config.Save(store2)
	if err := git.Apply("dev", "dev@example.com"); err != nil {
		t.Fatal(err)
	}

	if err := runSync([]string{}); err != nil {
		t.Fatalf("unexpected error during sync on second device: %v", err)
	}

	store2, _ = config.Load()
	dev2 := store2.FindUser("dev")
	if dev2 == nil || dev2.SSHKey == "" {
		t.Fatal("expected \"dev\" to have received the synced SSH key")
	}
	expectedSSHCommand := "ssh -i '" + dev2.SSHKey + "' -o IdentitiesOnly=yes"
	if git.CurrentSSHCommand() != expectedSSHCommand {
		t.Errorf("expected core.sshCommand to point at the newly synced key for the active identity, got %q, want %q", git.CurrentSSHCommand(), expectedSSHCommand)
	}
}
