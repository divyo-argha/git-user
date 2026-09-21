package testutil

import (
	"path/filepath"
	"testing"
)

// SetHomeDir points os.UserHomeDir() at dir for the duration of the test, on
// every OS: it sets both HOME (consulted on Linux/macOS, and by Go's own
// fallback) and USERPROFILE (consulted on Windows, where HOME is ignored
// entirely). Every test that fakes a home directory must call this (or
// Sandbox, which calls it) instead of setting HOME alone — a bare
// t.Setenv("HOME", ...) is silently a no-op for os.UserHomeDir() on Windows,
// which would let that test run against the real Windows user profile
// instead of its intended sandbox (and, for a test that also exercises a
// destructive path like uninstall/remove, against real files).
func SetHomeDir(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

// Sandbox redirects HOME, the git-user config path, the global git config and
// the SSH agent socket to a fresh temporary directory so tests can never read
// or write the developer's real configuration. It returns the sandbox dir.
func Sandbox(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	SetHomeDir(t, dir)
	t.Setenv("GIT_USER_CONFIG", filepath.Join(dir, ".git-users", "config.json"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, ".gitconfig"))
	t.Setenv("SSH_AUTH_SOCK", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, ".local", "share"))
	return dir
}
