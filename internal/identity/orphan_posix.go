//go:build !windows

package identity

import (
	"os"
	"syscall"
)

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// On Unix systems, we can check if a process exists by sending signal 0
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 (null signal) to check if process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist or we don't have permission
		return false
	}

	return true
}
