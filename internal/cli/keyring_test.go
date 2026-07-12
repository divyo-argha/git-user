package cli

import (
	"github.com/divyo-argha/git-user/internal/keyring"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func TestKeyringIntegration(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}

	setupTestEnv(t)

	// Create an identity
	store, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = store.AddUser("work", "work@example.com")
	if err != nil {
		t.Fatalf("failed to add user: %v", err)
	}
	err = config.Save(store)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create a dummy key
	keyPath := filepath.Join(t.TempDir(), "key")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-C", "test@example.com", "-f", keyPath, "-N", "").Run(); err != nil {
		t.Fatalf("generating key: %v", err)
	}

	if err := changeSSHKeyPassphrase(keyPath, "", "correct-passphrase"); err != nil {
		t.Fatalf("failed to set passphrase on key: %v", err)
	}

	// Bind the key and choose to save to keyring
	// Prompt mock for ui.Confirm: "Would you like to store the passphrase securely in your system keychain?" -> yes (true)
	ui.ConfirmFn = func(prompt string, defaultVal bool) bool {
		return true // yes, store in keychain
	}
	// Mock readPassphrase to return the correct passphrase
	readPassphraseFn = func(prompt string) (string, error) {
		return "correct-passphrase", nil
	}

	// Call checkAndPromptPassphrase
	checkAndPromptPassphrase("work", keyPath)

	// Verify it was stored in the mocked keyring
	secret, err := keyring.GetKeychainPassphrase("work")
	if err != nil {
		t.Fatalf("failed to retrieve stored passphrase: %v", err)
	}
	if secret != "correct-passphrase" {
		t.Errorf("expected secret to be 'correct-passphrase', got %q", secret)
	}

	// Test switch retrieves it automatically without prompt
	// Save the key path to the user config
	store, _ = config.Load()
	user := store.FindUser("work")
	user.SSHKey = keyPath
	_ = config.Save(store)

	// Mock readPassphrase to panic if it's called (because it should be retrieved from keychain!)
	readPassphraseFn = func(prompt string) (string, error) {
		t.Fatal("should not prompt for passphrase, should retrieve from keychain")
		return "", nil
	}

	// We need to bypass actual ssh-agent operations or stub them
	// Let's run switch
	err = runSwitch([]string{"work"})
	if err != nil {
		t.Fatalf("failed to switch user: %v", err)
	}

	// Test removal deletes it from keyring
	err = runRemove([]string{"work", "--force"})
	if err != nil {
		t.Fatalf("failed to remove user: %v", err)
	}

	// Verify deleted from keyring
	_, err = keyring.GetKeychainPassphrase("work")
	if err == nil {
		t.Fatal("expected keychain entry to be deleted, but it was found")
	}
}
