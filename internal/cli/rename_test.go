package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/keyring"
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

// TestRunRenameMigratesKeyringSecretsAndAskpass guards against a rename
// silently orphaning an identity's keyring-stored SSH passphrase and HTTPS
// token — both are keyed by identity name, which RenameUser alone never
// updates — and, for the active identity, leaving core.askpass pointing at
// the __askpass helper with a name config.json no longer has.
func TestRunRenameMigratesKeyringSecretsAndAskpass(t *testing.T) {
	setupTestEnv(t)
	store, _ := config.Load()
	_ = store.AddUser("old", "old@example.com")
	_ = store.SetCurrent("old")
	_ = config.Save(store)
	_ = git.Apply("old", "old@example.com")

	if err := keyring.SetKeychainPassphrase("old", "sshpass123"); err != nil {
		t.Fatalf("SetKeychainPassphrase: %v", err)
	}
	if err := keyring.SetHTTPSToken("old", "ghp_abc123"); err != nil {
		t.Fatalf("SetHTTPSToken: %v", err)
	}
	if err := git.ConfigureAskpass("'/usr/bin/git-user' __askpass 'old'"); err != nil {
		t.Fatalf("ConfigureAskpass: %v", err)
	}

	if err := runRename([]string{"old", "new"}); err != nil {
		t.Fatalf("runRename: %v", err)
	}

	if keyring.HasHTTPSToken("old") {
		t.Error("expected old identity's token to be migrated away, not left behind")
	}
	if pass, err := keyring.GetKeychainPassphrase("new"); err != nil || pass != "sshpass123" {
		t.Errorf("expected passphrase migrated to new name, got (%q, %v)", pass, err)
	}
	if token, err := keyring.GetHTTPSToken("new"); err != nil || token != "ghp_abc123" {
		t.Errorf("expected token migrated to new name, got (%q, %v)", token, err)
	}
	if askpass := git.CurrentAskpass(); !strings.Contains(askpass, "'new'") || strings.Contains(askpass, "'old'") {
		t.Errorf("expected core.askpass re-wired to the new name, got %q", askpass)
	}
}

// TestRunRenameUpdatesMatchingLocalOverride guards against a regression
// where renaming an identity only updated the global git config (if it was
// the active one), silently leaving a repo-local `switch --local` override
// pointing at the identity's now-stale old name/email.
func TestRunRenameUpdatesMatchingLocalOverride(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	tmpDir := setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("personal", "personal@example.com")
	_ = store.AddUser("other", "other@example.com")
	_ = store.SetCurrent("other")
	_ = config.Save(store)
	if err := git.Apply("other", "other@example.com"); err != nil {
		t.Fatal(err)
	}

	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}
	cwd, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	if err := runSwitch([]string{"--local", "personal"}); err != nil {
		t.Fatalf("local switch failed: %v", err)
	}

	if err := runRename([]string{"personal", "personal2"}); err != nil {
		t.Fatalf("runRename: %v", err)
	}

	if git.CurrentName() != "personal2" || git.CurrentEmail() != "personal@example.com" {
		t.Errorf("expected this repo's local override to follow the rename, got %s <%s>", git.CurrentName(), git.CurrentEmail())
	}

	// Global config (the unrelated active identity "other") must stay untouched.
	if git.CurrentGlobalName() != "other" {
		t.Errorf("expected global identity to remain \"other\", got %q", git.CurrentGlobalName())
	}
}
