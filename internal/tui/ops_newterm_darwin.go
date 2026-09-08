//go:build darwin

package tui

import (
	"fmt"
	"os/exec"
	"syscall"
)

// spawnIdentityTerminal opens a new Terminal.app window via AppleScript,
// running an isolated shell for the given identity.
func spawnIdentityTerminal(exePath, name string) error {
	script := `tell application "Terminal" to do script ` + appleScriptQuote(shellJoin(exePath, "shell", name))
	cmd := exec.Command("osascript", "-e", script, "-e", `tell application "Terminal" to activate`)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not open Terminal.app: %w", err)
	}
	return nil
}
