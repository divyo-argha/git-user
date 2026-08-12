package cli

import (
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func TestRunRekey_CancelledAndMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("GIT_USER_CONFIG", filepath.Join(tmpDir, ".git-users.json"))

	store := &config.Store{
		Users: []config.User{
			{Name: "tester", Email: "test@example.com"},
		},
	}
	if err := config.Save(store); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	// Test missing name argument
	if err := runRekey(nil); err == nil {
		t.Fatal("expected error on missing name argument")
	}

	// Test identity not found
	if err := runRekey([]string{"nonexistent"}); err == nil {
		t.Fatal("expected error when identity not found")
	}

	// Test user cancels rotation prompt
	oldConfirmFn := ui.ConfirmFn
	defer func() { ui.ConfirmFn = oldConfirmFn }()

	confirmed := false
	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		return false
	}

	err := runRekey([]string{"tester"})
	if err != nil {
		t.Fatalf("runRekey returned error when cancelled: %v", err)
	}
	if confirmed {
		t.Fatal("expected rotation to be cancelled")
	}
}
