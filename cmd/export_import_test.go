package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
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
	_ = store.AddUser("alice", "alice@example.com")
	_ = config.Save(store)

	// User not found
	err = runExport([]string{"bob"})
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
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.BindSSHKey("alice", keyPath)
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

	// Make sure config path is still configured correctly
	configFilePath := filepath.Join(tmpDir, ".git-users", "config.json")
	config.SetConfigPath(configFilePath)

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
	user := store.FindUser("alice")
	if user == nil {
		t.Fatal("user alice was not imported")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected email to be alice@example.com, got %s", user.Email)
	}

	// Verify imported SSH keys exist
	expectedKeyPath := filepath.Join(tmpDir, ".ssh", "git_alice")
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

func TestExportSkipsTemp(t *testing.T) {
	tmpDir := setupTestEnv(t)
	config.SetConfigPath(filepath.Join(tmpDir, "config.json"))

	store, _ := config.Load()
	_ = store.AddUser("perm", "perm@example.com")
	_ = store.AddUser("temp", "temp@example.com")
	
	u := store.FindUser("temp")
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
	if importedStore.FindUser("temp") != nil {
		t.Errorf("temporary profile was exported and imported")
	}
	if importedStore.FindUser("perm") == nil {
		t.Errorf("permanent profile was not imported")
	}
}
