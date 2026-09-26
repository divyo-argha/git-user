//go:build windows

package identity

import (
	"syscall"
)

// isProcessRunning checks if a process with the given PID is running on Windows
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	h, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		// If access is denied, the process exists and is running, but caller lacks query rights
		if errno, ok := err.(syscall.Errno); ok && errno == syscall.ERROR_ACCESS_DENIED {
			return true
		}
		return false
	}
	defer syscall.CloseHandle(h)

	var exitCode uint32
	if err := syscall.GetExitCodeProcess(h, &exitCode); err != nil {
		return false
	}
	// STILL_ACTIVE = 259 (0x103)
	return exitCode == 259
}
