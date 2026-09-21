package rekeyops

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestResolveOldKeyPath(t *testing.T) {
	store := &config.Store{
		Users: []config.User{
			{Name: "alice", SSHKey: "/path/to/alice_key"},
			{Name: "bob"},
		},
	}

	p, err := ResolveOldKeyPath(store, "alice")
	if err != nil || p != "/path/to/alice_key" {
		t.Fatalf("unexpected alice key path: %s, %v", p, err)
	}

	p, err = ResolveOldKeyPath(store, "bob")
	if err != nil || p == "" {
		t.Fatalf("unexpected bob default key path: %s, %v", p, err)
	}

	_, err = ResolveOldKeyPath(store, "nobody")
	if err == nil {
		t.Errorf("expected error for nonexistent user")
	}
}

func TestRotateSuccessAndRollback(t *testing.T) {
	tmpDir := t.TempDir()
	oldKey := filepath.Join(tmpDir, "id_old")
	newKey := filepath.Join(tmpDir, "id_new")

	if err := os.WriteFile(oldKey, []byte("old-private-key"), 0600); err != nil {
		t.Fatalf("failed to write old key: %v", err)
	}
	if err := os.WriteFile(oldKey+".pub", []byte("old-public-key"), 0644); err != nil {
		t.Fatalf("failed to write old pubkey: %v", err)
	}

	store := &config.Store{
		Users: []config.User{
			{
				Name:        "alice",
				SSHKey:      oldKey,
				SignKey:     oldKey,
				SignFormat:  "ssh",
				SignDisabled: false,
			},
		},
	}

	// 1. Failure in generateKey -> rollback
	genErr := errors.New("ssh-keygen failed")
	res, err := Rotate(store, "alice", newKey, func(p string) error {
		return genErr
	})
	if err == nil || !errors.Is(err, genErr) {
		t.Fatalf("expected error from failed generation, got %v", err)
	}
	if !res.HadOldKey {
		t.Errorf("expected HadOldKey to be true")
	}
	// Verify old key restored
	if _, err := os.Stat(oldKey); err != nil {
		t.Errorf("expected old key to be restored after failure: %v", err)
	}

	// 2. Success in generateKey
	res, err = Rotate(store, "alice", newKey, func(p string) error {
		if err := os.WriteFile(p, []byte("new-private-key"), 0600); err != nil {
			return err
		}
		return os.WriteFile(p+".pub", []byte("new-public-key"), 0644)
	})
	if err != nil {
		t.Fatalf("Rotate failed: %v", err)
	}
	if !res.SignKeyCarried {
		t.Errorf("expected SignKeyCarried to be true")
	}
	if res.NewKeyPath != newKey {
		t.Errorf("expected NewKeyPath to be %s, got %s", newKey, res.NewKeyPath)
	}

	// Verify store updated
	u := store.FindUser("alice")
	if u.SSHKey != newKey {
		t.Errorf("expected store SSHKey to be %s, got %s", newKey, u.SSHKey)
	}
	if u.SignKey != newKey {
		t.Errorf("expected store SignKey to be carried to %s, got %s", newKey, u.SignKey)
	}
}
