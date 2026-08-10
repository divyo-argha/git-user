package stats

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmdArgs := append([]string{"-c", "commit.gpgsign=false"}, args...)
	cmd := exec.Command("git", cmdArgs...)
	cmd.Dir = dir
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
