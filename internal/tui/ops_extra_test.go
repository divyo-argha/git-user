package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
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
	store := &config.Store{Users: []config.User{{Name: "work", Email: "work@corp.com"}}}

	res, err := opConfigList(store, "work")
	if err != nil {
		t.Fatalf("opConfigList empty: %v", err)
	}
	if !res.showReport {
		t.Error("expected report for empty config list")
	}

	if _, err := opConfigSet(store, "work", "init.defaultBranch", "main"); err != nil {
		t.Fatalf("opConfigSet: %v", err)
	}
	u := store.FindUser("work")
	if u.CustomConfig["init.defaultBranch"] != "main" {
		t.Error("expected custom config to be set")
	}

	res, err = opConfigList(store, "work")
	if err != nil {
		t.Fatalf("opConfigList: %v", err)
	}
	if res.detail == "" || res.detail == "No custom config keys set" {
		t.Error("expected config list to show the set key")
	}

	if _, err := opConfigSet(store, "missing", "k", "v"); err == nil {
		t.Error("expected error setting config for missing identity")
	}
	if _, err := opConfigSet(store, "work", "", "v"); err == nil {
		t.Error("expected error with empty key")
	}

	if _, err := opConfigUnset(store, "work", "init.defaultBranch"); err != nil {
		t.Fatalf("opConfigUnset: %v", err)
	}
	if _, ok := store.FindUser("work").CustomConfig["init.defaultBranch"]; ok {
		t.Error("expected key to be unset")
	}
	if _, err := opConfigUnset(store, "work", ""); err == nil {
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
	store := &config.Store{Users: []config.User{{Name: "work", Email: "work@corp.com"}}}
	if store.FindUserByEmail("work@corp.com") == nil {
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
