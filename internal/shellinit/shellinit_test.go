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
		{"cmd", Cmd},
		{"batch", Cmd},
	}
	for _, c := range cases {
		if got := Detect(c.explicit); got != c.want {
			t.Errorf("Detect(%q) = %q, want %q", c.explicit, got, c.want)
		}
	}
}

func TestDetect_WindowsGitBash(t *testing.T) {
	t.Setenv("PSModulePath", `C:\Program Files\WindowsPowerShell\Modules`)
	t.Setenv("SHELL", `/usr/bin/bash`)
	if got := Detect(""); got != Posix {
		t.Errorf("Detect with SHELL=/usr/bin/bash and PSModulePath got %v, want Posix", got)
	}

	t.Setenv("SHELL", "")
	t.Setenv("BASH", `/usr/bin/bash`)
	if got := Detect(""); got != Posix {
		t.Errorf("Detect with BASH set and PSModulePath got %v, want Posix", got)
	}

	t.Setenv("BASH", "")
	t.Setenv("MSYSTEM", "MINGW64")
	if got := Detect(""); got != Posix {
		t.Errorf("Detect with MSYSTEM set and PSModulePath got %v, want Posix", got)
	}
}

func TestPowerShellScript_CommandResolution(t *testing.T) {
	script := Script(PowerShell)
	if strings.Contains(script, "-CommandType Application") {
		t.Errorf("powerShellInitScript should not restrict command lookup with -CommandType Application: %s", script)
	}
	if !strings.Contains(script, "Where-Object") {
		t.Errorf("powerShellInitScript should filter functions using Where-Object: %s", script)
	}
}

func TestResolveShellPath(t *testing.T) {
	sh := ResolveShellPath()
	if sh == "" {
		t.Error("expected non-empty ResolveShellPath")
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
	if !strings.Contains(Script(Cmd), "@echo off") || !strings.Contains(Script(Cmd), "git-user.exe env") {
		t.Error("expected cmd script to contain @echo off and git-user.exe env")
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
