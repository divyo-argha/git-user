package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/gitenv"
	"github.com/divyo-argha/git-user/internal/tui/core"
)

// openIdentityShellCmd suspends the TUI, hands the real terminal to a fresh
// subshell scoped to the given identity's Git environment, and resumes the
// TUI when the subshell exits. Stdin/Stdout are deliberately left unset on
// the *exec.Cmd — tea.ExecProcess wires them to the program's own I/O as
// part of the suspend/resume handoff.
func openIdentityShellCmd(name string, user *config.User) tea.Cmd {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		if runtime.GOOS == "windows" {
			shellPath = "powershell.exe"
		} else {
			shellPath = "/bin/sh"
		}
	}

	vars := gitenv.Vars(user)
	env := os.Environ()
	for k, v := range vars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.Command(shellPath)
	cmd.Env = env

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return core.ShellSessionEndedMsg{Name: name, Err: err}
	})
}
