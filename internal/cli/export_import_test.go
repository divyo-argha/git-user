package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

func TestRunExport_Errors(t *testing.T) {
	setupTestEnv(t)

	// Missing args
	err := runExport([]string{})
	if err == nil {
		t.Fatal("expected error with no arguments, got nil")
	}

	// No users registered
	err = runExport([]string{"--all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Register a user
	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = config.Save(store)

	// User not found
	err = runExport([]string{"ops"})
	if err == nil {
		t.Fatal("expected error for nonexistent user, got nil")
	}

	// Passphrase mismatch
	readPassphraseFn = func(prompt string) (string, error) {
		if strings.Contains(prompt, "Confirm") {
			return "wrong-password", nil
		}
		return "secret123", nil
	}
	err = runExport([]string{"--all"})
	if err == nil {
		t.Fatal("expected error for passphrase mismatch, got nil")
	}

	// Empty passphrase
	readPassphraseFn = func(prompt string) (string, error) {
		return "", nil
	}
	err = runExport([]string{"--all"})
	if err == nil {
		t.Fatal("expected error for empty passphrase, got nil")
	}
}

func TestRunExportAndImport_Success(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Pre-create some SSH keys
	sshDir := filepath.Join(tmpDir, ".ssh")
	_ = os.MkdirAll(sshDir, 0700)
	keyPath := filepath.Join(sshDir, "id_ed25519")
	_ = os.WriteFile(keyPath, []byte("private key data"), 0600)
	_ = os.WriteFile(keyPath+".pub", []byte("public key data"), 0644)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.BindSSHKey("dev", keyPath)
	_ = config.Save(store)

	// Mock passphrase entry
	readPassphraseFn = func(prompt string) (string, error) {
		return "testpassword123", nil
	}

	err := runExport([]string{"--all"})
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	// Check if file is written to HOME (which is tmpDir)
	bundleName := "git-user-export-" + time.Now().Format("2006-01-02") + ".bundle"
	bundlePath := filepath.Join(tmpDir, bundleName)

	if _, err := os.Stat(bundlePath); err != nil {
		t.Fatalf("export bundle file not found: %s", bundlePath)
	}

	// Clean up config and SSH directory to simulate importing on a fresh environment
	os.RemoveAll(filepath.Join(tmpDir, ".git-users"))
	os.RemoveAll(sshDir)

	// Run import - missing bundle file
	err = runImport([]string{})
	if err == nil {
		t.Fatal("expected error with no arguments, got nil")
	}

	// Run import - non-existing file
	err = runImport([]string{filepath.Join(tmpDir, "nonexistent.bundle")})
	if err == nil {
		t.Fatal("expected error with nonexistent bundle file, got nil")
	}

	// Mock incorrect passphrase on import
	readPassphraseFn = func(prompt string) (string, error) {
		return "wrong-password", nil
	}
	err = runImport([]string{bundlePath})
	if err == nil {
		t.Fatal("expected decryption error with wrong passphrase, got nil")
	}

	// Run import successfully
	readPassphraseFn = func(prompt string) (string, error) {
		return "testpassword123", nil
	}
	err = runImport([]string{bundlePath})
	if err != nil {
		t.Fatalf("unexpected import error: %v", err)
	}

	// Verify imported config
	store, err = config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	user := store.FindUser("dev")
	if user == nil {
		t.Fatal("user dev was not imported")
	}
	if user.Email != "dev@example.com" {
		t.Errorf("expected email to be dev@example.com, got %s", user.Email)
	}

	// Verify imported SSH keys exist
	expectedKeyPath := filepath.Join(tmpDir, ".ssh", "git_dev")
	if _, err := os.Stat(expectedKeyPath); err != nil {
		t.Errorf("imported private key file does not exist: %s", expectedKeyPath)
	}
	if _, err := os.Stat(expectedKeyPath + ".pub"); err != nil {
		t.Errorf("imported public key file does not exist: %s", expectedKeyPath+".pub")
	}

	if user.SSHKey != expectedKeyPath {
		t.Errorf("expected bound SSH key to be %s, got %s", expectedKeyPath, user.SSHKey)
	}

	// Try importing again (should skip importing user but not crash/fail)
	err = runImport([]string{bundlePath})
	if err != nil {
		t.Fatalf("unexpected error re-importing: %v", err)
	}
}

// TestImportForceOverwriteRestoresActiveIdentity guards against a regression
// where re-importing (with --force) a bundle containing the identity that is
// currently active would remove-then-recreate it, but leave store.Current
// cleared and the live git config untouched — silently deactivating the
// user's own active identity via what looks like a routine backup restore.
func TestImportForceOverwriteRestoresActiveIdentity(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	tmpDir := setupTestEnv(t)

	sshDir := filepath.Join(tmpDir, ".ssh")
	_ = os.MkdirAll(sshDir, 0700)
	keyPath := filepath.Join(sshDir, "id_work")
	_ = os.WriteFile(keyPath, []byte("private key data"), 0600)
	_ = os.WriteFile(keyPath+".pub", []byte("public key data"), 0644)

	store, _ := config.Load()
	_ = store.AddUser("work", "work@example.com")
	_ = store.BindSSHKey("work", keyPath)
	_ = store.SetCurrent("work")
	_ = config.Save(store)
	if err := git.Apply("work", "work@example.com"); err != nil {
		t.Fatal(err)
	}

	readPassphraseFn = func(prompt string) (string, error) {
		return "testpassword123", nil
	}
	if err := runExport([]string{"work"}); err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}
	bundleName := "git-user-export-" + time.Now().Format("2006-01-02") + ".bundle"
	bundlePath := filepath.Join(tmpDir, bundleName)

	// Re-import the same bundle over the still-present, still-active "work"
	// identity — this forces the name-conflict overwrite path.
	if err := runImport([]string{"--force", bundlePath}); err != nil {
		t.Fatalf("unexpected import error: %v", err)
	}

	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if reloaded.Current != "work" {
		t.Errorf("expected active identity to remain %q after a force re-import, got %q", "work", reloaded.Current)
	}
	if git.CurrentName() != "work" || git.CurrentEmail() != "work@example.com" {
		t.Errorf("expected live git config to still reflect \"work\" after a force re-import, got %s <%s>", git.CurrentName(), git.CurrentEmail())
	}
}

func TestExportSkipsTemp(t *testing.T) {
	tmpDir := setupTestEnv(t)
	t.Setenv("GIT_USER_CONFIG", filepath.Join(tmpDir, "config.json"))

	store, _ := config.Load()
	_ = store.AddUser("perm", "perm@example.com")
	_ = store.AddUser("guest", "guest@example.com")

	u := store.FindUser("guest")
	u.IsTemporary = true
	_ = config.Save(store)

	readPassphraseFn = func(prompt string) (string, error) {
		return "testpassword123", nil
	}

	err := runExport([]string{"--all"})
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	bundleName := "git-user-export-" + time.Now().Format("2006-01-02") + ".bundle"
	bundlePath := filepath.Join(tmpDir, bundleName)

	if _, err := os.Stat(bundlePath); err != nil {
		t.Fatalf("export bundle file not found: %s", bundlePath)
	}

	// Read bundle and verify
	os.RemoveAll(filepath.Join(tmpDir, "config.json")) // ensure we import to blank
	config.DeleteTempConfig()

	readPassphraseFn = func(prompt string) (string, error) {
		return "testpassword123", nil
	}
	err = runImport([]string{bundlePath})
	if err != nil {
		t.Fatalf("unexpected import error: %v", err)
	}

	importedStore, _ := config.Load()
	if importedStore.FindUser("guest") != nil {
		t.Errorf("temporary profile was exported and imported")
	}
	if importedStore.FindUser("perm") == nil {
		t.Errorf("permanent profile was not imported")
	}
}
