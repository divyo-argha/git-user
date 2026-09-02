package tui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/keyring"
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

// TestOpRegisterFinishWarnsOnGitApplyFailure guards against a regression
// where auto-activating the first identity called git.Apply and, on failure,
// silently did nothing — no warning, yet the report still said "no active
// identity" style text with nothing indicating an activation attempt was
// even made. Forces git.Apply to fail by pointing GIT_CONFIG_GLOBAL at a
// directory instead of a file.
func TestOpRegisterFinishWarnsOnGitApplyFailure(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	withTempConfig(t)
	dir := os.Getenv("HOME")

	badGlobalConfig := filepath.Join(dir, "not-a-file")
	if err := os.MkdirAll(badGlobalConfig, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", badGlobalConfig)

	store := &config.Store{}
	res, err := opRegisterFinish(store, "solo", "solo@example.com", false, "", "", false)
	if err != nil {
		t.Fatalf("opRegisterFinish: %v", err)
	}
	if !strings.Contains(res.detail, "Could not apply git identity") {
		t.Errorf("expected an explicit warning that activation failed, got detail:\n%s", res.detail)
	}
	if store.Current == "solo" {
		t.Error("expected store.Current to NOT be set to an identity git.Apply failed for")
	}
}

// TestOpRefreshWarnsOnApplyFailureInsteadOfClaimingHealthy guards against a
// regression where a failed re-apply attempt (git.Apply/ConfigureSSH/
// ConfigureSigning errors were all discarded) could still fall through to
// "Git config already matched identity %q — nothing to fix." — actively
// telling the user their config is healthy when the fix attempt just failed.
func TestOpRefreshWarnsOnApplyFailureInsteadOfClaimingHealthy(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	withTempConfig(t)
	dir := os.Getenv("HOME")

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.SetCurrent("dev")
	_ = config.Save(store)

	badGlobalConfig := filepath.Join(dir, "not-a-file")
	if err := os.MkdirAll(badGlobalConfig, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", badGlobalConfig)

	res, err := opRefresh(store)
	if err != nil {
		t.Fatalf("opRefresh: %v", err)
	}
	if strings.Contains(res.detail, "nothing to fix") {
		t.Errorf("expected the failed apply attempt to NOT be reported as healthy, got detail:\n%s", res.detail)
	}
	if !strings.Contains(res.detail, "Could not") {
		t.Errorf("expected an explicit warning about the failed re-apply, got detail:\n%s", res.detail)
	}
}

// TestOpAttachKeyWarnsWhenKeychainStoreFails guards against a regression
// where a failed keyring.SetKeychainPassphrase during key generation was
// discarded, leaving the identity's PassphraseMode unset — which defaults to
// "persistent" (config.User.GetPassphraseMode's zero-value default) — even
// though nothing was actually persisted. The next switch would then try the
// keychain, silently fail to find anything, and fall back to an unexplained
// passphrase prompt with no record of why "persistent" mode never worked.
func TestOpAttachKeyWarnsWhenKeychainStoreFails(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}
	withTempConfig(t)

	oldSet := keyring.KeyringSet
	keyring.KeyringSet = func(service, user, password string) error {
		return errors.New("mock keychain unavailable")
	}
	t.Cleanup(func() { keyring.KeyringSet = oldSet })

	store, _ := config.Load()
	res, err := opAttachKey(store, "dev", "dev@example.com", "register", "generate", "secret123", "", false)
	if err != nil {
		t.Fatalf("opAttachKey failed: %v", err)
	}
	if !strings.Contains(res.detail, "Could not store the passphrase in the system keychain") {
		t.Errorf("expected an explicit warning about the failed keychain store, got detail:\n%s", res.detail)
	}
	u := store.FindUser("dev")
	if u == nil {
		t.Fatal("expected identity to be created")
	}
	if u.PassphraseMode != "everytime" {
		t.Errorf("expected PassphraseMode to fall back to %q since persistent storage failed, got %q", "everytime", u.PassphraseMode)
	}
}

func TestNeedsPassphraseForSwitchNoKey(t *testing.T) {
	store := &config.Store{Users: []config.User{{Name: "eng", Email: "eng@corp.com"}}}
	if needsPassphraseForSwitch(store, "eng") {
		t.Error("expected no passphrase needed when no SSH key")
	}
}

// TestOpSwitchWarnsWhenAgentUnreachable guards against a regression where
// switching to a passphrase-protected identity with no reachable ssh-agent
// (the test sandbox always clears SSH_AUTH_SOCK, exactly reproducing that
// case) silently returned success with no warning that the key was never
// actually loaded into any agent. EnsureSSHAgent prints its own message on
// failure, but that's a raw stdout write the TUI's alt-screen swallows — the
// only way the TUI user can find out is via the report/warnings this op
// returns, which used to stay empty in this exact case.
func TestOpSwitchWarnsWhenAgentUnreachable(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}
	withTempConfig(t)
	dir := os.Getenv("HOME")

	keyPath := filepath.Join(dir, "protected_key")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-C", "test@example.com", "-f", keyPath, "-N", "secret123").Run(); err != nil {
		t.Fatalf("generating protected key: %v", err)
	}

	store, _ := config.Load()
	_ = store.AddUser("protected", "protected@example.com")
	_ = store.BindSSHKey("protected", keyPath)
	_ = config.Save(store)

	res, err := opSwitch(store, "protected", "secret123")
	if err != nil {
		t.Fatalf("opSwitch failed: %v", err)
	}
	if !res.showReport {
		t.Error("expected the agent warning to force a report screen, not a fleeting toast")
	}
	if !strings.Contains(res.detail, "NOT loaded into any ssh-agent") {
		t.Errorf("expected an explicit warning that the key was not loaded into any ssh-agent, got detail:\n%s", res.detail)
	}
}

// TestOpRekeyWarnsWhenAgentUnreachable is opRekey's counterpart to
// TestOpSwitchWarnsWhenAgentUnreachable: rotating the active identity's key
// with a new passphrase but no reachable ssh-agent used to produce a report
// with no mention that the freshly rotated key was never loaded anywhere.
func TestOpRekeyWarnsWhenAgentUnreachable(t *testing.T) {
	withTempConfig(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.SetCurrent("dev")
	_ = config.Save(store)

	res, err := opRekey(store, "dev", "secret123")
	if err != nil {
		t.Fatalf("opRekey failed: %v", err)
	}
	if !strings.Contains(res.detail, "NOT loaded into any ssh-agent") {
		t.Errorf("expected an explicit warning that the new key was not loaded into any ssh-agent, got detail:\n%s", res.detail)
	}
}
