package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runInit(args []string) error {
	var shell string
	if len(args) == 0 {
		shell = DetectShell()
		if shell == "" {
			fmt.Println("usage: eval \"$(git-user init <zsh|bash>)\"")
			return nil
		}
	} else {
		shell = strings.ToLower(args[0])
	}

	exe, err := os.Executable()
	if err != nil {
		exe = "git-user" // fallback
	}

	switch shell {
	case "zsh", "zsh-p10k":
		// Check for p10k specifically
		if DetectShell() == "zsh-p10k" || shell == "zsh-p10k" {
			fmt.Print(strings.ReplaceAll(p10kInit, "{{EXE}}", exe))
		} else {
			fmt.Print(strings.ReplaceAll(zshInit, "{{EXE}}", exe))
		}
	case "bash":
		fmt.Print(strings.ReplaceAll(bashInit, "{{EXE}}", exe))
	default:
		fmt.Printf("unsupported shell: %s\n", shell)
	}

	return nil
}

const p10kInit = `
# Powerlevel10k custom segment for git-user
typeset -g POWERLEVEL9K_GIT_USER_FOREGROUND=14
typeset -g POWERLEVEL9K_GIT_USER_VISUAL_IDENTIFIER_COLOR=18
typeset -g POWERLEVEL9K_GIT_USER_BOLD=true

prompt_git_user() {
  local name=$({{EXE}} prompt --no-icon)
  if [[ -n "$name" ]]; then
    # We use p10k segment for a perfectly integrated look
    p10k segment -i '👤' -t "$name"
  fi
}

# Dynamically add to right prompt if not already there
if (( ${+POWERLEVEL9K_RIGHT_PROMPT_ELEMENTS} )); then
  if [[ ${POWERLEVEL9K_RIGHT_PROMPT_ELEMENTS[(r)git_user]} != "git_user" ]]; then
    # Add to the beginning of the right prompt
    POWERLEVEL9K_RIGHT_PROMPT_ELEMENTS=(git_user $POWERLEVEL9K_RIGHT_PROMPT_ELEMENTS)
  fi
fi
`

const zshInit = `
_git_user_prompt() {
    # If using standard RPROMPT
    if [[ -z "$GIT_USER_PROMPT_DISABLED" ]]; then
        # We fetch the prompt output using the absolute path to this binary
        export RPROMPT='$({{EXE}} prompt)'
    fi
}
# Hook into the pre-command execution to ensure it stays updated
autoload -Uz add-zsh-hook
add-zsh-hook precmd _git_user_prompt
`

const bashInit = `
_git_user_prompt() {
    local identity=$({{EXE}} prompt)
    if [[ -n "$identity" ]]; then
        # For Bash, we append it to PS1 skillfully
        export PS1="${PS1% } ($identity) "
    fi
}
export PROMPT_COMMAND="_git_user_prompt; $PROMPT_COMMAND"
`

func DetectShell() string {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return ""
	}

	if strings.Contains(shellPath, "zsh") {
		// Detect p10k
		home, _ := os.UserHomeDir()
		if _, err := os.Stat(filepath.Join(home, ".p10k.zsh")); err == nil {
			return "zsh-p10k"
		}
		return "zsh"
	}
	if strings.Contains(shellPath, "bash") {
		return "bash"
	}
	return ""
}
