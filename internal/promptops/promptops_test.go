package promptops

import (
	"os"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestTargetName(t *testing.T) {
	for _, target := range AllTargets() {
		name := TargetName(target)
		if name == "" {
			t.Errorf("TargetName for %s is empty", target)
		}
		desc := targetDescription(target)
		if desc == "" {
			t.Errorf("targetDescription for %s is empty", target)
		}
	}
}

func TestInstallAndUninstall_Targets(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	targets := []Target{
		TargetStarship,
		TargetZsh,
		TargetBash,
		TargetFish,
		TargetPowerShell,
		TargetNushell,
	}

	for _, target := range targets {
		t.Run(string(target), func(t *testing.T) {
			// Initially not installed
			info := CheckTarget(target)
			if info.Installed {
				t.Fatalf("expected target %s to be not installed initially", target)
			}

			// Install
			msg, err := Install(target)
			if err != nil {
				t.Fatalf("Install(%s) failed: %v", target, err)
			}
			if msg == "" {
				t.Errorf("expected non-empty message on install")
			}

			// Check installed
			info = CheckTarget(target)
			if !info.Installed {
				t.Fatalf("expected target %s to be installed after Install()", target)
			}

			// Install again (idempotent)
			msg2, err := Install(target)
			if err != nil {
				t.Fatalf("second Install(%s) failed: %v", target, err)
			}
			if !strings.Contains(msg2, "already") && !strings.Contains(msg2, "Installed") {
				t.Logf("second install msg: %s", msg2)
			}

			// Uninstall
			lines, err := Uninstall(target)
			if err != nil {
				t.Fatalf("Uninstall(%s) failed: %v", target, err)
			}
			if len(lines) == 0 {
				t.Errorf("expected lines from Uninstall(%s)", target)
			}

			// Check not installed
			info = CheckTarget(target)
			if info.Installed {
				t.Fatalf("expected target %s to not be installed after Uninstall()", target)
			}
		})
	}
}

func TestResolvePrompt(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)

	store := &config.Store{
		Current: "alice",
		Users: []config.User{
			{Name: "alice", Email: "alice@example.com"},
			{Name: "bob", Email: "bob@example.com"},
		},
	}

	// Case 1: Session environment override
	t.Setenv("GIT_USER_SESSION", "work-profile")
	prompt := ResolvePrompt(store, false, false, false)
	if prompt != "work-profile (session)" {
		t.Errorf("expected 'work-profile (session)', got %q", prompt)
	}

	// Plain session
	promptPlain := ResolvePrompt(store, false, true, false)
	if promptPlain != "work-profile" {
		t.Errorf("expected 'work-profile', got %q", promptPlain)
	}

	t.Setenv("GIT_USER_SESSION", "")

	// Case 2: Always show
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("GIT_USER_PROMPT_ICON", "test:")
	promptAlways := ResolvePrompt(store, true, false, true)
	if promptAlways != "test:alice" {
		t.Errorf("expected 'test:alice', got %q", promptAlways)
	}

	// Case 3: Empty when not in repo and not always
	t.Setenv("GIT_USER_PROMPT_ICON", "")
	promptOutside := ResolvePrompt(store, false, false, false)
	if promptOutside != "" {
		t.Errorf("expected empty string outside repo, got %q", promptOutside)
	}

	// Case 4: store.Prompt.Icon == "none" should produce no icon prefix
	store.Prompt = &config.PromptConfig{Icon: "none"}
	promptNoIcon := ResolvePrompt(store, true, false, true)
	if promptNoIcon != "alice" {
		t.Errorf("expected 'alice' without icon prefix, got %q", promptNoIcon)
	}

	// Case 5: store.Prompt.Plain == true should disable badges
	t.Setenv("GIT_USER_SESSION", "session-user")
	store.Prompt = &config.PromptConfig{Plain: true}
	promptConfigPlain := ResolvePrompt(store, false, false, false)
	if promptConfigPlain != "session-user" {
		t.Errorf("expected 'session-user' with plain=true from config, got %q", promptConfigPlain)
	}
	t.Setenv("GIT_USER_SESSION", "")
}

func TestUninstallAll(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Install a few targets
	_, _ = Install(TargetBash)
	_, _ = Install(TargetFish)

	lines, err := UninstallAll()
	if err != nil {
		t.Fatalf("UninstallAll failed: %v", err)
	}
	if len(lines) == 0 {
		t.Fatalf("expected output lines from UninstallAll")
	}

	// Verify all are uninstalled
	all := CheckAllTargets()
	for _, info := range all {
		if info.Installed {
			t.Errorf("expected target %s to be uninstalled after UninstallAll", info.Target)
		}
	}
}

func TestDetectActiveTarget_WindowsGitBash(t *testing.T) {
	t.Setenv("STARSHIP_SHELL", "")
	t.Setenv("PSModulePath", `C:\Program Files\WindowsPowerShell\Modules`)
	t.Setenv("SHELL", `/usr/bin/bash`)
	if got := DetectActiveTarget(); got != TargetBash {
		t.Errorf("DetectActiveTarget with SHELL=/usr/bin/bash and PSModulePath got %v, want TargetBash", got)
	}

	t.Setenv("SHELL", "")
	t.Setenv("BASH", `/usr/bin/bash`)
	if got := DetectActiveTarget(); got != TargetBash {
		t.Errorf("DetectActiveTarget with BASH set and PSModulePath got %v, want TargetBash", got)
	}

	t.Setenv("BASH", "")
	t.Setenv("MSYSTEM", "MINGW64")
	if got := DetectActiveTarget(); got != TargetBash {
		t.Errorf("DetectActiveTarget with MSYSTEM set and PSModulePath got %v, want TargetBash", got)
	}
}
