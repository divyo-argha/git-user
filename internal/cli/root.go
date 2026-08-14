package cli

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
	"github.com/divyo-argha/git-user/internal/version"
)

const usage = `git-user — manage multiple Git identities

QUICK START
  git-user register          Create a new identity (guided setup)
  git-user switch <name>     Switch to an identity
  git-user switch -c <name>  Create and switch in one command
  git-user list              Show all identities
  git-user current           Show active identity

COMMANDS

  Identities
    register                   Create new identity with SSH key
    switch <name>              Switch to an identity
    switch -c <name> [-e email] Create new identity and switch to it
    switch -c <name> --skip-ssh  Create and switch without SSH key setup
    switch --original          Import and switch to the original pre-git-user identity
    list                       List all identities (accepts --plain, --json)
    current                    Show active identity (accepts --plain, --json)
    prompt                     Output active identity for terminal integration
    remove <name>              Delete an identity
    edit <name> <email>        Update email
    rename <old-name> <new-name> Rename an identity

  SSH & Keys
    pubkey                     Show public key for active identity
    pubkey publish [platform]  Publish public SSH key to GitHub, GitLab, or Bitbucket
    bind-key <name> [--ssh-key <p>] Add/link SSH key (interactive if no path)
    bind-path <name> <path>    Bind a directory path to an identity for auto-switching
    unbind-path <name> <path>  Unbind a directory path from an identity
    passphrase                 Add/change passphrase for active, unlocked identity
    rekey <name>               Rotate SSH key

  Repos & Portability
    fix-remote                 Convert HTTPS remotes to SSH
    clone <repo-url> [dir]     Clone repository and auto-configure local identity
    export --all               Export all identities + SSH keys (encrypted)
    export <name> [name...]    Export specific identities (encrypted)
    import [--force] <file>    Import identities from a bundle
    sync                       Synchronize identities across devices using a private repository

  System
    doctor                     Check setup
    audit                      Run security audit
    stats                      Audit and show commit author identity stats
    sign <name>                Manage commit signing for an identity
    config <list|set|unset>    Manage custom git configurations for an identity
    hook <install|uninstall>   Manage git pre-commit hooks
    log [-n <count>|--all]     Show the identity-switch audit log
    logout                     Sign out and clear active identity
    tui                        Interactive menu
    completion <shell>         Generate shell completion (bash/zsh/fish)

ALIASES
  reg                        register
  ls                         list
  sw                         switch
  rm                         remove
  lo, signout                logout
  history                    log (hidden alias)
  import-original            switch --original (hidden alias)
  pubkey push                pubkey publish (hidden alias)
  security                   audit (hidden alias)
  -i, --interactive          tui
  (running git-user alone also opens the TUI on a terminal)
EXAMPLES
  git-user register                    # Guided setup with all options
  git-user switch -c work              # Quick create and switch
  git-user switch -c work -e me@work.com  # With email
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

	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}

	if shouldPromptFirstRunImport(sub) {
		if err := maybePromptFirstRunImport(); err != nil {
			return err
		}
	}

	if len(args) == 0 {
		if !ui.IsTTY() {
			printConciseStatus()
			return nil
		}
		return runTui()
	}

	if args[0] == "--help" || args[0] == "-h" {
		ui.PrintLogo()
		fmt.Print(usage)
		return nil
	}

	if args[0] == "help" {
		if len(args) > 1 {
			runSubcommandHelp(args[1])
		} else {
			ui.PrintLogo()
			fmt.Print(usage)
		}
		return nil
	}

	if args[0] == "--update" || args[0] == "update" {
		return RunUpdate()
	}

	if args[0] == "--version" || args[0] == "-v" {
		fmt.Printf("git-user %s\n", version.GetVersion())
		return nil
	}

	sub = args[0]
	rest := args[1:]

	if wantsHelp(rest) {
		runSubcommandHelp(sub)
		return nil
	}

	switch sub {
	case "register", "reg":
		return runRegister(rest)
	case "list", "ls":
		return runList(rest)
	case "switch", "sw":
		return runSwitch(rest)
	case "current":
		return runCurrent(rest)
	case "prompt":
		return runPrompt(rest)
	case "remove", "rm":
		return runRemove(rest)
	case "edit":
		return runEdit(rest)
	case "rename":
		return runRename(rest)
	case "pubkey":
		if len(rest) > 0 {
			switch rest[0] {
			case "publish":
				return runPubkeyPush(rest[1:])
			case "push":
				// Hidden backwards-compatible alias.
				return runPubkeyPush(rest[1:])
			}
		}
		return runPubkey(rest)
	case "bind-key", "bind":
		return runBind(rest)
	case "bind-path":
		return runBindPath(rest)
	case "unbind-path":
		return runUnbindPath(rest)
	case "passphrase":
		return runPassphrase(rest)
	case "rekey":
		return runRekey(rest)
	case "sign":
		return runSign(rest)
	case "fix-remote":
		return runFixRemote(rest)
	case "export":
		return runExport(rest)
	case "import-original":
		return runSwitchOriginal(rest)
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
	case "audit", "security":
		return runSecurityCheck(rest)
	case "logout", "lo", "signout":
		return runLogout(rest)
	case "clone":
		return runClone(rest)
	case "stats":
		return runStats(rest)
	case "config":
		return runConfig(rest)
	case "sync":
		return runSync(rest)
	case "log", "history":
		return runLog(rest)
	default:
		// Try as identity name → detail view
		if handleUnknownArg(sub) {
			return nil
		}
		ui.Errorf("unknown command %q — run 'git-user --help' for usage", sub)
		return fmt.Errorf("unknown command")
	}
}

// printConciseStatus prints a compact summary for non-interactive terminals.
func printConciseStatus() {
	store, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		return
	}

	fmt.Println(renderHeader(store, 30))
	fmt.Println()
	fmt.Println("Run in an interactive terminal to open the TUI dashboard.")
	fmt.Println("Run `git-user --help` to view all available commands.")
}
