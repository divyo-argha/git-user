package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runPrompt(args []string) error {
	if len(args) > 0 && args[0] == "install" {
		return runPromptInstall()
	}

	// Check if we are inside a git repository
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		// Not in a git repo or git is not installed; exit silently
		os.Exit(0)
	}

	// Load git-user config
	store, err := config.Load()
	if err != nil {
		// Error loading config, exit silently to avoid breaking the prompt
		os.Exit(0)
	}

	// Output the active identity name if there is one
	if store.Current != "" {
		u := store.CurrentUser()
		if u != nil && u.IsTemporary {
			fmt.Printf("%s (temp)", store.Current)
		} else {
			fmt.Print(store.Current)
		}
	}

	return nil
}

func runPromptInstall() error {
	ui.Banner("PROMPT INTEGRATION INSTALLER")
	fmt.Println()

	var options []string
	home, _ := os.UserHomeDir()
	starshipPath := filepath.Join(home, ".config", "starship.toml")

	hasStarship := false
	if _, err := os.Stat(starshipPath); err == nil {
		hasStarship = true
	}

	shellEnv := os.Getenv("SHELL")

	if hasStarship {
		options = append(options, "Starship Prompt (recommended - detected)")
	}

	if strings.Contains(shellEnv, "zsh") {
		options = append(options, "Zsh (recommended - active shell)")
		options = append(options, "Bash")
		options = append(options, "Fish")
	} else if strings.Contains(shellEnv, "fish") {
		options = append(options, "Fish (recommended - active shell)")
		options = append(options, "Zsh")
		options = append(options, "Bash")
	} else {
		options = append(options, "Bash (recommended - active shell)")
		options = append(options, "Zsh")
		options = append(options, "Fish")
	}

	if !hasStarship {
		options = append(options, "Starship Prompt")
	}

	options = append(options, "Cancel")

	idx, err := ui.Select("Choose where to install the git-user prompt integration:", options)
	if err != nil {
		return err
	}

	choice := options[idx]
	if choice == "Cancel" {
		ui.Info("Cancelled")
		return nil
	}

	if strings.Contains(choice, "Starship") {
		return installStarship()
	} else if strings.Contains(choice, "Zsh") {
		return installZsh()
	} else if strings.Contains(choice, "Bash") {
		return installBash()
	} else if strings.Contains(choice, "Fish") {
		return installFish()
	}

	return nil
}

func backupFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file for backup: %w", err)
	}
	backupPath := fmt.Sprintf("%s.bak-%s", path, time.Now().Format("20060102-150405"))
	err = os.WriteFile(backupPath, content, 0644)
	if err != nil {
		return fmt.Errorf("writing backup file: %w", err)
	}
	ui.Info(fmt.Sprintf("Backed up original config to %s", filepath.Base(backupPath)))
	return nil
}

func installStarship() error {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "starship.toml")

	// Check if already configured
	if content, err := os.ReadFile(path); err == nil {
		if strings.Contains(string(content), "[custom.gituser]") {
			ui.Warn("Starship prompt integration is already installed.")
			return nil
		}
	}

	if err := backupFile(path); err != nil {
		return err
	}

	// Create directory if not exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	block := `
[custom.gituser]
command = "git-user prompt"
when = "git rev-parse --is-inside-work-tree 2>/dev/null"
format = "[$output]($style) "
style = "bold blue"
`
	if _, err := f.WriteString(block); err != nil {
		return fmt.Errorf("writing configuration: %w", err)
	}

	ui.Success("Successfully appended git-user integration to starship.toml!")
	ui.Info("Starship automatically reloads configurations. Try navigating to a git repository now.")
	return nil
}

func installZsh() error {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".zshrc")

	if content, err := os.ReadFile(path); err == nil {
		if strings.Contains(string(content), "git-user prompt integration") {
			ui.Warn("Zsh prompt integration is already installed.")
			return nil
		}
	}

	if err := backupFile(path); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	block := `
# --- git-user prompt integration ---
function _git_user_prompt() {
  local user=$(git-user prompt 2>/dev/null)
  if [[ -n "$user" ]]; then
    echo "%F{blue} ${user}%f"
  fi
}
RPROMPT='$(_git_user_prompt)'
`
	if _, err := f.WriteString(block); err != nil {
		return fmt.Errorf("writing configuration: %w", err)
	}

	ui.Success("Successfully appended git-user integration to ~/.zshrc!")
	ui.Info("Run 'source ~/.zshrc' to apply the changes to your current terminal session.")
	return nil
}

func installBash() error {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".bashrc")

	if content, err := os.ReadFile(path); err == nil {
		if strings.Contains(string(content), "git-user prompt integration") {
			ui.Warn("Bash prompt integration is already installed.")
			return nil
		}
	}

	if err := backupFile(path); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	block := `
# --- git-user prompt integration ---
__git_user_prompt() {
  local user=$(git-user prompt 2>/dev/null)
  if [ -n "$user" ]; then
    echo -e "\033[1;34m ${user}\033[0m "
  fi
}
# Prepend to PS1 dynamically
PROMPT_COMMAND='PS1="$(__git_user_prompt)\u@\h:\w\$ "'
`
	if _, err := f.WriteString(block); err != nil {
		return fmt.Errorf("writing configuration: %w", err)
	}

	ui.Success("Successfully appended git-user integration to ~/.bashrc!")
	ui.Info("Run 'source ~/.bashrc' to apply the changes to your current terminal session.")
	return nil
}

func installFish() error {
	home, _ := os.UserHomeDir()
	functionsDir := filepath.Join(home, ".config", "fish", "functions")
	path := filepath.Join(functionsDir, "fish_right_prompt.fish")

	if content, err := os.ReadFile(path); err == nil {
		if strings.Contains(string(content), "git-user prompt") {
			ui.Warn("Fish prompt integration is already installed.")
			return nil
		}
	}

	if err := backupFile(path); err != nil {
		return err
	}

	if err := os.MkdirAll(functionsDir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	block := `function fish_right_prompt
  set -l git_user (git-user prompt 2>/dev/null)
  if test -n "$git_user"
    set_color blue
    echo -n " $git_user"
    set_color normal
  end
end
`
	if err := os.WriteFile(path, []byte(block), 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	ui.Success("Successfully created ~/.config/fish/functions/fish_right_prompt.fish!")
	ui.Info("Open a new fish session or run 'fish' to apply the changes.")
	return nil
}
