package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"user@example.com", true},
		{"user.name@example.com", true},
		{"user+tag@example.co.uk", true},
		{"user_name@example-domain.com", true},
		{"", false},
		{"invalid", false},
		{"@example.com", false},
		{"user@", false},
		{"user@.com", false},
		{"user @example.com", false},
		{"user@example", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			got := isValidEmail(tt.email)
			if got != tt.valid {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, got, tt.valid)
			}
		})
	}
}

func TestRegisterInteractiveSigning(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Pre-create a dummy key
	dummyKeyPath := filepath.Join(tmpDir, "dummy_id")
	err := os.WriteFile(dummyKeyPath, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\n...\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Mock UI inputs
	ui.PromptFn = func(label string) (string, error) {
		if strings.Contains(strings.ToLower(label), "name") {
			return "office", nil
		}
		if strings.Contains(strings.ToLower(label), "email") {
			return "office@example.com", nil
		}
		if strings.Contains(strings.ToLower(label), "ssh private key") {
			return dummyKeyPath, nil
		}
		return "", nil
	}

	ui.SelectFn = func(label string, options []string) (int, error) {
		return 1, nil // Use existing key
	}

	ui.ConfirmFn = func(label string, defaultVal bool) bool {
		if strings.Contains(label, "sign your Git commits automatically") {
			return true // Accept signing
		}
		return defaultVal
	}

	err = runRegister([]string{})
	if err != nil {
		t.Fatalf("runRegister failed: %v", err)
	}

	// Check config
	store, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	user := store.FindUser("office")
	if user == nil {
		t.Fatal("user office was not created")
	}

	if user.SignKey != dummyKeyPath || user.SignFormat != "ssh" || user.SignDisabled {
		t.Errorf("expected signing to be configured, got: %+v", user)
	}
}

// TestRegisterFirstIdentityAutoActivates verifies that registering an
// identity when none is active yet activates it immediately — otherwise a
// correctly-configured identity (with a bound SSH key) would sit inert until
// the user remembered a separate `switch`, and `git push` would keep failing
// even though "everything was set up".
func TestRegisterFirstIdentityAutoActivates(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	setupTestEnv(t)

	ui.PromptFn = func(label string) (string, error) {
		if strings.Contains(strings.ToLower(label), "name") {
			return "solo", nil
		}
		if strings.Contains(strings.ToLower(label), "email") {
			return "solo@example.com", nil
		}
		return "", nil
	}
	ui.SelectFn = func(label string, options []string) (int, error) {
		return 2, nil // Skip SSH key setup
	}

	if err := runRegister([]string{}); err != nil {
		t.Fatalf("runRegister failed: %v", err)
	}

	store, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if store.Current != "solo" {
		t.Errorf("expected store.Current to be %q after first registration, got %q", "solo", store.Current)
	}
	if git.CurrentName() != "solo" || git.CurrentEmail() != "solo@example.com" {
		t.Errorf("expected live git config to reflect the first registered identity, got %s <%s>", git.CurrentName(), git.CurrentEmail())
	}
}

// TestRegisterSecondIdentityStaysInactive verifies registration does NOT
// auto-activate when an identity is already active — auto-activation should
// only help the empty-state case, not silently switch an existing user away
// from whatever they're currently using every time they add a new profile.
func TestRegisterSecondIdentityStaysInactive(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("first", "first@example.com")
	_ = store.SetCurrent("first")
	_ = config.Save(store)
	_ = git.Apply("first", "first@example.com")

	ui.PromptFn = func(label string) (string, error) {
		if strings.Contains(strings.ToLower(label), "name") {
			return "second", nil
		}
		if strings.Contains(strings.ToLower(label), "email") {
			return "second@example.com", nil
		}
		return "", nil
	}
	ui.SelectFn = func(label string, options []string) (int, error) {
		return 2, nil // Skip SSH key setup
	}

	if err := runRegister([]string{}); err != nil {
		t.Fatalf("runRegister failed: %v", err)
	}

	store, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if store.Current != "first" {
		t.Errorf("expected store.Current to remain %q, got %q", "first", store.Current)
	}
	if git.CurrentName() != "first" {
		t.Errorf("expected live git config to remain \"first\", got %s", git.CurrentName())
	}
}
