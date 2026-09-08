package cli

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/validate"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns
// everything it printed.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = orig
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestRunDoctor_NoActive(t *testing.T) {
	setupTestEnv(t)

	err := runDoctor([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDoctor_GitConfigOutOfSync(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.SetCurrent("dev")
	_ = config.Save(store)

	_ = git.Apply("ops", "ops@example.com") // Mis-matched git config

	err := runDoctor([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDoctor_KeyFileNotFound(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.BindSSHKey("dev", "/nonexistent/key")
	_ = store.SetCurrent("dev")
	_ = config.Save(store)

	_ = git.Apply("dev", "dev@example.com")

	err := runDoctor([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRunDoctor_LocalOverrideNotFlaggedAsMismatch guards against a regression
// where doctor compared the active identity against `git config --global`
// directly. A legitimate `switch --local` deliberately makes the resolved
// git identity differ from the global active one in just this repo — doctor
// must not warn about that as if it were configuration drift.
func TestRunDoctor_LocalOverrideNotFlaggedAsMismatch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	tmpDir := setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("personal", "personal@example.com")
	_ = store.AddUser("eng", "eng@example.com")
	_ = store.SetCurrent("personal")
	_ = config.Save(store)
	if err := git.Apply("personal", "personal@example.com"); err != nil {
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

	if err := runSwitch([]string{"--local", "eng"}); err != nil {
		t.Fatalf("local switch failed: %v", err)
	}

	out := captureStdout(t, func() {
		if err := runDoctor([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if strings.Contains(out, "mismatch") {
		t.Errorf("expected no mismatch warning after a legitimate local override, got output:\n%s", out)
	}
	if !strings.Contains(out, "Local override active") {
		t.Errorf("expected doctor to note the local override, got output:\n%s", out)
	}
}

// TestRunDoctor_GenuineMismatchStillFlagged guards the other direction of the
// same fix: real drift between the active identity and the resolved git
// config (no local override involved) must still be reported.
func TestRunDoctor_GenuineMismatchStillFlagged(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.SetCurrent("dev")
	_ = config.Save(store)
	_ = git.Apply("ops", "ops@example.com") // Mis-matched git config, no local override anywhere

	out := captureStdout(t, func() {
		if err := runDoctor([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "mismatch") {
		t.Errorf("expected a genuine mismatch to still be flagged, got output:\n%s", out)
	}
}

// TestRunDoctor_FixResyncsGitConfig checks that `doctor --fix` actually
// corrects the drift `doctor` (without --fix) only reports — the same case
// as TestRunDoctor_GenuineMismatchStillFlagged, but this time asserting the
// git config is really re-applied afterward, not just that a warning was
// printed.
func TestRunDoctor_FixResyncsGitConfig(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.SetCurrent("dev")
	_ = config.Save(store)
	_ = git.Apply("ops", "ops@example.com") // drifted, no local override

	out := captureStdout(t, func() {
		if err := runDoctor([]string{"--fix"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Fixed") {
		t.Errorf("expected doctor --fix to report a fix, got output:\n%s", out)
	}
	if name := git.CurrentName(); name != "dev" {
		t.Errorf("expected git config re-synced to identity %q, got name %q", "dev", name)
	}
	if email := git.CurrentEmail(); email != "dev@example.com" {
		t.Errorf("expected git config re-synced to dev@example.com, got %q", email)
	}
}

// TestRunDoctor_FixCorrectsInsecurePermissions checks the other
// auto-correctable class doctor --fix handles: chmod'ing an insecure config
// file back to 0600.
func TestRunDoctor_FixCorrectsInsecurePermissions(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = config.Save(store)

	configPath := config.ConfigPath()
	if err := os.Chmod(configPath, 0644); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	if err := runDoctor([]string{"--fix"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected config file fixed to 0600, got %o", perm)
	}
}

// TestTokenExpiryWarning checks the pure date-math helper directly, across
// the boundary cases that matter: already expired, right at the warning
// threshold, safely in the future, and an unparseable date (doctor has no
// fix for a corrupted date, so it must stay silent rather than error out).
func TestTokenExpiryWarning(t *testing.T) {
	past := time.Now().AddDate(0, 0, -5).Format(validate.DateLayout)
	if w := tokenExpiryWarning(past); w == "" || !strings.Contains(w, "expired") {
		t.Errorf("expected an 'expired' warning for a past date, got %q", w)
	}

	soon := time.Now().AddDate(0, 0, tokenExpiryWarnDays-1).Format(validate.DateLayout)
	if w := tokenExpiryWarning(soon); w == "" {
		t.Error("expected a warning for a date inside the warning window")
	}

	farOut := time.Now().AddDate(0, 0, tokenExpiryWarnDays+30).Format(validate.DateLayout)
	if w := tokenExpiryWarning(farOut); w != "" {
		t.Errorf("expected no warning for a date well outside the window, got %q", w)
	}

	if w := tokenExpiryWarning("not-a-date"); w != "" {
		t.Errorf("expected no warning (and no crash) for an unparseable date, got %q", w)
	}
}

// TestRunDoctor_WarnsOnExpiringToken checks doctor surfaces an
// about-to-expire token for the active identity as an issue, not just a
// success line noting the token exists.
func TestRunDoctor_WarnsOnExpiringToken(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("work", "work@example.com")
	_ = store.SetCurrent("work")
	_ = store.SetHTTPSTokenExpiry("work", time.Now().AddDate(0, 0, 3).Format(validate.DateLayout))
	_ = config.Save(store)
	_ = git.Apply("work", "work@example.com")
	if err := keyring.SetHTTPSToken("work", "ghp_abc"); err != nil {
		t.Fatalf("SetHTTPSToken: %v", err)
	}

	out := captureStdout(t, func() {
		if err := runDoctor([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "expires in") {
		t.Errorf("expected doctor to warn about the approaching token expiry, got output:\n%s", out)
	}
}

// TestRunDoctor_SuggestsTokenWhenSSHFailsWithHTTPSRemote checks that doctor
// offers a token as an alternative to fix-remote when it has just watched
// SSH fail for the active identity — not just repeat "convert to SSH" as if
// SSH were guaranteed to be the fix.
func TestRunDoctor_SuggestsTokenWhenSSHFailsWithHTTPSRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	tmpDir := setupTestEnv(t)

	sshDir := filepath.Join(tmpDir, ".ssh")
	_ = os.MkdirAll(sshDir, 0700)
	keyPath := filepath.Join(sshDir, "id_ed25519")
	_ = os.WriteFile(keyPath, []byte("not a real key"), 0600)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.BindSSHKey("dev", keyPath)
	_ = store.SetCurrent("dev")
	_ = config.Save(store)
	_ = git.Apply("dev", "dev@example.com")

	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", repoDir, "init").Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if err := exec.Command("git", "-C", repoDir, "remote", "add", "origin", "https://github.com/example/repo.git").Run(); err != nil {
		t.Fatalf("git remote add: %v", err)
	}
	cwd, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	out := captureStdout(t, func() {
		if err := runDoctor([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "SSH connection failed") {
		t.Fatalf("expected the invalid key to fail the SSH check, got output:\n%s", out)
	}
	if !strings.Contains(out, "since SSH just failed above") {
		t.Errorf("expected doctor to suggest a token as an alternative, got output:\n%s", out)
	}
}

func TestRunDoctor_StaleBackupsAndRemotes(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Create a dummy key and a backup key
	sshDir := filepath.Join(tmpDir, ".ssh")
	_ = os.MkdirAll(sshDir, 0700)
	keyPath := filepath.Join(sshDir, "id_ed25519")
	backupPath := keyPath + ".backup"
	_ = os.WriteFile(keyPath, []byte("private key"), 0600)
	_ = os.WriteFile(backupPath, []byte("backup key"), 0600)

	store, _ := config.Load()
	_ = store.AddUser("dev", "dev@example.com")
	_ = store.BindSSHKey("dev", keyPath)
	_ = store.SetCurrent("dev")
	_ = config.Save(store)

	_ = git.Apply("dev", "dev@example.com")

	err := runDoctor([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
