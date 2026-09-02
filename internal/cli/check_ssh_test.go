package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestRunCheckSSH_NoActiveIdentity(t *testing.T) {
	setupTestEnv(t)
	err := runCheckSSH([]string{})
	if err == nil {
		t.Fatal("expected error with no active identity, got nil")
	}
}

func TestRunCheckSSH_IdentityNotFound(t *testing.T) {
	setupTestEnv(t)
	err := runCheckSSH([]string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error when identity not found, got nil")
	}
}

func TestRunCheckSSH_NoSSHKey(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.SetCurrent("dev")
	_ = config.Save(store)

	err := runCheckSSH([]string{})
	if err != nil {
		t.Fatalf("expected nil error when identity has no SSH key, got %v", err)
	}

	// Also test --json and --plain
	err = runCheckSSH([]string{"--json"})
	if err != nil {
		t.Fatalf("expected nil error for --json with no SSH key, got %v", err)
	}

	err = runCheckSSH([]string{"--plain"})
	if err != nil {
		t.Fatalf("expected nil error for --plain with no SSH key, got %v", err)
	}
}

func TestRunCheckSSH_KeyNotFound(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.BindSSHKey("dev", "/nonexistent/path/to/key")
	_ = store.SetCurrent("dev")
	_ = config.Save(store)

	err := runCheckSSH([]string{})
	if err == nil {
		t.Fatal("expected error when key file does not exist, got nil")
	}
}

func TestRunCheckSSH_WithKey(t *testing.T) {
	tmpDir := setupTestEnv(t)

	sshDir := filepath.Join(tmpDir, ".ssh")
	_ = os.MkdirAll(sshDir, 0700)
	keyPath := filepath.Join(sshDir, "id_ed25519")
	_ = os.WriteFile(keyPath, []byte("private key"), 0600)
	_ = os.WriteFile(keyPath+".pub", []byte("public key"), 0644)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.BindSSHKey("dev", keyPath)
	_ = store.SetCurrent("dev")
	_ = config.Save(store)

	// Test default execution
	err := runCheckSSH([]string{})
	if err != nil {
		t.Fatalf("unexpected error running check-ssh: %v", err)
	}

	// Test with explicit name
	err = runCheckSSH([]string{"dev"})
	if err != nil {
		t.Fatalf("unexpected error with explicit name: %v", err)
	}

	// Test with --json
	err = runCheckSSH([]string{"--json"})
	if err != nil {
		t.Fatalf("unexpected error with --json: %v", err)
	}

	// Test with --plain
	err = runCheckSSH([]string{"--plain"})
	if err != nil {
		t.Fatalf("unexpected error with --plain: %v", err)
	}
}

func TestRunCheckSSH_SubcommandAndAliases(t *testing.T) {
	tmpDir := setupTestEnv(t)

	sshDir := filepath.Join(tmpDir, ".ssh")
	_ = os.MkdirAll(sshDir, 0700)
	keyPath := filepath.Join(sshDir, "id_ed25519")
	_ = os.WriteFile(keyPath, []byte("private key"), 0600)
	_ = os.WriteFile(keyPath+".pub", []byte("public key"), 0644)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.BindSSHKey("dev", keyPath)
	_ = store.SetCurrent("dev")
	_ = config.Save(store)

	// Test pubkey check
	err := runPubkey([]string{"check"})
	if err != nil {
		t.Fatalf("pubkey check error = %v", err)
	}
}
