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
  git-user list              Show all identities
  git-user current           Show active identity

COMMANDS
  register                   Create new identity with SSH key
  switch <name>              Switch to an identity
  list                       List all identities
  current                    Show active identity
  remove <name>              Delete an identity
  edit <name> <email>        Update email
  bind <name> --ssh-key <p>  Link SSH key
  rekey <name>               Rotate SSH key
  doctor                     Check setup
  tui                        Interactive menu

ALIASES
  ls (list)  sw (switch)  rm (remove)

HELP
  git-user --help            Show this help
  git-user --version         Show version
  git-user doctor            Diagnose issues

Config: ~/.git-users/config.json
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
	case "bind":
		return runBind(rest)
	case "rekey":
		return runRekey(rest)
	case "doctor":
		return runDoctor(rest)
	case "tui", "-i", "--interactive":
		return runTui()
	default:
		ui.Errorf("unknown command %q — run 'git-user --help' for usage", sub)
		return fmt.Errorf("unknown command")
	}
}
