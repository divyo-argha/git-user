package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/policyops"
)

func withTempRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skip("git not available or failed to init repo")
	}
	oldDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(oldDir) })
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func TestOpHookCheckEnforcesRepoPolicy(t *testing.T) {
	withTempConfig(t)
	repoRoot := withTempRepo(t)

	store, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddUser("work", "work@example.com"); err != nil {
		t.Fatal(err)
	}
	store.Current = "work"
	if err := config.Save(store); err != nil {
		t.Fatal(err)
	}
	exec.Command("git", "config", "user.name", "work").Run()
	exec.Command("git", "config", "user.email", "work@example.com").Run()

	if err := policyops.WritePolicy(repoRoot, true, nil); err != nil {
		t.Fatalf("WritePolicy failed: %v", err)
	}

	// Signing is disabled by default, so `hook check` should block.
	if _, err := opHook("check"); err == nil {
		t.Fatal("expected opHook(check) to fail when repo policy requires signing but it's disabled")
	}

	// Enabling signing (without needing a real key/format resolution here)
	// via the store directly should let the same check pass.
	store, err = config.Load()
	if err != nil {
		t.Fatal(err)
	}
	store.SetSigningKey("work", filepath.Join(repoRoot, "fake_key"), "ssh")
	if err := config.Save(store); err != nil {
		t.Fatal(err)
	}
	if _, err := opHook("check"); err != nil {
		t.Fatalf("expected opHook(check) to pass once signing is enabled, got: %v", err)
	}
}
