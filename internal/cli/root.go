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
    register                   Create a new identity (guided setup)
    switch <name>              Switch to an identity
    switch -c <name> [-e email] Create new identity and switch to it
    switch -c <name> --skip-ssh  Create and switch without SSH key setup
    switch --original          Import and switch to the original pre-git-user identity
    switch --session <name>    Switch identity for current terminal session only
    list                       List all identities (accepts --plain, --json)
    current                    Show active identity (accepts --plain, --json)
    prompt                     Output active identity for terminal integration
    remove <name>              Delete an identity
    edit <name> <email>        Update email
    rename <old-name> <new-name> Rename an identity

  Terminal Sessions & Isolation
    env <name>                 Output export statements for current terminal session (eval "$(git-user env <name>)")
    env --unset                Output unset statements to restore global profile (eval "$(git-user env --unset)")
    shell <name>               Launch an isolated subshell for an identity
    exec <name> -- <cmd...>    Execute a single command using an identity
    init [shell]               Generate shell integration hook (eval "$(git-user init 2>/dev/null)")

  SSH & Keys
    pubkey                     Show public key for active identity
    pubkey publish [platform]  Publish public SSH key to GitHub, GitLab, or Bitbucket
    pubkey test [name]         Test connections on GitHub, GitLab, and Bitbucket
    connections [name]         Check connections to GitHub, GitLab, and Bitbucket
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
    doctor                     Check setup (read-only diagnostics)
    refresh                    Fix config conflicts doctor finds (re-syncs git config to match git-user's state)
    audit                      Run security audit
    stats                      Audit and show commit author identity stats
    sign <name>                Manage commit signing for an identity
    config <list|set|unset>    Manage custom git configurations for an identity
    hook <install|uninstall>   Manage git pre-commit hooks
    log [-n <count>|--all]     Show the identity-switch audit log
    logout                     Sign out and clear active identity
    uninstall [--yes]          Remove git-user entirely: identities, keys, config, restore original git identity
    tui                        Interactive menu
    completion <shell>         Generate shell completion (bash/zsh/fish)

ALIASES
  reg                        register
  ls                         list
  sw                         switch
  rm                         remove
  lo, signout                logout
  history                    log (hidden alias)
  check-ssh, check           connections (hidden alias)
  import-original            switch --original (hidden alias)
  pubkey push                pubkey publish (hidden alias)
  pubkey check               pubkey test (hidden alias)
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

	normSub := normalizeSubcommand(sub)

	if shouldPromptFirstRunImport(normSub) {
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

	if normSub == "help" {
		if len(args) > 1 {
			runSubcommandHelp(normalizeSubcommand(args[1]))
		} else {
			ui.PrintLogo()
			fmt.Print(usage)
		}
		return nil
	}

	if normSub == "update" {
		return RunUpdate()
	}

	if normSub == "version" {
		fmt.Printf("git-user %s\n", version.GetVersion())
		return nil
	}

	rest := args[1:]

	if wantsHelp(rest) {
		runSubcommandHelp(normSub)
		return nil
	}

	switch normSub {
	case "register":
		return runRegister(rest)
	case "list":
		return runList(rest)
	case "switch":
		return runSwitch(rest)
	case "current":
		return runCurrent(rest)
	case "prompt":
		return runPrompt(rest)
	case "remove":
		return runRemove(rest)
	case "edit":
		return runEdit(rest)
	case "rename":
		return runRename(rest)
	case "pubkey":
		if len(rest) > 0 {
			switch rest[0] {
			case "publish", "push", "--publish", "--push":
				return runPubkeyPush(rest[1:])
			case "check", "test", "status", "--check", "--test", "--status":
				return runCheckSSH(rest[1:])
			}
		}
		return runPubkey(rest)
	case "connections", "check-ssh":
		return runCheckSSH(rest)
	case "bind-key":
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
	case "refresh":
		return runRefresh(rest)
	case "uninstall":
		return runUninstall(rest)
	case "tui":
		return runTui()
	case "completion":
		return runCompletion(rest)
	case "hook":
		return runHook(rest)
	case "audit":
		return runSecurityCheck(rest)
	case "logout":
		return runLogout(rest)
	case "clone":
		return runClone(rest)
	case "stats":
		return runStats(rest)
	case "config":
		return runConfig(rest)
	case "sync":
		return runSync(rest)
	case "log":
		return runLog(rest)
	case "env":
		return runEnv(rest)
	case "shell":
		return runShell(rest)
	case "exec":
		return runExec(rest)
	case "init":
		return runInit(rest)
	default:
		// Try as identity name → detail view
		if handleUnknownArg(sub) {
			return nil
		}
		ui.Errorf("unknown command %q — run 'git-user --help' for usage", sub)
		return fmt.Errorf("unknown command")
	}
}

// normalizeSubcommand normalizes flag aliases and command variations to canonical command names.
func normalizeSubcommand(sub string) string {
	switch sub {
	// Help & Version
	case "--help", "-h", "-?", "help":
		return "help"
	case "--version", "-v", "-V", "version":
		return "version"
	case "--update", "-u", "update", "--upgrade":
		return "update"

	// Core Identity Commands
	case "current", "--current", "-c", "whoami", "--whoami", "active", "--active":
		return "current"
	case "list", "--list", "-l", "ls", "--ls":
		return "list"
	case "switch", "--switch", "-s", "sw", "--sw", "use", "--use":
		return "switch"
	case "register", "--register", "-r", "reg", "--reg", "add", "--add", "-a":
		return "register"
	case "remove", "--remove", "rm", "--rm", "-rm", "delete", "--delete", "-d", "del", "--del":
		return "remove"
	case "edit", "--edit":
		return "edit"
	case "rename", "--rename":
		return "rename"
	case "logout", "--logout", "signout", "--signout", "lo", "--lo":
		return "logout"

	// Terminal Sessions & Execution
	case "env", "--env":
		return "env"
	case "shell", "--shell":
		return "shell"
	case "exec", "--exec", "run", "--run":
		return "exec"
	case "init", "--init":
		return "init"

	// Keys & Signing
	case "pubkey", "--pubkey", "-k", "key", "--key":
		return "pubkey"
	case "bind-key", "--bind-key", "bind", "--bind":
		return "bind-key"
	case "bind-path", "--bind-path":
		return "bind-path"
	case "unbind-path", "--unbind-path":
		return "unbind-path"
	case "passphrase", "--passphrase":
		return "passphrase"
	case "rekey", "--rekey":
		return "rekey"
	case "sign", "--sign":
		return "sign"

	// Diagnostics & Security
	case "connections", "--connections", "check-ssh", "--check-ssh", "check", "--check", "test-ssh", "--test-ssh":
		return "connections"
	case "doctor", "--doctor":
		return "doctor"
	case "refresh", "--refresh", "repair", "--repair", "fix", "--fix":
		return "refresh"
	case "uninstall", "--uninstall", "purge", "--purge":
		return "uninstall"
	case "audit", "--audit", "security", "--security":
		return "audit"
	case "stats", "--stats":
		return "stats"
	case "log", "--log", "history", "--history":
		return "log"

	// Workflows & Integration
	case "prompt", "--prompt":
		return "prompt"
	case "clone", "--clone":
		return "clone"
	case "fix-remote", "--fix-remote":
		return "fix-remote"
	case "export", "--export":
		return "export"
	case "import", "--import":
		return "import"
	case "import-original", "--import-original":
		return "import-original"
	case "sync", "--sync":
		return "sync"
	case "hook", "--hook", "hooks", "--hooks":
		return "hook"
	case "config", "--config":
		return "config"
	case "completion", "--completion":
		return "completion"
	case "tui", "--tui", "-i", "--interactive":
		return "tui"
	default:
		return sub
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
