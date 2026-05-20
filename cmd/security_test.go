package cmd

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsSSHKeyPassphraseProtected(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}

	dir := t.TempDir()

	unprotectedKey := filepath.Join(dir, "unprotected")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-C", "test@example.com", "-f", unprotectedKey, "-N", "").Run(); err != nil {
		t.Fatalf("generating unprotected key: %v", err)
	}

	protectedKey := filepath.Join(dir, "protected")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-C", "test@example.com", "-f", protectedKey, "-N", "secret-passphrase").Run(); err != nil {
		t.Fatalf("generating protected key: %v", err)
	}

	protected, err := isSSHKeyPassphraseProtected(unprotectedKey)
	if err != nil {
		t.Fatalf("checking unprotected key: %v", err)
	}
	if protected {
		t.Fatal("unprotected key reported as protected")
	}

	protected, err = isSSHKeyPassphraseProtected(protectedKey)
	if err != nil {
		t.Fatalf("checking protected key: %v", err)
	}
	if !protected {
		t.Fatal("protected key reported as unprotected")
	}
}
