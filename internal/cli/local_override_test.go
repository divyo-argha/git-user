package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

func TestLocalSwitchOverride(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := setupTestEnv(t)

	// Create profiles in config
	store, _ := config.Load()
	_ = store.AddUser("personal", "personal@example.com")
	_ = store.AddUser("eng", "eng@example.com")
	_ = config.Save(store)

	// Set up global config
	err := runSwitch([]string{"personal"})
	if err != nil {
		t.Fatalf("global switch failed: %v", err)
	}

	// Verify global values
	if git.CurrentName() != "personal" || git.CurrentEmail() != "personal@example.com" {
		t.Fatalf("global configuration was not set correctly")
	}

	// Now create a temporary git repository directory
	repoDir := filepath.Join(tmpDir, "my-repo")
	err = os.MkdirAll(repoDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}

	// Change working directory of the test process to the repository directory
	cwd, _ := os.Getwd()
	err = os.Chdir(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chdir(cwd)
	})

	// Run switch locally
	err = runSwitch([]string{"--local", "eng"})
	if err != nil {
		t.Fatalf("local switch failed: %v", err)
	}

	// Verify resolved name in repository is "eng"
	if git.CurrentName() != "eng" || git.CurrentEmail() != "eng@example.com" {
		t.Errorf("expected local override to be eng, got %s / %s", git.CurrentName(), git.CurrentEmail())
	}

	// Verify global config remains "personal"
	globalName, _ := exec.Command("git", "config", "--global", "user.name").Output()
	globalEmail, _ := exec.Command("git", "config", "--global", "user.email").Output()

	gName := strings.TrimSpace(string(globalName))
	gEmail := strings.TrimSpace(string(globalEmail))

	if gName != "personal" || gEmail != "personal@example.com" {
		t.Errorf("global config was incorrectly modified to: %s / %s", gName, gEmail)
	}

	// Test git-user current displays local override
	err = runCurrent([]string{})
	if err != nil {
		t.Errorf("runCurrent failed: %v", err)
	}
}

// TestLocalSwitchDoesNotDeleteTemporaryGlobalIdentity guards against a
// regression where `switch --local` ran the same "auto-logout" cleanup as a
// global switch — including permanently deleting a *temporary* identity's key
// files — even though a local override never changes (or should affect) the
// global active identity. Deleting those files left the global identity
// still recorded in the config but pointing at a key that no longer existed.
func TestLocalSwitchDoesNotDeleteTemporaryGlobalIdentity(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	tmpDir := setupTestEnv(t)

	store, _ := config.Load()
	tempKeyPath := filepath.Join(tmpDir, ".ssh", "git_tempy")
	if err := os.MkdirAll(filepath.Dir(tempKeyPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tempKeyPath, []byte("private key"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tempKeyPath+".pub", []byte("ssh-ed25519 AAAA tempy"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = store.AddUser("tempy", "tempy@example.com")
	tempUser := store.FindUser("tempy")
	tempUser.IsTemporary = true
	tempUser.SSHKey = tempKeyPath
	store.Current = "tempy"
	_ = store.AddUser("other", "other@example.com")
	if err := config.Save(store); err != nil {
		t.Fatal(err)
	}
	if err := git.Apply("tempy", "tempy@example.com"); err != nil {
		t.Fatal(err)
	}

	repoDir := filepath.Join(tmpDir, "local-repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}
	cwd, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	if err := runSwitch([]string{"--local", "other"}); err != nil {
		t.Fatalf("local switch failed: %v", err)
	}

	if _, err := os.Stat(tempKeyPath); err != nil {
		t.Errorf("expected temporary identity's key file to survive a local switch, but it's gone: %v", err)
	}

	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if reloaded.FindUser("tempy") == nil {
		t.Error("expected temporary identity \"tempy\" to still be registered after a local switch")
	}
	if reloaded.Current != "tempy" {
		t.Errorf("expected global active identity to remain \"tempy\" after a local switch, got %q", reloaded.Current)
	}
}
