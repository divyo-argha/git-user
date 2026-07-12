package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestRunPubkeyPush_Errors(t *testing.T) {
	setupTestEnv(t)

	// Case 1: No active user
	err := runPubkeyPush(nil)
	if err == nil {
		t.Fatal("expected error when no active user, got nil")
	}

	// Setup active user without key
	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	// Case 2: No SSH key bound
	err = runPubkeyPush(nil)
	if err == nil {
		t.Fatal("expected error when no ssh key bound, got nil")
	}
}

func TestDetectPlatformFromRemotes(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Temporarily cd into tmpDir for the test
	oldWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Inside a non-git directory, should return empty
	plat, _ := detectPlatformFromRemotes()
	if plat != "" {
		t.Errorf("expected empty platform outside git repo, got %q", plat)
	}

	// Initialize git repo
	gitDir := filepath.Join(tmpDir, "repo")
	_ = os.Mkdir(gitDir, 0700)
	_ = os.Chdir(gitDir)

	_ = exec.Command("git", "init").Run()

	// Test 1: GitHub URL
	_ = exec.Command("git", "remote", "add", "origin", "git@github.com:user/repo.git").Run()
	plat, _ = detectPlatformFromRemotes()
	if plat != "github" {
		t.Errorf("expected github, got %q", plat)
	}

	// Test 2: GitLab custom URL
	_ = exec.Command("git", "remote", "set-url", "origin", "https://gitlab.my-company.com/org/project.git").Run()
	plat, host := detectPlatformFromRemotes()
	if plat != "gitlab" || host != "gitlab.my-company.com" {
		t.Errorf("expected gitlab with custom host, got platform=%q host=%q", plat, host)
	}

	// Test 3: Bitbucket URL
	_ = exec.Command("git", "remote", "set-url", "origin", "https://bitbucket.org/team/project.git").Run()
	plat, _ = detectPlatformFromRemotes()
	if plat != "bitbucket" {
		t.Errorf("expected bitbucket, got %q", plat)
	}
}
