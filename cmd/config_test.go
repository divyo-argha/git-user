package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestRunConfig_Errors(t *testing.T) {
	setupTestEnv(t)

	// Missing identity
	err := runConfig([]string{})
	if err == nil {
		t.Fatal("expected error with no identity, got nil")
	}

	// Nonexistent identity
	err = runConfig([]string{"alice"})
	if err == nil {
		t.Fatal("expected error with nonexistent identity, got nil")
	}

	// Register user
	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = config.Save(store)

	// Set missing key/value
	err = runConfig([]string{"alice", "set"})
	if err == nil {
		t.Fatal("expected error setting without key/value, got nil")
	}

	// Set missing value
	err = runConfig([]string{"alice", "set", "core.editor"})
	if err == nil {
		t.Fatal("expected error setting without value, got nil")
	}

	// Unset missing key
	err = runConfig([]string{"alice", "unset"})
	if err == nil {
		t.Fatal("expected error unsetting without key, got nil")
	}
}

func TestRunConfig_SetUnsetList(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = config.Save(store)

	// Set a custom config
	err := runConfig([]string{"alice", "set", "core.editor", "nano"})
	if err != nil {
		t.Fatalf("unexpected error setting config: %v", err)
	}

	// Verify in config store
	store, _ = config.Load()
	user := store.FindUser("alice")
	if user.CustomConfig["core.editor"] != "nano" {
		t.Fatalf("expected CustomConfig to have core.editor=nano, got %v", user.CustomConfig)
	}

	// List config (should print without error)
	err = runConfig([]string{"alice", "list"})
	if err != nil {
		t.Fatalf("unexpected error listing config: %v", err)
	}

	// Unset the custom config
	err = runConfig([]string{"alice", "unset", "core.editor"})
	if err != nil {
		t.Fatalf("unexpected error unsetting config: %v", err)
	}

	// Verify unset in config store
	store, _ = config.Load()
	user = store.FindUser("alice")
	if _, ok := user.CustomConfig["core.editor"]; ok {
		t.Fatal("expected CustomConfig to not contain core.editor")
	}
}

func TestRunConfig_SwitchIntegration(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.AddUser("bob", "bob@example.com")
	_ = config.Save(store)

	// Set custom config for alice
	_ = runConfig([]string{"alice", "set", "core.editor", "nano"})

	// Switch to alice (should apply core.editor=nano)
	err := runSwitch([]string{"alice"})
	if err != nil {
		t.Fatalf("unexpected error switching to alice: %v", err)
	}

	// Verify global git config contains core.editor=nano
	out, err := exec.Command("git", "config", "--global", "core.editor").Output()
	if err != nil {
		t.Fatalf("failed to read global core.editor: %v", err)
	}
	if strings.TrimSpace(string(out)) != "nano" {
		t.Errorf("expected global core.editor to be 'nano', got '%s'", strings.TrimSpace(string(out)))
	}

	// Switch to bob (should unset core.editor)
	err = runSwitch([]string{"bob"})
	if err != nil {
		t.Fatalf("unexpected error switching to bob: %v", err)
	}

	// Verify global git config no longer has core.editor
	out, err = exec.Command("git", "config", "--global", "core.editor").Output()
	// It should fail or return empty because the key was unset
	if err == nil && strings.TrimSpace(string(out)) == "nano" {
		t.Error("expected global core.editor to be unset, but it still has value 'nano'")
	}
}
