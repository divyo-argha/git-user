package cmd

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/ui"
)

const usage = `git-user — manage multiple Git identities

QUICK START
  git-user register          Create a new identity (guided setup)
  git-user switch <name>     Switch to an identity
  git-user switch -c <name>  Create and switch in one command
  git-user list              Show all identities
  git-user current           Show active identity

COMMANDS
  register                   Create new identity with SSH key
  switch <name>              Switch to an identity
  switch -c <name> [email]   Create new identity and switch to it
  list                       List all identities
  current                    Show active identity
  remove <name>              Delete an identity
  edit <name> <email>        Update email
  pubkey                     Show public key for active identity only
  bind <name> [--ssh-key <p>] Add/link SSH key (interactive if no path)
  passphrase                 Add/change passphrase for active, unlocked identity
  rekey <name>               Rotate SSH key
  fix-remote                 Convert HTTPS remotes to SSH
  export --all               Export all identities + SSH keys (encrypted)
  export <name> [name...]    Export specific identities (encrypted)
  import <file>              Import identities from a bundle
  doctor                     Check setup
  tui                        Interactive menu
  completion <shell>         Generate shell completion (bash/zsh/fish)
  hook <install|uninstall>   Manage git pre-commit hooks
  security                   Run security audit
  session start [name] [--ttl <duration>]  Load an identity's SSH key into ssh-agent
  session start --temp <name> <email> [--ttl] Temporary session — key not saved, deleted on stop
  session stop [name]          Unload an identity's SSH key; identity stays selected
  session stop --all           Remove all keys from ssh-agent
  session status               Show ssh-agent and loaded-key status

ALIASES
  ls (list)  sw (switch)  rm (remove)

EXAMPLES
  git-user register                    # Guided setup with all options
  git-user switch -c work              # Quick create and switch
  git-user switch -c work me@work.com  # With email
  git-user switch personal             # Switch to existing identity
  git-user fix-remote                  # Convert repo remotes to SSH
  git-user completion bash > /etc/bash_completion.d/git-user  # Enable completions

HELP
  git-user --help            Show this help
  git-user --version         Show version
  git-user --update          Update to latest version
  git-user doctor            Diagnose issues

Config: ~/.git-users/config.json
`

func Execute() error {
	args := os.Args[1:]

	autoCleanupExpiredTempSession()

	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		ui.PrintLogo()
		fmt.Print(usage)
		return nil
	}

	if args[0] == "--update" || args[0] == "update" {
		return RunUpdate()
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "register", "reg":
		return runRegister(rest)
	case "list", "ls":
		return runList(rest)
	case "switch", "sw":
		return runSwitch(rest)
	case "current":
		return runCurrent(rest)
	case "remove", "rm":
		return runRemove(rest)
	case "edit":
		return runEdit(rest)
	case "pubkey":
		return runPubkey(rest)
	case "bind":
		return runBind(rest)
	case "passphrase":
		return runPassphrase(rest)
	case "rekey":
		return runRekey(rest)
	case "fix-remote":
		return runFixRemote(rest)
	case "export":
		return runExport(rest)
	case "import":
		return runImport(rest)
	case "doctor":
		return runDoctor(rest)
	case "tui", "-i", "--interactive":
		return runTui()
	case "completion":
		return runCompletion(rest)
	case "hook":
		return runHook(rest)
	case "security":
		return runSecurityCheck(rest)
	case "session":
		return runSession(rest)
	default:
		ui.Errorf("unknown command %q — run 'git-user --help' for usage", sub)
		return fmt.Errorf("unknown command")
	}
}
