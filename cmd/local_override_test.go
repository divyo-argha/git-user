package cmd

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
	_ = store.AddUser("work", "work@example.com")
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
	err = runSwitch([]string{"--local", "work"})
	if err != nil {
		t.Fatalf("local switch failed: %v", err)
	}

	// Verify resolved name in repository is "work"
	if git.CurrentName() != "work" || git.CurrentEmail() != "work@example.com" {
		t.Errorf("expected local override to be work, got %s / %s", git.CurrentName(), git.CurrentEmail())
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
