package switchops

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/keyring"
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

func gitGlobal(t *testing.T, args ...string) string {
	t.Helper()
	out, _ := exec.Command("git", append([]string{"config", "--global"}, args...)...).Output()
	return strings.TrimSpace(string(out))
}

func unsetViaGit(key string, local bool) error {
	scope := "--global"
	if local {
		scope = "--local"
	}
	return exec.Command("git", "config", scope, "--unset-all", key).Run()
}

func TestLogout_NobodySignedIn(t *testing.T) {
	testutil.Sandbox(t)
	res, err := Logout(&config.Store{}, unsetViaGit)
	if res != nil || err != nil {
		t.Errorf("nobody signed in should be (nil, nil), got %v, %v", res, err)
	}
}

// Sign out must undo everything a switch applied — not just name and email.
func TestLogout_UndoesEverythingASwitchApplies(t *testing.T) {
	testutil.Sandbox(t)
	keyring.MockForTest(t)

	store := &config.Store{}
	if err := store.AddUser("work", "work@example.com"); err != nil {
		t.Fatal(err)
	}
	u := store.FindUser("work")
	u.SSHKey = filepath.Join(t.TempDir(), "id_work")
	u.SignKey = u.SSHKey + ".pub"
	u.SignFormat = "ssh"
	u.CustomConfig = map[string]string{"init.defaultBranch": "trunk"}
	if err := keyring.SetKeychainPassphrase("work", "s3cret"); err != nil {
		t.Fatal(err)
	}
	if err := keyring.SetHTTPSToken("work", "tok"); err != nil {
		t.Fatal(err)
	}

	// Apply the identity the way a switch would, including the HTTPS wiring.
	setCustom := func(k, v string, local bool) error {
		return exec.Command("git", "config", "--global", k, v).Run()
	}
	if _, err := ApplyIdentity(store, u, false, setCustom, unsetViaGit); err != nil {
		t.Fatalf("ApplyIdentity: %v", err)
	}
	for key, label := range map[string]string{
		"user.name": "name", "user.email": "email", "core.sshCommand": "ssh command",
		"user.signingkey": "signing key", "core.askpass": "askpass (HTTPS token)", "init.defaultBranch": "custom key",
	} {
		if gitGlobal(t, "--get", key) == "" {
			t.Fatalf("precondition: %s (%s) should be set after a switch", label, key)
		}
	}

	res, err := Logout(store, unsetViaGit)
	if err != nil || res == nil || res.Name != "work" || res.WasTemporary {
		t.Fatalf("Logout = %+v, %v", res, err)
	}

	for _, key := range []string{"user.name", "user.email", "core.sshCommand", "user.signingkey", "commit.gpgsign", "core.askpass", "init.defaultBranch"} {
		if got := gitGlobal(t, "--get", key); got != "" {
			t.Errorf("%s should be cleared after sign out, still %q", key, got)
		}
	}

	if store.Current != "" {
		t.Errorf("active identity should be cleared, got %q", store.Current)
	}
	saved, err := config.Load()
	if err != nil || saved.Current != "" {
		t.Errorf("sign-out should be persisted, got %v / %v", saved, err)
	}

	// Signing out is not forgetting credentials.
	if p, err := keyring.GetKeychainPassphrase("work"); err != nil || p != "s3cret" {
		t.Errorf("keychain passphrase should be kept, got %q, %v", p, err)
	}
	if !keyring.HasHTTPSToken("work") {
		t.Error("stored HTTPS token should be kept")
	}
	if store.FindUser("work") == nil {
		t.Error("a normal identity must survive sign out")
	}

	// And it is auditable.
	entries, _ := config.ReadSwitchLog()
	if len(entries) == 0 || !strings.Contains(entries[len(entries)-1], "work (signed out)") {
		t.Errorf("sign-out should be recorded in the switch log, got %v", entries)
	}
}

// A temporary identity is deleted on sign out, key material included.
func TestLogout_TemporaryIdentityIsDeleted(t *testing.T) {
	testutil.Sandbox(t)
	keyring.MockForTest(t)

	keyPath := filepath.Join(t.TempDir(), "id_guest")
	if err := os.WriteFile(keyPath, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath+".pub", []byte("public"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := &config.Store{}
	if err := store.AddUser("guest", "guest@example.com"); err != nil {
		t.Fatal(err)
	}
	u := store.FindUser("guest")
	u.IsTemporary = true
	u.SSHKey = keyPath
	_ = store.SetCurrent("guest")
	_ = keyring.SetKeychainPassphrase("guest", "pw")

	res, err := Logout(store, unsetViaGit)
	if err != nil || res == nil || !res.WasTemporary {
		t.Fatalf("Logout = %+v, %v", res, err)
	}
	if store.FindUser("guest") != nil {
		t.Error("temporary identity should be removed")
	}
	for _, p := range []string{keyPath, keyPath + ".pub"} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s should be deleted", p)
		}
	}
	if _, err := keyring.GetKeychainPassphrase("guest"); err == nil {
		t.Error("temporary identity's keychain passphrase should be deleted")
	}
}
