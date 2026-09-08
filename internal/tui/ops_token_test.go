package tui

import (
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/keyring"
	zkeyring "github.com/zalando/go-keyring"
)

// mockKeyringStore installs an in-memory keyring for the duration of the
// test, keyed by (service, user) like the real backend — the same pattern
// TestOpAttachKeyWarnsWhenKeychainStoreFails uses for a single mocked call.
func mockKeyringStore(t *testing.T) {
	t.Helper()
	oldGet, oldSet, oldDelete := keyring.KeyringGet, keyring.KeyringSet, keyring.KeyringDelete
	store := make(map[string]string)
	key := func(service, user string) string { return service + ":" + user }
	keyring.KeyringGet = func(service, user string) (string, error) {
		v, ok := store[key(service, user)]
		if !ok {
			return "", zkeyring.ErrNotFound
		}
		return v, nil
	}
	keyring.KeyringSet = func(service, user, password string) error {
		store[key(service, user)] = password
		return nil
	}
	keyring.KeyringDelete = func(service, user string) error {
		k := key(service, user)
		if _, ok := store[k]; !ok {
			return zkeyring.ErrNotFound
		}
		delete(store, k)
		return nil
	}
	t.Cleanup(func() {
		keyring.KeyringGet, keyring.KeyringSet, keyring.KeyringDelete = oldGet, oldSet, oldDelete
	})
}

func TestOpSetAndRemoveHTTPSToken(t *testing.T) {
	withTempConfig(t)
	mockKeyringStore(t)

	store, _ := config.Load()
	_ = store.AddUser("work", "work@example.com")
	_ = store.SetCurrent("work")
	_ = config.Save(store)
	_ = git.Apply("work", "work@example.com")

	res, err := opSetHTTPSToken(store, "work", "ghp_abc123", "octocat", "")
	if err != nil {
		t.Fatalf("opSetHTTPSToken: %v", err)
	}
	if !strings.Contains(res.detail, "Token stored securely") {
		t.Errorf("expected success detail, got: %s", res.detail)
	}
	if !keyring.HasHTTPSToken("work") {
		t.Error("expected token to be stored")
	}
	if u := store.FindUser("work"); u.HTTPSUsername != "octocat" {
		t.Errorf("expected username saved, got %q", u.HTTPSUsername)
	}
	if askpass := git.CurrentAskpass(); !strings.Contains(askpass, "__askpass") || !strings.Contains(askpass, "work") {
		t.Errorf("expected core.askpass wired for active identity, got %q", askpass)
	}

	res, err = opRemoveHTTPSToken(store, "work")
	if err != nil {
		t.Fatalf("opRemoveHTTPSToken: %v", err)
	}
	if !strings.Contains(res.detail, "removed") {
		t.Errorf("expected removal detail, got: %s", res.detail)
	}
	if keyring.HasHTTPSToken("work") {
		t.Error("expected token to be removed")
	}
	if askpass := git.CurrentAskpass(); askpass != "" {
		t.Errorf("expected core.askpass cleared after removing the active identity's token, got %q", askpass)
	}
}

func TestOpSetHTTPSToken_UnknownIdentity(t *testing.T) {
	withTempConfig(t)
	mockKeyringStore(t)

	store, _ := config.Load()
	if _, err := opSetHTTPSToken(store, "ghost", "tok", "", ""); err == nil {
		t.Error("expected an error for an unknown identity")
	}
}

// TestOpRenameMigratesKeyringSecretsAndAskpass mirrors
// TestRunRenameMigratesKeyringSecretsAndAskpass in internal/cli — the TUI's
// opRename must not orphan a renamed identity's keyring-stored SSH
// passphrase and HTTPS token, and must re-wire core.askpass to the new name
// when the renamed identity is the active one.
func TestOpRenameMigratesKeyringSecretsAndAskpass(t *testing.T) {
	withTempConfig(t)
	mockKeyringStore(t)

	store, _ := config.Load()
	_ = store.AddUser("old", "old@example.com")
	_ = store.SetCurrent("old")
	_ = config.Save(store)
	_ = git.Apply("old", "old@example.com")

	if err := keyring.SetKeychainPassphrase("old", "sshpass123"); err != nil {
		t.Fatalf("SetKeychainPassphrase: %v", err)
	}
	if err := keyring.SetHTTPSToken("old", "ghp_abc123"); err != nil {
		t.Fatalf("SetHTTPSToken: %v", err)
	}
	if err := git.ConfigureAskpass("'/usr/bin/git-user' __askpass 'old'"); err != nil {
		t.Fatalf("ConfigureAskpass: %v", err)
	}

	if err := opRename(store, "old", "new"); err != nil {
		t.Fatalf("opRename: %v", err)
	}

	if keyring.HasHTTPSToken("old") {
		t.Error("expected old identity's token to be migrated away, not left behind")
	}
	if pass, err := keyring.GetKeychainPassphrase("new"); err != nil || pass != "sshpass123" {
		t.Errorf("expected passphrase migrated to new name, got (%q, %v)", pass, err)
	}
	if token, err := keyring.GetHTTPSToken("new"); err != nil || token != "ghp_abc123" {
		t.Errorf("expected token migrated to new name, got (%q, %v)", token, err)
	}
	if askpass := git.CurrentAskpass(); !strings.Contains(askpass, "'new'") || strings.Contains(askpass, "'old'") {
		t.Errorf("expected core.askpass re-wired to the new name, got %q", askpass)
	}
}

// TestOpSetHTTPSToken_InactiveIdentityDoesNotTouchAskpass guards against
// wiring core.askpass for an identity that isn't the active one — that would
// make every credential prompt on this machine try the wrong identity's
// token until the next switch.
func TestOpSetHTTPSToken_InactiveIdentityDoesNotTouchAskpass(t *testing.T) {
	withTempConfig(t)
	mockKeyringStore(t)

	store, _ := config.Load()
	_ = store.AddUser("personal", "personal@example.com")
	_ = store.AddUser("work", "work@example.com")
	_ = store.SetCurrent("personal")
	_ = config.Save(store)
	_ = git.Apply("personal", "personal@example.com")

	if _, err := opSetHTTPSToken(store, "work", "ghp_xyz", "", ""); err != nil {
		t.Fatalf("opSetHTTPSToken: %v", err)
	}
	if askpass := git.CurrentAskpass(); askpass != "" {
		t.Errorf("expected no core.askpass change for a non-active identity, got %q", askpass)
	}
}

// TestOpSetHTTPSToken_ExpirySetAndClearedOnRemove checks the expiry field
// round-trips through opSetHTTPSToken and is cleared by opRemoveHTTPSToken —
// otherwise doctor would keep warning about an expiry date for a token that
// no longer exists.
func TestOpSetHTTPSToken_ExpirySetAndClearedOnRemove(t *testing.T) {
	withTempConfig(t)
	mockKeyringStore(t)

	store, _ := config.Load()
	_ = store.AddUser("work", "work@example.com")
	_ = config.Save(store)

	if _, err := opSetHTTPSToken(store, "work", "ghp_abc", "", "2099-01-01"); err != nil {
		t.Fatalf("opSetHTTPSToken: %v", err)
	}
	if u := store.FindUser("work"); u.HTTPSTokenExpiresAt != "2099-01-01" {
		t.Errorf("expected expiry recorded, got %q", u.HTTPSTokenExpiresAt)
	}

	if _, err := opRemoveHTTPSToken(store, "work"); err != nil {
		t.Fatalf("opRemoveHTTPSToken: %v", err)
	}
	if u := store.FindUser("work"); u.HTTPSTokenExpiresAt != "" {
		t.Errorf("expected expiry cleared after token removal, got %q", u.HTTPSTokenExpiresAt)
	}
}
