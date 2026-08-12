package cli

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
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
