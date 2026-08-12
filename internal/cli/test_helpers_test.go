package cli

import (
	"os/exec"
	"testing"

	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/testutil"
	"github.com/divyo-argha/git-user/internal/ui"
	zalando "github.com/zalando/go-keyring"
)

// setupTestEnv initializes a temporary HOME directory and redirects the git-user
// config path to isolate testing. It cleans up the environment automatically.
func setupTestEnv(t *testing.T) string {
	t.Helper()
	tmpDir := testutil.Sandbox(t)

	// Configure safe directory on the redirected HOME environment
	_ = exec.Command("git", "config", "--global", "--add", "safe.directory", "*").Run()

	// Save original keyring functions
	oldKeyringGet := keyring.KeyringGet
	oldKeyringSet := keyring.KeyringSet
	oldKeyringDelete := keyring.KeyringDelete

	// Mock keyring library
	mockKeyring := make(map[string]string)
	keyring.KeyringGet = func(service, user string) (string, error) {
		val, ok := mockKeyring[service+"/"+user]
		if !ok {
			return "", zalando.ErrNotFound
		}
		return val, nil
	}
	keyring.KeyringSet = func(service, user, password string) error {
		mockKeyring[service+"/"+user] = password
		return nil
	}
	keyring.KeyringDelete = func(service, user string) error {
		if _, ok := mockKeyring[service+"/"+user]; !ok {
			return zalando.ErrNotFound
		}
		delete(mockKeyring, service+"/"+user)
		return nil
	}
	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		return defaultYes
	}
	ui.SelectFn = func(label string, options []string) (int, error) {
		return 0, nil
	}
	readPassphraseFn = func(prompt string) (string, error) {
		return "", nil
	}

	// Reset mocked functions on cleanup
	t.Cleanup(func() {
		ui.PromptFn = nil
		ui.SelectFn = nil
		ui.ConfirmFn = nil
		readPassphraseFn = nil
		keyring.KeyringGet = oldKeyringGet
		keyring.KeyringSet = oldKeyringSet
		keyring.KeyringDelete = oldKeyringDelete
	})

	return tmpDir
}
