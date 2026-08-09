package cli

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestRunRename(t *testing.T) {
	setupTestEnv(t)
	store, _ := config.Load()
	_ = store.AddUser("old", "old@example.com")
	_ = store.SetCurrent("old")
	_ = config.Save(store)

	if err := runRename([]string{"old", "new"}); err != nil {
		t.Fatalf("runRename: %v", err)
	}

	store, _ = config.Load()
	if store.FindUser("new") == nil {
		t.Fatal("expected renamed user new")
	}
	if store.FindUser("old") != nil {
		t.Fatal("expected old name to be gone")
	}
	if store.Current != "new" {
		t.Errorf("expected current to move to new, got %q", store.Current)
	}
}

func TestRunRenameConflicts(t *testing.T) {
	setupTestEnv(t)
	store, _ := config.Load()
	_ = store.AddUser("a", "a@example.com")
	_ = store.AddUser("b", "b@example.com")
	_ = config.Save(store)

	if err := runRename([]string{"a", "b"}); err == nil {
		t.Fatal("expected error renaming to an existing name")
	}

	if err := runRename([]string{"missing", "x"}); err == nil {
		t.Fatal("expected error renaming a missing identity")
	}

	if err := runRename([]string{}); err == nil {
		t.Fatal("expected error with no args")
	}
}
