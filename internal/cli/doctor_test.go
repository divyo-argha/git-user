package cli

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
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
