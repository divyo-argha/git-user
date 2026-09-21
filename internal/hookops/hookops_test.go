package hookops

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestHookInstallAndUninstall(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)

	// Not in a repo
	if _, err := Install(nil); err == nil {
		t.Errorf("expected error when not in repo")
	}

	// Init repo
	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// First install
	results, err := Install(nil)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if len(results) != len(Specs) {
		t.Fatalf("expected %d results, got %d", len(Specs), len(results))
	}
	for _, r := range results {
		if r.Outcome != OutcomeInstalled {
			t.Errorf("expected OutcomeInstalled, got %v", r.Outcome)
		}
	}

	// Reinstall (already installed)
	results2, err := Install(nil)
	if err != nil {
		t.Fatalf("reinstall failed: %v", err)
	}
	for _, r := range results2 {
		if r.Outcome != OutcomeAlreadyInstalled {
			t.Errorf("expected OutcomeAlreadyInstalled, got %v", r.Outcome)
		}
	}

	// Uninstall
	unResults, err := Uninstall()
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	for _, r := range unResults {
		if r.Outcome != OutcomeRemoved {
			t.Errorf("expected OutcomeRemoved, got %v", r.Outcome)
		}
	}
}

func TestCheckIdentityAndPolicy(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)

	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	_ = exec.Command("git", "config", "user.name", "Alice").Run()
	_ = exec.Command("git", "config", "user.email", "alice@corp.com").Run()

	store := &config.Store{
		Current: "alice",
		Users: []config.User{
			{
				Name:         "alice",
				Email:        "alice@corp.com",
				SignKey:      "key123",
				SignDisabled: false,
			},
		},
	}

	// Identity matches
	if err := CheckIdentity(store); err != nil {
		t.Fatalf("CheckIdentity should succeed, got: %v", err)
	}

	// Identity mismatch
	_ = exec.Command("git", "config", "user.email", "wrong@corp.com").Run()
	if err := CheckIdentity(store); err == nil {
		t.Errorf("expected IdentityMismatchError")
	}

	// Policy enforcement - required signing
	userNoSign := &config.User{Name: "Bob", Email: "bob@corp.com", SignDisabled: true}
	policyPath := filepath.Join(tmpDir, config.RepoPolicyFileName)
	_ = os.WriteFile(policyPath, []byte("require_signing=true\nallowed_email_domains=corp.com\n"), 0644)

	if err := EnforceRepoPolicy(userNoSign); err == nil {
		t.Errorf("expected PolicyViolation for disabled signing")
	}

	// Policy enforcement - domain mismatch
	userWrongDomain := &config.User{Name: "Carol", Email: "carol@other.com", SignKey: "key"}
	if err := EnforceRepoPolicy(userWrongDomain); err == nil {
		t.Errorf("expected PolicyViolation for domain mismatch")
	}
}
