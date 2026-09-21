package diagnostics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
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
	t.Setenv("HOME", tmpHome)

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
