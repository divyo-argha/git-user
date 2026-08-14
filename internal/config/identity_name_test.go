package config_test

import (
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
