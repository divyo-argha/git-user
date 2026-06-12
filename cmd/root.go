package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
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
  switch --original          Restore the gitconfig state from before git-user was first used
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
  import-original [name]     Import original gitconfig identity into git-user
  import [--force] <file>    Import identities from a bundle
  doctor                     Check setup
  tui                        Interactive menu
  completion <shell>         Generate shell completion (bash/zsh/fish)
  hook <install|uninstall>   Manage git pre-commit hooks
  security                   Run security audit
  sign <name>                Manage commit signing for an identity
  logout                     Sign out and clear active identity

ALIASES

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

func init() {
	if os.Getenv("GIT_USER_ASKPASS_MODE") == "true" {
		prompt := ""
		if len(os.Args) > 1 {
			prompt = strings.ToLower(os.Args[1])
		}

		if strings.Contains(prompt, "old") || strings.Contains(prompt, "current") {
			fmt.Println(os.Getenv("GIT_USER_OLD_PASSPHRASE"))
		} else if strings.Contains(prompt, "new") || strings.Contains(prompt, "again") || strings.Contains(prompt, "confirm") {
			fmt.Println(os.Getenv("GIT_USER_NEW_PASSPHRASE"))
		} else {
			if val := os.Getenv("GIT_USER_PASSPHRASE"); val != "" {
				fmt.Println(val)
			} else if val := os.Getenv("GIT_USER_OLD_PASSPHRASE"); val != "" {
				fmt.Println(val)
			} else {
				fmt.Println(os.Getenv("GIT_USER_NEW_PASSPHRASE"))
			}
		}
		os.Exit(0)
	}
}

func Execute() error {

	args := os.Args[1:]

	autoSeedFromGitconfig() // first-run: import existing .gitconfig identity

	if len(args) == 0 {
		if !ui.IsTTY() {
			printConciseStatus()
			return nil
		}
		return runTui()
	}

	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
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
	case "sign":
		return runSign(rest)
	case "fix-remote":
		return runFixRemote(rest)
	case "export":
		return runExport(rest)
	case "import-original":
		return runImportOriginal(rest)
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
	case "logout", "lo", "signout":
		return runLogout(rest)
	default:
		// Try as identity name → detail view
		if handleUnknownArg(sub) {
			return nil
		}
		ui.Errorf("unknown command %q — run 'git-user --help' for usage", sub)
		return fmt.Errorf("unknown command")
	}
}

// autoSeedFromGitconfig is a no-op if any identities already exist.
func autoSeedFromGitconfig() {
	store, err := config.Load()
	if err != nil || len(store.Users) > 0 {
		return 
	}

	name := git.CurrentName()
	email := git.CurrentEmail()
	sshCommand := git.CurrentSSHCommand()

	if name == "" && email == "" {
		return 
	}

	importName := name
	if importName == "" {
		importName = "original"
	}

	store.SnapshotOriginal(name, email, sshCommand, git.CurrentSigningKey(), git.CurrentSignFormat(), git.CurrentCommitGPGSign())

	store.Users = append(store.Users, config.User{
		Name:   importName,
		Email:  email,
		SSHKey: extractSSHKeyFromCommand(sshCommand),
		Source: "original",
	})

	_ = config.Save(store)
}

func printConciseStatus() {
	store, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		return
	}

	fmt.Println("git-user — manage multiple Git identities")
	fmt.Println()

	// Active Identity
	if store.Current != "" {
		if u := store.CurrentUser(); u != nil {
			fmt.Printf("  Active Profile: \033[1;32m%s\033[0m (%s)\n", u.Name, u.Email)
		} else {
			fmt.Printf("  Active Profile: \033[1;31m%s (missing)\033[0m\n", store.Current)
		}
	} else {
		fmt.Println("  Active Profile: \033[1;30mNone (logged out)\033[0m")
	}

	// SSH Agent Connection
	_, conn, err := getAgentClient()
	if err == nil {
		defer conn.Close()
		fmt.Println("  SSH Agent     : \033[1;32mConnected\033[0m")
		if fingerprints, errList := loadedSSHKeyFingerprints(); errList == nil {
			fmt.Printf("  Loaded Keys   : %d\n", len(fingerprints))
		}
	} else {
		fmt.Println("  SSH Agent     : \033[1;31mNot reachable\033[0m")
	}

	fmt.Println("\nRun in an interactive terminal to open the TUI dashboard.")
	fmt.Println("Run `git-user --help` to view all available commands.")
}
