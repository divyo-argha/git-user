package cli

import (
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/keyring"
)

func TestRunToken_SetShowRemove(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("work", "work@example.com")
	_ = store.SetCurrent("work")
	_ = config.Save(store)
	_ = git.Apply("work", "work@example.com")

	readPassphraseFn = func(prompt string) (string, error) {
		return "ghp_supersecret", nil
	}

	// Status before set.
	out := captureStdout(t, func() {
		if err := runToken([]string{"work"}); err != nil {
			t.Fatalf("status: unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "no HTTPS token") {
		t.Errorf("expected no-token status, got: %s", out)
	}

	// Set.
	if err := runToken([]string{"work", "--set"}); err != nil {
		t.Fatalf("set: unexpected error: %v", err)
	}
	if !keyring.HasHTTPSToken("work") {
		t.Error("expected token to be stored")
	}
	token, err := keyring.GetHTTPSToken("work")
	if err != nil || token != "ghp_supersecret" {
		t.Errorf("expected stored token ghp_supersecret, got (%q, %v)", token, err)
	}

	// core.askpass should now be wired up since "work" is the active identity.
	if askpass := git.CurrentAskpass(); !strings.Contains(askpass, "__askpass") || !strings.Contains(askpass, "work") {
		t.Errorf("expected core.askpass wired to the __askpass helper for work, got: %q", askpass)
	}

	// Status after set.
	out = captureStdout(t, func() {
		if err := runToken([]string{"work"}); err != nil {
			t.Fatalf("status: unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "has an HTTPS token stored") {
		t.Errorf("expected token-stored status, got: %s", out)
	}

	// Remove.
	if err := runToken([]string{"work", "--remove"}); err != nil {
		t.Fatalf("remove: unexpected error: %v", err)
	}
	if keyring.HasHTTPSToken("work") {
		t.Error("expected token to be removed")
	}
	if askpass := git.CurrentAskpass(); askpass != "" {
		t.Errorf("expected core.askpass cleared after removing the only identity's token, got: %q", askpass)
	}
}

func TestRunToken_ExpiryFlag(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("work", "work@example.com")
	_ = config.Save(store)

	readPassphraseFn = func(prompt string) (string, error) {
		return "ghp_supersecret", nil
	}

	if err := runToken([]string{"work", "--set", "--expires", "2099-01-01"}); err != nil {
		t.Fatalf("set with expiry: unexpected error: %v", err)
	}
	store, _ = config.Load()
	if u := store.FindUser("work"); u.HTTPSTokenExpiresAt != "2099-01-01" {
		t.Errorf("expected expiry recorded, got %q", u.HTTPSTokenExpiresAt)
	}

	// Rejects a malformed date.
	if err := runToken([]string{"work", "--expires", "01/01/2099"}); err == nil {
		t.Error("expected an error for a non-YYYY-MM-DD date")
	}

	// Standalone --expires (no --set) updates the field without touching the token.
	if err := runToken([]string{"work", "--expires", "2100-06-15"}); err != nil {
		t.Fatalf("standalone expiry update: unexpected error: %v", err)
	}
	store, _ = config.Load()
	if u := store.FindUser("work"); u.HTTPSTokenExpiresAt != "2100-06-15" {
		t.Errorf("expected expiry updated, got %q", u.HTTPSTokenExpiresAt)
	}
	if token, err := keyring.GetHTTPSToken("work"); err != nil || token != "ghp_supersecret" {
		t.Errorf("expected token untouched by standalone expiry update, got (%q, %v)", token, err)
	}

	// Removing the token clears the recorded expiry too.
	if err := runToken([]string{"work", "--remove"}); err != nil {
		t.Fatalf("remove: unexpected error: %v", err)
	}
	store, _ = config.Load()
	if u := store.FindUser("work"); u.HTTPSTokenExpiresAt != "" {
		t.Errorf("expected expiry cleared after token removal, got %q", u.HTTPSTokenExpiresAt)
	}
}

func TestRunToken_RejectsEmptyToken(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("work", "work@example.com")
	_ = config.Save(store)

	readPassphraseFn = func(prompt string) (string, error) {
		return "   ", nil
	}

	if err := runToken([]string{"work", "--set"}); err == nil {
		t.Error("expected an error for an empty/whitespace-only token")
	}
	if keyring.HasHTTPSToken("work") {
		t.Error("expected no token stored after a rejected empty set")
	}
}

func TestRunAskpassHelper(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("work", "work@example.com")
	_ = store.SetHTTPSUsername("work", "octocat")
	_ = config.Save(store)
	if err := keyring.SetHTTPSToken("work", "ghp_xyz"); err != nil {
		t.Fatalf("SetHTTPSToken: %v", err)
	}

	out := captureStdout(t, func() {
		if err := runAskpassHelper([]string{"work", "Username for 'https://github.com': "}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if strings.TrimSpace(out) != "octocat" {
		t.Errorf("expected username 'octocat', got %q", out)
	}

	out = captureStdout(t, func() {
		if err := runAskpassHelper([]string{"work", "Password for 'https://octocat@github.com': "}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if strings.TrimSpace(out) != "ghp_xyz" {
		t.Errorf("expected token 'ghp_xyz', got %q", out)
	}
}

func TestRunAskpassHelper_UnknownIdentity(t *testing.T) {
	setupTestEnv(t)

	if err := runAskpassHelper([]string{"ghost", "Password for 'https://github.com': "}); err == nil {
		t.Error("expected an error for an unknown identity")
	}
}
