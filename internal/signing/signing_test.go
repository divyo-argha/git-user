package signing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestCurrentStatus(t *testing.T) {
	uDisabled := &config.User{Name: "alice", SignDisabled: true, SignKey: "somekey"}
	if st := CurrentStatus(uDisabled); st.Enabled {
		t.Errorf("expected disabled when SignDisabled is true")
	}

	uNoKey := &config.User{Name: "alice", SignDisabled: false, SignKey: ""}
	if st := CurrentStatus(uNoKey); st.Enabled {
		t.Errorf("expected disabled when SignKey is empty")
	}

	uEnabled := &config.User{Name: "alice", SignDisabled: false, SignKey: "key123", SignFormat: "ssh"}
	st := CurrentStatus(uEnabled)
	if !st.Enabled || st.Key != "key123" || st.Format != "ssh" {
		t.Errorf("unexpected status: %+v", st)
	}
}

func TestDisable(t *testing.T) {
	store := &config.Store{
		Users: []config.User{
			{Name: "alice", SignDisabled: false, SignKey: "key123"},
		},
	}
	Disable(store, "alice")
	u := store.FindUser("alice")
	if !u.SignDisabled {
		t.Errorf("expected SignDisabled to be true after Disable")
	}
}

func TestEnable(t *testing.T) {
	home, _ := os.UserHomeDir()
	store := &config.Store{
		Users: []config.User{
			{Name: "alice", SSHKey: "~/id_ed25519"},
			{Name: "bob"},
		},
	}

	// User not found
	if _, _, _, err := Enable(store, "nonexistent", "", ""); err == nil {
		t.Errorf("expected error when user not found")
	}

	// Bob has no bound key and no key provided
	if _, _, _, err := Enable(store, "bob", "", ""); err != ErrNoKeyBound {
		t.Errorf("expected ErrNoKeyBound, got %v", err)
	}

	// Alice auto-detects from bound SSH key
	resolvedKey, resolvedFormat, autoDetected, err := Enable(store, "alice", "", "")
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}
	if !autoDetected {
		t.Errorf("expected autoDetected to be true")
	}
	if resolvedFormat != "ssh" {
		t.Errorf("expected ssh format, got %s", resolvedFormat)
	}
	expectedPath := filepath.Join(home, "id_ed25519")
	if resolvedKey != expectedPath {
		t.Errorf("expected expanded path %s, got %s", expectedPath, resolvedKey)
	}

	// Bob with explicit GPG key
	resolvedKey, resolvedFormat, autoDetected, err = Enable(store, "bob", "3AA5C34371567BD2", "")
	if err != nil {
		t.Fatalf("Enable with GPG key failed: %v", err)
	}
	if autoDetected {
		t.Errorf("expected autoDetected to be false")
	}
	if resolvedFormat != "gpg" {
		t.Errorf("expected gpg format, got %s", resolvedFormat)
	}
	if resolvedKey != "3AA5C34371567BD2" {
		t.Errorf("unexpected key: %s", resolvedKey)
	}

	// Bob with explicit ssh format
	resolvedKey, resolvedFormat, _, err = Enable(store, "bob", "/path/to/key.pub", "")
	if err != nil {
		t.Fatalf("Enable with .pub key failed: %v", err)
	}
	if resolvedFormat != "ssh" {
		t.Errorf("expected ssh format for .pub key, got %s", resolvedFormat)
	}
}
