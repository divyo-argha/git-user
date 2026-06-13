package cmd

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
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	_ = git.Apply("alice", "alice@example.com")

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
	_ = store.AddUser("temp", "temp@example.com")
	u := store.FindUser("temp")
	u.IsTemporary = true
	_ = store.SetCurrent("temp")
	_ = config.Save(store)

	_ = git.Apply("temp", "temp@example.com")

	err := runLogout([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ = config.Load()
	if store.Current != "" {
		t.Errorf("expected current to be empty, got %s", store.Current)
	}
	if store.FindUser("temp") != nil {
		t.Errorf("expected temp profile to be deleted on logout")
	}
}
