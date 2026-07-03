package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
	"github.com/zalando/go-keyring"
)

// setupTestEnv initializes a temporary HOME directory and redirects the git-user
// config path to isolate testing. It cleans up the environment automatically.
func setupTestEnv(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Redirect HOME and config path
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	configFilePath := filepath.Join(tmpDir, ".git-users", "config.json")
	config.SetConfigPath(configFilePath)

	// Mock keyring library
	mockKeyring := make(map[string]string)
	keyringGet = func(service, user string) (string, error) {
		val, ok := mockKeyring[service+"/"+user]
		if !ok {
			return "", keyring.ErrNotFound
		}
		return val, nil
	}
	keyringSet = func(service, user, password string) error {
		mockKeyring[service+"/"+user] = password
		return nil
	}
	keyringDelete = func(service, user string) error {
		if _, ok := mockKeyring[service+"/"+user]; !ok {
			return keyring.ErrNotFound
		}
		delete(mockKeyring, service+"/"+user)
		return nil
	}
	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		return defaultYes
	}

	// Reset mocked functions and restore HOME on cleanup
	t.Cleanup(func() {
		os.Setenv("HOME", oldHome)
		ui.PromptFn = nil
		ui.SelectFn = nil
		ui.ConfirmFn = nil
		readPassphraseFn = nil
		keyringGet = keyring.Get
		keyringSet = keyring.Set
		keyringDelete = keyring.Delete
	})

	return tmpDir
}
