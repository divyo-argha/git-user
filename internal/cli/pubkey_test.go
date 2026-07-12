package cli

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func TestRunPubkey_NoActiveIdentity(t *testing.T) {
	setupTestEnv(t)
	err := runPubkey([]string{})
	if err == nil {
		t.Fatal("expected error with no active identity and no args, got nil")
	}
}

func TestRunPubkey_AccessDenied(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.AddUser("bob", "bob@example.com")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	err := runPubkey([]string{"bob"})
	if err == nil {
		t.Fatal("expected access denied error trying to view inactive user key, got nil")
	}
}

func TestRunPubkey_NoSSHKey(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	err := runPubkey([]string{})
	if err == nil {
		t.Fatal("expected error with no SSH key bound, got nil")
	}
}

func TestRunPubkey_KeyFileNotFound(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.BindSSHKey("alice", "/nonexistent/key")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	err := runPubkey([]string{})
	if err == nil {
		t.Fatal("expected error when key file does not exist, got nil")
	}
}

func TestRunPubkey_Success(t *testing.T) {
	_ = setupTestEnv(t)

	// We can generate a real key using generateAndDisplayKey to ensure ssh-keygen works
	ui.SelectFn = func(label string, options []string) (int, error) {
		return 0, nil // Auto-generate
	}
	err := runSwitch([]string{"-c", "alice", "alice@example.com"})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Now try running pubkey
	err = runPubkey([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test passing the active identity explicitly as argument
	err = runPubkey([]string{"alice"})
	if err != nil {
		t.Fatalf("unexpected error with explicit name: %v", err)
	}
}
