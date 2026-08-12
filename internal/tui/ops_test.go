package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/testutil"
)

func withTempConfig(t *testing.T) {
	t.Helper()
	testutil.Sandbox(t)
}

func TestExpandPath(t *testing.T) {
	private, _ := os.UserHomeDir()
	got := expandPath("~/foo")
	want := filepath.Join(private, "foo")
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
		Current: "eng",
		Users: []config.User{
			{Name: "eng", Email: "eng@corp.com"},
			{Name: "private", Email: "private@example.com"},
		},
	}

	if err := opRename(store, "eng", "eng-renamed"); err != nil {
		t.Fatalf("opRename: %v", err)
	}
	if store.Current != "eng-renamed" {
		t.Errorf("expected current to follow rename, got %q", store.Current)
	}
	if store.FindUser("eng-renamed") == nil {
		t.Error("expected renamed user to exist")
	}
	if err := opRename(store, "eng", "private"); err == nil {
		t.Error("expected error renaming to an existing name")
	}

	if err := opChangeEmail(store, "private", "new@private.com"); err != nil {
		t.Fatalf("opChangeEmail: %v", err)
	}
	if store.FindUser("private").Email != "new@private.com" {
		t.Error("expected email to be updated")
	}
	if err := opChangeEmail(store, "private", "eng@corp.com"); err == nil {
		t.Error("expected error when email is already used")
	}
}

func TestOpBindPathAndUnbindPath(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{
		Users: []config.User{{Name: "eng", Email: "eng@corp.com"}},
	}
	dir := t.TempDir()

	if err := opBindPath(store, "eng", dir); err != nil {
		t.Fatalf("opBindPath: %v", err)
	}
	if len(store.FindUser("eng").BindPaths) != 1 {
		t.Error("expected one bound path")
	}
	if err := opBindPath(store, "eng", "definitely/missing/dir"); err == nil {
		t.Error("expected error binding a missing directory")
	}
	if err := opBindPath(store, "eng", "/etc/hosts"); err == nil {
		t.Error("expected error binding a file path")
	}

	if err := opUnbindPath(store, "eng", dir); err != nil {
		t.Fatalf("opUnbindPath: %v", err)
	}
	if len(store.FindUser("eng").BindPaths) != 0 {
		t.Error("expected no bound paths after unbind")
	}
}

func TestOpUnbindAndRemove(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{
		Current: "eng",
		Users: []config.User{
			{Name: "eng", Email: "eng@corp.com", SSHKey: "/fake/key"},
			{Name: "private", Email: "private@example.com"},
		},
	}

	// Unbind the active identity's key.
	if err := opUnbind(store, "eng"); err != nil {
		t.Fatalf("opUnbind: %v", err)
	}
	if store.FindUser("eng").SSHKey != "" {
		t.Error("expected SSH key binding to be removed")
	}

	// Remove a non-active identity (avoids clearing real git config in tests).
	key, err := opRemove(store, "private")
	if err != nil {
		t.Fatalf("opRemove: %v", err)
	}
	if key != "" {
		t.Errorf("expected no key for removed user, got %q", key)
	}
	if store.FindUser("private") != nil {
		t.Error("expected private user to be removed")
	}
	if store.Current != "eng" {
		t.Error("expected current identity unchanged")
	}
}

func TestOpRemoveInactiveWhileSignedOut(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{
		Current: "",
		Users: []config.User{
			{Name: "private", Email: "private@example.com"},
		},
	}
	key, err := opRemove(store, "private")
	if err != nil {
		t.Fatalf("opRemove: %v", err)
	}
	if key != "" {
		t.Errorf("expected no key path, got %q", key)
	}
	if store.FindUser("private") != nil {
		t.Error("expected user to be removed")
	}
	if store.Current != "" {
		t.Errorf("expected Current to remain empty, got %q", store.Current)
	}
}

func TestOpImportOriginalValidation(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{
		Current: "",
		Users: []config.User{
			{Name: "eng", Email: "eng@corp.com"},
		},
	}

	// Duplicate name.
	if _, err := opImportOriginal(store, "eng", "orig@x.com"); err == nil {
		t.Error("expected error for duplicate name")
	}
	// Duplicate email.
	if _, err := opImportOriginal(store, "orig", "eng@corp.com"); err == nil {
		t.Error("expected error for duplicate email")
	}
	// Invalid email.
	if _, err := opImportOriginal(store, "orig", "not-an-email"); err == nil {
		t.Error("expected error for invalid email")
	}
	// Already imported original source.
	store.Users = append(store.Users, config.User{Name: "orig", Email: "orig@x.com", Source: "original"})
	if _, err := opImportOriginal(store, "orig2", "orig2@x.com"); err == nil {
		t.Error("expected error when the original identity was already imported")
	}

	// Valid import.
	store.Users = store.Users[:1]
	if _, err := opImportOriginal(store, "orig", "orig@x.com"); err != nil {
		t.Fatalf("opImportOriginal valid import: %v", err)
	}
	u := store.FindUser("orig")
	if u == nil || u.Source != "original" {
		t.Error("expected imported identity tagged as original")
	}
	if store.Current != "orig" {
		t.Errorf("expected imported identity to be active, got %q", store.Current)
	}
}

func TestOpRegisterFinishNoKey(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{}
	res, err := opRegisterFinish(store, "guest", "guest@corp.com", true, "", "", false)
	if err != nil {
		t.Fatalf("opRegisterFinish: %v", err)
	}
	if !res.showReport {
		t.Error("expected report screen for registration")
	}
	u := store.FindUser("guest")
	if u == nil {
		t.Fatal("expected user to be created")
	}
	if !u.IsTemporary {
		t.Error("expected temporary flag")
	}
}

func TestNeedsPassphraseForSwitchNoKey(t *testing.T) {
	store := &config.Store{Users: []config.User{{Name: "eng", Email: "eng@corp.com"}}}
	if needsPassphraseForSwitch(store, "eng") {
		t.Error("expected no passphrase needed when no SSH key")
	}
}
