package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func TestRunBind_Errors(t *testing.T) {
	setupTestEnv(t)

	// Missing args
	err := runBind([]string{})
	if err == nil {
		t.Fatal("expected error with no arguments, got nil")
	}

	// User not found
	err = runBind([]string{"alice"})
	if err == nil {
		t.Fatal("expected error with nonexistent user, got nil")
	}

	// Register user
	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = config.Save(store)

	// SSH key file does not exist
	err = runBind([]string{"alice", "--ssh-key", "/nonexistent/key"})
	if err == nil {
		t.Fatal("expected error with nonexistent key file, got nil")
	}
}

func TestRunBind_Success(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Pre-create user and a dummy key
	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	keyPath := filepath.Join(tmpDir, "dummy_id")
	_ = os.WriteFile(keyPath, []byte("private key"), 0600)

	// Run bind on active user
	err := runBind([]string{"alice", "--ssh-key", keyPath})
	if err != nil {
		t.Fatalf("unexpected error binding key: %v", err)
	}

	// Verify key was bound in config
	store, _ = config.Load()
	user := store.FindUser("alice")
	if user.SSHKey != keyPath {
		t.Errorf("expected bound SSH key to be %s, got %s", keyPath, user.SSHKey)
	}

	// Verify git sshCommand was updated
	expectedSSHCmd := "ssh -i \"" + keyPath + "\" -o IdentitiesOnly=yes"
	if git.CurrentSSHCommand() != expectedSSHCmd {
		t.Errorf("expected git core.sshCommand to be %q, got %q", expectedSSHCmd, git.CurrentSSHCommand())
	}

	// Verify signing was enabled by default
	if user.SignKey != keyPath || user.SignFormat != "ssh" || user.SignDisabled {
		t.Errorf("expected commit signing to be enabled with ssh key, got %v", user)
	}
}

func TestRunBind_NoSign(t *testing.T) {
	tmpDir := setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("bob", "bob@example.com")
	_ = config.Save(store)

	keyPath := filepath.Join(tmpDir, "dummy_id2")
	_ = os.WriteFile(keyPath, []byte("private key"), 0600)

	err := runBind([]string{"bob", "--ssh-key", keyPath, "--no-sign"})
	if err != nil {
		t.Fatalf("unexpected error binding key: %v", err)
	}

	store, _ = config.Load()
	user := store.FindUser("bob")
	if user.SSHKey != keyPath {
		t.Errorf("expected bound SSH key to be %s", keyPath)
	}
	if !user.SignDisabled {
		t.Errorf("expected signing to be disabled, got %v", user)
	}
}

func TestRunBind_Interactive(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Pre-create user and key
	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = config.Save(store)

	keyPath := filepath.Join(tmpDir, "dummy_id")
	_ = os.WriteFile(keyPath, []byte("private key"), 0600)

	// Scenario 1: Select Cancel (index 2)
	ui.SelectFn = func(label string, options []string) (int, error) {
		return 2, nil
	}
	err := runBind([]string{"alice"})
	if err != nil {
		t.Fatalf("unexpected error on interactive cancel: %v", err)
	}
	store, _ = config.Load()
	if store.FindUser("alice").SSHKey != "" {
		t.Fatal("expected user to have no SSH key bound")
	}

	// Scenario 2: Select Use existing key (index 1) with valid path
	ui.SelectFn = func(label string, options []string) (int, error) {
		return 1, nil
	}
	ui.PromptFn = func(label string) (string, error) {
		return keyPath, nil
	}
	err = runBind([]string{"alice"})
	if err != nil {
		t.Fatalf("unexpected error on interactive bind: %v", err)
	}

	store, _ = config.Load()
	if store.FindUser("alice").SSHKey != keyPath {
		t.Errorf("expected SSH key path to be bound to %s", keyPath)
	}
}
