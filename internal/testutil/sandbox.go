package testutil

import (
	"path/filepath"
	"testing"
)

// Sandbox redirects HOME, the git-user config path, the global git config and
// the SSH agent socket to a fresh temporary directory so tests can never read
// or write the developer's real configuration. It returns the sandbox dir.
func Sandbox(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("GIT_USER_CONFIG", filepath.Join(dir, ".git-users", "config.json"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, ".gitconfig"))
	t.Setenv("SSH_AUTH_SOCK", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, ".local", "share"))
	return dir
}
