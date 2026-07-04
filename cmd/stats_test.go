package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestRunStats_NotInRepo(t *testing.T) {
	tmpDir := setupTestEnv(t)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldWd)

	err = runStats([]string{})
	if err == nil {
		t.Fatal("expected error running stats outside a repository, got nil")
	}
}

func TestRunStats_Success(t *testing.T) {
	tmpDir := setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("work", "work@example.com")
	_ = config.Save(store)

	// Initialize git repo in the temp directory
	runGitCmd(t, tmpDir, "init")
	runGitCmd(t, tmpDir, "config", "user.name", "work")
	runGitCmd(t, tmpDir, "config", "user.email", "work@example.com")

	// Create a commit
	_ = os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("data"), 0644)
	runGitCmd(t, tmpDir, "add", "file.txt")
	runGitCmd(t, tmpDir, "commit", "-m", "add file")

	// Switch working directory to run stats
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldWd)

	// Run stats subcommand
	err = runStats([]string{})
	if err != nil {
		t.Fatalf("unexpected error running stats: %v", err)
	}
}
