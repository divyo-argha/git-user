package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
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

func TestRunSecurityCheck_Fix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file-mode bits are unreliable on Windows")
	}
	tmpDir := setupTestEnv(t)

	keyPath := filepath.Join(tmpDir, ".ssh", "id_ed25519")
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("private key"), 0644); err != nil {
		t.Fatal(err)
	}

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.BindSSHKey("dev", keyPath)
	_ = config.Save(store)

	if err := os.Chmod(config.ConfigPath(), 0644); err != nil {
		t.Fatal(err)
	}

	// Without --fix, insecure permissions are reported but left untouched.
	if err := runSecurityCheck([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	info, _ := os.Stat(keyPath)
	if info.Mode().Perm() != 0644 {
		t.Fatalf("expected key permissions untouched without --fix, got %o", info.Mode().Perm())
	}

	// With --fix (and the default-Yes Confirm mock from setupTestEnv), both
	// the SSH key and the config file should be corrected to 0600.
	if err := runSecurityCheck([]string{"--fix"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected SSH key permissions fixed to 0600, got %o", info.Mode().Perm())
	}
	configInfo, err := os.Stat(config.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if configInfo.Mode().Perm() != 0600 {
		t.Errorf("expected config file permissions fixed to 0600, got %o", configInfo.Mode().Perm())
	}
}
