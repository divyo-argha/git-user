package cli

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/ui"
)

// commandSpec is the single source of truth for shell completion generation.
// Every CLI command (including aliases) lives here, so bash/zsh/fish completions
// stay in sync automatically whenever a command is added or renamed.
type commandSpec struct {
	name    string
	desc    string
	takesID bool // completes with identity names
}

// commands is the canonical command list used to generate completions.
var commands = []commandSpec{
	{name: "register", desc: "Create a new identity (guided setup with SSH)"},
	{name: "reg", desc: "Create a new identity (alias)"},
	{name: "switch", desc: "Switch to an identity", takesID: true},
	{name: "sw", desc: "Switch to an identity (alias)", takesID: true},
	{name: "list", desc: "List all identities"},
	{name: "ls", desc: "List all identities (alias)"},
	{name: "current", desc: "Show active identity"},
	{name: "prompt", desc: "Output active identity for terminal integration"},
	{name: "remove", desc: "Delete an identity", takesID: true},
	{name: "rm", desc: "Delete an identity (alias)", takesID: true},
	{name: "edit", desc: "Update email", takesID: true},
	{name: "rename", desc: "Rename an identity", takesID: true},
	{name: "pubkey", desc: "Show the public key of the active identity"},
	{name: "bind-key", desc: "Add/link an SSH key to an identity", takesID: true},
	{name: "bind", desc: "Add/link an SSH key (alias)", takesID: true},
	{name: "bind-path", desc: "Bind a directory to an identity", takesID: true},
	{name: "unbind-path", desc: "Remove a directory binding", takesID: true},
	{name: "passphrase", desc: "Manage the passphrase for the active identity"},
	{name: "rekey", desc: "Rotate SSH key", takesID: true},
	{name: "sign", desc: "Manage commit signing for an identity", takesID: true},
	{name: "fix-remote", desc: "Convert HTTPS remotes to SSH"},
	{name: "export", desc: "Export identities (encrypted bundle)", takesID: true},
	{name: "import", desc: "Import identities from a bundle"},
	{name: "import-original", desc: "Import and switch to the original identity (alias for switch --original)"},
	{name: "doctor", desc: "Diagnose common setup issues"},
	{name: "audit", desc: "Run a security audit"},
	{name: "security", desc: "Run a security audit (alias)"},
	{name: "logout", desc: "Sign out and clear the active identity"},
	{name: "lo", desc: "Sign out (alias)"},
	{name: "signout", desc: "Sign out (alias)"},
	{name: "clone", desc: "Clone a repository and auto-configure the local identity"},
	{name: "stats", desc: "Audit commit author identity stats"},
	{name: "config", desc: "Manage custom git config for an identity", takesID: true},
	{name: "sync", desc: "Synchronize identities across devices"},
	{name: "hook", desc: "Manage git pre-commit hooks"},
	{name: "completion", desc: "Generate shell completion (bash/zsh/fish)"},
	{name: "tui", desc: "Interactive menu"},
}

// commandsThatTakeIDs returns the subcommand names (including aliases) that
// complete against registered identity names.
func commandsThatTakeIDs() []string {
	var out []string
	for _, c := range commands {
		if c.takesID {
			out = append(out, c.name)
		}
	}
	return out
}

// idCommandsPattern returns a case/pattern string listing identity-taking
// commands, e.g. "switch|sw|remove|rm|...".
func idCommandsPattern() string {
	out := ""
	for i, c := range commandsThatTakeIDs() {
		if i > 0 {
			out += "|"
		}
		out += c
	}
	return out
}

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
		fmt.Print(bashCompletion())
	case "zsh":
		fmt.Print(zshCompletion())
	case "fish":
		fmt.Print(fishCompletion())
	default:
		ui.Errorf("unsupported shell: %s", shell)
		ui.Info("Supported shells: bash, zsh, fish")
		return fmt.Errorf("unsupported shell")
	}

	return nil
}

func bashCompletion() string {
	cmdList := ""
	for i, c := range commands {
		if i > 0 {
			cmdList += " "
		}
		cmdList += c.name
	}

	return fmt.Sprintf(`# bash completion for git-user

_git_user_completions() {
    local cur prev commands
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    commands="%s"

    # Complete commands
    if [ $COMP_CWORD -eq 1 ]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi

    # Complete identity names for commands that need them
    case "${prev}" in
        %s)
            local identities=$(git-user list --plain 2>/dev/null | awk '{print $1}' | grep -v "^$")
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
`, cmdList, idCommandsPattern())
}

func zshCompletion() string {
	descLines := ""
	for _, c := range commands {
		descLines += fmt.Sprintf("        '%s:%s'\n", c.name, c.desc)
	}

	return fmt.Sprintf(`#compdef git-user

_git_user() {
    local -a commands identities
    commands=(
%s    )

    _arguments -C \
        '1: :->command' \
        '*:: :->args'

    case $state in
        command)
            _describe 'command' commands
            ;;
        args)
            case $words[1] in
                %s)
                    identities=(${(f)"$(git-user list --plain 2>/dev/null | awk '{print $1}' | grep -v '^$')"})
                    _describe 'identity' identities
                    ;;
                completion)
                    _values 'shell' bash zsh fish
                    ;;
                export)
                    if (( CURRENT == 2 )); then
                        _values 'option' --all
                    fi
                    identities=(${(f)"$(git-user list --plain 2>/dev/null | awk '{print $1}' | grep -v '^$')"})
                    _describe 'identity' identities
                    ;;
            esac
            ;;
    esac
}

_git_user "$@"
`, descLines, idCommandsPattern())
}

func fishCompletion() string {
	out := "# fish completion for git-user\n\n# Commands\n"
	for _, c := range commands {
		out += fmt.Sprintf("complete -c git-user -f -n \"__fish_use_subcommand\" -a \"%s\" -d \"%s\"\n", c.name, c.desc)
	}

	out += "\n# Identity name completions\n"
	out += "function __git_user_identities\n    git-user list --plain 2>/dev/null | awk '{print $1}' | grep -v '^$'\nend\n\n"

	for _, name := range commandsThatTakeIDs() {
		out += fmt.Sprintf("complete -c git-user -f -n \"__fish_seen_subcommand_from %s\" -a \"(__git_user_identities)\"\n", name)
	}

	out += "\n# Completion shell types\n"
	out += "complete -c git-user -f -n \"__fish_seen_subcommand_from completion\" -a \"bash zsh fish\"\n\n"

	out += "# Export --all flag\n"
	out += "complete -c git-user -f -n \"__fish_seen_subcommand_from export\" -l \"all\" -d \"Export all identities\"\n\n"

	out += "# Switch flags\n"
	out += "complete -c git-user -f -n \"__fish_seen_subcommand_from switch sw\" -s \"c\" -d \"Create and switch\"\n"
	out += "complete -c git-user -f -n \"__fish_seen_subcommand_from switch sw\" -s \"e\" -l \"email\" -d \"Email address\"\n"
	out += "complete -c git-user -f -n \"__fish_seen_subcommand_from switch sw\" -s \"l\" -l \"local\" -d \"Switch only for this repository\"\n"
	out += "complete -c git-user -f -n \"__fish_seen_subcommand_from switch sw\" -l \"original\" -d \"Import and switch to the original identity\"\n"
	out += "complete -c git-user -f -n \"__fish_seen_subcommand_from switch sw\" -l \"skip-ssh\" -d \"Skip SSH key setup (with -c)\"\n"

	return out
}
