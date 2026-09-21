package policyops

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestBuildAndWritePolicy(t *testing.T) {
	tmpDir := t.TempDir()

	content := BuildPolicyFile(true, []string{"company.com", "org.net"})
	if !strings.Contains(string(content), "require_signing=true") {
		t.Errorf("expected require_signing=true in content")
	}
	if !strings.Contains(string(content), "allowed_email_domains=company.com,org.net") {
		t.Errorf("expected allowed_email_domains in content")
	}

	err := WritePolicy(tmpDir, true, []string{"company.com"})
	if err != nil {
		t.Fatalf("WritePolicy failed: %v", err)
	}

	read, err := config.LoadRepoPolicy(tmpDir)
	if err != nil {
		t.Fatalf("LoadRepoPolicy failed: %v", err)
	}
	if !read.RequireSigning {
		t.Errorf("expected RequireSigning=true")
	}
	if len(read.AllowedEmailDomains) != 1 || read.AllowedEmailDomains[0] != "company.com" {
		t.Errorf("unexpected AllowedEmailDomains: %v", read.AllowedEmailDomains)
	}
}

func TestResolveSignerFromIdentityAndEmail(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519")
	pubKeyContent := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGeneratedKeyForTesting test@example.com"
	if err := os.WriteFile(keyPath+".pub", []byte(pubKeyContent), 0644); err != nil {
		t.Fatalf("failed to write pubkey: %v", err)
	}

	store := &config.Store{
		Current: "alice",
		Users: []config.User{
			{
				Name:    "alice",
				Email:   "alice@example.com",
				Aliases: []string{"work-alice@example.com"},
				SSHKey:  keyPath,
			},
		},
	}

	principals, keyBlob, err := ResolveSignerFromIdentity(store, "alice")
	if err != nil {
		t.Fatalf("ResolveSignerFromIdentity failed: %v", err)
	}
	if len(principals) != 2 || principals[0] != "alice@example.com" {
		t.Errorf("unexpected principals: %v", principals)
	}
	if keyBlob != pubKeyContent {
		t.Errorf("unexpected keyBlob: %q", keyBlob)
	}

	// Missing identity
	if _, _, err := ResolveSignerFromIdentity(store, "bob"); err == nil {
		t.Errorf("expected error for missing identity")
	}

	// From Email
	principals, keyBlob, err = ResolveSignerFromEmail("carol@example.com", keyPath+".pub")
	if err != nil {
		t.Fatalf("ResolveSignerFromEmail failed: %v", err)
	}
	if len(principals) != 1 || principals[0] != "carol@example.com" {
		t.Errorf("unexpected principals: %v", principals)
	}
	if keyBlob != pubKeyContent {
		t.Errorf("unexpected keyBlob: %q", keyBlob)
	}

	// Missing pubkey file
	if _, _, err := ResolveSignerFromEmail("carol@example.com", ""); err == nil {
		t.Errorf("expected error when pubkey file is empty")
	}
}

func TestSignerUpsertRemoveAndWire(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo in tmpDir
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git init in %s: %v", tmpDir, err)
	}

	pubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestSigner test@example.com"

	// Upsert signer
	err := UpsertSigner(tmpDir, []string{"alice@example.com"}, pubKey)
	if err != nil {
		t.Fatalf("UpsertSigner failed: %v", err)
	}

	// Check file was created
	entries, err := config.LoadAllowedSigners(tmpDir)
	if err != nil {
		t.Fatalf("LoadAllowedSigners failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 signer entry, got %d", len(entries))
	}

	// Wire config
	changed, err := WireAllowedSignersConfig(tmpDir)
	if err != nil {
		t.Fatalf("WireAllowedSignersConfig failed: %v", err)
	}
	if !changed {
		t.Errorf("expected changed=true on initial wire")
	}

	// Wire again - idempotent
	changed2, err := WireAllowedSignersConfig(tmpDir)
	if err != nil {
		t.Fatalf("second WireAllowedSignersConfig failed: %v", err)
	}
	if changed2 {
		t.Errorf("expected changed=false on second wire")
	}

	// Remove signer
	removed, err := RemoveSigner(tmpDir, "alice@example.com")
	if err != nil {
		t.Fatalf("RemoveSigner failed: %v", err)
	}
	if !removed {
		t.Errorf("expected removed=true")
	}

	// Remove non-existent signer
	removed2, err := RemoveSigner(tmpDir, "nobody@example.com")
	if err != nil {
		t.Fatalf("RemoveSigner for non-existent failed: %v", err)
	}
	if removed2 {
		t.Errorf("expected removed=false for non-existent signer")
	}
}
