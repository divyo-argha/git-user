package cmd

import (
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestRunSignEnable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	config.SetConfigPath(path)

	store := &config.Store{}
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.BindSSHKey("alice", "/home/alice/.ssh/id_ed25519")
	config.Save(store)

	// Test enabling signing with bound SSH key automatically
	args := []string{"alice", "--on"}
	if err := runSign(args); err != nil {
		t.Fatalf("runSign failed: %v", err)
	}

	loaded, _ := config.Load()
	u := loaded.FindUser("alice")
	if u.SignKey != "/home/alice/.ssh/id_ed25519" || u.SignFormat != "ssh" || u.SignDisabled {
		t.Errorf("expected signing to be enabled with ssh key, got %v", u)
	}
}

func TestRunSignDisable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	config.SetConfigPath(path)

	store := &config.Store{}
	_ = store.AddUser("bob", "bob@example.com")
	_ = store.SetSigningKey("bob", "key_123", "gpg")
	config.Save(store)

	args := []string{"bob", "--off"}
	if err := runSign(args); err != nil {
		t.Fatalf("runSign failed: %v", err)
	}

	loaded, _ := config.Load()
	u := loaded.FindUser("bob")
	if !u.SignDisabled {
		t.Errorf("expected signing to be disabled, got %v", u)
	}
}

func TestRunSignExplicitKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	config.SetConfigPath(path)

	store := &config.Store{}
	_ = store.AddUser("carol", "carol@example.com")
	config.Save(store)

	args := []string{"carol", "--key", "ABCD1234EFGH", "--format", "gpg"}
	if err := runSign(args); err != nil {
		t.Fatalf("runSign failed: %v", err)
	}

	loaded, _ := config.Load()
	u := loaded.FindUser("carol")
	if u.SignKey != "ABCD1234EFGH" || u.SignFormat != "gpg" || u.SignDisabled {
		t.Errorf("expected explicit gpg key to be set, got %v", u)
	}
}
