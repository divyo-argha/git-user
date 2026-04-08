package cmd

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/ui"
)

const usage = `git-user — manage multiple Git identities

USAGE
  git-user <command> [arguments]
  git-user tui             Open interactive menu

COMMANDS
  add     [name] [email]   Add a new Git identity (interactive if no args)
  list                     List all saved identities
  switch  [-c] <name> [e] Switch or create and switch identity
  current                  Show the currently active identity
  current --sign-out       Sign out and enter void state (no commits/pushes)
  sign-in [--remember]     Sign in with optional credential persistence
  remove  <name>           Remove a saved identity
  edit    <name> <email>   Update the email for an existing identity
  bind    <name> [flags]   Associate an SSH key or Signing key
  platform <add/remove>    Manage platform accounts (github, gitlab, etc.)
  discover                 Scan system for existing Git/SSH identities
  tui                      Open an interactive management menu
  prompt                   Output current identity for shell prompts (git repos only)
  setup-prompt             Automated shell prompt configuration
  remove-prompt            Remove automated shell configurations
  reload                   Refresh shell prompt configuration
  config [flags]           Manage global git-user settings
  init    <zsh|bash>       Generate shell integration script

ALIASES
  ls      alias for list
  sw      alias for switch
  rm      alias for remove
  reg     alias for add (interactive)
  register alias for add (interactive)

FLAGS
  --ssh-key <path> (bind) Link SSH key
  --signing-key <k> (add/bind) Link GPG/SSH key
  --method <gpg|ssh>(add/bind) Set signing method
  --unset-signing  (bind) Remove signing key
  --remember       (sign-in) Persist credentials across profile switches
  --force           (remove) Force-remove the active user
  -i, --interactive Open TUI (shortcut for tui)
  --help            Show this help text

SETUP AS GIT SUBCOMMAND
  Place git-user on your PATH so you can run:
    git user <command>

  (Git automatically delegates "git user" to "git-user" if the binary is on PATH.)

SHELL PROMPT INTEGRATION
  To show the active user in your prompt, add this to your .zshrc or .bashrc:
    eval "$(git-user init zsh)"   # for Zsh
    eval "$(git-user init bash)"  # for Bash

  The prompt will only display when inside a git repository.

EXAMPLES
  git-user add work   work@company.com
  git-user add home   me@gmail.com
  git-user list
  git-user switch work
  git-user switch -c personal me@gmail.com
  git-user current
  git-user current --sign-out
  git-user sign-in --remember
  git-user edit home  personal@gmail.com
  git-user remove home
  git-user bind work  --ssh-key ~/.ssh/id_rsa_work

Config stored at: ~/.git-users/config.json
`

// Execute is the top-level entry point.
func Execute() error {
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Print(usage)
		return nil
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "add", "register", "reg":
		return runAdd(rest)
	case "list", "ls":
		return runList(rest)
	case "switch", "sw":
		return runSwitch(rest)
	case "current":
		return runCurrent(rest)
	case "sign-in":
		return runSignIn(rest)
	case "remove", "rm":
		return runRemove(rest)
	case "edit":
		return runEdit(rest)
	case "bind":
		return runBind(rest)
	case "platform":
		return runPlatform(rest)
	case "discover":
		return runDiscover(rest)
	case "tui":
		return runTui()
	case "prompt":
		return runPrompt(rest)
	case "init":
		return runInit(rest)
	case "setup-prompt":
		return runSetupPrompt(rest)
	case "remove-prompt":
		return runRemovePrompt(rest)
	case "reload":
		return runReload(rest)
	case "config":
		return runConfig(rest)
	default:
		if sub == "-i" || sub == "--interactive" {
			return runTui()
		}
		ui.Errorf("unknown command %q — run 'git-user --help' for usage", sub)
		return fmt.Errorf("unknown command")
	}
}
