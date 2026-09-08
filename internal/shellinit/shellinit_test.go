package shellinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	cases := []struct {
		explicit string
		want     Shell
	}{
		{"fish", Fish},
		{"zsh", Posix},
		{"bash", Posix},
		{"pwsh", PowerShell},
		{"powershell", PowerShell},
	}
	for _, c := range cases {
		if got := Detect(c.explicit); got != c.want {
			t.Errorf("Detect(%q) = %q, want %q", c.explicit, got, c.want)
		}
	}
}

func TestScript(t *testing.T) {
	if !strings.Contains(Script(Posix), "git-user()") {
		t.Error("expected posix script to define a git-user() wrapper function")
	}
	if !strings.Contains(Script(Fish), "function git-user") {
		t.Error("expected fish script to define a git-user function")
	}
	if !strings.Contains(Script(PowerShell), "function git-user") {
		t.Error("expected powershell script to define a git-user function")
	}
}

func TestInstallFreshUpgradeAlready(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome) // os.UserHomeDir() on Windows

	bashrc := filepath.Join(tmpHome, ".bashrc")
	zshrc := filepath.Join(tmpHome, ".zshrc")

	// Fresh install into bashrc.
	results, err := Install(Posix, "bash")
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(results) != 1 || results[0].File != bashrc || results[0].Status != StatusInstalled {
		t.Fatalf("expected a single StatusInstalled result for %s, got %+v", bashrc, results)
	}
	content, err := os.ReadFile(bashrc)
	if err != nil {
		t.Fatalf("reading bashrc: %v", err)
	}
	if !strings.Contains(string(content), `command -v git-user >/dev/null 2>&1 && eval "$(git-user init 2>/dev/null)"`) {
		t.Fatalf("expected safe integration snippet in bashrc, got:\n%s", content)
	}

	// Installing again is a no-op (already installed).
	results, err = Install(Posix, "bash")
	if err != nil {
		t.Fatalf("Install (already): %v", err)
	}
	if len(results) != 1 || results[0].Status != StatusAlready {
		t.Fatalf("expected StatusAlready on reinstall, got %+v", results)
	}

	// A legacy unshielded snippet in zshrc gets upgraded in place.
	if err := os.WriteFile(zshrc, []byte("# custom\neval \"$(git-user init)\"\n"), 0644); err != nil {
		t.Fatalf("writing legacy zshrc: %v", err)
	}
	results, err = Install(Posix, "zsh")
	if err != nil {
		t.Fatalf("Install (upgrade): %v", err)
	}
	if len(results) != 1 || results[0].File != zshrc || results[0].Status != StatusUpgraded {
		t.Fatalf("expected StatusUpgraded for %s, got %+v", zshrc, results)
	}
	zshContent, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf("reading upgraded zshrc: %v", err)
	}
	if strings.Contains(string(zshContent), `eval "$(git-user init)"`) && !strings.Contains(string(zshContent), "command -v git-user") {
		t.Fatalf("legacy snippet was not upgraded, got:\n%s", zshContent)
	}
}
