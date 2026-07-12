package cli

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

func TestRunEdit_MissingArgs(t *testing.T) {
	setupTestEnv(t)
	err := runEdit([]string{})
	if err == nil {
		t.Fatal("expected error with no arguments, got nil")
	}
	err = runEdit([]string{"alice"})
	if err == nil {
		t.Fatal("expected error with missing email argument, got nil")
	}
}

func TestRunEdit_InvalidEmail(t *testing.T) {
	setupTestEnv(t)
	err := runEdit([]string{"alice", "invalid-email"})
	if err == nil {
		t.Fatal("expected error with invalid email format, got nil")
	}
}

func TestRunEdit_UserNotFound(t *testing.T) {
	setupTestEnv(t)
	err := runEdit([]string{"alice", "alice@example.com"})
	if err == nil {
		t.Fatal("expected error with nonexistent user, got nil")
	}
}

func TestRunEdit_EmailAlreadyInUse(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.AddUser("bob", "bob@example.com")
	_ = config.Save(store)

	err := runEdit([]string{"alice", "bob@example.com"})
	if err == nil {
		t.Fatal("expected error when updating email to one already in use, got nil")
	}
}

func TestRunEdit_SuccessInactiveUser(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.AddUser("bob", "bob@example.com")
	_ = store.SetCurrent("bob")
	_ = config.Save(store)

	_ = git.Apply("bob", "bob@example.com")

	err := runEdit([]string{"alice", "alice-new@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ = config.Load()
	user := store.FindUser("alice")
	if user.Email != "alice-new@example.com" {
		t.Errorf("expected email to be updated to alice-new@example.com, got %s", user.Email)
	}

	// Verify git config is still bob's email
	if git.CurrentEmail() != "bob@example.com" {
		t.Errorf("expected git user.email to remain bob@example.com, got %s", git.CurrentEmail())
	}
}

func TestRunEdit_SuccessActiveUser(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	// Since we set current to alice, apply alice's details initially
	_ = git.Apply("alice", "alice@example.com")

	err := runEdit([]string{"alice", "alice-new@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ = config.Load()
	user := store.FindUser("alice")
	if user.Email != "alice-new@example.com" {
		t.Errorf("expected email to be updated to alice-new@example.com, got %s", user.Email)
	}

	// Verify git config was updated automatically
	if git.CurrentEmail() != "alice-new@example.com" {
		t.Errorf("expected git user.email to have been updated to alice-new@example.com, got %s", git.CurrentEmail())
	}
}
