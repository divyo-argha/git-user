package cmd

import (
	"os/exec"
	"path/filepath"
	"testing"
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

	if err := changeSSHKeyPassphrase(keyPath, "another-secret", ""); err != nil {
		t.Fatalf("removing passphrase: %v", err)
	}

	protected, err = isSSHKeyPassphraseProtected(keyPath)
	if err != nil {
		t.Fatalf("checking unprotected key: %v", err)
	}
	if protected {
		t.Fatal("key should be unprotected after removing passphrase")
	}
}
