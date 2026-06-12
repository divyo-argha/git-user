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
