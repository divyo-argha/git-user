package cli

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

	// Run fixremote (default non-interactive should be explicit rewrite for backwards compat)
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

func TestRunFixRemote_Implicit(t *testing.T) {
	tmpDir := setupTestEnv(t)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	originalURL := "https://github.com/divyo-argha/git-user.git"
	cmd = exec.Command("git", "remote", "add", "origin", originalURL)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	// Run fixremote with --implicit
	err = runFixRemote([]string{"--implicit"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Remote fetch URL should remain unchanged HTTPS
	fetchURL, err := git.GetRemoteURL("origin")
	if err != nil {
		t.Fatalf("failed to get remote URL: %v", err)
	}
	if fetchURL != originalURL {
		t.Errorf("expected fetch URL to be %q, got %q", originalURL, fetchURL)
	}

	// Push URL should resolve to SSH
	pushURL, err := git.GetPushRemoteURL("origin")
	if err != nil {
		t.Fatalf("failed to get push remote URL: %v", err)
	}
	expectedPush := "git@github.com:divyo-argha/git-user.git"
	if pushURL != expectedPush {
		t.Errorf("expected push URL to be %q, got %q", expectedPush, pushURL)
	}

	// HasHTTPSPushRemotes should now be false
	if git.HasHTTPSPushRemotes() {
		t.Errorf("expected HasHTTPSPushRemotes to be false after implicit push configuration")
	}
}

func TestRunFixRemote_Global(t *testing.T) {
	tmpDir := setupTestEnv(t)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	originalURL := "https://github.com/divyo-argha/git-user.git"
	cmd = exec.Command("git", "remote", "add", "origin", originalURL)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	defer func() {
		for _, host := range git.DefaultPushInsteadOfHosts() {
			git.RemovePushInsteadOf(host, false)
		}
	}()

	// Run fixremote with --global
	err = runFixRemote([]string{"--global"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Push URL should resolve to SSH via global config
	pushURL, err := git.GetPushRemoteURL("origin")
	if err != nil {
		t.Fatalf("failed to get push remote URL: %v", err)
	}
	expectedPush := "git@github.com:divyo-argha/git-user.git"
	if pushURL != expectedPush {
		t.Errorf("expected push URL to be %q, got %q", expectedPush, pushURL)
	}
}

func TestRunFixRemote_ConflictingFlags(t *testing.T) {
	err := runFixRemote([]string{"--implicit", "--explicit"})
	if err == nil {
		t.Fatal("expected error with conflicting flags, got nil")
	}
}

func TestRunFixRemote_Help(t *testing.T) {
	err := runFixRemote([]string{"--help"})
	if err != nil {
		t.Fatalf("expected nil error for --help, got %v", err)
	}
}
