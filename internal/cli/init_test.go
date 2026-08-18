package cli

import (
	"os"
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
