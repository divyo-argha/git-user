package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func withTempConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	old := config.ConfigPath()
	config.SetConfigPath(filepath.Join(dir, "config.json"))
	t.Cleanup(func() { config.SetConfigPath(old) })
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := expandPath("~/foo")
	want := filepath.Join(home, "foo")
	if got != want {
		t.Errorf("expandPath ~/foo = %q, want %q", got, want)
	}
	if expandPath("/abs/path") != "/abs/path" {
		t.Error("expandPath should leave absolute paths untouched")
	}
}

func TestIsValidEmail(t *testing.T) {
	cases := []struct {
		email string
		want  bool
	}{
		{"user@example.com", true},
		{"user.name+tag@sub.example.co", true},
		{"not-an-email", false},
		{"user@", false},
		{"@example.com", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isValidEmail(c.email); got != c.want {
			t.Errorf("isValidEmail(%q) = %v, want %v", c.email, got, c.want)
		}
	}
}

func TestOpRenameAndChangeEmail(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{
		Current: "work",
		Users: []config.User{
			{Name: "work", Email: "work@corp.com"},
			{Name: "home", Email: "home@personal.com"},
		},
	}

	if err := opRename(store, "work", "work2"); err != nil {
		t.Fatalf("opRename: %v", err)
	}
	if store.Current != "work2" {
		t.Errorf("expected current to follow rename, got %q", store.Current)
	}
	if store.FindUser("work2") == nil {
		t.Error("expected renamed user to exist")
	}
	if err := opRename(store, "work", "home"); err == nil {
		t.Error("expected error renaming to an existing name")
	}

	if err := opChangeEmail(store, "home", "new@home.com"); err != nil {
		t.Fatalf("opChangeEmail: %v", err)
	}
	if store.FindUser("home").Email != "new@home.com" {
		t.Error("expected email to be updated")
	}
	if err := opChangeEmail(store, "home", "work@corp.com"); err == nil {
		t.Error("expected error when email is already used")
	}
}

func TestOpBindPathAndUnbindPath(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{
		Users: []config.User{{Name: "work", Email: "work@corp.com"}},
	}
	dir := t.TempDir()

	if err := opBindPath(store, "work", dir); err != nil {
		t.Fatalf("opBindPath: %v", err)
	}
	if len(store.FindUser("work").BindPaths) != 1 {
		t.Error("expected one bound path")
	}
	if err := opBindPath(store, "work", "definitely/missing/dir"); err == nil {
		t.Error("expected error binding a missing directory")
	}
	if err := opBindPath(store, "work", "/etc/hosts"); err == nil {
		t.Error("expected error binding a file path")
	}

	if err := opUnbindPath(store, "work", dir); err != nil {
		t.Fatalf("opUnbindPath: %v", err)
	}
	if len(store.FindUser("work").BindPaths) != 0 {
		t.Error("expected no bound paths after unbind")
	}
}

func TestOpUnbindAndRemove(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{
		Current: "work",
		Users: []config.User{
			{Name: "work", Email: "work@corp.com", SSHKey: "/fake/key"},
			{Name: "home", Email: "home@personal.com"},
		},
	}

	// Unbind the active identity's key.
	if err := opUnbind(store, "work"); err != nil {
		t.Fatalf("opUnbind: %v", err)
	}
	if store.FindUser("work").SSHKey != "" {
		t.Error("expected SSH key binding to be removed")
	}

	// Remove a non-active identity (avoids clearing real git config in tests).
	key, err := opRemove(store, "home")
	if err != nil {
		t.Fatalf("opRemove: %v", err)
	}
	if key != "" {
		t.Errorf("expected no key for removed user, got %q", key)
	}
	if store.FindUser("home") != nil {
		t.Error("expected home user to be removed")
	}
	if store.Current != "work" {
		t.Error("expected current identity unchanged")
	}
}

func TestOpRemoveInactiveWhileSignedOut(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{
		Current: "",
		Users: []config.User{
			{Name: "home", Email: "home@personal.com"},
		},
	}
	key, err := opRemove(store, "home")
	if err != nil {
		t.Fatalf("opRemove: %v", err)
	}
	if key != "" {
		t.Errorf("expected no key path, got %q", key)
	}
	if store.FindUser("home") != nil {
		t.Error("expected user to be removed")
	}
	if store.Current != "" {
		t.Errorf("expected Current to remain empty, got %q", store.Current)
	}
}

func TestOpRegisterFinishNoKey(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{}
	res, err := opRegisterFinish(store, "temp", "temp@corp.com", true, "", "", false)
	if err != nil {
		t.Fatalf("opRegisterFinish: %v", err)
	}
	if !res.showReport {
		t.Error("expected report screen for registration")
	}
	u := store.FindUser("temp")
	if u == nil {
		t.Fatal("expected user to be created")
	}
	if !u.IsTemporary {
		t.Error("expected temporary flag")
	}
}

func TestNeedsPassphraseForSwitchNoKey(t *testing.T) {
	store := &config.Store{Users: []config.User{{Name: "work", Email: "work@corp.com"}}}
	if needsPassphraseForSwitch(store, "work") {
		t.Error("expected no passphrase needed when no SSH key")
	}
}
