package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestRunBindPath_Errors(t *testing.T) {
	setupTestEnv(t)

	// Missing args
	err := runBindPath([]string{})
	if err == nil {
		t.Fatal("expected error with no arguments, got nil")
	}

	// Only 1 arg
	err = runBindPath([]string{"work"})
	if err == nil {
		t.Fatal("expected error with only 1 argument, got nil")
	}

	// User not found
	err = runBindPath([]string{"work", "/tmp"})
	if err == nil {
		t.Fatal("expected error with nonexistent user, got nil")
	}

	// Register user
	store, _ := config.Load()
	_ = store.AddUser("work", "work@example.com")
	_ = config.Save(store)

	// Nonexistent path
	err = runBindPath([]string{"work", "/nonexistent/dir"})
	if err == nil {
		t.Fatal("expected error with nonexistent directory, got nil")
	}

	// Path is a file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	_ = os.WriteFile(filePath, []byte("test"), 0600)
	err = runBindPath([]string{"work", filePath})
	if err == nil {
		t.Fatal("expected error when path is a file, got nil")
	}
}

func TestRunBindPath_Success(t *testing.T) {
	tmpDir := setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("work", "work@example.com")
	_ = config.Save(store)

	workDir := filepath.Join(tmpDir, "work-repo")
	_ = os.Mkdir(workDir, 0700)

	// Bind path
	err := runBindPath([]string{"work", workDir})
	if err != nil {
		t.Fatalf("unexpected error binding path: %v", err)
	}

	// Verify bind paths in config
	store, _ = config.Load()
	user := store.FindUser("work")
	if len(user.BindPaths) != 1 || user.BindPaths[0] != workDir {
		t.Fatalf("expected bind path %q, got %v", workDir, user.BindPaths)
	}

	// Verify snippet file exists and contains correct content
	configDir := filepath.Dir(config.ConfigPath())
	snippetPath := filepath.Join(configDir, "profile-work.gitconfig")
	content, err := os.ReadFile(snippetPath)
	if err != nil {
		t.Fatalf("failed to read snippet file: %v", err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, "name = work") || !strings.Contains(contentStr, "email = work@example.com") {
		t.Errorf("snippet content mismatch:\n%s", contentStr)
	}

	// Verify git config includeIf entry is set
	cmd := exec.Command("git", "config", "--global", "--get-regexp", `includeif\..*\.path`)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to read git config: %v", err)
	}
	outStr := string(out)
	if !strings.Contains(outStr, "includeif.gitdir/i:") || !strings.Contains(outStr, "profile-work.gitconfig") {
		t.Errorf("expected git config to contain includeif matching profile-work.gitconfig, got:\n%s", outStr)
	}

	// Unbind path
	err = runUnbindPath([]string{"work", workDir})
	if err != nil {
		t.Fatalf("unexpected error unbinding path: %v", err)
	}

	// Verify configuration unbind
	store, _ = config.Load()
	user = store.FindUser("work")
	if len(user.BindPaths) != 0 {
		t.Fatalf("expected bind paths to be empty, got %v", user.BindPaths)
	}

	// Verify snippet file is deleted because no bind paths remain
	if _, err := os.Stat(snippetPath); !os.IsNotExist(err) {
		t.Errorf("expected snippet file to be deleted, but it still exists")
	}

	// Verify git config includeIf entry is unset
	cmd = exec.Command("git", "config", "--global", "--get-regexp", `includeif\..*\.path`)
	out, _ = cmd.Output()
	if strings.Contains(string(out), "profile-work.gitconfig") {
		t.Errorf("expected git config to clear includeif, but it still contains it:\n%s", string(out))
	}
}
