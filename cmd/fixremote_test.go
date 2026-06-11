package cmd

import (
	"os"
	"os/exec"
	"testing"

	"github.com/divyo-argha/git-user/internal/git"
)

func TestRunFixRemote_NotInRepo(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Change working directory to temp dir (which is not a git repo)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	err = runFixRemote([]string{})
	if err == nil {
		t.Fatal("expected error when not in repository, got nil")
	}
}

func TestRunFixRemote_Success(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Change working directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	// Initialize a git repo
	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Add remote
	cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/divyo-argha/git-user.git")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	// Run fixremote
	err = runFixRemote([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert remote has been updated to SSH
	newURL, err := git.GetRemoteURL("origin")
	if err != nil {
		t.Fatalf("failed to get remote URL: %v", err)
	}

	expected := "git@github.com:divyo-argha/git-user.git"
	if newURL != expected {
		t.Errorf("expected URL to be %q, got %q", expected, newURL)
	}
}
