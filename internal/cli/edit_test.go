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
	err = runEdit([]string{"dev"})
	if err == nil {
		t.Fatal("expected error with missing email argument, got nil")
	}
}

func TestRunEdit_InvalidEmail(t *testing.T) {
	setupTestEnv(t)
	err := runEdit([]string{"dev", "invalid-email"})
	if err == nil {
		t.Fatal("expected error with invalid email format, got nil")
	}
}

func TestRunEdit_UserNotFound(t *testing.T) {
	setupTestEnv(t)
	err := runEdit([]string{"dev", "dev@example.com"})
	if err == nil {
		t.Fatal("expected error with nonexistent user, got nil")
	}
}

func TestRunEdit_EmailAlreadyInUse(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.AddUser("ops", "ops@example.com")
	_ = config.Save(store)

	err := runEdit([]string{"dev", "ops@example.com"})
	if err == nil {
		t.Fatal("expected error when updating email to one already in use, got nil")
	}
}

func TestRunEdit_SuccessInactiveUser(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.AddUser("ops", "ops@example.com")
	_ = store.SetCurrent("ops")
	_ = config.Save(store)

	_ = git.Apply("ops", "ops@example.com")

	err := runEdit([]string{"dev", "dev-new@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ = config.Load()
	user := store.FindUser("dev")
	if user.Email != "dev-new@example.com" {
		t.Errorf("expected email to be updated to dev-new@example.com, got %s", user.Email)
	}

	// Verify git config is still ops's email
	if git.CurrentEmail() != "ops@example.com" {
		t.Errorf("expected git user.email to remain ops@example.com, got %s", git.CurrentEmail())
	}
}

func TestRunEdit_SuccessActiveUser(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.SetCurrent("dev")
	_ = config.Save(store)

	// Since we set current to dev, apply dev's details initially
	_ = git.Apply("dev", "dev@example.com")

	err := runEdit([]string{"dev", "dev-new@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ = config.Load()
	user := store.FindUser("dev")
	if user.Email != "dev-new@example.com" {
		t.Errorf("expected email to be updated to dev-new@example.com, got %s", user.Email)
	}

	// Verify git config was updated automatically
	if git.CurrentEmail() != "dev-new@example.com" {
		t.Errorf("expected git user.email to have been updated to dev-new@example.com, got %s", git.CurrentEmail())
	}
}

func TestRunEdit_SameEmail(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = config.Save(store)

	err := runEdit([]string{"dev", "dev@example.com"})
	if err != nil {
		t.Fatalf("expected editing with same email to succeed, got error: %v", err)
	}
}
