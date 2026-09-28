package tui

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/git"
)

// setupFixRemoteTestRepo creates a fresh git repo in a temp dir, chdir's into
// it, and returns a cleanup func that restores the original working
// directory. Mirrors internal/cli/fixremote_test.go's setup so behavior stays
// directly comparable between the CLI and TUI paths.
func setupFixRemoteTestRepo(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	withTempConfig(t)
	_ = exec.Command("git", "config", "--global", "--add", "safe.directory", "*").Run()

	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	if err := exec.Command("git", "init", "-q").Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
}

// TestOpFixRemote_NonDestructive guards against opFixRemote reverting to the
// old behavior of running `git remote set-url` (which permanently discards
// the HTTPS URL from .git/config). It must instead set a local
// url.<ssh>.insteadOf rule per host so push, pull, and fetch all
// transparently route over SSH while the stored remote URL stays untouched
// and the fix stays reversible (RemoveInsteadOf) without touching the remote
// again.
func TestOpFixRemote_NonDestructive(t *testing.T) {
	setupFixRemoteTestRepo(t)

	originalURL := "https://github.com/divyo-argha/git-user.git"
	if err := exec.Command("git", "remote", "add", "origin", originalURL).Run(); err != nil {
		t.Fatalf("git remote add: %v", err)
	}

	res, err := opFixRemote()
	if err != nil {
		t.Fatalf("opFixRemote: %v", err)
	}
	if !strings.Contains(res.detail, "routed over SSH") {
		t.Errorf("expected report to mention SSH routing, got:\n%s", res.detail)
	}

	// The stored URL itself must be untouched — this is what makes it
	// reversible, unlike the old `git remote set-url` rewrite.
	stored, err := exec.Command("git", "config", "--local", "--get", "remote.origin.url").Output()
	if err != nil {
		t.Fatalf("reading stored remote url: %v", err)
	}
	if strings.TrimSpace(string(stored)) != originalURL {
		t.Errorf("expected stored remote.origin.url to remain %q, got %q", originalURL, strings.TrimSpace(string(stored)))
	}

	// Push must now resolve to SSH.
	pushURL, err := git.GetPushRemoteURL("origin")
	if err != nil {
		t.Fatalf("GetPushRemoteURL: %v", err)
	}
	if pushURL != "git@github.com:divyo-argha/git-user.git" {
		t.Errorf("expected push URL to resolve to SSH, got %q", pushURL)
	}

	// Fetch must now also resolve to SSH — the whole point of using plain
	// insteadOf instead of pushInsteadOf, so pull/fetch/clone are covered
	// too, not just push.
	fetchURL, err := git.GetRemoteURL("origin")
	if err != nil {
		t.Fatalf("GetRemoteURL: %v", err)
	}
	if fetchURL != "git@github.com:divyo-argha/git-user.git" {
		t.Errorf("expected fetch URL to also resolve to SSH, got %q", fetchURL)
	}

	if git.HasHTTPSPushRemotes() {
		t.Error("expected HasHTTPSPushRemotes to be false after opFixRemote")
	}
}

// TestOpFixRemote_AlreadySSH guards against a misleading "fixed" report when
// there's genuinely nothing to do.
func TestOpFixRemote_AlreadySSH(t *testing.T) {
	setupFixRemoteTestRepo(t)

	if err := exec.Command("git", "remote", "add", "origin", "git@github.com:divyo-argha/git-user.git").Run(); err != nil {
		t.Fatalf("git remote add: %v", err)
	}

	res, err := opFixRemote()
	if err != nil {
		t.Fatalf("opFixRemote: %v", err)
	}
	if !strings.Contains(res.detail, "already use SSH") {
		t.Errorf("expected an 'already use SSH' report, got:\n%s", res.detail)
	}
}

// TestOpFixRemote_MultipleRemotesSameHost guards against configuring the
// same host's insteadOf rule more than once when several remotes point at
// it (e.g. origin + upstream both on github.com) — git config would accept
// duplicate values, but the report should still describe exactly one fix.
func TestOpFixRemote_MultipleRemotesSameHost(t *testing.T) {
	setupFixRemoteTestRepo(t)

	if err := exec.Command("git", "remote", "add", "origin", "https://github.com/divyo-argha/git-user.git").Run(); err != nil {
		t.Fatalf("git remote add origin: %v", err)
	}
	if err := exec.Command("git", "remote", "add", "upstream", "https://github.com/someone-else/git-user.git").Run(); err != nil {
		t.Fatalf("git remote add upstream: %v", err)
	}

	res, err := opFixRemote()
	if err != nil {
		t.Fatalf("opFixRemote: %v", err)
	}
	if strings.Count(res.detail, "now routed over SSH") != 1 {
		t.Errorf("expected exactly one host to be configured (both remotes share github.com), got report:\n%s", res.detail)
	}
}
