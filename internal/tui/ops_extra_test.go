package tui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/keyring"
)

func TestRepoDirName(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"git@github.com:foo/bar.git", "bar"},
		{"https://github.com/foo/baz.git", "baz"},
		{"https://github.com/foo/qux", "qux"},
		{"git@github.com:foo/bar", "bar"},
	}
	for _, c := range cases {
		if got := repoDirName(c.url); got != c.want {
			t.Errorf("repoDirName(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

func TestOpConfigListSetUnset(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{Users: []config.User{{Name: "eng", Email: "eng@corp.com"}}}

	res, err := opConfigList(store, "eng")
	if err != nil {
		t.Fatalf("opConfigList empty: %v", err)
	}
	if !res.showReport {
		t.Error("expected report for empty config list")
	}

	if _, err := opConfigSet(store, "eng", "init.defaultBranch", "main"); err != nil {
		t.Fatalf("opConfigSet: %v", err)
	}
	u := store.FindUser("eng")
	if u.CustomConfig["init.defaultBranch"] != "main" {
		t.Error("expected custom config to be set")
	}

	res, err = opConfigList(store, "eng")
	if err != nil {
		t.Fatalf("opConfigList: %v", err)
	}
	if res.detail == "" || res.detail == "No custom config keys set" {
		t.Error("expected config list to show the set key")
	}

	if _, err := opConfigSet(store, "missing", "k", "v"); err == nil {
		t.Error("expected error setting config for missing identity")
	}
	if _, err := opConfigSet(store, "eng", "", "v"); err == nil {
		t.Error("expected error with empty key")
	}

	if _, err := opConfigUnset(store, "eng", "init.defaultBranch"); err != nil {
		t.Fatalf("opConfigUnset: %v", err)
	}
	if _, ok := store.FindUser("eng").CustomConfig["init.defaultBranch"]; ok {
		t.Error("expected key to be unset")
	}
	if _, err := opConfigUnset(store, "eng", ""); err == nil {
		t.Error("expected error unsetting empty key")
	}
}

func TestHookHelpers(t *testing.T) {
	if isGitUserHook([]byte("#!/bin/sh\n# git-user identity verification hook")) != true {
		t.Error("expected git-user hook content to be recognized")
	}
	if isGitUserHook([]byte("#!/bin/sh\n# some other hook")) {
		t.Error("expected non git-user hook to be rejected")
	}
	if got := stringsTrimNewline("abc\n"); got != "abc" {
		t.Errorf("stringsTrimNewline = %q", got)
	}
	if got := stringsTrimNewline("abc"); got != "abc" {
		t.Errorf("stringsTrimNewline without newline = %q", got)
	}
}

func TestFindUserByEmail(t *testing.T) {
	store := &config.Store{Users: []config.User{{Name: "eng", Email: "eng@corp.com"}}}
	if store.FindUserByEmail("eng@corp.com") == nil {
		t.Error("expected user found by email")
	}
	if store.FindUserByEmail("nope@corp.com") != nil {
		t.Error("expected nil for unknown email")
	}
}

func TestOpCloneRequiresIdentity(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{}
	_, err := opClone(store, "git@github.com:foo/bar.git", "", "missing", false)
	if err == nil {
		t.Error("expected error cloning with unknown identity")
	}
}

func TestOpHookUnknownAction(t *testing.T) {
	_, err := opHook("bogus")
	if err == nil {
		t.Error("expected error for unknown hook action")
	}
}

func TestOpStatsNotInRepo(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{}
	// Run from the temp config dir (not a git repo) to assert the guard.
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)
	_, err := opStats(store)
	if err == nil {
		t.Error("expected error when not in a git repository")
	}
}

func TestOpSyncNotConfigured(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{}
	_, err := opSync(store, "", "")
	if err == nil {
		t.Error("expected error when sync is not configured")
	}
}

// TestOpSyncWarnsWhenKeychainStoreFails guards against a regression where a
// failed keyring.KeyringSet during first-time sync setup was discarded via a
// literal `if err != nil { _ = err }` no-op — sync would appear to configure
// successfully with no indication the passphrase was never actually
// persisted, so a later plain `sync` (no passphrase) fails on
// keyring.KeyringGet with an unexplained "passphrase required".
func TestOpSyncWarnsWhenKeychainStoreFails(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	withTempConfig(t)
	dir := os.Getenv("HOME")

	remoteRepoDir := filepath.Join(dir, "remote-backup-repo")
	if err := os.Mkdir(remoteRepoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "init", "--bare", remoteRepoDir).Run(); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}
	_ = exec.Command("git", "config", "--global", "user.name", "Test User").Run()
	_ = exec.Command("git", "config", "--global", "user.email", "test@example.com").Run()

	oldSet := keyring.KeyringSet
	keyring.KeyringSet = func(service, user, password string) error {
		return errors.New("mock keychain unavailable")
	}
	t.Cleanup(func() { keyring.KeyringSet = oldSet })

	store, _ := config.Load()
	res, err := opSync(store, remoteRepoDir, "secretpass")
	if err != nil {
		t.Fatalf("opSync failed: %v", err)
	}
	if !strings.Contains(res.detail, "Could not store the sync passphrase") {
		t.Errorf("expected an explicit warning about the failed keychain store, got detail:\n%s", res.detail)
	}
}

func TestOpConfigMissingIdentity(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{}
	if _, err := opConfigList(store, "nope"); err == nil {
		t.Error("expected error for missing identity")
	}
	if _, err := opConfigSet(store, "nope", "k", "v"); err == nil {
		t.Error("expected error for missing identity")
	}
	if _, err := opConfigUnset(store, "nope", "k"); err == nil {
		t.Error("expected error for missing identity")
	}
}

func TestRunCapturedCapturesOutput(t *testing.T) {
	out, err := runCaptured("", "sh", "-c", "echo hello&&echo err 1>&2&&exit 0")
	if err != nil {
		t.Fatalf("runCaptured: %v", err)
	}
	if !strings.Contains(out, "hello") || !strings.Contains(out, "err") {
		t.Errorf("expected both stdout and stderr captured, got %q", out)
	}
}

func TestRunCapturedErrors(t *testing.T) {
	_, err := runCaptured("", "sh", "-c", "echo boom&&false")
	if err == nil {
		t.Error("expected error for failing command")
	}
}

func TestRunCapturedSetsTerminalPromptDisabled(t *testing.T) {
	out, err := runCaptured("", "sh", "-c", "printf '%s' \"$GIT_TERMINAL_PROMPT\"")
	if err != nil {
		t.Fatalf("runCaptured: %v", err)
	}
	if out != "0" {
		t.Errorf("expected GIT_TERMINAL_PROMPT=0, got %q", out)
	}
}
