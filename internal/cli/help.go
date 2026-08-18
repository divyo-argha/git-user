package cli

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/ui"
)

func commandUsage(sub string) string {
	usage := map[string]string{
		"register": `Usage: git-user register [flags]

Create a new identity (guided setup).

Flags:
  -n, --name <name>            Identity name (e.g., 'work', 'personal')
  -e, --email <email>          Email address
  -t, --temp                   Create as a temporary identity
  -h, --help                   Show this help

Examples:
  git-user register
  git-user register -n work -e me@work.com
  git-user register --temp -n guest -e guest@corp.com`,
		"switch": `Usage: git-user switch <name> [flags]

Switch to an identity.

Flags:
  -c <name> [-e email]       Create a new identity and switch to it
  -e, --email <email>        Email address (with -c)
  -t, --temp                 Create as a temporary identity (with -c)
  --skip-ssh                 Skip SSH key setup (with -c)
  -l, --local                Switch only for the current repository
  --original                 Import and switch to the original pre-git-user identity
  -h, --help                 Show this help

Examples:
  git-user switch work
  git-user switch work --local
  git-user switch -c work -e me@work.com`,
		"list": `Usage: git-user list [flags]

List all identities.

Flags:
  --plain                    Plain, machine-readable output (name <email> [# active])
  --json                     JSON output
  -h, --help                 Show this help`,
		"current": `Usage: git-user current [flags]

Show the active identity.

Flags:
  --plain                    Plain, machine-readable output (name <email>)
  --json                     JSON output
  -h, --help                 Show this help`,
		"prompt": `Usage: git-user prompt

Output the active identity for terminal integration.`,
		"remove": `Usage: git-user remove <name> [flags]

Delete an identity.

Flags:
  -f, --force                  Skip confirmation
  -h, --help                   Show this help`,
		"edit": `Usage: git-user edit <name> <email>

Update an identity's email address.`,
		"rename": `Usage: git-user rename <old-name> <new-name>

Rename an identity. The active git config user.name is updated automatically
when the active identity is renamed. Useful for resolving an import conflict.`,
		"pubkey": `Usage: git-user pubkey [publish [platform]]

Show the public SSH key for the active identity.
  pubkey publish [platform]  Publish the key to github, gitlab, or bitbucket.
  (push is a hidden alias for publish)

Flags:
  -h, --help                   Show this help`,
		"bind-key": `Usage: git-user bind-key <name> [flags]

Add or link an SSH key to an identity. ` + "`bind`" + ` is a hidden alias.

Flags:
  --ssh-key <path>             Path to an existing SSH private key
  -h, --help                   Show this help`,
		"bind": `Usage: git-user bind <name> [flags]

Hidden alias for git-user bind-key.`,
		"bind-path": `Usage: git-user bind-path <name> <path>

Bind a directory path to an identity for auto-switching.`,
		"unbind-path": `Usage: git-user unbind-path <name> <path>

Remove a directory binding from an identity.`,
		"passphrase": `Usage: git-user passphrase [flags]

Manage the passphrase for the active, unlocked identity.

Flags:
  -s, --set                    Set/change the passphrase
  -r, --remove                 Remove the passphrase
  --verify                     Verify the passphrase (deprecated short form: -v)
  -m, --mode <mode>            persistent | login | everytime
  -h, --help                   Show this help`,
		"rekey": `Usage: git-user rekey <name> [flags]

Rotate the SSH key for an identity.

Flags:
  -f, --force                  Skip confirmation
  -h, --help                   Show this help`,
		"sign": `Usage: git-user sign <name>

Manage commit signing for an identity.`,
		"fix-remote": `Usage: git-user fix-remote

Convert HTTPS remotes in the current repository to SSH.`,
		"export": `Usage: git-user export [name...] [flags]

Export identities + SSH keys as an encrypted bundle.

Flags:
  --all                        Export all identities
  -h, --help                   Show this help`,
		"import-original": `Usage: git-user import-original [name]

Hidden alias for 'git-user switch --original'.

Imports the original pre-git-user gitconfig identity into git-user and
switches to it. A name argument is optional; when omitted you are prompted
to choose one.`,
		"import": `Usage: git-user import <file> [flags]

Import identities from a bundle.

Flags:
  -f, --force                  Overwrite existing identities
  -h, --help                   Show this help`,
		"doctor": `Usage: git-user doctor

Diagnose common setup and configuration issues.`,
		"tui": `Usage: git-user tui

Launch the interactive terminal UI.`,
		"completion": `Usage: git-user completion <shell>

Generate shell completion (bash, zsh, or fish).`,
		"hook": `Usage: git-user hook <install|uninstall|check>

Manage the git pre-commit identity-checking hook.`,
		"audit": `Usage: git-user audit [--fix]

Run a security audit on your identity setup. ` + "`security`" + ` is a hidden alias.

Flags:
  --fix    Prompt to automatically chmod misconfigured config/key files to 0600`,
		"security": `Usage: git-user security

Hidden alias for git-user audit.`,
		"log": `Usage: git-user log [flags]

Show the audit log of identity switches (timestamp, identity, directory).
` + "`history`" + ` is a hidden alias.

Flags:
  -n, --limit <count>          Show only the last <count> entries (default 20)
  --all                        Show all recorded entries
  --plain                      Tab-separated output for scripting

Examples:
  git-user log
  git-user log --all
  git-user log -n 5 --plain`,
		"history": `Usage: git-user history

Hidden alias for git-user log.`,
		"logout": `Usage: git-user logout

Sign out and clear the active identity.`,
		"clone": `Usage: git-user clone <repo-url> [dir] [flags]

Clone a repository and auto-configure the local identity.

Flags:
  -a, --as <name>              Identity to configure locally (default: active)
  -b, --bind                   Bind the clone directory to the identity
  -h, --help                   Show this help`,
		"stats": `Usage: git-user stats

Audit and show commit author identity stats for the current repository.`,
		"config": `Usage: git-user config <list|set|unset>

Manage custom git configuration for an identity.

Commands:
  config list [identity]            List custom config (defaults to active)
  config set <identity> <key> <val> Set a custom config key
  config unset <identity> <key>     Remove a custom config key

Examples:
  git-user config list work
  git-user config set work init.defaultBranch main
  git-user config unset work init.defaultBranch`,
		"sync": `Usage: git-user sync

Synchronize identities across devices using a private repository.`,
		"env": `Usage: git-user env [name] [flags]

Output shell export statements to isolate an identity to the current terminal session.

Flags:
  -u, --unset                  Output unset statements to restore global identity
  --shell <sh|fish|pwsh|cmd>   Target shell format (auto-detected if omitted)
  --fish                       Shortcut for --shell fish
  --powershell, --pwsh         Shortcut for --shell powershell
  --cmd                        Shortcut for --shell cmd
  -h, --help                   Show this help

Examples:
  eval "$(git-user env work)"         # Activate 'work' profile in this terminal only
  eval "$(git-user env --unset)"      # Revert to global active profile
  git-user env work --fish | source   # Fish shell activation`,
		"shell": `Usage: git-user shell <name>

Launch an isolated subshell locked to the specified Git identity.
All git commands executed in this subshell will use this identity.

Type 'exit' to leave the subshell and return to your default environment.

Examples:
  git-user shell work
  git-user shell personal`,
		"exec": `Usage: git-user exec <name> -- <command...>

Execute a single command using the specified Git identity's environment.

Examples:
  git-user exec work -- git push origin main
  git-user exec personal -- git commit -m "personal commit"`,
		"init": `Usage: git-user init [shell]

Generate a shell integration wrapper function to enable seamless per-terminal switching.

Supported shells: bash, zsh, fish, powershell

Installation:
  # Bash / Zsh (add to ~/.bashrc or ~/.zshrc):
  eval "$(git-user init)"

  # Fish (add to ~/.config/fish/config.fish):
  git-user init fish | source

  # PowerShell (add to $PROFILE):
  Invoke-Expression (& git-user init powershell)`,
	}
	if u, ok := usage[sub]; ok {
		return u
	}
	return ""
}

func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" || a == "help" {
			return true
		}
	}
	return false
}

func runSubcommandHelp(sub string) {
	ui.PrintLogo()
	fmt.Println()
	u := commandUsage(sub)
	if u == "" {
		ui.Errorf("unknown command %q — run 'git-user --help' for usage", sub)
		return
	}
	fmt.Println(u)
}
