package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ssh"
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

	if err := ssh.ChangeKeyPassphrase(keyPath, "", "new-secret"); err != nil {
		t.Fatalf("adding passphrase: %v", err)
	}

	protected, err = isSSHKeyPassphraseProtected(keyPath)
	if err != nil {
		t.Fatalf("checking protected key: %v", err)
	}
	if !protected {
		t.Fatal("key should be protected after adding passphrase")
	}

	if err := ssh.ChangeKeyPassphrase(keyPath, "wrong-secret", "another-secret"); err == nil {
		t.Fatal("expected wrong current passphrase to fail")
	}

	if err := ssh.ChangeKeyPassphrase(keyPath, "new-secret", "another-secret"); err != nil {
		t.Fatalf("changing passphrase: %v", err)
	}

	if err := ssh.ChangeKeyPassphrase(keyPath, "another-secret", ""); err != nil {
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

// TestRunPassphraseModeErrorsOnSaveFailure guards against a regression where
// changing passphrase mode discarded config.Save's error entirely and always
// printed a success message — a security-relevant setting could silently
// fail to persist. Forces the save (but not the preceding load) to fail by
// making the config directory read-only after priming it.
func TestRunPassphraseModeErrorsOnSaveFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission-based failure injection doesn't work as root")
	}
	tmpDir := setupTestEnv(t)

	keyPath := filepath.Join(tmpDir, "dev_key")
	if err := os.WriteFile(keyPath, []byte("dummy"), 0600); err != nil {
		t.Fatal(err)
	}

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.BindSSHKey("dev", keyPath)
	if err := config.Save(store); err != nil {
		t.Fatalf("priming config: %v", err)
	}

	configDir := filepath.Dir(config.ConfigPath())
	if err := os.Chmod(configDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(configDir, 0700) })

	err := runPassphrase([]string{"dev", "--mode", "login"})
	if err == nil {
		t.Fatal("expected an error when config.Save fails, got nil")
	}
}
