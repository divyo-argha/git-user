package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHookInstallUninstall(t *testing.T) {
	// Create a temporary git repo for testing
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skip("Git not available or failed to init repo")
	}

	// Save current directory
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	// Change to temp repo
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Test install
	if err := installHook(); err != nil {
		t.Errorf("installHook() failed: %v", err)
	}

	// Verify hook was created
	hookPath := filepath.Join(tmpDir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Error("Hook file was not created")
	}

	// Verify hook is executable
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Error("Hook file is not executable")
	}

	// Verify hook content
	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 {
		t.Error("Hook file is empty")
	}

	// Test uninstall
	if err := uninstallHook(); err != nil {
		t.Errorf("uninstallHook() failed: %v", err)
	}

	// Verify hook was removed
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("Hook file was not removed")
	}
}

func TestCheckIdentity(t *testing.T) {
	// This test depends on having a configured identity
	// We just verify it doesn't panic
	err := checkIdentity()
	// Don't assert success/failure since it depends on user's setup
	t.Logf("checkIdentity() result: %v", err)
}
