package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runShell(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		runSubcommandHelp("shell")
		return nil
	}

	targetName := args[0]
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

	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		if runtime.GOOS == "windows" {
			shellPath = "powershell.exe"
		} else {
			shellPath = "/bin/sh"
		}
	}

	vars := EnvVars(user)
	env := os.Environ()
	for k, v := range vars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	ui.Banner(fmt.Sprintf("ISOLATED SHELL: %s", user.Name))
	fmt.Printf("  Identity : %s <%s>\n", user.Name, user.Email)
	if user.SSHKey != "" {
		fmt.Printf("  SSH Key  : %s\n", user.SSHKey)
	}
	fmt.Println()
	ui.Info("All Git commands in this subshell will use this identity.")
	ui.Info("Type 'exit' to leave this session and return to your default environment.")
	fmt.Println()

	cmd := exec.Command(shellPath)
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

	ui.Success(fmt.Sprintf("Exited isolated session for %q. Returned to default profile.", user.Name))
	return nil
}
