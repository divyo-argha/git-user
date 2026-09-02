package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

// TestOpUninstallRestoresOriginalAndKeepsKeys verifies the TUI uninstall path
// restores the pre-git-user git identity, removes git-user's own config
// directory, and — matching its deliberately conservative scope — leaves SSH
// key files on disk untouched rather than silently deleting them.
func TestOpUninstallRestoresOriginalAndKeepsKeys(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	withTempConfig(t)
	dir := os.Getenv("HOME")

	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(sshDir, "git_dev")
	if err := os.WriteFile(keyPath, []byte("private key"), 0600); err != nil {
		t.Fatal(err)
	}

	store, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	store.SnapshotOriginal("Original User", "original@example.com", "", "", "", "")
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.BindSSHKey("dev", keyPath)
	_ = store.SetCurrent("dev")
	if err := config.Save(store); err != nil {
		t.Fatal(err)
	}
	if err := git.Apply("dev", "dev@example.com"); err != nil {
		t.Fatal(err)
	}

	result, err := opUninstall(store)
	if err != nil {
		t.Fatalf("opUninstall failed: %v", err)
	}
	if !result.showReport {
		t.Error("expected opUninstall to produce a report")
	}

	if git.CurrentName() != "Original User" || git.CurrentEmail() != "original@example.com" {
		t.Errorf("expected git config to be restored to the original identity, got %s <%s>", git.CurrentName(), git.CurrentEmail())
	}

	if _, err := os.Stat(config.ConfigPath()); !os.IsNotExist(err) {
		t.Errorf("expected git-user's config directory to be removed, config still readable (err=%v)", err)
	}

	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("expected SSH key file to be kept by default, but it's gone: %v", err)
	}
}
