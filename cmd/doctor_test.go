package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

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
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	_ = git.Apply("bob", "bob@example.com") // Mis-matched git config

	err := runDoctor([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDoctor_KeyFileNotFound(t *testing.T) {
	setupTestEnv(t)

	store, _ := config.Load()
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.BindSSHKey("alice", "/nonexistent/key")
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	_ = git.Apply("alice", "alice@example.com")

	err := runDoctor([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
	_ = store.AddUser("alice", "alice@example.com")
	_ = store.BindSSHKey("alice", keyPath)
	_ = store.SetCurrent("alice")
	_ = config.Save(store)

	_ = git.Apply("alice", "alice@example.com")

	err := runDoctor([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
