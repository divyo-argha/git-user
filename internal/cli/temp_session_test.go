package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestTemporarySessionCleanup(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Switch with temporary flag
	err := runSwitch([]string{"-c", "temp-user", "temp@example.com", "--temp"})
	if err != nil {
		t.Fatalf("failed to quick switch temp: %v", err)
	}

	// Load store to verify temporary user exists
	store, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	user := store.FindUser("temp-user")
	if user == nil {
		t.Fatalf("temp-user not found in store")
	}
	if !user.IsTemporary {
		t.Errorf("expected user to be marked temporary")
	}

	// Verify key files exist
	keyPath := filepath.Join(tmpDir, ".ssh", "git_temp-user")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Errorf("expected private key file to exist at %s", keyPath)
	}
	if _, err := os.Stat(keyPath + ".pub"); os.IsNotExist(err) {
		t.Errorf("expected public key file to exist at %s.pub", keyPath)
	}

	// Run logout
	err = runLogout([]string{})
	if err != nil {
		t.Fatalf("runLogout failed: %v", err)
	}

	// Reload store and check
	store, _ = config.Load()
	if store.FindUser("temp-user") != nil {
		t.Errorf("expected temp-user to be deleted from store")
	}

	// Verify key files are deleted
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Errorf("expected private key file to be deleted, but it still exists")
	}
	if _, err := os.Stat(keyPath + ".pub"); !os.IsNotExist(err) {
		t.Errorf("expected public key file to be deleted, but it still exists")
	}
}
