package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/bundle"
	"github.com/divyo-argha/git-user/internal/config"
)

// TestRunSync_RejectsPathTraversalIdentityName proves that a malicious sync
// remote cannot use a crafted identity Name (e.g. "../evil-marker") to make
// `git-user sync` write a file outside ~/.ssh. The write in runSync's merge
// loop must go through config.DefaultSSHKeyPath, which validates the name
// before any path is ever constructed — this test fails if that validation
// is ever removed or bypassed by a future edit.
func TestRunSync_RejectsPathTraversalIdentityName(t *testing.T) {
	tmpDir := setupTestEnv(t)

	_ = exec.Command("git", "config", "--global", "user.name", "Test User").Run()
	_ = exec.Command("git", "config", "--global", "user.email", "test@example.com").Run()

	// A bare remote repo standing in for the (potentially compromised or
	// malicious-collaborator-controlled) sync remote.
	remoteRepoDir := filepath.Join(tmpDir, "remote-backup-repo")
	if err := os.Mkdir(remoteRepoDir, 0755); err != nil {
		t.Fatalf("failed to create remote dir: %v", err)
	}
	runGitCmd(t, remoteRepoDir, "init", "--bare")
	runGitCmd(t, remoteRepoDir, "symbolic-ref", "HEAD", "refs/heads/main")

	passphrase := "secretpass"

	// Craft a malicious bundle directly (bypassing config.Store.AddUser
	// entirely, since a real attacker controlling the sync remote is not
	// bound by this codebase's own validation) with a path-traversal name.
	// "../../authorized_keys" resolves through the vulnerable
	// filepath.Join(home, ".ssh", fmt.Sprintf("git_%s", name)) pattern to
	// exactly ~/.ssh/authorized_keys — i.e. an attacker-chosen "private key"
	// byte blob would silently become the victim's SSH authorized_keys file,
	// granting the attacker direct SSH access to the account.
	maliciousIdentities := []bundle.Identity{
		{
			Name:       "../../authorized_keys",
			Email:      "attacker@example.com",
			PrivateKey: []byte("ssh-ed25519 AAAAATTACKERCONTROLLEDKEY attacker@evil\n"),
		},
	}
	encrypted, err := bundle.Encrypt(maliciousIdentities, passphrase)
	if err != nil {
		t.Fatalf("failed to craft malicious bundle: %v", err)
	}

	// Push the malicious bundle to the bare remote, exactly as a legitimate
	// sync push would, so it's what a victim's `git-user sync` pulls next.
	pushDir := filepath.Join(tmpDir, "attacker-push-clone")
	if err := os.MkdirAll(pushDir, 0700); err != nil {
		t.Fatalf("failed to create push clone dir: %v", err)
	}
	runGitCmd(t, pushDir, "init")
	runGitCmd(t, pushDir, "config", "user.name", "attacker")
	runGitCmd(t, pushDir, "config", "user.email", "attacker@example.com")
	runGitCmd(t, pushDir, "remote", "add", "origin", remoteRepoDir)
	runGitCmd(t, pushDir, "branch", "-M", "main")
	if err := os.WriteFile(filepath.Join(pushDir, "backup.bundle"), encrypted, 0600); err != nil {
		t.Fatalf("failed to write malicious bundle: %v", err)
	}
	runGitCmd(t, pushDir, "add", "backup.bundle")
	runGitCmd(t, pushDir, "commit", "-m", "malicious update")
	runGitCmd(t, pushDir, "push", "origin", "main")

	// Now the victim configures sync against the compromised remote and runs
	// a normal sync.
	store, _ := config.Load()
	store.Sync = &config.SyncConfig{RepoURL: remoteRepoDir, DeviceName: "victim-device"}
	_ = config.Save(store)

	readPassphraseFn = func(prompt string) (string, error) {
		return passphrase, nil
	}
	t.Cleanup(func() { readPassphraseFn = nil })

	if err := runSync([]string{}); err != nil {
		t.Fatalf("unexpected error during sync: %v", err)
	}

	// Sanity check that the test actually exercised the merge path (i.e. the
	// malicious bundle really was pulled and decrypted) rather than passing
	// vacuously because nothing was fetched.
	home0, _ := os.UserHomeDir()
	syncedBundlePath := filepath.Join(home0, ".git-users", "sync", "backup.bundle")
	if _, err := os.Stat(syncedBundlePath); err != nil {
		t.Fatalf("expected malicious backup.bundle to have been pulled to %s: %v", syncedBundlePath, err)
	}

	// The identity must not have been merged into the local store at all —
	// its invalid name is rejected before store.AddUser is ever called.
	store, _ = config.Load()
	if u := store.FindUser("../../authorized_keys"); u != nil {
		t.Fatal("malicious identity with a path-traversal name should not have been imported")
	}

	// The actual traversal target: filepath.Join(home, ".ssh",
	// fmt.Sprintf("git_%s", "../../authorized_keys")) resolves to exactly
	// ~/.ssh/authorized_keys. It must not exist — proving the attacker's
	// "private key" bytes were never written there.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}
	traversalTarget := filepath.Join(home, ".ssh", "authorized_keys")
	if _, statErr := os.Stat(traversalTarget); statErr == nil {
		t.Fatalf("path-traversal write succeeded — found attacker-controlled file at %s", traversalTarget)
	}
}

// TestRunSync_RejectsConfigInjectionEmail proves that a malicious sync remote
// cannot use a crafted identity Email containing embedded newlines to inject
// arbitrary directives into the hand-written .gitconfig snippet that
// syncIncludeIfs generates for bind-path identities (e.g. smuggling in a
// [core] sshCommand override). config.AddUser now validates email format
// before an identity is ever merged into the store, so a malicious email
// never reaches the snippet writer in the first place.
func TestRunSync_RejectsConfigInjectionEmail(t *testing.T) {
	tmpDir := setupTestEnv(t)

	_ = exec.Command("git", "config", "--global", "user.name", "Test User").Run()
	_ = exec.Command("git", "config", "--global", "user.email", "test@example.com").Run()

	remoteRepoDir := filepath.Join(tmpDir, "remote-backup-repo")
	if err := os.Mkdir(remoteRepoDir, 0755); err != nil {
		t.Fatalf("failed to create remote dir: %v", err)
	}
	runGitCmd(t, remoteRepoDir, "init", "--bare")
	runGitCmd(t, remoteRepoDir, "symbolic-ref", "HEAD", "refs/heads/main")

	passphrase := "secretpass"

	// A valid name, but an Email crafted to break out of the snippet's
	// `email = ...` line and inject a new [core] section if it were ever
	// written unescaped.
	maliciousEmail := "attacker@example.com\"\n[core]\n\tsshCommand = evil-command\n"
	maliciousIdentities := []bundle.Identity{
		{
			Name:  "attacker-profile",
			Email: maliciousEmail,
		},
	}
	encrypted, err := bundle.Encrypt(maliciousIdentities, passphrase)
	if err != nil {
		t.Fatalf("failed to craft malicious bundle: %v", err)
	}

	pushDir := filepath.Join(tmpDir, "attacker-push-clone")
	if err := os.MkdirAll(pushDir, 0700); err != nil {
		t.Fatalf("failed to create push clone dir: %v", err)
	}
	runGitCmd(t, pushDir, "init")
	runGitCmd(t, pushDir, "config", "user.name", "attacker")
	runGitCmd(t, pushDir, "config", "user.email", "attacker@example.com")
	runGitCmd(t, pushDir, "remote", "add", "origin", remoteRepoDir)
	runGitCmd(t, pushDir, "branch", "-M", "main")
	if err := os.WriteFile(filepath.Join(pushDir, "backup.bundle"), encrypted, 0600); err != nil {
		t.Fatalf("failed to write malicious bundle: %v", err)
	}
	runGitCmd(t, pushDir, "add", "backup.bundle")
	runGitCmd(t, pushDir, "commit", "-m", "malicious update")
	runGitCmd(t, pushDir, "push", "origin", "main")

	store, _ := config.Load()
	store.Sync = &config.SyncConfig{RepoURL: remoteRepoDir, DeviceName: "victim-device"}
	_ = config.Save(store)

	readPassphraseFn = func(prompt string) (string, error) {
		return passphrase, nil
	}
	t.Cleanup(func() { readPassphraseFn = nil })

	if err := runSync([]string{}); err != nil {
		t.Fatalf("unexpected error during sync: %v", err)
	}

	// The identity must not have been merged — its malformed email is
	// rejected by config.AddUser before it can ever reach a config file.
	store, _ = config.Load()
	if u := store.FindUser("attacker-profile"); u != nil {
		t.Fatal("identity with a config-injection email should not have been imported")
	}
}
