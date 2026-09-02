package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit_Posix(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runInit([]string{"bash"})
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "git-user()") {
		t.Errorf("expected wrapper function in posix init script, got:\n%s", output)
	}
	if !strings.Contains(output, "eval \"$(command git-user env \"$@\")\"") {
		t.Errorf("expected eval call in posix init script, got:\n%s", output)
	}
}

func TestRunInit_Fish(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runInit([]string{"fish"})
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runInit fish failed: %v", err)
	}

	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "function git-user") {
		t.Errorf("expected fish function in fish init script, got:\n%s", output)
	}
}

func TestRunInit_PowerShell(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runInit([]string{"powershell"})
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runInit powershell failed: %v", err)
	}

	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "function git-user") {
		t.Errorf("expected powershell function in powershell init script, got:\n%s", output)
	}
}

func TestRunInit_InstallAndUpgrade(t *testing.T) {
	tmpDir := setupTestEnv(t)
	bashrc := filepath.Join(tmpDir, ".bashrc")
	zshrc := filepath.Join(tmpDir, ".zshrc")

	// 1. Install fresh into bashrc
	if err := runInit([]string{"install", "--shell", "bash"}); err != nil {
		t.Fatalf("runInit install failed: %v", err)
	}

	content, err := os.ReadFile(bashrc)
	if err != nil {
		t.Fatalf("failed to read bashrc: %v", err)
	}
	if !strings.Contains(string(content), "command -v git-user >/dev/null 2>&1 && eval \"$(git-user init 2>/dev/null)\"") {
		t.Fatalf("expected safe shell integration in bashrc, got:\n%s", string(content))
	}

	// 2. Simulate legacy snippet in zshrc
	if err := os.WriteFile(zshrc, []byte("# some other config\neval \"$(git-user init)\"\n"), 0644); err != nil {
		t.Fatalf("failed to write legacy zshrc: %v", err)
	}

	// 3. Upgrade legacy snippet in zshrc
	if err := runInit([]string{"install", "--shell", "zsh"}); err != nil {
		t.Fatalf("runInit install upgrade failed: %v", err)
	}

	zshContent, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf("failed to read upgraded zshrc: %v", err)
	}
	if strings.Contains(string(zshContent), "eval \"$(git-user init)\"") && !strings.Contains(string(zshContent), "command -v git-user") {
		t.Fatalf("expected legacy snippet to be upgraded, got:\n%s", string(zshContent))
	}
	if !strings.Contains(string(zshContent), "command -v git-user >/dev/null 2>&1 && eval \"$(git-user init 2>/dev/null)\"") {
		t.Fatalf("expected safe shell integration in zshrc, got:\n%s", string(zshContent))
	}

	// 4. Test removeShellIntegration cleans both up
	removeShellIntegration()

	bashCleaned, _ := os.ReadFile(bashrc)
	if strings.Contains(string(bashCleaned), "git-user init") {
		t.Fatalf("expected bashrc to be cleaned up after removeShellIntegration, got:\n%s", string(bashCleaned))
	}

	zshCleaned, _ := os.ReadFile(zshrc)
	if strings.Contains(string(zshCleaned), "git-user init") {
		t.Fatalf("expected zshrc to be cleaned up after removeShellIntegration, got:\n%s", string(zshCleaned))
	}
}

