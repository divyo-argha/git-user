package git_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/testutil"
)

func TestIsIdentityInSync(t *testing.T) {
	// Both empty never reports in sync.
	if git.IsIdentityInSync("", "") {
		t.Error("IsIdentityInSync(\"\", \"\") should be false")
	}

	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// No git config yet → out of sync.
	if git.IsIdentityInSync("eng", "eng@example.com") {
		t.Error("expected out of sync with no git config")
	}

	// Matching identity → in sync.
	cfg := "[user]\n\tname = eng\n\temail = eng@example.com\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitconfig"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if !git.IsIdentityInSync("eng", "eng@example.com") {
		t.Error("expected in sync when git config matches")
	}

	// Drifted identity (only name matches) → out of sync.
	if git.IsIdentityInSync("eng", "other@example.com") {
		t.Error("expected out of sync when email drifted")
	}
}

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
		// embedded credentials
		{"https://user:token@github.com/foo/bar.git", "git@github.com:user/repo.git", true}, // Note: we'll test the actual host parsing
	}

	for _, c := range cases {
		// Skip mock credential test checking since it expects a specific format
		if c.input == "https://user:token@github.com/foo/bar.git" {
			got, changed := git.ConvertHTTPSToSSH(c.input)
			if !changed || got != "git@github.com:foo/bar.git" {
				t.Errorf("ConvertHTTPSToSSH(%q): got %q, want %q", c.input, got, "git@github.com:foo/bar.git")
			}
			continue
		}
		got, changed := git.ConvertHTTPSToSSH(c.input)
		if changed != c.changed {
			t.Errorf("ConvertHTTPSToSSH(%q): changed=%v, want %v", c.input, changed, c.changed)
		}
		if got != c.want {
			t.Errorf("ConvertHTTPSToSSH(%q): got %q, want %q", c.input, got, c.want)
		}
	}
}

func TestCurrentBranchAndRepo(t *testing.T) {
	// Since we are running in a Git repository, these should return correct non-empty values
	branch := git.CurrentBranch()
	if branch == "" {
		t.Errorf("Expected non-empty current branch name")
	}

	repoName := git.CurrentRepoName()
	if repoName != "git-user" {
		t.Errorf("Expected current repo name 'git-user', got %q", repoName)
	}
}

func TestIsInstalled(t *testing.T) {
	// Git should be installed for this project to eng
	if !git.IsInstalled() {
		t.Error("IsInstalled() = false, but git should be available")
	}
}

func TestIsInRepo(t *testing.T) {
	// This test depends on whether we're in a git repo
	// Just verify it doesn't panic
	_ = git.IsInRepo()
}
func TestApply(t *testing.T) {
	dir := testutil.Sandbox(t)

	// Test applying new config
	testName := "tester"
	testEmail := "tester@example.com"

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

	// The write must land in the sandboxed git config, never the real one.
	data, err := os.ReadFile(filepath.Join(dir, ".gitconfig"))
	if err != nil {
		t.Fatalf("reading sandboxed .gitconfig: %v", err)
	}
	if !strings.Contains(string(data), testName) {
		t.Errorf("sandboxed .gitconfig missing applied identity:\n%s", data)
	}
}

func TestConfigureSSH(t *testing.T) {
	testutil.Sandbox(t)
	testKeyPath := filepath.Join(t.TempDir(), "test_key")

	if err := git.ConfigureSSH(testKeyPath); err != nil {
		t.Fatalf("ConfigureSSH() failed: %v", err)
	}

	if got := git.CurrentSSHCommand(); !strings.Contains(got, testKeyPath) {
		t.Errorf("CurrentSSHCommand() = %q, want it to contain %q", got, testKeyPath)
	}

	// Clean up
	if err := git.RemoveSSHConfig(); err != nil {
		t.Fatalf("RemoveSSHConfig() failed: %v", err)
	}
}

func TestRemoveSSHConfig(t *testing.T) {
	testutil.Sandbox(t)

	// Set a test SSH config
	testKeyPath := filepath.Join(t.TempDir(), "test_key")
	if err := git.ConfigureSSH(testKeyPath); err != nil {
		t.Fatalf("ConfigureSSH() failed: %v", err)
	}

	// Remove it
	if err := git.RemoveSSHConfig(); err != nil {
		t.Fatalf("RemoveSSHConfig() failed: %v", err)
	}

	if got := git.CurrentSSHCommand(); got != "" {
		t.Errorf("CurrentSSHCommand() = %q after remove, want empty", got)
	}
}

func TestConfigureSigning(t *testing.T) {
	dir := testutil.Sandbox(t)
	testKeyPath := filepath.Join(t.TempDir(), "test_key")

	if err := git.ConfigureSigning(testKeyPath, "ssh"); err != nil {
		t.Fatalf("ConfigureSigning() failed: %v", err)
	}

	if got := git.CurrentSigningKey(); got != testKeyPath {
		t.Errorf("CurrentSigningKey() = %q, want %q", got, testKeyPath)
	}
	if got := git.CurrentSignFormat(); got != "ssh" {
		t.Errorf("CurrentSignFormat() = %q, want %q", got, "ssh")
	}
	if got := git.CurrentCommitGPGSign(); got != "true" {
		t.Errorf("CurrentCommitGPGSign() = %q, want %q", got, "true")
	}

	// Clean up
	git.RemoveSigningConfig()

	if got := git.CurrentSigningKey(); got != "" {
		t.Errorf("Expected empty signing key after remove, got %q", got)
	}

	// Nothing may have escaped the sandbox.
	if data, err := os.ReadFile(filepath.Join(dir, ".gitconfig")); err == nil {
		if strings.Contains(string(data), "/private/") || strings.Contains(string(data), "/Users/") {
			t.Errorf("sandboxed gitconfig unexpectedly contains real paths:\n%s", data)
		}
	}
}
