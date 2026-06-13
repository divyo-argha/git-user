package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestAddAndFind(t *testing.T) {
	s := &config.Store{}
	if err := s.AddUser("alice", "alice@example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	u := s.FindUser("alice")
	if u == nil {
		t.Fatal("user not found after add")
	}
	if u.Email != "alice@example.com" {
		t.Errorf("expected alice@example.com, got %s", u.Email)
	}
}

func TestDuplicateAdd(t *testing.T) {
	s := &config.Store{}
	_ = s.AddUser("bob", "bob@example.com")
	if err := s.AddUser("bob", "bob2@example.com"); err == nil {
		t.Fatal("expected duplicate error, got nil")
	}
}

func TestRemoveActive(t *testing.T) {
	s := &config.Store{}
	_ = s.AddUser("carol", "carol@example.com")
	_ = s.SetCurrent("carol")

	if err := s.RemoveUser("carol", false); err == nil {
		t.Fatal("expected error removing active user without force")
	}
	if err := s.RemoveUser("carol", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.FindUser("carol") != nil {
		t.Fatal("user still present after force remove")
	}
	if s.Current != "" {
		t.Fatal("current should be cleared after removing active user")
	}
}

func TestUpdateUser(t *testing.T) {
	s := &config.Store{}
	_ = s.AddUser("dave", "old@example.com")
	if err := s.UpdateUser("dave", "new@example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u := s.FindUser("dave"); u.Email != "new@example.com" {
		t.Errorf("expected new@example.com, got %s", u.Email)
	}
}

func TestBindSSHKey(t *testing.T) {
	s := &config.Store{}
	_ = s.AddUser("eve", "eve@example.com")
	if err := s.BindSSHKey("eve", "/home/eve/.ssh/id_ed25519"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u := s.FindUser("eve"); u.SSHKey != "/home/eve/.ssh/id_ed25519" {
		t.Errorf("unexpected ssh key: %s", u.SSHKey)
	}
}

func TestSigningConfig(t *testing.T) {
	s := &config.Store{}
	_ = s.AddUser("eve", "eve@example.com")
	
	if err := s.SetSigningKey("eve", "key_123", "ssh"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	u := s.FindUser("eve")
	if u.SignKey != "key_123" || u.SignFormat != "ssh" || u.SignDisabled {
		t.Errorf("signing key setup failed: %v", u)
	}

	if err := s.ToggleSigning("eve", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !u.SignDisabled {
		t.Error("expected sign disabled")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	s := &config.Store{}
	_ = s.AddUser("frank", "frank@example.com")
	_ = s.SetCurrent("frank")

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var loaded config.Store
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if loaded.Current != "frank" {
		t.Errorf("expected current=frank, got %s", loaded.Current)
	}
	if u := loaded.FindUser("frank"); u == nil || u.Email != "frank@example.com" {
		t.Error("user not preserved across save/load")
	}
}

func TestRealSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	config.SetConfigPath(path)

	if config.ConfigPath() != path {
		t.Errorf("ConfigPath() = %s, want %s", config.ConfigPath(), path)
	}

	s := &config.Store{}
	_ = s.AddUser("grace", "grace@example.com")
	_ = s.SetCurrent("grace")

	if err := config.Save(s); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Current != "grace" {
		t.Errorf("expected current to be grace, got %s", loaded.Current)
	}

	// Test loading non-existent config path returns empty store
	config.SetConfigPath(filepath.Join(dir, "nonexistent.json"))
	nonexistent, err := config.Load()
	if err != nil {
		t.Fatalf("Load on nonexistent path should succeed, got error: %v", err)
	}
	if len(nonexistent.Users) != 0 {
		t.Errorf("expected empty store on nonexistent file load, got users count %d", len(nonexistent.Users))
	}
}

func TestTempProfile(t *testing.T) {
	dir := t.TempDir()
	config.SetConfigPath(filepath.Join(dir, "config.json"))

	// Create original temp dir inside test so we don't clobber OS temp
	oldTemp := os.Getenv("TMPDIR")
	defer os.Setenv("TMPDIR", oldTemp)
	os.Setenv("TMPDIR", dir)

	// In test, TempDir() relies on env var or falls back
	// But os.TempDir() caches its result if we don't manipulate env early enough.
	// Wait, instead of hacking env vars, let's just make sure the TempConfigPath doesn't overwrite real temp config?
	// The function `config.TempConfigPath()` uses `os.TempDir()`. If we run tests concurrently, they might collide.
	// To make this testable, we should add `SetTempConfigPath` to `config` or just allow testing.
	// Let's just create a store, add a temp user, save, and verify it's written properly.
	
	s := &config.Store{}
	_ = s.AddUser("perm", "perm@example.com")
	_ = s.AddUser("temp", "temp@example.com")
	
	// Mark temp user
	u := s.FindUser("temp")
	u.IsTemporary = true
	
	if err := config.Save(s); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify perm is in config.json
	data, _ := os.ReadFile(config.ConfigPath())
	var stored config.Store
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("failed to parse config.json: %v", err)
	}
	if stored.FindUser("temp") != nil {
		t.Errorf("temp user should not be in config.json")
	}

	// Verify temp is in temp config
	tempData, _ := os.ReadFile(config.TempConfigPath())
	var tempUsers []config.User
	if err := json.Unmarshal(tempData, &tempUsers); err != nil {
		t.Fatalf("failed to parse temp config: %v", err)
	}
	if len(tempUsers) != 1 || tempUsers[0].Name != "temp" {
		t.Errorf("temp config does not contain temp user")
	}

	// Verify Load merges them
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.FindUser("perm") == nil || loaded.FindUser("temp") == nil {
		t.Errorf("Load did not merge users correctly")
	}
	if !loaded.FindUser("temp").IsTemporary {
		t.Errorf("Loaded temp user missing IsTemporary flag")
	}

	// Cleanup
	config.DeleteTempConfig()
}
