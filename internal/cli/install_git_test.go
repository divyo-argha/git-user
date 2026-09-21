package cli

import (
	"os"
	"strings"
	"testing"
)

func TestRunInstallGit_WhenInstalled(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runInstallGit(nil)
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runInstallGit failed: %v", err)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "already installed") {
		t.Errorf("expected 'already installed' message when git is present, got:\n%s", output)
	}
}
