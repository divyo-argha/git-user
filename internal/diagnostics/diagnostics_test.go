package diagnostics

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/testutil"
	"github.com/divyo-argha/git-user/internal/validate"
)

func TestTokenExpiryMessage(t *testing.T) {
	// Invalid date
	if msg := TokenExpiryMessage("invalid-date"); msg != "" {
		t.Errorf("expected empty string for invalid date, got %q", msg)
	}

	// Expired date
	past := time.Now().AddDate(0, 0, -5).Format(validate.DateLayout)
	if msg := TokenExpiryMessage(past); !strings.Contains(msg, "expired 5 day(s) ago") {
		t.Errorf("expected expired message, got %q", msg)
	}

	// Expiring soon (5 days from now)
	soon := time.Now().AddDate(0, 0, 5).Format(validate.DateLayout)
	if msg := TokenExpiryMessage(soon); !strings.Contains(msg, "expires in ") || !strings.Contains(msg, soon) {
		t.Errorf("expected expires in message, got %q", msg)
	}

	// Not expiring soon (60 days from now)
	future := time.Now().AddDate(0, 0, 60).Format(validate.DateLayout)
	if msg := TokenExpiryMessage(future); msg != "" {
		t.Errorf("expected empty string for future date, got %q", msg)
	}
}

func TestSigningDisabledMessage(t *testing.T) {
	uDisabled := &config.User{SignDisabled: true}
	if msg := SigningDisabledMessage(uDisabled); !strings.Contains(msg, "disabled") {
		t.Errorf("expected disabled message, got %q", msg)
	}

	uNoKey := &config.User{SignDisabled: false, SignKey: ""}
	if msg := SigningDisabledMessage(uNoKey); !strings.Contains(msg, "No commit signing key") {
		t.Errorf("expected no key message, got %q", msg)
	}

	uOK := &config.User{SignDisabled: false, SignKey: "my-key"}
	if msg := SigningDisabledMessage(uOK); msg != "" {
		t.Errorf("expected empty message for configured signing, got %q", msg)
	}
}

func TestRunDiagnostics(t *testing.T) {
	tmpHome := t.TempDir()
	testutil.SetHomeDir(t, tmpHome)

	keyPath := filepath.Join(tmpHome, "id_test")
	_ = os.WriteFile(keyPath, []byte("test-key"), 0600)

	store := &config.Store{
		Current: "alice",
		Users: []config.User{
			{
				Name:         "alice",
				Email:        "alice@example.com",
				SSHKey:       keyPath,
				SignKey:      keyPath,
				SignDisabled: false,
			},
		},
	}

	opts := Options{
		Fix: false,
		VerifySSH: func(k string) error {
			return nil
		},
	}

	report, err := Run(store, opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(report.Checks) == 0 {
		t.Errorf("expected checks in report")
	}

	// Verify some check results
	foundSSH := false
	for _, c := range report.Checks {
		if c.ID == "ssh-connectivity" && c.Status == StatusPass {
			foundSSH = true
		}
	}
	if !foundSSH {
		t.Errorf("expected ssh-connectivity check to pass with mock VerifySSH")
	}
}

// TestIsLikelySharedMachine covers the sibling-directory-count heuristic
// directly, since Run's use of it depends on the real filesystem around
// os.UserHomeDir() and can't be forced deterministically from inside Run
// itself.
func TestIsLikelySharedMachine(t *testing.T) {
	parent := t.TempDir()
	home := filepath.Join(parent, "alice")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	testutil.SetHomeDir(t, home)

	if isLikelySharedMachine() {
		t.Error("expected false with no sibling directories")
	}

	if err := os.MkdirAll(filepath.Join(parent, "bob"), 0755); err != nil {
		t.Fatal(err)
	}
	if isLikelySharedMachine() {
		t.Error("expected false with only one sibling directory")
	}

	if err := os.MkdirAll(filepath.Join(parent, "carol"), 0755); err != nil {
		t.Fatal(err)
	}
	if !isLikelySharedMachine() {
		t.Error("expected true with two sibling directories")
	}
}

// TestIsLikelySharedMachineSkipsKnownNonUserDirs guards against a false
// positive from the platform-standard non-personal directories that live
// alongside real user profiles (e.g. "Shared"/"Guest"/"Public" on macOS,
// "Default"/"All Users" on Windows) — those must not count toward the
// "looks shared" threshold.
func TestIsLikelySharedMachineSkipsKnownNonUserDirs(t *testing.T) {
	parent := t.TempDir()
	home := filepath.Join(parent, "alice")
	for _, dir := range []string{home, filepath.Join(parent, "Shared"), filepath.Join(parent, "Guest")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	testutil.SetHomeDir(t, home)

	if isLikelySharedMachine() {
		t.Error("expected false — Shared/Guest are skip-listed, not real user siblings")
	}
}

// TestRunDiagnosticsHardensSharedMachine covers the profile-hardening check
// end to end: a StatusNotice suggestion without --fix, and an actual,
// persisted settings change (verified by reloading the config from disk)
// with --fix.
func TestRunDiagnosticsHardensSharedMachine(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}

	parent := t.TempDir()
	home := filepath.Join(parent, "alice")
	for _, dir := range []string{home, filepath.Join(parent, "bob"), filepath.Join(parent, "carol")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	testutil.SetHomeDir(t, home)
	t.Setenv("GIT_USER_CONFIG", filepath.Join(home, ".git-users", "config.json"))

	keyPath := filepath.Join(home, "id_test")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-C", "t", "-f", keyPath, "-N", "secret123").Run(); err != nil {
		t.Fatalf("generating key: %v", err)
	}

	store := &config.Store{
		Current: "alice",
		Users: []config.User{
			{Name: "alice", Email: "alice@example.com", SSHKey: keyPath, PassphraseMode: "persistent"},
		},
	}

	report, err := Run(store, Options{Fix: false, VerifySSH: func(string) error { return nil }})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, c := range report.Checks {
		if c.ID == "profile-hardening" && c.Status == StatusNotice {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a profile-hardening StatusNotice check on a machine that looks shared")
	}

	report, err = Run(store, Options{Fix: true, Interactive: true, VerifySSH: func(string) error { return nil }})
	if err != nil {
		t.Fatalf("Run --fix: %v", err)
	}
	fixed := false
	for _, c := range report.Checks {
		if c.ID == "profile-hardening" && c.Fixed && c.Status == StatusPass {
			fixed = true
		}
	}
	if !fixed {
		t.Errorf("expected profile-hardening to be Fixed under --fix")
	}

	u := store.FindUser("alice")
	if u.PassphraseMode != "everytime" {
		t.Errorf("expected PassphraseMode=everytime after --fix, got %q", u.PassphraseMode)
	}
	if u.AgentTTL != config.HardenedAgentTTL {
		t.Errorf("expected AgentTTL=%q after --fix, got %q", config.HardenedAgentTTL, u.AgentTTL)
	}
	if !u.AgentConfirmBeforeUse {
		t.Error("expected AgentConfirmBeforeUse=true after --fix")
	}

	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("reloading config: %v", err)
	}
	ru := reloaded.FindUser("alice")
	if ru == nil || ru.PassphraseMode != "everytime" {
		t.Errorf("expected hardening to persist to disk, got %+v", ru)
	}
}

// TestRunDiagnosticsFixNeverAppliesHardeningUnattended guards the
// Interactive gate: Fix:true alone must never apply the shared-device
// hardening bundle — auto-changing passphrase-unlock behavior from an
// unattended/scripted `doctor --fix` (e.g. in CI or a container) is exactly
// the surprise this gate exists to prevent. It must still report the
// suggestion, just not apply it, and say so.
func TestRunDiagnosticsFixNeverAppliesHardeningUnattended(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}

	parent := t.TempDir()
	home := filepath.Join(parent, "alice")
	for _, dir := range []string{home, filepath.Join(parent, "bob"), filepath.Join(parent, "carol")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	testutil.SetHomeDir(t, home)
	t.Setenv("GIT_USER_CONFIG", filepath.Join(home, ".git-users", "config.json"))

	keyPath := filepath.Join(home, "id_test")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-C", "t", "-f", keyPath, "-N", "secret123").Run(); err != nil {
		t.Fatalf("generating key: %v", err)
	}

	store := &config.Store{
		Current: "alice",
		Users: []config.User{
			{Name: "alice", Email: "alice@example.com", SSHKey: keyPath, PassphraseMode: "persistent"},
		},
	}

	// Fix:true but Interactive left at its zero value (false) — simulates an
	// unattended `doctor --fix` run.
	report, err := Run(store, Options{Fix: true, VerifySSH: func(string) error { return nil }})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	u := store.FindUser("alice")
	if u.PassphraseMode != "persistent" {
		t.Errorf("expected PassphraseMode to be left untouched by a non-interactive --fix, got %q", u.PassphraseMode)
	}
	if u.AgentConfirmBeforeUse {
		t.Error("expected AgentConfirmBeforeUse to be left untouched by a non-interactive --fix")
	}

	found := false
	for _, c := range report.Checks {
		if c.ID == "profile-hardening" {
			if c.Fixed || c.Status != StatusNotice {
				t.Errorf("expected the hardening check to be reported, not applied, got %+v", c)
			}
			if !strings.Contains(c.Message, "non-interactive") {
				t.Errorf("expected the message to explain why it wasn't auto-applied, got %q", c.Message)
			}
			found = true
		}
	}
	if !found {
		t.Error("expected a profile-hardening check to still be reported")
	}
}

// TestRunDiagnosticsNoHardeningCheckForUnprotectedKey guards the
// protected-gate: an identity with no passphrase at all must never get a
// profile-hardening suggestion — the check only applies once there's a
// passphrase whose exposure window is worth bounding.
func TestRunDiagnosticsNoHardeningCheckForUnprotectedKey(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}

	parent := t.TempDir()
	home := filepath.Join(parent, "alice")
	for _, dir := range []string{home, filepath.Join(parent, "bob"), filepath.Join(parent, "carol")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	testutil.SetHomeDir(t, home)
	t.Setenv("GIT_USER_CONFIG", filepath.Join(home, ".git-users", "config.json"))

	keyPath := filepath.Join(home, "id_test")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-C", "t", "-f", keyPath, "-N", "").Run(); err != nil {
		t.Fatalf("generating key: %v", err)
	}

	store := &config.Store{
		Current: "alice",
		Users: []config.User{
			{Name: "alice", Email: "alice@example.com", SSHKey: keyPath},
		},
	}

	report, err := Run(store, Options{Fix: false, VerifySSH: func(string) error { return nil }})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, c := range report.Checks {
		if c.ID == "profile-hardening" {
			t.Errorf("expected no profile-hardening check for an unprotected key, got %+v", c)
		}
	}
}
