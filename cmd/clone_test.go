package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestRunClone_Errors(t *testing.T) {
	setupTestEnv(t)

	// No arguments
	err := runClone([]string{})
	if err == nil {
		t.Fatal("expected error with no repo URL, got nil")
	}

	// No registered identities
	err = runClone([]string{"https://github.com/divyo-argha/git-user.git"})
	if err == nil {
		t.Fatal("expected error with no registered identities, got nil")
	}
}

func TestRunClone_Success(t *testing.T) {
	tmpDir := setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("work", "work@example.com")
	_ = config.Save(store)

	// Create a mock remote repo to clone from
	remoteRepoDir := filepath.Join(tmpDir, "remote-repo")
	err := os.Mkdir(remoteRepoDir, 0755)
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Initialize git repo in the remote directory
	runGitCmd(t, remoteRepoDir, "init")
	runGitCmd(t, remoteRepoDir, "config", "user.name", "Remote Owner")
	runGitCmd(t, remoteRepoDir, "config", "user.email", "remote@owner.com")
	// Make a commit so it's a cloneable repo
	_ = os.WriteFile(filepath.Join(remoteRepoDir, "readme.md"), []byte("Hello remote"), 0644)
	runGitCmd(t, remoteRepoDir, "add", "readme.md")
	runGitCmd(t, remoteRepoDir, "commit", "-m", "initial commit")

	destRepoDir := filepath.Join(tmpDir, "cloned-repo")

	// Run clone subcommand
	err = runClone([]string{remoteRepoDir, destRepoDir, "--as", "work"})
	if err != nil {
		t.Fatalf("unexpected error cloning: %v", err)
	}

	// Verify local identity configuration in the destination repo
	nameCmd := exec.Command("git", "config", "--local", "user.name")
	nameCmd.Dir = destRepoDir
	nameOut, err := nameCmd.Output()
	if err != nil {
		t.Fatalf("failed to read local user.name: %v", err)
	}
	if strings.TrimSpace(string(nameOut)) != "work" {
		t.Errorf("expected local user.name to be 'work', got '%s'", strings.TrimSpace(string(nameOut)))
	}

	emailCmd := exec.Command("git", "config", "--local", "user.email")
	emailCmd.Dir = destRepoDir
	emailOut, err := emailCmd.Output()
	if err != nil {
		t.Fatalf("failed to read local user.email: %v", err)
	}
	if strings.TrimSpace(string(emailOut)) != "work@example.com" {
		t.Errorf("expected local user.email to be 'work@example.com', got '%s'", strings.TrimSpace(string(emailOut)))
	}
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git command failed: %v, output: %s", err, string(out))
	}
}
