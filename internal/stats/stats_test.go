package stats

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func filterEnv(env []string) []string {
	var filtered []string
	for _, e := range env {
		if strings.HasPrefix(e, "GIT_AUTHOR_NAME=") && e == "GIT_AUTHOR_NAME=" {
			continue
		}
		if strings.HasPrefix(e, "GIT_AUTHOR_EMAIL=") && e == "GIT_AUTHOR_EMAIL=" {
			continue
		}
		if strings.HasPrefix(e, "GIT_COMMITTER_NAME=") {
			continue
		}
		if strings.HasPrefix(e, "GIT_COMMITTER_EMAIL=") {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmdArgs := append([]string{"-c", "commit.gpgsign=false"}, args...)
	cmd := exec.Command("git", cmdArgs...)
	cmd.Dir = dir
	cmd.Env = append(filterEnv(os.Environ()),
		"GIT_COMMITTER_NAME=Test Committer",
		"GIT_COMMITTER_EMAIL=committer@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %v\nOutput: %s", args, dir, err, string(out))
	}
}

func TestAuditRepository_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "commit.gpgsign", "false")
	runGit(t, tmpDir, "config", "user.name", "Niloy")
	runGit(t, tmpDir, "config", "user.email", "tonmoy@shellbeehaken.com")

	// Commit 1 with Name "Niloy"
	file1 := filepath.Join(tmpDir, "file1.txt")
	_ = os.WriteFile(file1, []byte("content 1"), 0644)
	runGit(t, tmpDir, "add", "file1.txt")
	runGit(t, tmpDir, "commit", "-m", "commit 1")

	// Commit 2 with Name "Niloy Rashid" but SAME email
	file2 := filepath.Join(tmpDir, "file2.txt")
	_ = os.WriteFile(file2, []byte("content 2"), 0644)
	runGit(t, tmpDir, "add", "file2.txt")
	runGit(t, tmpDir, "commit", "--author=Niloy Rashid <tonmoy@shellbeehaken.com>", "-m", "commit 2")

	// Commit 3 with secondary email mapped to user alias
	file3 := filepath.Join(tmpDir, "file3.txt")
	_ = os.WriteFile(file3, []byte("content 3"), 0644)
	runGit(t, tmpDir, "add", "file3.txt")
	runGit(t, tmpDir, "commit", "--author=Niloy Rashid <niloyrashid71@gmail.com>", "-m", "commit 3")

	// Set up config store with registered user & alias
	store := &config.Store{
		Users: []config.User{
			{
				Name:    "Niloy Rashid Profile",
				Email:   "tonmoy@shellbeehaken.com",
				Aliases: []string{"niloyrashid71@gmail.com"},
			},
		},
	}

	// Change working dir for git calls inside AuditRepository
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	_ = os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	results, err := AuditRepository(store, "")
	if err != nil {
		t.Fatalf("unexpected error auditing repository: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 unified author group due to deduplication, got %d", len(results))
	}

	stat := results[0]
	if stat.Commits != 3 {
		t.Errorf("expected 3 total commits, got %d", stat.Commits)
	}

	if stat.DisplayName != "Niloy Rashid Profile" {
		t.Errorf("expected DisplayName 'Niloy Rashid Profile', got %q", stat.DisplayName)
	}

	if stat.VerifiedUser == nil {
		t.Errorf("expected VerifiedUser to be non-nil")
	}
}

func TestAuditRepository_PathFiltering(t *testing.T) {
	tmpDir := t.TempDir()

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "commit.gpgsign", "false")
	runGit(t, tmpDir, "config", "user.name", "Test Committer")
	runGit(t, tmpDir, "config", "user.email", "committer@example.com")

	subDir := filepath.Join(tmpDir, "subdir")
	_ = os.MkdirAll(subDir, 0755)

	// Commit 1 in root dir by User A
	_ = os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("root"), 0644)
	runGit(t, tmpDir, "add", "root.txt")
	runGit(t, tmpDir, "commit", "--author=User A <usera@example.com>", "-m", "root commit")

	// Commit 2 in subdir by User B
	_ = os.WriteFile(filepath.Join(subDir, "sub.txt"), []byte("sub"), 0644)
	runGit(t, tmpDir, "add", "subdir/sub.txt")
	runGit(t, tmpDir, "commit", "--author=User B <userb@example.com>", "-m", "sub commit")

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	_ = os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Audit path "subdir"
	results, err := AuditRepository(nil, "subdir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 author stat for subdir, got %d", len(results))
	}

	if results[0].Email != "userb@example.com" {
		t.Errorf("expected userb@example.com, got %s", results[0].Email)
	}
}

func TestAuditRepository_CodeLinesAndCommentFiltering(t *testing.T) {
	tmpDir := t.TempDir()

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "commit.gpgsign", "false")
	runGit(t, tmpDir, "config", "user.name", "Test Committer")
	runGit(t, tmpDir, "config", "user.email", "committer@example.com")

	// Commit 1 with Python comments and code
	pyFile := filepath.Join(tmpDir, "app.py")
	pyContent := "# This is a comment\n\nprint('hello')\n\"\"\" docstring \"\"\"\nval = 42\n"
	_ = os.WriteFile(pyFile, []byte(pyContent), 0644)
	runGit(t, tmpDir, "add", "app.py")
	runGit(t, tmpDir, "commit", "--author=Dev Py <py@example.com>", "-m", "py commit")

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	_ = os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	results, err := AuditRepositoryMode(nil, "", SortByLines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	res := results[0]
	// Expected code lines: print('hello') and val = 42 -> 2 code lines! (comments and blank lines excluded)
	if res.CodeLinesAdded != 2 {
		t.Errorf("expected 2 code lines added (excluding comments & blanks), got %d", res.CodeLinesAdded)
	}
}

// genSSHKey generates a fresh ed25519 SSH keypair in dir and returns the
// private key path (the public key is at path+".pub").
func genSSHKey(t *testing.T, dir, name string) string {
	t.Helper()
	keyPath := filepath.Join(dir, name)
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-C", name, "-f", keyPath, "-q")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen failed: %v\n%s", err, out)
	}
	return keyPath
}

func TestAuditRepositoryMode_SSHSignedCommitByUnregisteredKeyCountsAsSigned(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := genSSHKey(t, tmpDir, "id_ed25519")

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.name", "SSH Signer")
	runGit(t, tmpDir, "config", "user.email", "sshsigner@example.com")

	file := filepath.Join(tmpDir, "f.txt")
	_ = os.WriteFile(file, []byte("content"), 0644)
	addCmd := exec.Command("git", "add", "f.txt")
	addCmd.Dir = tmpDir
	if out, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}
	commitCmd := exec.Command("git", "-c", "gpg.format=ssh", "-c", "commit.gpgsign=true", "-c", "user.signingkey="+keyPath, "commit", "-m", "ssh signed commit")
	commitCmd.Dir = tmpDir
	commitCmd.Env = append(filterEnv(os.Environ()),
		"GIT_COMMITTER_NAME=Test Committer",
		"GIT_COMMITTER_EMAIL=committer@example.com",
	)
	if out, err := commitCmd.CombinedOutput(); err != nil {
		t.Fatalf("ssh signed commit failed: %v\n%s", err, out)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	_ = os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// No store registered — this key belongs to nobody git-user knows about.
	// It must still count as Signed (the commit really is SSH-signed), even
	// though the signer is not a trusted/known principal.
	results, err := AuditRepository(nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	stat := results[0]
	if stat.SignedCommits != 1 {
		t.Errorf("expected 1 SignedCommits for an SSH-signed commit (even by an unregistered key), got %d", stat.SignedCommits)
	}
	if stat.UnsignedCommits != 0 {
		t.Errorf("expected 0 UnsignedCommits, got %d", stat.UnsignedCommits)
	}
	if stat.IsRegisteredIdentity() {
		t.Errorf("expected identity to be unregistered (no matching config.User)")
	}
}

func TestAuditRepositoryMode_SSHSignedCommitByRegisteredKeyIsSignedAndRegistered(t *testing.T) {
	tmpDir := t.TempDir()
	keyDir := t.TempDir()
	keyPath := genSSHKey(t, keyDir, "id_ed25519")

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.name", "Registered Signer")
	runGit(t, tmpDir, "config", "user.email", "registered@example.com")

	file := filepath.Join(tmpDir, "f.txt")
	_ = os.WriteFile(file, []byte("content"), 0644)
	addCmd := exec.Command("git", "add", "f.txt")
	addCmd.Dir = tmpDir
	if out, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}
	commitCmd := exec.Command("git", "-c", "gpg.format=ssh", "-c", "commit.gpgsign=true", "-c", "user.signingkey="+keyPath, "commit", "-m", "ssh signed by registered key")
	commitCmd.Dir = tmpDir
	commitCmd.Env = append(filterEnv(os.Environ()),
		"GIT_COMMITTER_NAME=Test Committer",
		"GIT_COMMITTER_EMAIL=committer@example.com",
	)
	if out, err := commitCmd.CombinedOutput(); err != nil {
		t.Fatalf("ssh signed commit failed: %v\n%s", err, out)
	}

	store := &config.Store{
		Users: []config.User{
			{Name: "Registered Signer Profile", Email: "registered@example.com", SSHKey: keyPath},
		},
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	_ = os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	results, err := AuditRepository(store, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	stat := results[0]
	if stat.SignedCommits != 1 {
		t.Errorf("expected 1 SignedCommits for an SSH-signed commit by a registered key, got %d", stat.SignedCommits)
	}
	if !stat.IsRegisteredIdentity() {
		t.Errorf("expected identity to be registered (email matches config.User)")
	}
}

func requireGPG(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("gpg"); err != nil {
		t.Skip("gpg binary not available in PATH; skipping real-signature test")
	}
}

// genGPGKey creates a fresh ed25519 signing-only GPG key in an ephemeral
// GNUPGHOME (set via t.Setenv so it's inherited by the git subprocesses
// AuditRepositoryMode invokes) and returns the GNUPGHOME dir and the key's
// fingerprint. Callers must explicitly pass -c gpg.format=openpgp on every
// git invocation that signs a commit — do NOT rely on ambient gitconfig,
// which may default gpg.format to "ssh" on some machines.
func genGPGKey(t *testing.T, email string) (gnupgHome, fingerprint string) {
	t.Helper()
	gnupgHome = t.TempDir()
	t.Setenv("GNUPGHOME", gnupgHome)

	cmd := exec.Command("gpg", "--batch", "--passphrase", "", "--quick-generate-key", "Test Signer <"+email+">", "ed25519", "sign", "0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gpg key generation failed: %v\n%s", err, out)
	}

	out, err := exec.Command("gpg", "--list-secret-keys", "--with-colons").Output()
	if err != nil {
		t.Fatalf("gpg --list-secret-keys failed: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "fpr:") {
			fields := strings.Split(line, ":")
			if len(fields) > 9 {
				fingerprint = fields[9]
				break
			}
		}
	}
	if fingerprint == "" {
		t.Fatalf("could not determine fingerprint of generated GPG key; output was:\n%s", out)
	}
	return gnupgHome, fingerprint
}

func TestAuditRepositoryMode_UnsignedCommitNotCountedAsSigned(t *testing.T) {
	tmpDir := t.TempDir()
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "commit.gpgsign", "false")
	runGit(t, tmpDir, "config", "user.name", "Plain Dev")
	runGit(t, tmpDir, "config", "user.email", "plain@example.com")

	file := filepath.Join(tmpDir, "f.txt")
	_ = os.WriteFile(file, []byte("content"), 0644)
	runGit(t, tmpDir, "add", "f.txt")
	runGit(t, tmpDir, "commit", "-m", "unsigned commit")

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	_ = os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	results, err := AuditRepository(nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	stat := results[0]
	if stat.SignedCommits != 0 {
		t.Errorf("expected 0 SignedCommits for an unsigned commit, got %d", stat.SignedCommits)
	}
	if stat.UnsignedCommits != 1 {
		t.Errorf("expected 1 UnsignedCommits, got %d", stat.UnsignedCommits)
	}
	if stat.RevokedSignatureCommits != 0 || stat.BadSignatureCommits != 0 || stat.UnverifiableCommits != 0 {
		t.Errorf("expected no revoked/bad/unverifiable commits, got R=%d B=%d E=%d", stat.RevokedSignatureCommits, stat.BadSignatureCommits, stat.UnverifiableCommits)
	}
}

func TestAuditRepositoryMode_RealGPGSignedCommitCountsAsSigned(t *testing.T) {
	requireGPG(t)
	tmpDir := t.TempDir()
	_, fpr := genGPGKey(t, "signer@example.com")

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.name", "Test Signer")
	runGit(t, tmpDir, "config", "user.email", "signer@example.com")

	file := filepath.Join(tmpDir, "f.txt")
	_ = os.WriteFile(file, []byte("content"), 0644)
	addCmd := exec.Command("git", "add", "f.txt")
	addCmd.Dir = tmpDir
	if out, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}
	// Explicit gpg.format=openpgp overrides any ambient gitconfig default.
	commitCmd := exec.Command("git", "-c", "gpg.format=openpgp", "-c", "commit.gpgsign=true", "-c", "user.signingkey="+fpr, "commit", "-m", "signed commit")
	commitCmd.Dir = tmpDir
	commitCmd.Env = append(filterEnv(os.Environ()),
		"GIT_COMMITTER_NAME=Test Committer",
		"GIT_COMMITTER_EMAIL=committer@example.com",
	)
	if out, err := commitCmd.CombinedOutput(); err != nil {
		t.Fatalf("signed commit failed: %v\n%s", err, out)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	_ = os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	results, err := AuditRepository(nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	stat := results[0]
	if stat.SignedCommits != 1 {
		t.Errorf("expected 1 SignedCommits for a real gpg-signed commit, got %d", stat.SignedCommits)
	}
	if stat.UnsignedCommits != 0 || stat.RevokedSignatureCommits != 0 || stat.BadSignatureCommits != 0 {
		t.Errorf("expected no unsigned/revoked/bad commits, got Unsigned=%d Revoked=%d Bad=%d", stat.UnsignedCommits, stat.RevokedSignatureCommits, stat.BadSignatureCommits)
	}
}

func TestAuditRepositoryMode_RevokedKeySignatureNotCountedAsSigned(t *testing.T) {
	requireGPG(t)
	tmpDir := t.TempDir()
	gnupgHome, fpr := genGPGKey(t, "revoked@example.com")

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.name", "Revoked Signer")
	runGit(t, tmpDir, "config", "user.email", "revoked@example.com")
	file := filepath.Join(tmpDir, "f.txt")
	_ = os.WriteFile(file, []byte("content"), 0644)
	addCmd := exec.Command("git", "add", "f.txt")
	addCmd.Dir = tmpDir
	if out, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}
	commitCmd := exec.Command("git", "-c", "gpg.format=openpgp", "-c", "commit.gpgsign=true", "-c", "user.signingkey="+fpr, "commit", "-m", "will be revoked")
	commitCmd.Dir = tmpDir
	commitCmd.Env = append(filterEnv(os.Environ()),
		"GIT_COMMITTER_NAME=Test Committer",
		"GIT_COMMITTER_EMAIL=committer@example.com",
	)
	if out, err := commitCmd.CombinedOutput(); err != nil {
		t.Fatalf("signed commit failed: %v\n%s", err, out)
	}

	// Find and import the auto-generated revocation certificate for this key,
	// stripping the leading safety colon gpg prepends to "-----BEGIN...".
	revFiles, err := filepath.Glob(filepath.Join(gnupgHome, "openpgp-revocs.d", "*.rev"))
	if err != nil || len(revFiles) == 0 {
		t.Fatalf("could not find generated revocation certificate: %v", err)
	}
	raw, err := os.ReadFile(revFiles[0])
	if err != nil {
		t.Fatalf("failed to read revocation cert: %v", err)
	}
	cleaned := strings.Replace(string(raw), ":-----BEGIN", "-----BEGIN", 1)
	cleanedPath := filepath.Join(t.TempDir(), "revoke.asc")
	if err := os.WriteFile(cleanedPath, []byte(cleaned), 0644); err != nil {
		t.Fatalf("failed to write cleaned revocation cert: %v", err)
	}
	importCmd := exec.Command("gpg", "--batch", "--yes", "--import", cleanedPath)
	if out, err := importCmd.CombinedOutput(); err != nil {
		t.Fatalf("gpg import of revocation cert failed: %v\n%s", err, out)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	_ = os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	results, err := AuditRepository(nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	stat := results[0]
	if stat.SignedCommits != 0 {
		t.Errorf("a commit signed by a since-revoked key must NOT count as SignedCommits, got %d", stat.SignedCommits)
	}
	if stat.RevokedSignatureCommits != 1 {
		t.Errorf("expected 1 RevokedSignatureCommits, got %d", stat.RevokedSignatureCommits)
	}
}
