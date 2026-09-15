package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func withTempRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skip("git not available or failed to init repo")
	}
	oldDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(oldDir) })
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func TestOpHookCheckEnforcesRepoPolicy(t *testing.T) {
	withTempConfig(t)
	repoRoot := withTempRepo(t)

	store, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddUser("work", "work@example.com"); err != nil {
		t.Fatal(err)
	}
	store.Current = "work"
	if err := config.Save(store); err != nil {
		t.Fatal(err)
	}
	exec.Command("git", "config", "user.name", "work").Run()
	exec.Command("git", "config", "user.email", "work@example.com").Run()

	if _, err := opPolicyWrite(repoRoot, true, nil); err != nil {
		t.Fatalf("opPolicyWrite failed: %v", err)
	}

	// Signing is disabled by default, so `hook check` should block.
	if _, err := opHook("check"); err == nil {
		t.Fatal("expected opHook(check) to fail when repo policy requires signing but it's disabled")
	}

	// Enabling signing (without needing a real key/format resolution here)
	// via the store directly should let the same check pass.
	store, err = config.Load()
	if err != nil {
		t.Fatal(err)
	}
	store.SetSigningKey("work", filepath.Join(repoRoot, "fake_key"), "ssh")
	if err := config.Save(store); err != nil {
		t.Fatal(err)
	}
	if _, err := opHook("check"); err != nil {
		t.Fatalf("expected opHook(check) to pass once signing is enabled, got: %v", err)
	}
}

func TestOpPolicyWrite(t *testing.T) {
	repoRoot := withTempRepo(t)

	if _, err := opPolicyWrite(repoRoot, true, []string{"example.com", "Other.com"}); err != nil {
		t.Fatalf("opPolicyWrite failed: %v", err)
	}

	policy, err := config.LoadRepoPolicy(repoRoot)
	if err != nil {
		t.Fatalf("LoadRepoPolicy failed: %v", err)
	}
	if !policy.RequireSigning {
		t.Error("expected RequireSigning true")
	}
	if len(policy.AllowedEmailDomains) != 2 || policy.AllowedEmailDomains[0] != "example.com" || policy.AllowedEmailDomains[1] != "other.com" {
		t.Errorf("unexpected domains: %#v", policy.AllowedEmailDomains)
	}
}

func TestOpSignerAddIdentityAndRemove(t *testing.T) {
	withTempConfig(t)
	repoRoot := withTempRepo(t)

	store, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddUser("work", "work@example.com"); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(repoRoot, "id_ed25519")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath).Run(); err != nil {
		t.Skip("ssh-keygen not available")
	}
	if err := store.BindSSHKey("work", keyPath); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(store); err != nil {
		t.Fatal(err)
	}

	if _, err := opSignerAddIdentity(store, repoRoot, "work"); err != nil {
		t.Fatalf("opSignerAddIdentity failed: %v", err)
	}

	entries, err := config.LoadAllowedSigners(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Principals[0] != "work@example.com" {
		t.Fatalf("unexpected entries: %#v", entries)
	}

	// The local git config should now point at the committed file.
	out, _ := exec.Command("git", "config", "--local", "gpg.ssh.allowedSignersFile").Output()
	if got := string(out); got == "" {
		t.Error("expected gpg.ssh.allowedSignersFile to be wired after adding a signer")
	}

	res, err := opSignerRemove(repoRoot, "work@example.com")
	if err != nil {
		t.Fatalf("opSignerRemove failed: %v", err)
	}
	if res.detail == "" {
		t.Error("expected a non-empty result detail")
	}
	entries, err = config.LoadAllowedSigners(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected entry removed, got %#v", entries)
	}
}

func TestOpSignerAddEmail(t *testing.T) {
	repoRoot := withTempRepo(t)

	keyPath := filepath.Join(repoRoot, "other_id_ed25519")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath).Run(); err != nil {
		t.Skip("ssh-keygen not available")
	}

	if _, err := opSignerAddEmail(repoRoot, "outside@example.com", keyPath+".pub"); err != nil {
		t.Fatalf("opSignerAddEmail failed: %v", err)
	}

	entries, err := config.LoadAllowedSigners(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Principals[0] != "outside@example.com" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}
