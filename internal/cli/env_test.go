package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestEnvVars(t *testing.T) {
	user := &config.User{
		Name:          "alice",
		Email:         "alice@example.com",
		SSHKey:     "~/.ssh/git_alice",
		SignKey:    "KEY123",
		SignFormat: "ssh",
		CustomConfig: map[string]string{
			"init.defaultBranch": "main",
		},
	}

	vars := EnvVars(user)

	if vars["GIT_AUTHOR_NAME"] != "alice" {
		t.Errorf("GIT_AUTHOR_NAME = %q, want alice", vars["GIT_AUTHOR_NAME"])
	}
	if vars["GIT_AUTHOR_EMAIL"] != "alice@example.com" {
		t.Errorf("GIT_AUTHOR_EMAIL = %q, want alice@example.com", vars["GIT_AUTHOR_EMAIL"])
	}
	if vars["GIT_COMMITTER_NAME"] != "alice" {
		t.Errorf("GIT_COMMITTER_NAME = %q, want alice", vars["GIT_COMMITTER_NAME"])
	}
	if vars["GIT_COMMITTER_EMAIL"] != "alice@example.com" {
		t.Errorf("GIT_COMMITTER_EMAIL = %q, want alice@example.com", vars["GIT_COMMITTER_EMAIL"])
	}
	if vars["GIT_USER_SESSION"] != "alice" {
		t.Errorf("GIT_USER_SESSION = %q, want alice", vars["GIT_USER_SESSION"])
	}
	if !strings.Contains(vars["GIT_SSH_COMMAND"], "ssh -i") || !strings.Contains(vars["GIT_SSH_COMMAND"], "git_alice") {
		t.Errorf("GIT_SSH_COMMAND = %q", vars["GIT_SSH_COMMAND"])
	}
	if !strings.Contains(vars["GIT_CONFIG_PARAMETERS"], "user.name=alice") {
		t.Errorf("GIT_CONFIG_PARAMETERS = %q, want user.name=alice", vars["GIT_CONFIG_PARAMETERS"])
	}
	if !strings.Contains(vars["GIT_CONFIG_PARAMETERS"], "user.email=alice@example.com") {
		t.Errorf("GIT_CONFIG_PARAMETERS = %q, want user.email=alice@example.com", vars["GIT_CONFIG_PARAMETERS"])
	}
	if !strings.Contains(vars["GIT_CONFIG_PARAMETERS"], "user.signingkey=KEY123") {
		t.Errorf("GIT_CONFIG_PARAMETERS = %q", vars["GIT_CONFIG_PARAMETERS"])
	}
	if !strings.Contains(vars["GIT_CONFIG_PARAMETERS"], "init.defaultBranch=main") {
		t.Errorf("GIT_CONFIG_PARAMETERS missing custom config: %q", vars["GIT_CONFIG_PARAMETERS"])
	}
}

func TestRunEnv_Output(t *testing.T) {
	setupTestEnv(t)

	store := &config.Store{
		Current: "dev",
		Users: []config.User{
			{Name: "dev", Email: "dev@example.com", SSHKey: "/tmp/key_dev"},
		},
	}
	if err := config.Save(store); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEnv([]string{"dev", "--posix"})
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runEnv failed: %v", err)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "export GIT_AUTHOR_NAME=\"dev\"") {
		t.Errorf("expected export GIT_AUTHOR_NAME in output, got:\n%s", output)
	}
	if !strings.Contains(output, "export GIT_AUTHOR_EMAIL=\"dev@example.com\"") {
		t.Errorf("expected export GIT_AUTHOR_EMAIL in output, got:\n%s", output)
	}
}

func TestRunEnv_Unset(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runEnv([]string{"--unset", "--posix"})
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runEnv --unset failed: %v", err)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "unset GIT_AUTHOR_NAME") {
		t.Errorf("expected unset statement in output, got:\n%s", output)
	}
}
