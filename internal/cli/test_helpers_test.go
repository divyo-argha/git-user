package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ui"
	zalando "github.com/zalando/go-keyring"
)

// setupTestEnv initializes a temporary HOME directory and redirects the git-user
// config path to isolate testing. It cleans up the environment automatically.
func setupTestEnv(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Redirect HOME and config path, and isolate SSH agent
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Setenv("SSH_AUTH_SOCK", "")

	configFilePath := filepath.Join(tmpDir, ".git-users", "config.json")
	config.SetConfigPath(configFilePath)

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

	// Reset mocked functions and restore HOME on cleanup
	t.Cleanup(func() {
		os.Setenv("HOME", oldHome)
		ui.PromptFn = nil
		ui.SelectFn = nil
		ui.ConfirmFn = nil
		readPassphraseFn = nil
		keyring.KeyringGet = keyring.KeyringGet
		keyring.KeyringSet = keyring.KeyringSet
		keyring.KeyringDelete = keyring.KeyringDelete
	})

	return tmpDir
}
