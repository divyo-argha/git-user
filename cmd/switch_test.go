package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func TestRunSwitch_MissingArgs(t *testing.T) {
	setupTestEnv(t)
	err := runSwitch([]string{})
	if err == nil {
		t.Fatal("expected error with no arguments, got nil")
	}
}

func TestRunSwitch_UserNotFound(t *testing.T) {
	setupTestEnv(t)
	err := runSwitch([]string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error switching to nonexistent user, got nil")
	}
}

func TestRunSwitch_CreateAndSwitch(t *testing.T) {
	_ = setupTestEnv(t)

	// Mock UI inputs:
	// Email is passed as argument, so it shouldn't ask for email.
	// UI Select will ask "Choose SSH key setup:" -> Select Skip (index 2)
	ui.SelectFn = func(label string, options []string) (int, error) {
		return 2, nil // Skip SSH
	}

	err := runSwitch([]string{"-c", "alice", "alice@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify config
	store, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if store.Current != "alice" {
		t.Errorf("expected current to be alice, got %q", store.Current)
	}

	user := store.FindUser("alice")
	if user == nil {
		t.Fatal("user alice was not created in config")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected email to be alice@example.com, got %q", user.Email)
	}

	// Verify git config
	if got := git.CurrentName(); got != "alice" {
		t.Errorf("expected git user.name to be alice, got %q", got)
	}
	if got := git.CurrentEmail(); got != "alice@example.com" {
		t.Errorf("expected git user.email to be alice@example.com, got %q", got)
	}

	// Verify that switch again works without -c
	err = runSwitch([]string{"alice"})
	if err != nil {
		t.Fatalf("unexpected error switching to existing user: %v", err)
	}

	// Let's create another user and switch to it
	ui.SelectFn = func(label string, options []string) (int, error) {
		return 2, nil // Skip
	}
	err = runSwitch([]string{"-c", "bob", "bob@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if git.CurrentName() != "bob" {
		t.Errorf("expected current git user to be bob, got %s", git.CurrentName())
	}

	// Switch back to alice
	err = runSwitch([]string{"alice"})
	if err != nil {
		t.Fatalf("unexpected error switching back to alice: %v", err)
	}
	if git.CurrentName() != "alice" {
		t.Errorf("expected current git user to be alice, got %s", git.CurrentName())
	}

	// Verify original snapshot is saved and can be restored
	err = runSwitch([]string{"--original"})
	if err != nil {
		t.Fatalf("unexpected error switching to original: %v", err)
	}

	// Since we started with empty git config in tmp directory,
	// user.name and user.email should now be empty (restored).
	if got := git.CurrentName(); got != "" {
		t.Errorf("expected restored git user.name to be empty, got %q", got)
	}
	if got := git.CurrentEmail(); got != "" {
		t.Errorf("expected restored git user.email to be empty, got %q", got)
	}

	store, _ = config.Load()
	if store.Current != "" {
		t.Errorf("expected current identity to be empty after restoring original, got %q", store.Current)
	}
}

func TestRunSwitch_CreateAndSwitch_WithSSHKeyGen(t *testing.T) {
	_ = setupTestEnv(t)

	// Mock UI inputs:
	// Select index 0: Auto-generate SSH key
	ui.SelectFn = func(label string, options []string) (int, error) {
		return 0, nil
	}

	// Let's run switch
	err := runSwitch([]string{"-c", "charlie", "charlie@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify key was generated
	store, _ := config.Load()
	user := store.FindUser("charlie")
	if user == nil {
		t.Fatal("user charlie not found")
	}

	if user.SSHKey == "" {
		t.Error("expected SSH key path to be set, got empty")
	}

	if _, err := os.Stat(user.SSHKey); err != nil {
		t.Errorf("SSH key file does not exist at path: %s", user.SSHKey)
	}

	// Verify public key also exists
	pubKeyPath := user.SSHKey + ".pub"
	if _, err := os.Stat(pubKeyPath); err != nil {
		t.Errorf("SSH public key file does not exist: %s", pubKeyPath)
	}
}

func TestRunSwitch_ExistingKeyBind(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Pre-create a dummy ssh key
	dummyKeyPath := filepath.Join(tmpDir, "dummy_id")
	err := os.WriteFile(dummyKeyPath, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\n...\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Mock UI inputs:
	// Select index 1: Use existing key
	ui.SelectFn = func(label string, options []string) (int, error) {
		return 1, nil
	}
	// Prompt for key path
	ui.PromptFn = func(label string) (string, error) {
		return dummyKeyPath, nil
	}

	err = runSwitch([]string{"-c", "dave", "dave@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ := config.Load()
	user := store.FindUser("dave")
	if user == nil || user.SSHKey != dummyKeyPath {
		t.Errorf("expected user SSH key path to be %s, got %s", dummyKeyPath, user.SSHKey)
	}
}
