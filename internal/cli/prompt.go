package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

const (
	zshPromptBlock = `
# --- git-user prompt integration ---
setopt PROMPT_SUBST 2>/dev/null

function _git_user_prompt() {
  local user
  user=$(git-user prompt 2>/dev/null)
  if [[ -n "$user" ]]; then
    local icon=" "
    if [[ "$TERM" == "linux" || "$TERM" == "dumb" ]]; then
      icon="git:"
    fi
    if [[ -n "$GIT_USER_PROMPT_ICON" ]]; then
      icon="$GIT_USER_PROMPT_ICON"
    fi
    echo "%F{blue}${icon}${user}%f"
  fi
}

autoload -Uz add-zsh-hook 2>/dev/null

_git_user_setup_prompt() {
  local p="$(_git_user_prompt)"
  if [[ -n "$p" ]]; then
    if [[ -z "$_GIT_USER_ORIG_RPROMPT" && -n "$RPROMPT" && "$RPROMPT" != *"$p"* ]]; then
      _GIT_USER_ORIG_RPROMPT="$RPROMPT"
    fi
    RPROMPT="${p}${_GIT_USER_ORIG_RPROMPT:+ $_GIT_USER_ORIG_RPROMPT}"
  elif [[ -n "$_GIT_USER_ORIG_RPROMPT" ]]; then
    RPROMPT="$_GIT_USER_ORIG_RPROMPT"
  fi
}

if functions add-zsh-hook >/dev/null 2>&1; then
  add-zsh-hook precmd _git_user_setup_prompt
else
  RPROMPT='$(_git_user_prompt)'
fi
`

	bashPromptBlock = `
# --- git-user prompt integration ---
__git_user_prompt() {
  local user=$(git-user prompt 2>/dev/null)
  if [ -n "$user" ]; then
    local icon=" "
    if [ "$TERM" = "linux" ] || [ "$TERM" = "dumb" ]; then
      icon="git:"
    fi
    if [ -n "$GIT_USER_PROMPT_ICON" ]; then
      icon="$GIT_USER_PROMPT_ICON"
    fi
    printf "\001\033[1;34m\002%s%s\001\033[0m\002 " "$icon" "$user"
  fi
}

__git_user_update_ps1() {
  if [ -z "$__GIT_USER_ORIG_PS1" ]; then
    __GIT_USER_ORIG_PS1="$PS1"
  fi
  local p=$(__git_user_prompt)
  PS1="${p}${__GIT_USER_ORIG_PS1}"
}

if [[ ! "$PROMPT_COMMAND" =~ __git_user_update_ps1 ]]; then
  PROMPT_COMMAND="__git_user_update_ps1${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
fi
`

	fishPromptBlock = `# --- git-user prompt integration ---
if status is-interactive
    if functions -q fish_right_prompt; and not functions -q __git_user_orig_right_prompt
        functions -c fish_right_prompt __git_user_orig_right_prompt
    end

    function fish_right_prompt -d "Display active git-user profile in right prompt"
        set -l git_user (git-user prompt 2>/dev/null)
        if test -n "$git_user"
            set -l icon " "
            if test "$TERM" = "linux" -o "$TERM" = "dumb"
                set icon "git:"
            end
            if set -q GIT_USER_PROMPT_ICON
                set icon "$GIT_USER_PROMPT_ICON"
            end
            set_color blue
            echo -n "$icon$git_user"
            set_color normal
        end
        if functions -q __git_user_orig_right_prompt
            echo -n " "
            __git_user_orig_right_prompt
        end
    end
end
`

	starshipPromptBlock = `
[custom.gituser]
command = "git-user prompt"
when = "git rev-parse --is-inside-work-tree 2>/dev/null"
format = "[$output]($style) "
style = "bold blue"
`

	powerShellPromptBlock = `
# --- git-user prompt integration ---
if (Get-Command git-user -ErrorAction SilentlyContinue) {
    if (-not (Test-Path Function:\__git_user_orig_prompt)) {
        if (Test-Path Function:\prompt) {
            Copy-Item Function:\prompt Function:\__git_user_orig_prompt
        }
    }
    function global:prompt {
        $u = $(git-user prompt 2>$null)
        if ($u) {
            $icon = " "
            if ($env:TERM -eq "linux" -or $env:TERM -eq "dumb") {
                $icon = "git:"
            }
            if ($env:GIT_USER_PROMPT_ICON) {
                $icon = $env:GIT_USER_PROMPT_ICON
            }
            Write-Host -NoNewline -ForegroundColor Blue "$icon$u "
        }
        if (Test-Path Function:\__git_user_orig_prompt) {
            & (Get-Item Function:\__git_user_orig_prompt)
        } else {
            "PS $($executionContext.SessionState.Path.CurrentLocation)$('>' * ($nestedPromptLevel + 1)) "
        }
    }
}
`

	nushellPromptBlock = `
# --- git-user prompt integration ---
$env.PROMPT_COMMAND_RIGHT = {||
    let user = (do -i { git-user prompt } | complete)
    if ($user.exit_code == 0) and ($user.stdout != "") {
        let u = ($user.stdout | str trim)
        let icon = (if ($env.TERM? == "linux" or $env.TERM? == "dumb") { "git:" } else { " " })
        let icon = (if ($env.GIT_USER_PROMPT_ICON? != null) { $env.GIT_USER_PROMPT_ICON } else { $icon })
        $"(ansi blue)($icon)($u)(ansi reset)"
    } else {
        ""
    }
}
`
)

func runPrompt(args []string) error {
	withIcon := false
	always := false
	plain := false

	for _, arg := range args {
		switch arg {
		case "install":
			return runPromptInstall()
		case "--icon", "-i":
			withIcon = true
		case "--always", "-a":
			always = true
		case "--plain", "-p":
			plain = true
		}
	}

	// Output only inside git repos unless --always was specified
	if !always && !git.IsInRepo() {
		return nil
	}

	// Load config store
	store, err := config.Load()
	if err != nil {
		return nil
	}

	var name string
	var badge string

	// 1. Session override
	if session := os.Getenv("GIT_USER_SESSION"); session != "" {
		name = session
		badge = "session"
	} else if authorName := os.Getenv("GIT_AUTHOR_NAME"); authorName != "" {
		name = authorName
		badge = "session"
	} else if git.IsInRepo() {
		// 2. Local repository override or directory binding override
		gitName := git.CurrentName()
		gitEmail := git.CurrentEmail()
		if gitName != "" || gitEmail != "" {
			for _, u := range store.Users {
				if u.Name == gitName || (gitEmail != "" && u.Email == gitEmail) {
					name = u.Name
					if u.IsTemporary {
						badge = "temp"
					} else if git.HasLocalOverride() && u.Name != store.Current {
						badge = "local"
					}
					break
				}
			}
			if name == "" && gitName != "" {
				name = gitName
				if git.HasLocalOverride() {
					badge = "local"
				}
			}
		}
	}

	// 3. Global active profile
	if name == "" && store.Current != "" {
		name = store.Current
		if u := store.CurrentUser(); u != nil && u.IsTemporary {
			badge = "temp"
		}
	}

	if name == "" {
		return nil
	}

	out := name
	if !plain && badge != "" {
		out = fmt.Sprintf("%s (%s)", name, badge)
	}

	if withIcon {
		icon := os.Getenv("GIT_USER_PROMPT_ICON")
		if icon == "" {
			term := os.Getenv("TERM")
			if term == "linux" || term == "dumb" || term == "cons25" {
				icon = "git:"
			} else {
				icon = " "
			}
		}
		out = icon + out
	}

	fmt.Print(out)
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

	if runtime.GOOS == "windows" {
		options = append(options, "PowerShell (recommended - Windows)")
		options = append(options, "Nushell")
	} else if strings.Contains(shellEnv, "zsh") {
		options = append(options, "Zsh (recommended - active shell)")
		options = append(options, "Bash")
		options = append(options, "Fish")
		options = append(options, "PowerShell")
		options = append(options, "Nushell")
	} else if strings.Contains(shellEnv, "fish") {
		options = append(options, "Fish (recommended - active shell)")
		options = append(options, "Zsh")
		options = append(options, "Bash")
		options = append(options, "PowerShell")
		options = append(options, "Nushell")
	} else if strings.Contains(shellEnv, "nu") {
		options = append(options, "Nushell (recommended - active shell)")
		options = append(options, "Zsh")
		options = append(options, "Bash")
		options = append(options, "Fish")
		options = append(options, "PowerShell")
	} else {
		options = append(options, "Bash (recommended - active shell)")
		options = append(options, "Zsh")
		options = append(options, "Fish")
		options = append(options, "PowerShell")
		options = append(options, "Nushell")
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
	} else if strings.Contains(choice, "PowerShell") {
		return installPowerShell()
	} else if strings.Contains(choice, "Nushell") {
		return installNushell()
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

	if content, err := os.ReadFile(path); err == nil {
		if strings.Contains(string(content), "[custom.gituser]") {
			ui.Warn("Starship prompt integration is already installed.")
			return nil
		}
	}

	if err := backupFile(path); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(starshipPromptBlock); err != nil {
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
		str := string(content)
		if strings.Contains(str, "_git_user_setup_prompt") {
			ui.Warn("Zsh prompt integration is already installed and up to date.")
			return nil
		}
		if strings.Contains(str, "# --- git-user prompt integration ---") || strings.Contains(str, "_git_user_prompt") {
			if err := backupFile(path); err != nil {
				return err
			}
			// Replace legacy integration block
			start := strings.Index(str, "# --- git-user prompt integration ---")
			if start != -1 {
				end := strings.Index(str[start:], "RPROMPT=")
				if end != -1 {
					lineEnd := strings.Index(str[start+end:], "\n")
					if lineEnd != -1 {
						str = str[:start] + strings.TrimLeft(zshPromptBlock, "\n") + str[start+end+lineEnd+1:]
						if err := os.WriteFile(path, []byte(str), 0644); err == nil {
							ui.Success("Successfully upgraded git-user integration in ~/.zshrc!")
							ui.Info("Run 'source ~/.zshrc' to apply the changes to your current terminal session.")
							return nil
						}
					}
				}
			}
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

	if _, err := f.WriteString(zshPromptBlock); err != nil {
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
		str := string(content)
		if strings.Contains(str, "__git_user_update_ps1") {
			ui.Warn("Bash prompt integration is already installed and up to date.")
			return nil
		}
		if strings.Contains(str, "# --- git-user prompt integration ---") || strings.Contains(str, "__git_user_prompt") {
			if err := backupFile(path); err != nil {
				return err
			}
			start := strings.Index(str, "# --- git-user prompt integration ---")
			if start != -1 {
				end := strings.Index(str[start:], "PROMPT_COMMAND=")
				if end != -1 {
					lineEnd := strings.Index(str[start+end:], "\n")
					if lineEnd != -1 {
						str = str[:start] + strings.TrimLeft(bashPromptBlock, "\n") + str[start+end+lineEnd+1:]
						if err := os.WriteFile(path, []byte(str), 0644); err == nil {
							ui.Success("Successfully upgraded git-user integration in ~/.bashrc!")
							ui.Info("Run 'source ~/.bashrc' to apply the changes to your current terminal session.")
							return nil
						}
					}
				}
			}
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

	if _, err := f.WriteString(bashPromptBlock); err != nil {
		return fmt.Errorf("writing configuration: %w", err)
	}

	ui.Success("Successfully appended git-user integration to ~/.bashrc!")
	ui.Info("Run 'source ~/.bashrc' to apply the changes to your current terminal session.")
	return nil
}

func installFish() error {
	home, _ := os.UserHomeDir()
	confDir := filepath.Join(home, ".config", "fish", "conf.d")
	functionsDir := filepath.Join(home, ".config", "fish", "functions")

	confPath := filepath.Join(confDir, "git_user_prompt.fish")
	funcPath := filepath.Join(functionsDir, "fish_right_prompt.fish")

	if err := os.MkdirAll(confDir, 0755); err != nil {
		return fmt.Errorf("creating fish conf.d directory: %w", err)
	}
	if err := os.MkdirAll(functionsDir, 0755); err != nil {
		return fmt.Errorf("creating fish functions directory: %w", err)
	}

	_ = backupFile(confPath)
	_ = backupFile(funcPath)

	if err := os.WriteFile(confPath, []byte(fishPromptBlock), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", confPath, err)
	}

	// Also write to functions/fish_right_prompt.fish for maximum fish version compatibility
	if err := os.WriteFile(funcPath, []byte(fishPromptBlock), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", funcPath, err)
	}

	ui.Success("Successfully installed fish prompt integration to:")
	ui.Info("  • " + confPath)
	ui.Info("  • " + funcPath)
	ui.Info("Open a new fish session or run 'source " + confPath + "' to apply.")
	return nil
}

func installPowerShell() error {
	home, _ := os.UserHomeDir()
	var profilePaths []string

	if runtime.GOOS == "windows" {
		profilePaths = []string{
			filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
			filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		}
	} else {
		profilePaths = []string{
			filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"),
		}
	}

	installedAny := false
	for _, path := range profilePaths {
		if content, err := os.ReadFile(path); err == nil {
			if strings.Contains(string(content), "__git_user_orig_prompt") {
				ui.Warn("PowerShell prompt integration is already installed in " + path)
				installedAny = true
				continue
			}
		}

		_ = backupFile(path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			continue
		}

		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			continue
		}

		_, _ = f.WriteString(powerShellPromptBlock)
		f.Close()
		installedAny = true
		ui.Success("Successfully appended git-user integration to " + path)
	}

	if !installedAny {
		primary := profilePaths[0]
		_ = os.MkdirAll(filepath.Dir(primary), 0755)
		if err := os.WriteFile(primary, []byte(powerShellPromptBlock), 0644); err != nil {
			return fmt.Errorf("writing powershell profile: %w", err)
		}
		ui.Success("Successfully created PowerShell profile with git-user integration: " + primary)
	}

	ui.Info("Restart your PowerShell session or run '. $PROFILE' to apply.")
	return nil
}

func installNushell() error {
	home, _ := os.UserHomeDir()
	var path string
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		path = filepath.Join(appData, "nushell", "env.nu")
	} else {
		path = filepath.Join(home, ".config", "nushell", "env.nu")
	}

	if content, err := os.ReadFile(path); err == nil {
		str := string(content)
		if strings.Contains(str, "git-user prompt") {
			ui.Warn("Nushell prompt integration is already installed in " + path)
			return nil
		}
	}

	_ = backupFile(path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating nushell config directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(nushellPromptBlock); err != nil {
		return fmt.Errorf("writing configuration: %w", err)
	}

	ui.Success("Successfully appended git-user integration to " + path)
	ui.Info("Restart Nushell to see your active profile in the right prompt.")
	return nil
}
