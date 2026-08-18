package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runExec(args []string) error {
	if len(args) < 2 || args[0] == "-h" || args[0] == "--help" {
		runSubcommandHelp("exec")
		return nil
	}

	targetName := args[0]
	cmdArgs := args[1:]
	if len(cmdArgs) > 0 && cmdArgs[0] == "--" {
		cmdArgs = cmdArgs[1:]
	}

	if len(cmdArgs) == 0 {
		ui.Errorf("no command provided to execute")
		return fmt.Errorf("missing command")
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := store.FindUser(targetName)
	if user == nil {
		ui.Errorf("identity %q not found", targetName)
		return fmt.Errorf("identity not found: %s", targetName)
	}

	vars := EnvVars(user)
	env := os.Environ()
	for k, v := range vars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}
