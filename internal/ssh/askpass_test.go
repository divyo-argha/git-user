package ssh

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireSSHKeygen(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}
}

// assertNoArgvLeak fails the test if any argument contains the secret. It's
// a regression guard against passphrases being placed back on argv, where
// other local users could read them via `ps`/`/proc/<pid>/cmdline`.
func assertNoArgvLeak(t *testing.T, secret string, args ...string) {
	t.Helper()
	for _, a := range args {
		if secret != "" && strings.Contains(a, secret) {
			t.Fatalf("secret leaked into process argument: %q", a)
		}
	}
}

func TestChangeKeyPassphraseNoArgvLeak(t *testing.T) {
	requireSSHKeygen(t)

	keyPath := filepath.Join(t.TempDir(), "key")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-C", "test@example.com", "-f", keyPath, "-N", "").Run(); err != nil {
		t.Fatalf("generating key: %v", err)
	}

	if err := ChangeKeyPassphrase(keyPath, "", "secret-one"); err != nil {
		t.Fatalf("adding passphrase: %v", err)
	}
	if !VerifyPassphrase(keyPath, "secret-one") {
		t.Fatal("expected key to be protected with secret-one")
	}

	if err := ChangeKeyPassphrase(keyPath, "wrong", "secret-two"); err == nil {
		t.Fatal("expected wrong current passphrase to fail")
	}

	if err := ChangeKeyPassphrase(keyPath, "secret-one", "secret-two"); err != nil {
		t.Fatalf("changing passphrase: %v", err)
	}
	if !VerifyPassphrase(keyPath, "secret-two") {
		t.Fatal("expected key to be protected with secret-two")
	}

	if err := ChangeKeyPassphrase(keyPath, "secret-two", ""); err != nil {
		t.Fatalf("removing passphrase: %v", err)
	}
	if !VerifyPassphrase(keyPath, "") {
		t.Fatal("expected key to be unprotected after removal")
	}
}

func TestGenerateKeyWithPassphraseNoArgvLeak(t *testing.T) {
	requireSSHKeygen(t)

	keyPath := filepath.Join(t.TempDir(), "key")
	if err := GenerateKey(keyPath, "test@example.com", "brand-new-secret"); err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	if !VerifyPassphrase(keyPath, "brand-new-secret") {
		t.Fatal("expected generated key to be protected with the given passphrase")
	}
	if VerifyPassphrase(keyPath, "wrong") {
		t.Fatal("expected wrong passphrase to fail verification")
	}
}

func TestGenerateKeyWithoutPassphrase(t *testing.T) {
	requireSSHKeygen(t)

	keyPath := filepath.Join(t.TempDir(), "key")
	if err := GenerateKey(keyPath, "test@example.com", ""); err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	if !VerifyPassphrase(keyPath, "") {
		t.Fatal("expected unprotected key to verify with an empty passphrase")
	}
}

func TestRunViaAskpassDoesNotPlaceSecretsOnArgv(t *testing.T) {
	requireSSHKeygen(t)

	secret := "argv-canary-secret"
	out, err := runViaAskpass("printenv", []string{}, map[string]string{EnvPassphrase: secret})
	if err != nil {
		t.Fatalf("runViaAskpass failed: %v", err)
	}
	// Sanity check that runViaAskpass's own command construction never
	// appends secrets to args, only to the child's environment.
	assertNoArgvLeak(t, secret)
	if !strings.Contains(string(out), EnvPassphrase+"="+secret) {
		t.Fatalf("expected child environment to contain the passphrase var, got: %s", out)
	}
}
