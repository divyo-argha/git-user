// Package shellinit holds the shell-integration wrapper templates and rc-file
// installer shared between internal/cli (`git-user init [install]`) and
// internal/tui (installing the same snippet on request from a menu action).
// It has no dependency on either, so both can import it without a cycle.
package shellinit

import (
	"fmt"
	"os"
	"os/exec"
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
	Cmd        Shell = "cmd"
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
		case "cmd", "batch":
			return Cmd
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
	if strings.Contains(shellEnv, "bash") || strings.Contains(shellEnv, "zsh") || strings.Contains(shellEnv, "sh") ||
		os.Getenv("BASH") != "" || os.Getenv("MSYSTEM") != "" {
		return Posix
	}

	if runtime.GOOS == "windows" || os.Getenv("PSModulePath") != "" || os.Getenv("PROMPT") != "" {
		if os.Getenv("PROMPT") != "" && os.Getenv("PSExecutionPolicyPreference") == "" {
			return Cmd
		}
		if os.Getenv("PSModulePath") != "" {
			return PowerShell
		}
		return Cmd
	}

	return Posix
}

// ResolveShellPath determines the shell executable path to launch for isolated subshells.
// On Windows, Git Bash and MSYS2 provide virtual Unix paths (e.g. /usr/bin/bash) which
// cannot be directly launched by Windows CreateProcess without resolving to a Win32 executable.
func ResolveShellPath() string {
	shellPath := os.Getenv("SHELL")
	if runtime.GOOS == "windows" {
		if shellPath != "" {
			if (strings.HasPrefix(shellPath, "/") || strings.HasPrefix(shellPath, "\\")) && !strings.Contains(shellPath, ":") {
				base := filepath.Base(shellPath)
				if lp, err := exec.LookPath(base); err == nil {
					return lp
				}
				if lp, err := exec.LookPath(base + ".exe"); err == nil {
					return lp
				}
			}
			if lp, err := exec.LookPath(shellPath); err == nil {
				return lp
			}
		}
		if os.Getenv("BASH") != "" || os.Getenv("MSYSTEM") != "" {
			if lp, err := exec.LookPath("bash.exe"); err == nil {
				return lp
			}
			if lp, err := exec.LookPath("bash"); err == nil {
				return lp
			}
		}
		// If running in standard CMD (PROMPT set, and not inside PowerShell or Bash):
		if os.Getenv("PROMPT") != "" && os.Getenv("PSExecutionPolicyPreference") == "" {
			if comspec := os.Getenv("ComSpec"); comspec != "" {
				if _, err := os.Stat(comspec); err == nil {
					return comspec
				}
			}
			if lp, err := exec.LookPath("cmd.exe"); err == nil {
				return lp
			}
		}
		if lp, err := exec.LookPath("powershell.exe"); err == nil {
			return lp
		}
		if comspec := os.Getenv("ComSpec"); comspec != "" {
			return comspec
		}
		return "cmd.exe"
	}

	if shellPath == "" {
		return "/bin/sh"
	}
	return shellPath
}

// Script returns the shell-function wrapper source for `git-user init
// <shell>` to print (and the caller to `eval`/`source`).
func Script(sh Shell) string {
	switch sh {
	case Fish:
		return fishInitScript
	case PowerShell:
		return powerShellInitScript
	case Cmd:
		return cmdInitScript
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
		initSnippet = "\n# git-user shell integration\nif (Get-Command git-user -ErrorAction SilentlyContinue) { Invoke-Expression (& git-user init powershell 2>$null) }\n"
		if runtime.GOOS == "windows" {
			targetFiles = []string{
				filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
				filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
			}
		} else {
			targetFiles = []string{filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")}
		}
	case Cmd:
		guPath := filepath.Join(home, "gu.cmd")
		if content, err := os.ReadFile(guPath); err == nil {
			if strings.Contains(string(content), "git-user.exe env") {
				return []Result{{File: guPath, Status: StatusAlready}}, nil
			}
		}
		if err := os.WriteFile(guPath, []byte(cmdInitScript), 0755); err != nil {
			return nil, fmt.Errorf("could not write %s: %w", guPath, err)
		}
		return []Result{{File: guPath, Status: StatusInstalled}}, nil
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

# Also seamlessly support 'gu switch -s ...'
gu() {
    git-user "$@"
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

function gu --wraps=git-user --description 'Git identity manager alias'
    git-user $argv
end

function git --wraps=git --description 'Git version control system'
    if test "$argv[1]" = "user"
        git-user $argv[2..-1]
        return $status
    end
    command git $argv
end
`

const powerShellInitScript = `# git-user shell integration for PowerShell
# Generated by: git-user init powershell
# Add to $PROFILE:  if (Get-Command git-user -ErrorAction SilentlyContinue) { Invoke-Expression (& git-user init powershell 2>$null) }

function git-user {
    param([Parameter(ValueFromRemainingArguments = $true)]$args)
    $exe = (Get-Command git-user -ErrorAction SilentlyContinue | Where-Object { $_.CommandType -ne 'Function' } | Select-Object -First 1)
    if (-not $exe) {
        Write-Error "git-user: command not found"
        return
    }
    if ($args.Count -gt 0) {
        $sub = $args[0]
        if ($sub -eq "switch" -or $sub -eq "sw" -or $sub -eq "--switch" -or $sub -eq "-s") {
            if ($args -contains "--session" -or $args -contains "-s") {
                $script = & $exe env --powershell @args
                Invoke-Expression ($script -join "` + "`" + `n")
                return
            }
        } elseif ($sub -eq "logout" -or $sub -eq "signout" -or $sub -eq "lo") {
            if ($args -contains "--session" -or $args -contains "-s") {
                $script = & $exe env --powershell --unset
                Invoke-Expression ($script -join "` + "`" + `n")
                return
            }
        }
    }
    & $exe @args
}

function gu {
    param([Parameter(ValueFromRemainingArguments = $true)]$args)
    git-user @args
}

function git {
    param([Parameter(ValueFromRemainingArguments = $true)]$args)
    if ($args.Count -gt 0 -and $args[0] -eq "user") {
        git-user @($args | Select-Object -Skip 1)
        return
    }
    $gitExe = (Get-Command git -ErrorAction SilentlyContinue | Where-Object { $_.CommandType -ne 'Function' } | Select-Object -First 1)
    if (-not $gitExe) {
        Write-Error "git: command not found"
        return
    }
    & $gitExe @args
}
`

const cmdInitScript = `@echo off
rem git-user shell integration for Windows Command Prompt (CMD)
rem Generated by: git-user init cmd
rem Save as 'gu.cmd' in a directory on your PATH (e.g. %APPDATA%\npm\gu.cmd)

if "%1"=="" (
    git-user.exe
    goto :eof
)

if "%1"=="switch" if "%2"=="-s" goto :session
if "%1"=="switch" if "%2"=="--session" goto :session
if "%1"=="sw" if "%2"=="-s" goto :session
if "%1"=="sw" if "%2"=="--session" goto :session
if "%1"=="logout" if "%2"=="-s" goto :logout
if "%1"=="logout" if "%2"=="--session" goto :logout
if "%1"=="lo" if "%2"=="-s" goto :logout
if "%1"=="env" if "%2"=="--eval" goto :eval
if "%1"=="env" if "%2"=="-e" goto :eval

git-user.exe %*
goto :eof

:session
for /f "delims=" %%i in ('git-user.exe env %3 --cmd') do %%i
goto :eof

:logout
for /f "delims=" %%i in ('git-user.exe env --unset --cmd') do %%i
goto :eof

:eval
for /f "delims=" %%i in ('git-user.exe env %3 %4 %5 --cmd') do %%i
goto :eof
`
