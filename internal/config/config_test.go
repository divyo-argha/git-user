package config_test

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
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

func TestBindSSHKeyClearsPreservedCommand(t *testing.T) {
	s := &config.Store{}
	_ = s.AddUser("eve", "eve@example.com")
	u := s.FindUser("eve")
	u.SSHCommand = "ssh -i /old/key -o SomeFlag"
	u.SSHKey = "/old/key"

	if err := s.BindSSHKey("eve", "/new/key"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.SSHKey != "/new/key" {
		t.Errorf("expected ssh key /new/key, got %s", u.SSHKey)
	}
	if u.SSHCommand != "" {
		t.Errorf("expected preserved sshCommand to be cleared on re-bind, got %q", u.SSHCommand)
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
	t.Setenv("GIT_USER_CONFIG", path)

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
	nonexistentPath := filepath.Join(dir, "nonexistent.json")
	t.Setenv("GIT_USER_CONFIG", nonexistentPath)
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
	t.Setenv("GIT_USER_CONFIG", filepath.Join(dir, "config.json"))

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

func TestSaveGuardDetectsExternalChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("GIT_USER_CONFIG", path)

	// Load, then modify the file behind the store's back.
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := loaded.AddUser("alice", "alice@example.com"); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	if err := config.Save(loaded); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}

	// Simulate another session editing the file (e.g. removing a profile or
	// restoring an old snapshot).
	external, err := config.Load()
	if err != nil {
		t.Fatalf("external Load failed: %v", err)
	}
	if err := external.RemoveUser("alice", true); err != nil {
		t.Fatalf("external RemoveUser failed: %v", err)
	}
	if err := config.Save(external); err != nil {
		t.Fatalf("external Save failed: %v", err)
	}

	// The original store is now stale — saving it must be refused so it cannot
	// resurrect alice (or any profile the external session removed).
	if err := loaded.AddUser("bob", "bob@example.com"); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	if err := config.Save(loaded); !errors.Is(err, config.ErrConfigChanged) {
		t.Fatalf("expected ErrConfigChanged, got %v", err)
	}

	// The on-disk state must be untouched.
	onDisk, err := config.Load()
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if onDisk.FindUser("bob") != nil {
		t.Error("stale store wrote to disk despite guard")
	}
	if onDisk.FindUser("alice") != nil {
		t.Error("stale store resurrected removed profile")
	}
}

func TestSaveGuardAllowsConsecutiveSaves(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("GIT_USER_CONFIG", path)

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := loaded.AddUser("carol", "carol@example.com"); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	// Saving the same loaded store twice must succeed (the hash is updated on
	// each save, so consecutive saves by the same session stay valid).
	for i := 0; i < 2; i++ {
		if err := config.Save(loaded); err != nil {
			t.Fatalf("Save #%d failed: %v", i+1, err)
		}
	}
}

func TestSaveGuardSkipsProgrammaticStores(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("GIT_USER_CONFIG", path)

	// Stores built without Load() have no baseline and are allowed to write.
	s := &config.Store{}
	if err := s.AddUser("dave", "dave@example.com"); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	if err := config.Save(s); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
}

func TestSyncIncludeIfsHonorsEnvConfigPath(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	t.Setenv("GIT_USER_CONFIG", filepath.Join(dir, "config.json"))

	s := &config.Store{}
	_ = s.AddUser("work", "work@corp.com")
	_ = s.BindPathToUser("work", dir)

	if err := config.Save(s); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// The profile snippet must be written into the directory that ConfigPath()
	// resolves to (honoring GIT_USER_CONFIG), not the default ~/.git-users dir.
	snippet := filepath.Join(dir, "profile-work.gitconfig")
	if _, err := os.Stat(snippet); err != nil {
		t.Errorf("expected profile-work.gitconfig in env config dir: %v", err)
	}

	// Clean up the includeIf entry added to the real global gitconfig.
	key := "includeif.gitdir/i:" + dir + "/.path"
	_ = exec.Command("git", "config", "--global", "--unset-all", key).Run()
	_ = os.Remove(snippet)
}
