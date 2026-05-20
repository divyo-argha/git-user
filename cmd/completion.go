package cmd

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/ui"
)

func runCompletion(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user completion <bash|zsh|fish>")
		fmt.Println()
		ui.Info("To enable completions:")
		fmt.Println()
		fmt.Println("  Bash:")
		fmt.Println("    git-user completion bash > /etc/bash_completion.d/git-user")
		fmt.Println("    # or for user only:")
		fmt.Println("    git-user completion bash > ~/.local/share/bash-completion/completions/git-user")
		fmt.Println()
		fmt.Println("  Zsh:")
		fmt.Println("    git-user completion zsh > \"${fpath[1]}/_git-user\"")
		fmt.Println("    # or add to ~/.zshrc:")
		fmt.Println("    source <(git-user completion zsh)")
		fmt.Println()
		fmt.Println("  Fish:")
		fmt.Println("    git-user completion fish > ~/.config/fish/completions/git-user.fish")
		return fmt.Errorf("missing shell type")
	}

	shell := args[0]

	switch shell {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	case "fish":
		fmt.Print(fishCompletion)
	default:
		ui.Errorf("unsupported shell: %s", shell)
		ui.Info("Supported shells: bash, zsh, fish")
		return fmt.Errorf("unsupported shell")
	}

	return nil
}

const bashCompletion = `# bash completion for git-user

_git_user_completions() {
    local cur prev commands
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    commands="register switch list current remove edit bind rekey fix-remote export import doctor tui completion"
    
    # Complete commands
    if [ $COMP_CWORD -eq 1 ]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi
    
    # Complete identity names for commands that need them
    case "${prev}" in
        switch|sw|remove|rm|edit|bind|rekey|export)
            local identities=$(git-user list 2>/dev/null | grep -v "^No identities" | awk '{print $1}' | grep -v "^$")
            COMPREPLY=( $(compgen -W "${identities}" -- ${cur}) )
            return 0
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) )
            return 0
            ;;
        export)
            if [ "${COMP_WORDS[1]}" = "export" ]; then
                COMPREPLY=( $(compgen -W "--all" -- ${cur}) )
            fi
            return 0
            ;;
    esac
}

complete -F _git_user_completions git-user
`

const zshCompletion = `#compdef git-user

_git_user() {
    local -a commands identities
    commands=(
        'register:Create new identity with SSH key'
        'switch:Switch to an identity'
        'list:List all identities'
        'ls:List all identities (alias)'
        'current:Show active identity'
        'remove:Delete an identity'
        'rm:Delete an identity (alias)'
        'edit:Update email'
        'bind:Add/link SSH key'
        'rekey:Rotate SSH key'
        'fix-remote:Convert HTTPS remotes to SSH'
        'export:Export identities'
        'import:Import identities'
        'doctor:Check setup'
        'tui:Interactive menu'
        'completion:Generate shell completion'
    )
    
    _arguments -C \
        '1: :->command' \
        '*:: :->args'
    
    case $state in
        command)
            _describe 'command' commands
            ;;
        args)
            case $words[1] in
                switch|sw|remove|rm|edit|bind|rekey|export)
                    identities=(${(f)"$(git-user list 2>/dev/null | grep -v '^No identities' | awk '{print $1}' | grep -v '^$')"})
                    _describe 'identity' identities
                    ;;
                completion)
                    _values 'shell' bash zsh fish
                    ;;
                export)
                    if (( CURRENT == 2 )); then
                        _values 'option' --all
                    fi
                    identities=(${(f)"$(git-user list 2>/dev/null | grep -v '^No identities' | awk '{print $1}' | grep -v '^$')"})
                    _describe 'identity' identities
                    ;;
            esac
            ;;
    esac
}

_git_user "$@"
`

const fishCompletion = `# fish completion for git-user

# Commands
complete -c git-user -f -n "__fish_use_subcommand" -a "register" -d "Create new identity with SSH key"
complete -c git-user -f -n "__fish_use_subcommand" -a "switch" -d "Switch to an identity"
complete -c git-user -f -n "__fish_use_subcommand" -a "sw" -d "Switch to an identity (alias)"
complete -c git-user -f -n "__fish_use_subcommand" -a "list" -d "List all identities"
complete -c git-user -f -n "__fish_use_subcommand" -a "ls" -d "List all identities (alias)"
complete -c git-user -f -n "__fish_use_subcommand" -a "current" -d "Show active identity"
complete -c git-user -f -n "__fish_use_subcommand" -a "remove" -d "Delete an identity"
complete -c git-user -f -n "__fish_use_subcommand" -a "rm" -d "Delete an identity (alias)"
complete -c git-user -f -n "__fish_use_subcommand" -a "edit" -d "Update email"
complete -c git-user -f -n "__fish_use_subcommand" -a "bind" -d "Add/link SSH key"
complete -c git-user -f -n "__fish_use_subcommand" -a "rekey" -d "Rotate SSH key"
complete -c git-user -f -n "__fish_use_subcommand" -a "fix-remote" -d "Convert HTTPS remotes to SSH"
complete -c git-user -f -n "__fish_use_subcommand" -a "export" -d "Export identities"
complete -c git-user -f -n "__fish_use_subcommand" -a "import" -d "Import identities"
complete -c git-user -f -n "__fish_use_subcommand" -a "doctor" -d "Check setup"
complete -c git-user -f -n "__fish_use_subcommand" -a "tui" -d "Interactive menu"
complete -c git-user -f -n "__fish_use_subcommand" -a "completion" -d "Generate shell completion"

# Identity name completions
function __git_user_identities
    git-user list 2>/dev/null | grep -v "^No identities" | awk '{print $1}' | grep -v "^$"
end

complete -c git-user -f -n "__fish_seen_subcommand_from switch sw" -a "(__git_user_identities)"
complete -c git-user -f -n "__fish_seen_subcommand_from remove rm" -a "(__git_user_identities)"
complete -c git-user -f -n "__fish_seen_subcommand_from edit" -a "(__git_user_identities)"
complete -c git-user -f -n "__fish_seen_subcommand_from bind" -a "(__git_user_identities)"
complete -c git-user -f -n "__fish_seen_subcommand_from rekey" -a "(__git_user_identities)"
complete -c git-user -f -n "__fish_seen_subcommand_from export" -a "(__git_user_identities)"

# Completion shell types
complete -c git-user -f -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"

# Export --all flag
complete -c git-user -f -n "__fish_seen_subcommand_from export" -l "all" -d "Export all identities"

# Switch -c flag
complete -c git-user -f -n "__fish_seen_subcommand_from switch sw" -s "c" -d "Create and switch"
`
