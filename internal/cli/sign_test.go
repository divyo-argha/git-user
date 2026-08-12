package cli

import (
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/testutil"
)

func TestRunSignEnable(t *testing.T) {
testutil.Sandbox(t)
		dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("GIT_USER_CONFIG", path)

	store := &config.Store{}
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.BindSSHKey("dev", "/private/dev/.ssh/id_ed25519")
	config.Save(store)

	// Test enabling signing with bound SSH key automatically
	args := []string{"dev", "--on"}
	if err := runSign(args); err != nil {
		t.Fatalf("runSign failed: %v", err)
	}

	loaded, _ := config.Load()
	u := loaded.FindUser("dev")
	if u.SignKey != "/private/dev/.ssh/id_ed25519" || u.SignFormat != "ssh" || u.SignDisabled {
		t.Errorf("expected signing to be enabled with ssh key, got %v", u)
	}
}

func TestRunSignDisable(t *testing.T) {
testutil.Sandbox(t)
		dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("GIT_USER_CONFIG", path)

	store := &config.Store{}
	_ = store.AddUser("ops", "ops@example.com")
	_ = store.SetSigningKey("ops", "key_123", "gpg")
	config.Save(store)

	args := []string{"ops", "--off"}
	if err := runSign(args); err != nil {
		t.Fatalf("runSign failed: %v", err)
	}

	loaded, _ := config.Load()
	u := loaded.FindUser("ops")
	if !u.SignDisabled {
		t.Errorf("expected signing to be disabled, got %v", u)
	}
}

func TestRunSignExplicitKey(t *testing.T) {
testutil.Sandbox(t)
		dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("GIT_USER_CONFIG", path)

	store := &config.Store{}
	_ = store.AddUser("qa", "qa@example.com")
	config.Save(store)

	args := []string{"qa", "--key", "ABCD1234EFGH", "--format", "gpg"}
	if err := runSign(args); err != nil {
		t.Fatalf("runSign failed: %v", err)
	}

	loaded, _ := config.Load()
	u := loaded.FindUser("qa")
	if u.SignKey != "ABCD1234EFGH" || u.SignFormat != "gpg" || u.SignDisabled {
		t.Errorf("expected explicit gpg key to be set, got %v", u)
	}
}
