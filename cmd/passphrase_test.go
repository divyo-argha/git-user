package cmd

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestChangeSSHKeyPassphrase(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}

	keyPath := filepath.Join(t.TempDir(), "key")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-C", "test@example.com", "-f", keyPath, "-N", "").Run(); err != nil {
		t.Fatalf("generating key: %v", err)
	}

	protected, err := isSSHKeyPassphraseProtected(keyPath)
	if err != nil {
		t.Fatalf("checking initial key: %v", err)
	}
	if protected {
		t.Fatal("new key should start unprotected")
	}

	if err := changeSSHKeyPassphrase(keyPath, "", "new-secret"); err != nil {
		t.Fatalf("adding passphrase: %v", err)
	}

	protected, err = isSSHKeyPassphraseProtected(keyPath)
	if err != nil {
		t.Fatalf("checking protected key: %v", err)
	}
	if !protected {
		t.Fatal("key should be protected after adding passphrase")
	}

	if err := changeSSHKeyPassphrase(keyPath, "wrong-secret", "another-secret"); err == nil {
		t.Fatal("expected wrong current passphrase to fail")
	}

	if err := changeSSHKeyPassphrase(keyPath, "new-secret", "another-secret"); err != nil {
		t.Fatalf("changing passphrase: %v", err)
	}
}

func TestRequireActivePassphraseIdentity(t *testing.T) {
	store := &config.Store{}
	if err := store.AddUser("work", "work@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddUser("personal", "me@example.com"); err != nil {
		t.Fatal(err)
	}

	work := store.FindUser("work")

	if err := requireActivePassphraseIdentity(store, work); err == nil {
		t.Fatal("expected error when no identity is active")
	}

	if err := store.SetCurrent("personal"); err != nil {
		t.Fatal(err)
	}
	if err := requireActivePassphraseIdentity(store, work); err == nil {
		t.Fatal("expected error when another identity is active")
	}

	if err := store.SetCurrent("work"); err != nil {
		t.Fatal(err)
	}
	if err := requireActivePassphraseIdentity(store, work); err != nil {
		t.Fatalf("expected active identity to pass: %v", err)
	}
}
