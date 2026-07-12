package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func TestRunRemove_MissingArgs(t *testing.T) {
	setupTestEnv(t)
	err := runRemove([]string{})
	if err == nil {
		t.Fatal("expected error with no arguments, got nil")
	}
}

func TestRunRemove_UserNotFound(t *testing.T) {
	setupTestEnv(t)
	err := runRemove([]string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error with nonexistent user, got nil")
	}
}

func TestRunRemove_InactiveUser(t *testing.T) {
	setupTestEnv(t)

	// Add users
	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.AddUser("bob", "bob@example.com")
	_ = store.SetCurrent("bob")
	_ = config.Save(store)

	// Remove inactive user alice
	err := runRemove([]string{"alice"})
	if err != nil {
		t.Fatalf("unexpected error removing inactive user: %v", err)
	}

	// Verify alice is gone, bob is still current
	store, _ = config.Load()
	if store.FindUser("alice") != nil {
		t.Fatal("alice should be removed")
	}
	if store.Current != "bob" {
		t.Errorf("current user should be bob, got %s", store.Current)
	}
}

func TestRunRemove_ActiveUserWithoutForce(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	// Attempt to remove active user without force
	err := runRemove([]string{"alice"})
	if err == nil {
		t.Fatal("expected error removing active user without force, got nil")
	}

	// Verify user is not removed
	store, _ = config.Load()
	if store.FindUser("alice") == nil {
		t.Fatal("alice should still exist")
	}
}

func TestRunRemove_ActiveUserWithForce(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	// Remove active user with force
	err := runRemove([]string{"alice", "--force"})
	if err != nil {
		t.Fatalf("unexpected error removing active user with force: %v", err)
	}

	store, _ = config.Load()
	if store.FindUser("alice") != nil {
		t.Fatal("alice should be removed")
	}
	if store.Current != "" {
		t.Errorf("current should be empty, got %s", store.Current)
	}
}

func TestRunRemove_DeleteSSHKeys(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Create dummy SSH keys
	keyPath := filepath.Join(tmpDir, "test_id")
	pubPath := keyPath + ".pub"
	_ = os.WriteFile(keyPath, []byte("private key"), 0600)
	_ = os.WriteFile(pubPath, []byte("public key"), 0644)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.BindSSHKey("alice", keyPath)
	_ = config.Save(store)

	// Test Case 1: Say 'No' to key deletion
	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		return false
	}
	err := runRemove([]string{"alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify files still exist
	if _, err := os.Stat(keyPath); err != nil {
		t.Error("private key file should still exist")
	}
	if _, err := os.Stat(pubPath); err != nil {
		t.Error("public key file should still exist")
	}

	// Re-add and bind key for Case 2
	store, _ = config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.BindSSHKey("alice", keyPath)
	_ = config.Save(store)

	// Test Case 2: Say 'Yes' to key deletion
	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		return true
	}
	err = runRemove([]string{"alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify files are deleted
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Error("private key file should have been deleted")
	}
	if _, err := os.Stat(pubPath); !os.IsNotExist(err) {
		t.Error("public key file should have been deleted")
	}
}
