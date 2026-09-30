package cli

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func TestRunLogout_LoggedOut(t *testing.T) {
	setupTestEnv(t)

	err := runLogout([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunLogout_LoggedIn(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.SetCurrent("dev")
	_ = config.Save(store)

	_ = git.Apply("dev", "dev@example.com")

	err := runLogout([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ = config.Load()
	if store.Current != "" {
		t.Errorf("expected current to be empty, got %s", store.Current)
	}

	if git.CurrentName() != "" {
		t.Errorf("expected git user.name to be empty, got %s", git.CurrentName())
	}
	if git.CurrentEmail() != "" {
		t.Errorf("expected git user.email to be empty, got %s", git.CurrentEmail())
	}
}

func TestRunLogout_TempProfile(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("guest", "guest@example.com")
	u := store.FindUser("guest")
	u.IsTemporary = true
	_ = store.SetCurrent("guest")
	_ = config.Save(store)

	_ = git.Apply("guest", "guest@example.com")

	err := runLogout([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ = config.Load()
	if store.Current != "" {
		t.Errorf("expected current to be empty, got %s", store.Current)
	}
	if store.FindUser("guest") != nil {
		t.Errorf("expected temp profile to be deleted on logout")
	}
}

func setupTempProfile(t *testing.T) {
	t.Helper()
	store, _ := config.Load()
	_ = store.AddUser("guest", "guest@example.com")
	store.FindUser("guest").IsTemporary = true
	_ = store.SetCurrent("guest")
	_ = config.Save(store)
	_ = git.Apply("guest", "guest@example.com")
}

// In a terminal, signing out of a temporary identity (which deletes it for
// good) asks first; declining keeps everything as it was.
func TestRunLogout_TempProfileAsksInTerminal(t *testing.T) {
	setupTestEnv(t)
	forceTTY(t)
	setupTempProfile(t)

	asked := false
	origConfirm := ui.ConfirmFn
	t.Cleanup(func() { ui.ConfirmFn = origConfirm })
	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		asked = true
		if defaultYes {
			t.Error("the destructive confirmation must default to No")
		}
		return false
	}

	if err := runLogout(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !asked {
		t.Fatal("expected a confirmation prompt for a temporary identity")
	}
	store, _ := config.Load()
	if store.Current != "guest" || store.FindUser("guest") == nil {
		t.Error("declining must leave the identity active and intact")
	}
	if git.CurrentName() != "guest" {
		t.Error("declining must leave git config alone")
	}

	// Accepting signs out and deletes it.
	ui.ConfirmFn = func(string, bool) bool { return true }
	if err := runLogout(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	store, _ = config.Load()
	if store.Current != "" || store.FindUser("guest") != nil {
		t.Error("accepting should sign out and delete the temporary identity")
	}
}

func TestRunLogout_YesSkipsPrompt(t *testing.T) {
	setupTestEnv(t)
	forceTTY(t)
	setupTempProfile(t)

	origConfirm := ui.ConfirmFn
	t.Cleanup(func() { ui.ConfirmFn = origConfirm })
	ui.ConfirmFn = func(string, bool) bool {
		t.Error("--yes must not prompt")
		return false
	}
	if err := runLogout([]string{"--yes"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store, _ := config.Load(); store.Current != "" || store.FindUser("guest") != nil {
		t.Error("--yes should sign out and delete the temporary identity")
	}
}

// A normal identity never prompts and is kept.
func TestRunLogout_NormalIdentityNeverPrompts(t *testing.T) {
	setupTestEnv(t)
	forceTTY(t)
	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.SetCurrent("dev")
	_ = config.Save(store)
	_ = git.Apply("dev", "dev@example.com")

	origConfirm := ui.ConfirmFn
	t.Cleanup(func() { ui.ConfirmFn = origConfirm })
	ui.ConfirmFn = func(string, bool) bool {
		t.Error("a normal identity must not prompt")
		return false
	}
	if err := runLogout(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	store, _ = config.Load()
	if store.Current != "" || store.FindUser("dev") == nil {
		t.Error("signed out but the identity itself must remain")
	}
}

// Without the shell wrapper, `logout --session` must not fall through to a
// global sign-out — that is a much bigger action than was asked for.
func TestRunLogout_SessionFlagNeverSignsOutGlobally(t *testing.T) {
	setupTestEnv(t)
	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.SetCurrent("dev")
	_ = config.Save(store)
	_ = git.Apply("dev", "dev@example.com")

	for _, flag := range []string{"--session", "-s"} {
		if err := runLogout([]string{flag}); err == nil {
			t.Errorf("logout %s without the shell integration should fail", flag)
		}
		if s, _ := config.Load(); s.Current != "dev" {
			t.Fatalf("logout %s signed out globally", flag)
		}
		if git.CurrentName() != "dev" {
			t.Fatalf("logout %s cleared the global git config", flag)
		}
	}
}
