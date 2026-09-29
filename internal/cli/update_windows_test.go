//go:build windows

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallBinary_Success(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "git-user.exe")
	newPath := filepath.Join(tmpDir, "git-user-new.exe")

	if err := os.WriteFile(execPath, []byte("version-1"), 0755); err != nil {
		t.Fatalf("writing old binary: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("version-2"), 0755); err != nil {
		t.Fatalf("writing new binary: %v", err)
	}

	msg, err := installBinary(execPath, newPath)
	if err != nil {
		t.Fatalf("installBinary failed: %v", err)
	}
	if msg != "" {
		t.Errorf("expected empty msg, got %q", msg)
	}

	content, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("reading updated binary: %v", err)
	}
	if string(content) != "version-2" {
		t.Errorf("expected binary content %q, got %q", "version-2", string(content))
	}
}

func TestInstallBinary_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "git-user.exe")
	nonExistentNew := filepath.Join(tmpDir, "does-not-exist.exe")

	if err := os.WriteFile(execPath, []byte("version-1"), 0755); err != nil {
		t.Fatalf("writing old binary: %v", err)
	}

	_, err := installBinary(execPath, nonExistentNew)
	if err == nil {
		t.Fatal("expected error with non-existent new binary, got nil")
	}

	content, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("reading rolled back binary: %v", err)
	}
	if string(content) != "version-1" {
		t.Errorf("expected rolled back content %q, got %q", "version-1", string(content))
	}
}

func TestCleanStaleBackups(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "git-user.exe")
	old1 := filepath.Join(tmpDir, "git-user.exe.old")
	old2 := filepath.Join(tmpDir, "git-user.exe.old.1234")
	keep := filepath.Join(tmpDir, "other.txt")

	_ = os.WriteFile(execPath, []byte("active"), 0755)
	_ = os.WriteFile(old1, []byte("old1"), 0755)
	_ = os.WriteFile(old2, []byte("old2"), 0755)
	_ = os.WriteFile(keep, []byte("keep"), 0644)

	cleanStaleBackups(execPath)

	if _, err := os.Stat(old1); !os.IsNotExist(err) {
		t.Errorf("expected %s to be removed", old1)
	}
	if _, err := os.Stat(old2); !os.IsNotExist(err) {
		t.Errorf("expected %s to be removed", old2)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("expected %s to remain untouched", keep)
	}
}

func TestScheduleNpmUpdateWindows_UnsafeVersion(t *testing.T) {
	err := scheduleNpmUpdateWindows("1.0.0; calc.exe")
	if err == nil {
		t.Error("expected error for unsafe version, got nil")
	}
}
