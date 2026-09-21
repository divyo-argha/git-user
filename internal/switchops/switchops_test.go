package switchops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/testutil"
)

func TestAlreadyActive(t *testing.T) {
	testutil.Sandbox(t)

	alice := &config.User{Name: "alice", Email: "alice@example.com"}
	store := &config.Store{Users: []config.User{*alice}}

	// Not current yet.
	if AlreadyActive(store, alice, false) {
		t.Errorf("expected not already active before any switch")
	}

	// A local switch is never "already active" — it always re-applies,
	// since it targets a possibly different repo's config.
	store.Current = "alice"
	if AlreadyActive(store, alice, true) {
		t.Errorf("expected local mode to never report already-active")
	}

	// Current by name, but git config not applied yet -> not in sync.
	if AlreadyActive(store, alice, false) {
		t.Errorf("expected not-in-sync identity to not be already-active")
	}

	if err := git.Apply(alice.Name, alice.Email); err != nil {
		t.Fatalf("git.Apply: %v", err)
	}
	if !AlreadyActive(store, alice, false) {
		t.Errorf("expected already-active once current and in sync")
	}
}

func TestBoundKeyMissing(t *testing.T) {
	noKey := &config.User{Name: "nokey"}
	if BoundKeyMissing(noKey) {
		t.Errorf("expected no-key identity to never report a missing key")
	}

	dir := t.TempDir()
	missing := &config.User{Name: "missing", SSHKey: filepath.Join(dir, "does-not-exist")}
	if !BoundKeyMissing(missing) {
		t.Errorf("expected missing key file to be reported")
	}

	present := filepath.Join(dir, "present_key")
	if err := os.WriteFile(present, []byte("key"), 0600); err != nil {
		t.Fatalf("writing fixture key: %v", err)
	}
	found := &config.User{Name: "found", SSHKey: present}
	if BoundKeyMissing(found) {
		t.Errorf("expected present key file to not be reported as missing")
	}
}

func TestLogoutPreviousNoOpCases(t *testing.T) {
	testutil.Sandbox(t)

	store := &config.Store{Users: []config.User{{Name: "alice"}}}

	// No current identity at all.
	if notices := LogoutPrevious(store, "alice"); notices != nil {
		t.Errorf("expected no notices when there is no current identity, got %v", notices)
	}

	// Switching to the identity that's already current is not a logout.
	store.Current = "alice"
	if notices := LogoutPrevious(store, "alice"); notices != nil {
		t.Errorf("expected no notices when switching to the already-current identity, got %v", notices)
	}
}

func TestLogoutPreviousDeletesTemporaryIdentity(t *testing.T) {
	testutil.Sandbox(t)

	dir := t.TempDir()
	tempKey := filepath.Join(dir, "temp_key")
	if err := os.WriteFile(tempKey, []byte("private"), 0600); err != nil {
		t.Fatalf("writing fixture key: %v", err)
	}
	if err := os.WriteFile(tempKey+".pub", []byte("public"), 0644); err != nil {
		t.Fatalf("writing fixture pubkey: %v", err)
	}

	store := &config.Store{
		Current: "temp-alice",
		Users: []config.User{
			{Name: "temp-alice", SSHKey: tempKey, IsTemporary: true},
			{Name: "bob"},
		},
	}

	notices := LogoutPrevious(store, "bob")
	if len(notices) == 0 {
		t.Fatalf("expected at least one notice about the deleted temporary identity")
	}
	if store.FindUser("temp-alice") != nil {
		t.Errorf("expected temporary identity to be removed from the store")
	}
	if _, err := os.Stat(tempKey); !os.IsNotExist(err) {
		t.Errorf("expected temporary key file to be securely deleted, stat err = %v", err)
	}
}

func TestApplyIdentityGlobalSwitch(t *testing.T) {
	testutil.Sandbox(t)

	alice := &config.User{Name: "alice", Email: "alice@example.com"}
	store := &config.Store{Users: []config.User{*alice}}

	noop := func(key, value string, local bool) error { return nil }
	noopUnset := func(key string, local bool) error { return nil }

	warnings, err := ApplyIdentity(store, store.FindUser("alice"), false, noop, noopUnset)
	if err != nil {
		t.Fatalf("ApplyIdentity: %v (warnings: %v)", err, warnings)
	}
	if store.Current != "alice" {
		t.Errorf("expected store.Current to be set to alice, got %q", store.Current)
	}
	if got := git.CurrentName(); got != "alice" {
		t.Errorf("expected git user.name to be applied, got %q", got)
	}
	if got := git.CurrentEmail(); got != "alice@example.com" {
		t.Errorf("expected git user.email to be applied, got %q", got)
	}
}
