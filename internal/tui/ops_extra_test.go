package tui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestOpHookInstallUninstall(t *testing.T) {
	withTempRepo(t)

	res, err := opHook("install")
	if err != nil {
		t.Fatalf("opHook(install) failed: %v", err)
	}
	for _, hook := range []string{"pre-commit", "pre-push", "post-merge"} {
		if !strings.Contains(res.detail, hook) {
			t.Errorf("expected install report to mention %q, got: %s", hook, res.detail)
		}
	}

	// Installing again should recognize the existing git-user hooks rather
	// than erroring or duplicating them.
	res, err = opHook("install")
	if err != nil {
		t.Fatalf("second opHook(install) failed: %v", err)
	}
	if !strings.Contains(res.detail, "already installed") {
		t.Errorf("expected second install to report already-installed hooks, got: %s", res.detail)
	}

	res, err = opHook("uninstall")
	if err != nil {
		t.Fatalf("opHook(uninstall) failed: %v", err)
	}
	for _, hook := range []string{"pre-commit", "pre-push", "post-merge"} {
		if !strings.Contains(res.detail, hook+" hook removed") {
			t.Errorf("expected uninstall report to mention removing %q, got: %s", hook, res.detail)
		}
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

func TestOpSwitchSession_ShellFormatting(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{Users: []config.User{{Name: "alice", Email: "alice@example.com"}}}

	// Test CMD shell
	t.Setenv("SHELL", "")
	t.Setenv("BASH", "")
	t.Setenv("MSYSTEM", "")
	t.Setenv("PSModulePath", "")
	t.Setenv("PROMPT", "$P$G")
	res, err := opSwitchSession(store, "alice")
	if err != nil {
		t.Fatalf("opSwitchSession for CMD: %v", err)
	}
	if !strings.Contains(res.detail, "gu alice") {
		t.Errorf("expected 'gu alice' for CMD, got:\n%s", res.detail)
	}

	// Test PowerShell
	t.Setenv("PROMPT", "")
	t.Setenv("PSModulePath", `C:\Program Files\PowerShell\Modules`)
	res, err = opSwitchSession(store, "alice")
	if err != nil {
		t.Fatalf("opSwitchSession for PowerShell: %v", err)
	}
	if !strings.Contains(res.detail, "Invoke-Expression (& git-user env alice --pwsh)") {
		t.Errorf("expected Invoke-Expression for PowerShell, got:\n%s", res.detail)
	}

	// Test Fish
	t.Setenv("PSModulePath", "")
	t.Setenv("SHELL", "/usr/bin/fish")
	res, err = opSwitchSession(store, "alice")
	if err != nil {
		t.Fatalf("opSwitchSession for Fish: %v", err)
	}
	if !strings.Contains(res.detail, "git-user env alice --fish | source") {
		t.Errorf("expected fish source for Fish, got:\n%s", res.detail)
	}

	// Test Posix / Bash
	t.Setenv("SHELL", "/bin/bash")
	res, err = opSwitchSession(store, "alice")
	if err != nil {
		t.Fatalf("opSwitchSession for Bash: %v", err)
	}
	if !strings.Contains(res.detail, `eval "$(git-user env alice)"`) {
		t.Errorf("expected eval for Bash, got:\n%s", res.detail)
	}
}


// fakeSSH puts an `ssh` stub first on PATH that records its arguments and
// fails, so a test can see which key a clone tried to authenticate with
// without any network access. Returns the log file path.
func fakeSSH(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell stub")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "ssh.log")
	script := "#!/bin/sh\necho \"$@\" >> " + log + "\nexit 255\n"
	if err := os.WriteFile(filepath.Join(dir, "ssh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIT_SSH_COMMAND", "") // make sure only git-user's setting is in play
	t.Setenv("GIT_SSH", "")
	return log
}

// The clone itself must authenticate with the chosen identity's key, not
// whatever key the agent happens to offer.
func TestOpCloneUsesIdentityKey(t *testing.T) {
	withTempConfig(t)
	log := fakeSSH(t)

	workKey := filepath.Join(t.TempDir(), "id_work")
	store := &config.Store{}
	for _, n := range []string{"work", "home"} {
		if err := store.AddUser(n, n+"@example.com"); err != nil {
			t.Fatal(err)
		}
	}
	store.FindUser("work").SSHKey = workKey
	store.FindUser("home").SSHKey = filepath.Join(t.TempDir(), "id_home")

	dest := filepath.Join(t.TempDir(), "repo")
	// The stub ssh fails, so the clone errors — what matters is what it was asked to use.
	if _, err := opClone(store, "git@example.com:org/repo.git", dest, "work", false); err == nil {
		t.Fatal("expected the stubbed ssh to make the clone fail")
	}

	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("ssh was never invoked: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, workKey) || !strings.Contains(got, "IdentitiesOnly=yes") {
		t.Errorf("clone should use the work key with IdentitiesOnly, ssh args were: %s", got)
	}
	if strings.Contains(got, "id_home") {
		t.Errorf("clone must not use another identity's key: %s", got)
	}
}

// An identity with a custom ssh command keeps it for the clone too.
func TestOpCloneHonoursCustomSSHCommand(t *testing.T) {
	withTempConfig(t)
	log := fakeSSH(t)

	store := &config.Store{}
	if err := store.AddUser("work", "work@example.com"); err != nil {
		t.Fatal(err)
	}
	store.FindUser("work").SSHCommand = "ssh -o Marker=custom-cmd"

	_, _ = opClone(store, "git@example.com:org/repo.git", filepath.Join(t.TempDir(), "r"), "work", false)
	data, _ := os.ReadFile(log)
	if !strings.Contains(string(data), "Marker=custom-cmd") {
		t.Errorf("custom ssh command should be used for the clone, got: %s", data)
	}
}

// An identity without a key leaves ssh to its defaults rather than pinning one.
func TestOpCloneWithoutKeyDoesNotPinSSH(t *testing.T) {
	withTempConfig(t)
	log := fakeSSH(t)

	store := &config.Store{}
	if err := store.AddUser("plain", "plain@example.com"); err != nil {
		t.Fatal(err)
	}
	_, _ = opClone(store, "git@example.com:org/repo.git", filepath.Join(t.TempDir(), "r"), "plain", false)
	data, _ := os.ReadFile(log)
	if strings.Contains(string(data), "IdentitiesOnly") {
		t.Errorf("no key means no pinning, got: %s", data)
	}
}
