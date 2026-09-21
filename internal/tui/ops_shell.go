package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/gitenv"
	"github.com/divyo-argha/git-user/internal/shellinit"
	"github.com/divyo-argha/git-user/internal/tui/core"
)

// openIdentityShellCmd suspends the TUI, hands the real terminal to a fresh
// subshell scoped to the given identity's Git environment, and resumes the
// TUI when the subshell exits. Stdin/Stdout are deliberately left unset on
// the *exec.Cmd — tea.ExecProcess wires them to the program's own I/O as
// part of the suspend/resume handoff.
func openIdentityShellCmd(name string, user *config.User) tea.Cmd {
	shellPath := shellinit.ResolveShellPath()

	vars := gitenv.Vars(user)
	if strings.Contains(strings.ToLower(shellPath), "cmd") {
		vars["PROMPT"] = fmt.Sprintf("(%s) $P$G", user.Name)
	}
	env := make([]string, 0, len(vars)+len(os.Environ()))
	for k, v := range vars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	env = append(env, os.Environ()...)

	cmd := exec.Command(shellPath)
	cmd.Env = env

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return core.ShellSessionEndedMsg{Name: name, Err: err}
	})
}
