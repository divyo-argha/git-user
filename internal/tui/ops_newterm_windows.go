//go:build windows

package tui

import (
	"fmt"
	"os/exec"
	"syscall"
)

const (
	detachedProcess       = 0x00000008 // DETACHED_PROCESS
	createNewProcessGroup = 0x00000200 // CREATE_NEW_PROCESS_GROUP
)

// spawnIdentityTerminal opens a new console window running an isolated shell
// for the given identity — Windows Terminal if available (forcing a new
// window with -w -1, since a plain new-tab would land inside whatever
// window is already focused), otherwise a plain console via `start`.
func spawnIdentityTerminal(exePath, name string) error {
	if wt, err := exec.LookPath("wt.exe"); err == nil {
		cmd := exec.Command(wt, "-w", "-1", exePath, "shell", name)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess | createNewProcessGroup}
		if err := cmd.Start(); err == nil {
			return nil
		}
	}

	cmd := exec.Command("cmd", "/c", "start", "git-user: "+name, exePath, "shell", name)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess | createNewProcessGroup}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not open a new console window: %w", err)
	}
	return nil
}
