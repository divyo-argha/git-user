package promptops

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

// Target represents a supported shell or prompt framework.
type Target string

const (
	TargetFish       Target = "fish"
	TargetZsh        Target = "zsh"
	TargetBash       Target = "bash"
	TargetStarship   Target = "starship"
	TargetPowerShell Target = "powershell"
	TargetNushell    Target = "nushell"
)

const (
	ZshPromptBlock = `
# --- git-user prompt integration ---
setopt PROMPT_SUBST 2>/dev/null

# Clean up any stale or duplicated prompt tokens from previous sessions
setopt EXTENDED_GLOB 2>/dev/null
if [[ "$RPROMPT" == *'%F{blue}'* ]]; then
  RPROMPT="${RPROMPT//(#b)%F\{blue\}[^%]##%f([[:space:]]#)/}"
fi
unset _GIT_USER_ORIG_RPROMPT 2>/dev/null
_GIT_USER_PREV_PROMPT=""

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
  if [[ -n "$_GIT_USER_PREV_PROMPT" ]]; then
    RPROMPT="${RPROMPT#$_GIT_USER_PREV_PROMPT }"
    RPROMPT="${RPROMPT#$_GIT_USER_PREV_PROMPT}"
    RPROMPT="${RPROMPT% $_GIT_USER_PREV_PROMPT}"
    RPROMPT="${RPROMPT%$_GIT_USER_PREV_PROMPT}"
  fi
  _GIT_USER_PREV_PROMPT="$p"
  if [[ -n "$p" ]]; then
    if [[ -n "$RPROMPT" ]]; then
      RPROMPT="${p} ${RPROMPT}"
    else
      RPROMPT="${p}"
    fi
  fi
}

if functions add-zsh-hook >/dev/null 2>&1; then
  add-zsh-hook precmd _git_user_setup_prompt
else
  RPROMPT='$(_git_user_prompt)'
fi
`

	ZshPromptBlockV1 = `
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

	ZshPromptBlockLegacy = `
# --- git-user prompt integration ---
function _git_user_prompt() {
  local user=$(git-user prompt 2>/dev/null)
  if [ -n "$user" ]; then
    echo "%F{blue} ${user}%f"
  fi
}
setopt PROMPT_SUBST
RPROMPT='$( _git_user_prompt )'
`

	BashPromptBlock = `
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
  local p=$(__git_user_prompt)
  if [ -n "$__GIT_USER_PREV_P" ]; then
    PS1="${PS1#$__GIT_USER_PREV_P}"
  fi
  __GIT_USER_PREV_P="$p"
  if [ -n "$p" ]; then
    PS1="${p}${PS1}"
  fi
}

if [[ ! "$PROMPT_COMMAND" =~ __git_user_update_ps1 ]]; then
  PROMPT_COMMAND="__git_user_update_ps1${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
fi
`

	BashPromptBlockV1 = `
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

	BashPromptBlockLegacy = `
# --- git-user prompt integration ---
__git_user_prompt() {
  local user=$(git-user prompt 2>/dev/null)
  if [ -n "$user" ]; then
    echo -e "\033[1;34m $user\033[0m "
  fi
}
if [[ ! "$PROMPT_COMMAND" =~ __git_user_prompt ]]; then
  PROMPT_COMMAND="__git_user_prompt; $PROMPT_COMMAND"
fi
`

	FishPromptBlock = `# --- git-user prompt integration ---
if status is-interactive
    if functions -q __git_user_orig_right_prompt
        if functions __git_user_orig_right_prompt | string match -q "*git-user prompt*"
            functions -e __git_user_orig_right_prompt
        end
    end

    if functions -q fish_right_prompt; and not functions -q __git_user_orig_right_prompt
        if not functions fish_right_prompt | string match -q "*git-user prompt*"
            functions -c fish_right_prompt __git_user_orig_right_prompt
        end
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
            if functions -q __git_user_orig_right_prompt
                echo -n " "
            end
        end
        if functions -q __git_user_orig_right_prompt
            __git_user_orig_right_prompt
        end
    end
end
`

	FishPromptFileLegacy = `function fish_right_prompt -d "Display active git-user profile in right prompt"
    set -l git_user (git-user prompt 2>/dev/null)
    if test -n "$git_user"
        set_color blue
        echo -n " $git_user"
        set_color normal
    end
end
`

	StarshipPromptBlock = `
[custom.gituser]
command = "git-user prompt"
when = "git rev-parse --is-inside-work-tree 2>/dev/null"
format = "[$output]($style) "
style = "bold blue"
`

	PowerShellPromptBlock = `
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

	NushellPromptBlock = `
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

// TargetInfo provides descriptive and status metadata for a target shell.
type TargetInfo struct {
	Target      Target
	Name        string
	ConfigPath  string
	Installed   bool
	IsActive    bool
	Description string
}

// AllTargets returns all supported prompt integration targets in recommended order.
func AllTargets() []Target {
	return []Target{
		TargetFish,
		TargetZsh,
		TargetBash,
		TargetStarship,
		TargetPowerShell,
		TargetNushell,
	}
}

// TargetName returns a human-friendly name for the given target.
func TargetName(t Target) string {
	switch t {
	case TargetFish:
		return "Fish Shell"
	case TargetZsh:
		return "Zsh / Oh My Zsh"
	case TargetBash:
		return "Bash"
	case TargetStarship:
		return "Starship Prompt"
	case TargetPowerShell:
		return "PowerShell"
	case TargetNushell:
		return "Nushell"
	default:
		return string(t)
	}
}

// DetectActiveTarget detects the shell currently active in the user's environment.
func DetectActiveTarget() Target {
	home, _ := os.UserHomeDir()
	starshipPath := filepath.Join(home, ".config", "starship.toml")
	if _, err := os.Stat(starshipPath); err == nil && os.Getenv("STARSHIP_SHELL") != "" {
		return TargetStarship
	}

	shellEnv := strings.ToLower(os.Getenv("SHELL"))
	if strings.Contains(shellEnv, "fish") {
		return TargetFish
	}
	if strings.Contains(shellEnv, "zsh") {
		return TargetZsh
	}
	if strings.Contains(shellEnv, "nu") {
		return TargetNushell
	}
	if strings.Contains(shellEnv, "pwsh") || strings.Contains(shellEnv, "powershell") {
		return TargetPowerShell
	}
	if runtime.GOOS == "windows" {
		if os.Getenv("PSModulePath") != "" {
			return TargetPowerShell
		}
	}
	if strings.Contains(shellEnv, "bash") {
		return TargetBash
	}
	if runtime.GOOS == "darwin" {
		return TargetZsh
	}
	return TargetBash
}

// PrimaryConfigPath returns the canonical configuration file path for a target.
func PrimaryConfigPath(t Target) string {
	home, _ := os.UserHomeDir()
	switch t {
	case TargetFish:
		return filepath.Join(home, ".config", "fish", "conf.d", "git_user_prompt.fish")
	case TargetZsh:
		return filepath.Join(home, ".zshrc")
	case TargetBash:
		return filepath.Join(home, ".bashrc")
	case TargetStarship:
		return filepath.Join(home, ".config", "starship.toml")
	case TargetPowerShell:
		if runtime.GOOS == "windows" {
			return filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
		}
		return filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
	case TargetNushell:
		if runtime.GOOS == "windows" {
			appData := os.Getenv("APPDATA")
			if appData == "" {
				appData = filepath.Join(home, "AppData", "Roaming")
			}
			return filepath.Join(appData, "nushell", "env.nu")
		}
		return filepath.Join(home, ".config", "nushell", "env.nu")
	default:
		return ""
	}
}

// CheckTarget returns status metadata for a specific target.
func CheckTarget(t Target) TargetInfo {
	path := PrimaryConfigPath(t)
	active := DetectActiveTarget() == t
	installed := false

	home, _ := os.UserHomeDir()

	switch t {
	case TargetFish:
		// Check both conf.d and functions
		confPath := filepath.Join(home, ".config", "fish", "conf.d", "git_user_prompt.fish")
		funcPath := filepath.Join(home, ".config", "fish", "functions", "fish_right_prompt.fish")
		if data, err := os.ReadFile(confPath); err == nil && strings.Contains(string(data), "git-user prompt") {
			installed = true
		} else if data, err := os.ReadFile(funcPath); err == nil && strings.Contains(string(data), "git-user prompt") {
			installed = true
		}
	case TargetPowerShell:
		paths := []string{
			path,
			filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"),
			filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
			filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		}
		for _, p := range paths {
			if data, err := os.ReadFile(p); err == nil && strings.Contains(string(data), "git-user prompt") {
				installed = true
				path = p
				break
			}
		}
	default:
		if data, err := os.ReadFile(path); err == nil && strings.Contains(string(data), "git-user prompt") {
			installed = true
		}
	}

	return TargetInfo{
		Target:      t,
		Name:        TargetName(t),
		ConfigPath:  path,
		Installed:   installed,
		IsActive:    active,
		Description: targetDescription(t),
	}
}

// CheckAllTargets returns the status of all supported targets.
func CheckAllTargets() []TargetInfo {
	targets := AllTargets()
	results := make([]TargetInfo, len(targets))
	for i, t := range targets {
		results[i] = CheckTarget(t)
	}
	return results
}

func targetDescription(t Target) string {
	switch t {
	case TargetFish:
		return "Startup configuration in conf.d with right-prompt integration"
	case TargetZsh:
		return "Dynamic precmd hook with PROMPT_SUBST and RPROMPT support"
	case TargetBash:
		return "PROMPT_COMMAND PS1 wrapper with readline escape guards"
	case TargetStarship:
		return "Native custom module [custom.gituser] in starship.toml"
	case PowerShellPromptBlock:
		return "Cross-platform prompt wrapper function in PowerShell profile"
	case TargetNushell:
		return "Right prompt closure hook ($env.PROMPT_COMMAND_RIGHT) in env.nu"
	default:
		return ""
	}
}

// BackupFile creates a timestamped backup of path if it exists.
func BackupFile(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file for backup: %w", err)
	}
	backupPath := fmt.Sprintf("%s.bak-%s", path, time.Now().Format("20060102-150405"))
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return "", fmt.Errorf("writing backup file: %w", err)
	}
	return backupPath, nil
}

// Install installs the prompt snippet for the specified target.
func Install(t Target) (string, error) {
	switch t {
	case TargetStarship:
		return installStarship()
	case TargetZsh:
		return installZsh()
	case TargetBash:
		return installBash()
	case TargetFish:
		return installFish()
	case TargetPowerShell:
		return installPowerShell()
	case TargetNushell:
		return installNushell()
	default:
		return "", fmt.Errorf("unsupported prompt target: %s", t)
	}
}

func installStarship() (string, error) {
	path := PrimaryConfigPath(TargetStarship)
	if content, err := os.ReadFile(path); err == nil {
		if strings.Contains(string(content), "[custom.gituser]") {
			return fmt.Sprintf("Starship prompt integration is already configured in %s", path), nil
		}
	}
	_, _ = BackupFile(path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(StarshipPromptBlock); err != nil {
		return "", fmt.Errorf("writing configuration: %w", err)
	}
	return fmt.Sprintf("Appended Starship prompt configuration to %s", path), nil
}

func installZsh() (string, error) {
	path := PrimaryConfigPath(TargetZsh)
	if content, err := os.ReadFile(path); err == nil {
		str := string(content)
		if strings.Contains(str, "_GIT_USER_PREV_PROMPT") {
			return fmt.Sprintf("Zsh prompt integration is already up to date in %s", path), nil
		}
		for _, old := range []string{ZshPromptBlockV1, ZshPromptBlockLegacy} {
			if strings.Contains(str, old) {
				str = strings.Replace(str, old, ZshPromptBlock, 1)
				_, _ = BackupFile(path)
				if err := os.WriteFile(path, []byte(str), 0644); err != nil {
					return "", fmt.Errorf("updating configuration: %w", err)
				}
				return fmt.Sprintf("Upgraded Zsh prompt integration in %s", path), nil
			}
		}
		if strings.Contains(str, "_git_user_setup_prompt") || strings.Contains(str, "_git_user_prompt") {
			if idx := strings.Index(str, "# --- git-user prompt integration ---"); idx != -1 {
				endMarker := "RPROMPT='$(_git_user_prompt)'\nfi\n"
				if endIdx := strings.Index(str[idx:], endMarker); endIdx != -1 {
					oldBlock := str[idx : idx+endIdx+len(endMarker)]
					str = strings.Replace(str, oldBlock, strings.TrimPrefix(ZshPromptBlock, "\n"), 1)
					_, _ = BackupFile(path)
					if err := os.WriteFile(path, []byte(str), 0644); err != nil {
						return "", fmt.Errorf("updating configuration: %w", err)
					}
					return fmt.Sprintf("Upgraded Zsh prompt integration in %s", path), nil
				}
			}
		}
	}
	_, _ = BackupFile(path)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(ZshPromptBlock); err != nil {
		return "", fmt.Errorf("writing configuration: %w", err)
	}
	return fmt.Sprintf("Appended Zsh prompt integration to %s", path), nil
}

func installBash() (string, error) {
	path := PrimaryConfigPath(TargetBash)
	if content, err := os.ReadFile(path); err == nil {
		str := string(content)
		if strings.Contains(str, "__GIT_USER_PREV_P") {
			return fmt.Sprintf("Bash prompt integration is already up to date in %s", path), nil
		}
		for _, old := range []string{BashPromptBlockV1, BashPromptBlockLegacy} {
			if strings.Contains(str, old) {
				str = strings.Replace(str, old, BashPromptBlock, 1)
				_, _ = BackupFile(path)
				if err := os.WriteFile(path, []byte(str), 0644); err != nil {
					return "", fmt.Errorf("updating configuration: %w", err)
				}
				return fmt.Sprintf("Upgraded Bash prompt integration in %s", path), nil
			}
		}
	}
	_, _ = BackupFile(path)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(BashPromptBlock); err != nil {
		return "", fmt.Errorf("writing configuration: %w", err)
	}
	return fmt.Sprintf("Appended Bash prompt integration to %s", path), nil
}

func installFish() (string, error) {
	home, _ := os.UserHomeDir()
	confDir := filepath.Join(home, ".config", "fish", "conf.d")
	confPath := filepath.Join(confDir, "git_user_prompt.fish")

	if err := os.MkdirAll(confDir, 0755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}
	if err := os.WriteFile(confPath, []byte(FishPromptBlock), 0644); err != nil {
		return "", fmt.Errorf("writing fish configuration: %w", err)
	}

	// Also install to functions directory for backward compatibility
	funcDir := filepath.Join(home, ".config", "fish", "functions")
	funcPath := filepath.Join(funcDir, "fish_right_prompt.fish")
	if err := os.MkdirAll(funcDir, 0755); err == nil {
		_ = os.WriteFile(funcPath, []byte(FishPromptBlock), 0644)
	}

	return fmt.Sprintf("Installed Fish prompt integration to %s", confPath), nil
}

func installPowerShell() (string, error) {
	path := PrimaryConfigPath(TargetPowerShell)
	if content, err := os.ReadFile(path); err == nil {
		str := string(content)
		if strings.Contains(str, "git-user prompt") {
			return fmt.Sprintf("PowerShell prompt integration is already installed in %s", path), nil
		}
	}
	_, _ = BackupFile(path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("creating powershell profile directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(PowerShellPromptBlock); err != nil {
		return "", fmt.Errorf("writing configuration: %w", err)
	}
	return fmt.Sprintf("Appended PowerShell prompt integration to %s", path), nil
}

func installNushell() (string, error) {
	path := PrimaryConfigPath(TargetNushell)
	if content, err := os.ReadFile(path); err == nil {
		str := string(content)
		if strings.Contains(str, "git-user prompt") {
			return fmt.Sprintf("Nushell prompt integration is already installed in %s", path), nil
		}
	}
	_, _ = BackupFile(path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("creating nushell config directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(NushellPromptBlock); err != nil {
		return "", fmt.Errorf("writing configuration: %w", err)
	}
	return fmt.Sprintf("Appended Nushell prompt integration to %s", path), nil
}

// Uninstall removes the prompt integration block from the specified target.
func Uninstall(t Target) ([]string, error) {
	home, _ := os.UserHomeDir()
	var lines []string

	removeFileBlock := func(filePath, label string, blocks ...string) {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return
		}
		str := string(content)
		for _, b := range blocks {
			if b != "" && strings.Contains(str, b) {
				str = strings.Replace(str, b, "", 1)
				_ = os.WriteFile(filePath, []byte(str), 0644)
				lines = append(lines, fmt.Sprintf("Removed %s prompt block from %s", label, filePath))
				return
			}
		}
		if idx := strings.Index(str, "# --- git-user prompt integration ---"); idx != -1 {
			for _, endMarker := range []string{"RPROMPT='$(_git_user_prompt)'\nfi\n", "fi\n", "\n"} {
				if endIdx := strings.Index(str[idx:], endMarker); endIdx != -1 {
					fullEnd := idx + endIdx + len(endMarker)
					str = str[:idx] + str[fullEnd:]
					_ = os.WriteFile(filePath, []byte(str), 0644)
					lines = append(lines, fmt.Sprintf("Removed %s prompt block from %s", label, filePath))
					return
				}
			}
		}
		if strings.Contains(str, "git-user prompt") {
			lines = append(lines, fmt.Sprintf("Notice: %s has a customized git-user prompt block. Please edit manually.", filePath))
		}
	}

	switch t {
	case TargetFish:
		confPath := filepath.Join(home, ".config", "fish", "conf.d", "git_user_prompt.fish")
		if _, err := os.Stat(confPath); err == nil {
			_ = os.Remove(confPath)
			lines = append(lines, fmt.Sprintf("Removed Fish prompt file: %s", confPath))
		}
		funcPath := filepath.Join(home, ".config", "fish", "functions", "fish_right_prompt.fish")
		if content, err := os.ReadFile(funcPath); err == nil {
			trimmed := strings.TrimSpace(string(content))
			if strings.Contains(trimmed, "git-user prompt") || trimmed == strings.TrimSpace(FishPromptBlock) || trimmed == strings.TrimSpace(FishPromptFileLegacy) {
				_ = os.Remove(funcPath)
				lines = append(lines, fmt.Sprintf("Removed Fish right prompt function: %s", funcPath))
			}
		}
	case TargetZsh:
		removeFileBlock(filepath.Join(home, ".zshrc"), "Zsh", ZshPromptBlock, ZshPromptBlockV1, ZshPromptBlockLegacy)
	case TargetBash:
		removeFileBlock(filepath.Join(home, ".bashrc"), "Bash", BashPromptBlock, BashPromptBlockV1, BashPromptBlockLegacy)
	case TargetStarship:
		removeFileBlock(filepath.Join(home, ".config", "starship.toml"), "Starship", StarshipPromptBlock)
	case TargetPowerShell:
		paths := []string{
			filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"),
			filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
			filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		}
		for _, p := range paths {
			removeFileBlock(p, "PowerShell", PowerShellPromptBlock)
		}
	case TargetNushell:
		paths := []string{
			filepath.Join(home, ".config", "nushell", "env.nu"),
		}
		if runtime.GOOS == "windows" {
			appData := os.Getenv("APPDATA")
			if appData != "" {
				paths = append(paths, filepath.Join(appData, "nushell", "env.nu"))
			}
		}
		for _, p := range paths {
			removeFileBlock(p, "Nushell", NushellPromptBlock)
		}
	}

	if len(lines) == 0 {
		lines = append(lines, fmt.Sprintf("No active git-user prompt configuration found for %s", TargetName(t)))
	}
	return lines, nil
}

// UninstallAll removes prompt integration from all supported shells.
func UninstallAll() ([]string, error) {
	var allLines []string
	for _, t := range AllTargets() {
		lines, err := Uninstall(t)
		if err != nil {
			return allLines, err
		}
		allLines = append(allLines, lines...)
	}
	return allLines, nil
}

// ResolvePrompt computes the prompt text for the current git context.
func ResolvePrompt(store *config.Store, withIcon, plain, always bool) string {
	var name, badge string

	if store != nil && store.Prompt != nil {
		if !plain && store.Prompt.Plain {
			plain = true
		}
		if !always && store.Prompt.Always {
			always = true
		}
	}

	// If not inside a repo and not always, check if session is active
	inRepo := git.IsInRepo()
	session := os.Getenv("GIT_USER_SESSION")
	authorName := os.Getenv("GIT_AUTHOR_NAME")

	if !inRepo && !always && session == "" && authorName == "" {
		return ""
	}

	// 1. Session environment overrides
	if session != "" {
		name = session
		badge = "session"
	} else if authorName != "" {
		name = authorName
		badge = "session"
	} else if inRepo {
		// 2. Local repository override or directory binding
		gitName := git.CurrentName()
		gitEmail := git.CurrentEmail()
		if gitName != "" || gitEmail != "" {
			if store != nil {
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
	if name == "" && store != nil && store.Current != "" {
		name = store.Current
		if u := store.CurrentUser(); u != nil && u.IsTemporary {
			badge = "temp"
		}
	}

	if name == "" {
		return ""
	}

	out := name
	if !plain && badge != "" {
		out = fmt.Sprintf("%s (%s)", name, badge)
	}

	if withIcon {
		icon := os.Getenv("GIT_USER_PROMPT_ICON")
		if icon == "" && store != nil && store.Prompt != nil && store.Prompt.Icon != "" {
			icon = store.Prompt.Icon
		}
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

	return out
}
