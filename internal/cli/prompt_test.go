package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/ui"
)

func TestPromptCommand(t *testing.T) {
	setupTestEnv(t)

	// Run standard prompt command (should exit silently or output nothing)
	// since we are not in a git repo in standard test context, it might exit early.
	err := runPrompt([]string{})
	if err != nil {
		t.Fatalf("runPrompt failed: %v", err)
	}
}

func TestPromptInstall_Zsh(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Pre-create a dummy .zshrc
	zshrcPath := filepath.Join(tmpDir, ".zshrc")
	err := os.WriteFile(zshrcPath, []byte("export FOO=bar\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write dummy .zshrc: %v", err)
	}

	// Mock UI select:
	// Let's mock Select to choose the Zsh option.
	ui.SelectFn = func(label string, options []string) (int, error) {
		for idx, opt := range options {
			if strings.Contains(opt, "Zsh") {
				return idx, nil
			}
		}
		return 0, nil
	}

	err = runPrompt([]string{"install"})
	if err != nil {
		t.Fatalf("runPrompt install failed: %v", err)
	}

	// Verify backup file exists
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}
	hasBackup := false
	for _, f := range files {
		if strings.HasPrefix(f.Name(), ".zshrc.bak-") {
			hasBackup = true
			break
		}
	}
	if !hasBackup {
		t.Error("expected .zshrc backup file to be created")
	}

	// Verify content appended to .zshrc
	content, err := os.ReadFile(zshrcPath)
	if err != nil {
		t.Fatalf("failed to read .zshrc: %v", err)
	}
	if !strings.Contains(string(content), "function _git_user_prompt()") {
		t.Error("expected .zshrc to contain prompt integration function")
	}
}

func TestPromptInstall_Starship(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// Pre-create starship.toml
	starshipDir := filepath.Join(tmpDir, ".config")
	_ = os.MkdirAll(starshipDir, 0755)
	tomlPath := filepath.Join(starshipDir, "starship.toml")
	err := os.WriteFile(tomlPath, []byte("[directory]\ntruncate_to_repo = true\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write dummy starship.toml: %v", err)
	}

	ui.SelectFn = func(label string, options []string) (int, error) {
		for idx, opt := range options {
			if strings.Contains(opt, "Starship") {
				return idx, nil
			}
		}
		return 0, nil
	}

	err = runPrompt([]string{"install"})
	if err != nil {
		t.Fatalf("runPrompt install failed: %v", err)
	}

	// Verify backup file exists
	files, err := os.ReadDir(starshipDir)
	if err != nil {
		t.Fatalf("failed to read config dir: %v", err)
	}
	hasBackup := false
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "starship.toml.bak-") {
			hasBackup = true
			break
		}
	}
	if !hasBackup {
		t.Error("expected starship.toml backup file to be created")
	}

	// Verify content appended
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("failed to read starship.toml: %v", err)
	}
	if !strings.Contains(string(content), "[custom.gituser]") {
		t.Error("expected starship.toml to contain [custom.gituser] module")
	}
}

func TestPromptInstall_Fish(t *testing.T) {
	tmpDir := setupTestEnv(t)

	ui.SelectFn = func(label string, options []string) (int, error) {
		for idx, opt := range options {
			if strings.Contains(opt, "Fish") {
				return idx, nil
			}
		}
		return 0, nil
	}

	err := runPrompt([]string{"install"})
	if err != nil {
		t.Fatalf("runPrompt install failed: %v", err)
	}

	confPath := filepath.Join(tmpDir, ".config", "fish", "conf.d", "git_user_prompt.fish")
	funcPath := filepath.Join(tmpDir, ".config", "fish", "functions", "fish_right_prompt.fish")

	if _, err := os.Stat(confPath); err != nil {
		t.Errorf("expected fish conf.d file to exist: %v", err)
	}
	if _, err := os.Stat(funcPath); err != nil {
		t.Errorf("expected fish functions file to exist: %v", err)
	}

	content, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", confPath, err)
	}
	if !strings.Contains(string(content), "fish_right_prompt") {
		t.Errorf("expected fish_right_prompt in %s", confPath)
	}
}

func TestPromptInstall_Bash(t *testing.T) {
	tmpDir := setupTestEnv(t)

	bashrcPath := filepath.Join(tmpDir, ".bashrc")
	_ = os.WriteFile(bashrcPath, []byte("# bashrc\n"), 0644)

	ui.SelectFn = func(label string, options []string) (int, error) {
		for idx, opt := range options {
			if strings.Contains(opt, "Bash") {
				return idx, nil
			}
		}
		return 0, nil
	}

	err := runPrompt([]string{"install"})
	if err != nil {
		t.Fatalf("runPrompt install failed: %v", err)
	}

	content, err := os.ReadFile(bashrcPath)
	if err != nil {
		t.Fatalf("failed to read .bashrc: %v", err)
	}
	if !strings.Contains(string(content), "__git_user_update_ps1") {
		t.Errorf("expected .bashrc to contain __git_user_update_ps1")
	}
}

func TestPromptInstall_PowerShell(t *testing.T) {
	tmpDir := setupTestEnv(t)

	ui.SelectFn = func(label string, options []string) (int, error) {
		for idx, opt := range options {
			if strings.Contains(opt, "PowerShell") {
				return idx, nil
			}
		}
		return 0, nil
	}

	err := runPrompt([]string{"install"})
	if err != nil {
		t.Fatalf("runPrompt install failed: %v", err)
	}

	psPath := filepath.Join(tmpDir, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
	if _, err := os.Stat(psPath); err != nil {
		t.Fatalf("expected powershell profile to be created at %s", psPath)
	}

	content, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("failed to read powershell profile: %v", err)
	}
	if !strings.Contains(string(content), "git-user prompt") {
		t.Errorf("expected powershell profile to contain git-user prompt")
	}
}

