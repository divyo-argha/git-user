package git_test

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/git"
)

func TestConvertHTTPSToSSH(t *testing.T) {
	cases := []struct {
		input   string
		want    string
		changed bool
	}{
		{"https://github.com/user/repo.git", "git@github.com:user/repo.git", true},
		{"https://gitlab.com/org/project.git", "git@gitlab.com:org/project.git", true},
		{"https://bitbucket.org/team/repo.git", "git@bitbucket.org:team/repo.git", true},
		// already SSH — should be unchanged
		{"git@github.com:user/repo.git", "git@github.com:user/repo.git", false},
		// no .git suffix
		{"https://github.com/user/repo", "git@github.com:user/repo.git", true},
	}

	for _, c := range cases {
		got, changed := git.ConvertHTTPSToSSH(c.input)
		if changed != c.changed {
			t.Errorf("ConvertHTTPSToSSH(%q): changed=%v, want %v", c.input, changed, c.changed)
		}
		if got != c.want {
			t.Errorf("ConvertHTTPSToSSH(%q): got %q, want %q", c.input, got, c.want)
		}
	}
}

func TestIsInstalled(t *testing.T) {
	// Git should be installed for this project to work
	if !git.IsInstalled() {
		t.Error("IsInstalled() = false, but git should be available")
	}
}

func TestApply(t *testing.T) {
	// Save current git config
	oldName := git.CurrentName()
	oldEmail := git.CurrentEmail()
	
	// Ensure we restore it
	defer func() {
		if oldName != "" {
			git.Apply(oldName, oldEmail)
		}
	}()

	// Test applying new config
	testName := "Test User"
	testEmail := "test@example.com"
	
	if err := git.Apply(testName, testEmail); err != nil {
		t.Fatalf("Apply() failed: %v", err)
	}

	// Verify it was applied
	if got := git.CurrentName(); got != testName {
		t.Errorf("CurrentName() = %q, want %q", got, testName)
	}
	if got := git.CurrentEmail(); got != testEmail {
		t.Errorf("CurrentEmail() = %q, want %q", got, testEmail)
	}
}

func TestConfigureSSH(t *testing.T) {
	testKeyPath := "/home/user/.ssh/test_key"
	
	if err := git.ConfigureSSH(testKeyPath); err != nil {
		t.Fatalf("ConfigureSSH() failed: %v", err)
	}

	// Clean up
	defer git.RemoveSSHConfig()
}

func TestRemoveSSHConfig(t *testing.T) {
	// Set a test SSH config
	testKeyPath := "/tmp/test_key"
	git.ConfigureSSH(testKeyPath)

	// Remove it
	if err := git.RemoveSSHConfig(); err != nil {
		t.Fatalf("RemoveSSHConfig() failed: %v", err)
	}
}

func TestIsInRepo(t *testing.T) {
	// This test depends on whether we're in a git repo
	// Just verify it doesn't panic
	_ = git.IsInRepo()
}
