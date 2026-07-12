package cli

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

func TestRunList_Empty(t *testing.T) {
	setupTestEnv(t)
	err := runList([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunList_WithUsers(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.AddUser("bob", "bob@example.com")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	err := runList([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCurrent_NoActive(t *testing.T) {
	setupTestEnv(t)
	err := runCurrent([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCurrent_ActiveInSync(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	// Keep git config in sync
	_ = git.Apply("alice", "alice@example.com")

	err := runCurrent([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCurrent_ActiveOutOfSync(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	// Git config out of sync (empty or different name/email)
	_ = git.Apply("different", "different@example.com")

	err := runCurrent([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
