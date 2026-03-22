package cmd

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/ui"
)

const usage = `git-user — manage multiple Git identities

USAGE
  git-user <command> [arguments]

COMMANDS
  add     <name> <email>   Add a new Git identity
  list                     List all saved identities
  switch  <name>           Switch the active Git identity
  current                  Show the currently active identity
  remove  <name>           Remove a saved identity
  edit    <name> <email>   Update the email for an existing identity
  bind    <name> [flags]   Associate an SSH key or Signing key

ALIASES
  ls      alias for list
  sw      alias for switch
  rm      alias for remove

FLAGS
  --ssh-key <path> (bind) Link SSH key
  --signing-key <k> (add/bind) Link GPG/SSH key
  --method <gpg|ssh>(add/bind) Set signing method
  --unset-signing  (bind) Remove signing key
  --force           (remove) Force-remove the active user
  --help            Show this help text

SETUP AS GIT SUBCOMMAND
  Place git-user on your PATH so you can run:
    git user <command>

  (Git automatically delegates "git user" to "git-user" if the binary is on PATH.)

EXAMPLES
  git-user add work   work@company.com
  git-user add home   me@gmail.com
  git-user list
  git-user switch work
  git-user current
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
	case "add":
		return runAdd(rest)
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
	case "bind":
		return runBind(rest)
	default:
		ui.Errorf("unknown command %q — run 'git-user --help' for usage", sub)
		return fmt.Errorf("unknown command")
	}
}
