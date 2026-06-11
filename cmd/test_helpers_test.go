package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
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

	// Reset mocked functions and restore HOME on cleanup
	t.Cleanup(func() {
		os.Setenv("HOME", oldHome)
		ui.PromptFn = nil
		ui.SelectFn = nil
		ui.ConfirmFn = nil
		readPassphraseFn = nil
	})

	return tmpDir
}
