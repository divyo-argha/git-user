package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestValidIdentityName(t *testing.T) {
	valid := []string{"work", "oss.alt", "team-2", "a", "Dev_1"}
	for _, name := range valid {
		if !config.ValidIdentityName(name) {
			t.Errorf("expected %q to be valid", name)
		}
	}

	invalid := []string{
		"",
		"../../../.bashrc",
		"/etc/passwd",
		"..",
		".",
		".hidden",
		"-leading-dash",
		"has/slash",
		"has\\backslash",
		"trailing space ",
		"unïcode",
		string(make([]byte, 100)), // way over length, all NUL bytes
	}
	for _, name := range invalid {
		if config.ValidIdentityName(name) {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}

func TestDefaultSSHKeyPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := config.DefaultSSHKeyPath("work")
	if err != nil {
		t.Fatalf("unexpected error for valid name: %v", err)
	}
	want := filepath.Join(home, ".ssh", "git_work")
	if path != want {
		t.Errorf("got %q, want %q", path, want)
	}
}

func TestDefaultSSHKeyPathRejectsTraversal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	for _, name := range []string{"../../../.bashrc", "/etc/passwd", "..", "has/slash"} {
		if path, err := config.DefaultSSHKeyPath(name); err == nil {
			t.Errorf("expected error for %q, got path %q", name, path)
		}
	}
}

func TestSSHKeyPathForFilename(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := config.SSHKeyPathForFilename("git_work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, ".ssh", "git_work")
	if path != want {
		t.Errorf("got %q, want %q", path, want)
	}

	for _, name := range []string{"", "../escape", "a/b", "id.pub", "id.backup", "known_hosts"} {
		if _, err := config.SSHKeyPathForFilename(name); err == nil {
			t.Errorf("expected %q to be rejected as a key filename", name)
		}
	}
}

func TestSuggestSSHKeyFilenameAvoidsCollision(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}

	// No existing file: suggestion is the plain default.
	got, err := config.SuggestSSHKeyFilename("work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "git_work" {
		t.Errorf("expected %q, got %q", "git_work", got)
	}

	// Occupy the default name (as a stale leftover from a renamed/deleted
	// identity would) — the suggestion must skip it, not point back at it,
	// since silently reusing it is exactly the "attaches to previous key"
	// behavior this function exists to avoid steering the user into.
	if err := os.WriteFile(filepath.Join(sshDir, "git_work"), []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = config.SuggestSSHKeyFilename("work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "git_work_2" {
		t.Errorf("expected suggestion to skip the occupied name, got %q", got)
	}
}

func TestListSSHKeyFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}

	// A real key pair: private + matching .pub with a comment.
	if err := os.WriteFile(filepath.Join(sshDir, "git_work"), []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "git_work.pub"), []byte("ssh-ed25519 AAAA... work@corp.com\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Non-key files that happen to live in ~/.ssh: must be skipped since
	// neither has a .pub sibling.
	if err := os.WriteFile(filepath.Join(sshDir, "known_hosts"), []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	keys, err := config.ListSSHKeyFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected exactly 1 key file, got %d: %+v", len(keys), keys)
	}
	if keys[0].Name != "git_work" {
		t.Errorf("expected name %q, got %q", "git_work", keys[0].Name)
	}
	if keys[0].Comment != "work@corp.com" {
		t.Errorf("expected comment %q, got %q", "work@corp.com", keys[0].Comment)
	}
}

func TestAddUserRejectsInvalidName(t *testing.T) {
	s := &config.Store{}
	if err := s.AddUser("../../../.bashrc", "attacker@example.com"); err == nil {
		t.Fatal("expected AddUser to reject a path-traversal name")
	}
	if s.FindUser("../../../.bashrc") != nil {
		t.Fatal("invalid identity should not have been added")
	}
}

func TestRenameUserRejectsInvalidName(t *testing.T) {
	s := &config.Store{}
	_ = s.AddUser("work", "work@example.com")
	if err := s.RenameUser("work", "../../../.bashrc"); err == nil {
		t.Fatal("expected RenameUser to reject a path-traversal name")
	}
	if u := s.FindUser("work"); u == nil || u.Name != "work" {
		t.Fatal("identity should be unchanged after a rejected rename")
	}
}
