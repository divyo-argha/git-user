// Package shellinit holds the shell-integration wrapper templates and rc-file
// installer shared between internal/cli (`git-user init [install]`) and
// internal/tui (installing the same snippet on request from a menu action).
// It has no dependency on either, so both can import it without a cycle.
package shellinit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Shell identifies which shell dialect to generate/install an integration
// snippet for.
type Shell string

const (
	Posix      Shell = "posix" // bash, zsh, sh, ksh, dash
	Fish       Shell = "fish"
	PowerShell Shell = "powershell"
)

// Detect picks a shell dialect from an explicit override if given, else the
// environment ($SHELL / $PSModulePath), else the platform default.
func Detect(explicit string) Shell {
	if explicit != "" {
		switch strings.ToLower(strings.TrimSpace(explicit)) {
		case "fish":
			return Fish
		case "pwsh", "powershell", "ps":
			return PowerShell
		case "bash", "zsh", "sh", "posix", "dash", "ksh":
			return Posix
		}
	}

	shellEnv := strings.ToLower(os.Getenv("SHELL"))
	if strings.Contains(shellEnv, "fish") {
		return Fish
	}
	if strings.Contains(shellEnv, "pwsh") || strings.Contains(shellEnv, "powershell") {
		return PowerShell
	}

	if runtime.GOOS == "windows" {
		if os.Getenv("PSModulePath") != "" {
			return PowerShell
		}
	}

	return Posix
}

// Script returns the shell-function wrapper source for `git-user init
// <shell>` to print (and the caller to `eval`/`source`).
func Script(sh Shell) string {
	switch sh {
	case Fish:
		return fishInitScript
	case PowerShell:
		return powerShellInitScript
	default:
		return posixInitScript
	}
}

// Status describes what Install did to one rc file.
type Status string

const (
	StatusInstalled Status = "installed" // snippet freshly appended
	StatusUpgraded  Status = "upgraded"  // legacy unshielded eval replaced with the safe form
	StatusAlready   Status = "already"   // snippet (or its legacy form) was already present
)

// Result reports what Install did to a single rc file.
type Result struct {
	File   string
	Status Status
}

// Install appends the shell-integration snippet to the target rc file(s) for
// sh, upgrading a legacy unshielded `eval "$(git-user init)"` in place and
// leaving an already-installed file untouched. explicitShell, when it is
// exactly "bash" or "zsh", narrows a Posix install to just that one rc file;
// otherwise both ~/.zshrc and ~/.bashrc are targeted (whichever already
// exist, or ~/.zshrc if neither does).
func Install(sh Shell, explicitShell string) ([]Result, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not locate home directory: %w", err)
	}

	var targetFiles []string
	var initSnippet string

	switch sh {
	case Fish:
		targetFiles = []string{filepath.Join(home, ".config", "fish", "config.fish")}
		initSnippet = "\n# git-user shell integration\ncommand -q git-user; and git-user init fish 2>/dev/null | source\n"
	case PowerShell:
		if runtime.GOOS == "windows" {
			targetFiles = []string{filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")}
		} else {
			targetFiles = []string{filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")}
		}
		initSnippet = "\n# git-user shell integration\nif (Get-Command git-user -ErrorAction SilentlyContinue) { Invoke-Expression (& git-user init powershell 2>$null) }\n"
	default: // Posix
		initSnippet = "\n# git-user shell integration\ncommand -v git-user >/dev/null 2>&1 && eval \"$(git-user init 2>/dev/null)\"\n"
		if explicitShell == "bash" {
			targetFiles = []string{filepath.Join(home, ".bashrc")}
		} else if explicitShell == "zsh" {
			targetFiles = []string{filepath.Join(home, ".zshrc")}
		} else {
			zshrc := filepath.Join(home, ".zshrc")
			bashrc := filepath.Join(home, ".bashrc")
			if _, err := os.Stat(zshrc); err == nil {
				targetFiles = append(targetFiles, zshrc)
			}
			if _, err := os.Stat(bashrc); err == nil {
				targetFiles = append(targetFiles, bashrc)
			}
			if len(targetFiles) == 0 {
				targetFiles = []string{zshrc}
			}
		}
	}

	var results []Result
	for _, targetFile := range targetFiles {
		if content, err := os.ReadFile(targetFile); err == nil {
			str := string(content)
			if strings.Contains(str, "eval \"$(git-user init)\"") {
				updated := strings.Replace(str, "eval \"$(git-user init)\"", "command -v git-user >/dev/null 2>&1 && eval \"$(git-user init 2>/dev/null)\"", 1)
				if err := os.WriteFile(targetFile, []byte(updated), 0644); err == nil {
					results = append(results, Result{File: targetFile, Status: StatusUpgraded})
					continue
				}
			}
			if strings.Contains(str, "git-user init") {
				results = append(results, Result{File: targetFile, Status: StatusAlready})
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
			return results, fmt.Errorf("could not create directory for %s: %w", targetFile, err)
		}
		f, err := os.OpenFile(targetFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return results, fmt.Errorf("could not write to %s: %w", targetFile, err)
		}
		if _, err := f.WriteString(initSnippet); err != nil {
			f.Close()
			return results, fmt.Errorf("could not append to %s: %w", targetFile, err)
		}
		f.Close()
		results = append(results, Result{File: targetFile, Status: StatusInstalled})
	}

	return results, nil
}

const posixInitScript = `# git-user shell integration
# Generated by: git-user init
# Add to ~/.zshrc or ~/.bashrc:  command -v git-user >/dev/null 2>&1 && eval "$(git-user init 2>/dev/null)"

git-user() {
    case "$1" in
        switch|sw|--switch|-s)
            has_session=0
            for arg in "$@"; do
                if [ "$arg" = "--session" ] || [ "$arg" = "-s" ]; then
                    has_session=1
                    break
                fi
            done
            if [ "$has_session" -eq 1 ]; then
                eval "$(command git-user env "$@")"
                return $?
            fi
            command git-user "$@"
            ;;
        logout|signout|lo|--logout|--signout)
            if [ "$2" = "--session" ] || [ "$2" = "-s" ]; then
                eval "$(command git-user env --unset)"
                return $?
            fi
            command git-user "$@"
            ;;
        env)
            if [ "$2" = "--eval" ] || [ "$2" = "-e" ]; then
                shift 2
                eval "$(command git-user env "$@")"
                return $?
            fi
            command git-user "$@"
            ;;
        *)
            command git-user "$@"
            ;;
    esac
}

# Also seamlessly support 'git user switch -s ...'
git() {
    if [ "$1" = "user" ]; then
        shift
        git-user "$@"
        return $?
    fi
    command git "$@"
}
`

const fishInitScript = `# git-user shell integration for fish
# Generated by: git-user init fish
# Add to ~/.config/fish/config.fish:  command -q git-user; and git-user init fish 2>/dev/null | source

function git-user --wraps=git-user --description 'Git identity manager'
    switch $argv[1]
        case switch sw --switch -s
            if contains -- --session $argv; or contains -- -s $argv
                command git-user env --fish $argv | source
                return $status
            end
            command git-user $argv
        case logout signout lo --logout --signout
            if contains -- --session $argv; or contains -- -s $argv
                command git-user env --fish --unset | source
                return $status
            end
            command git-user $argv
        case env
            if contains -- --eval $argv; or contains -- -e $argv
                command git-user env --fish $argv[2..-1] | source
                return $status
            end
            command git-user $argv
        case '*'
            command git-user $argv
    end
end
`

const powerShellInitScript = `# git-user shell integration for PowerShell
# Generated by: git-user init powershell
# Add to $PROFILE:  if (Get-Command git-user -ErrorAction SilentlyContinue) { Invoke-Expression (& git-user init powershell 2>$null) }

function git-user {
    param([Parameter(ValueFromRemainingArguments = $true)]$args)
    if ($args.Count -gt 0) {
        $sub = $args[0]
        if ($sub -eq "switch" -or $sub -eq "sw" -or $sub -eq "--switch" -or $sub -eq "-s") {
            if ($args -contains "--session" -or $args -contains "-s") {
                $script = & (Get-Command git-user -CommandType Application) env --powershell @args
                Invoke-Expression ($script -join "` + "`" + `n")
                return
            }
        } elseif ($sub -eq "logout" -or $sub -eq "signout" -or $sub -eq "lo") {
            if ($args -contains "--session" -or $args -contains "-s") {
                $script = & (Get-Command git-user -CommandType Application) env --powershell --unset
                Invoke-Expression ($script -join "` + "`" + `n")
                return
            }
        }
    }
    & (Get-Command git-user -CommandType Application) @args
}
`
