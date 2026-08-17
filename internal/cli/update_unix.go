//go:build !windows

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/divyo-argha/git-user/internal/ui"
)

// installBinary replaces the installed binary on macOS and Linux.
//
// Renaming over the currently-running executable is safe on both platforms
// (the OS keeps the old inode alive for the running process), so the swap
// works even while git-user is executing.
//
// When the install directory is not writable by this user (e.g. a sudo
// install into /usr/local/bin), the replacement is promoted with sudo.
func installBinary(execPath, newBinary string) (string, error) {
	if filepath.Dir(newBinary) == filepath.Dir(execPath) {
		// Temp file was created next to the binary, so the directory is
		// writable: swap in place without sudo.
		backupPath := execPath + ".bak"
		if err := os.Rename(execPath, backupPath); err != nil {
			return "", fmt.Errorf("backing up current binary: %w", err)
		}
		if err := os.Rename(newBinary, execPath); err != nil {
			// Rollback
			os.Rename(backupPath, execPath)
			return "", fmt.Errorf("installing new binary: %w", err)
		}
		os.Remove(backupPath)
		return "", nil
	}

	// Install directory requires root: promote the swap with sudo.
	ui.Info(fmt.Sprintf("Updating %s requires administrator (sudo) privileges.", execPath))

	// Shell script performing the swap atomically with rollback.
	// Positional params: $1 = installed binary, $2 = new binary.
	script := `if mv -- "$1" "$1.bak"; then
  if mv -- "$2" "$1"; then
    rm -f -- "$1.bak"
    exit 0
  fi
  mv -f -- "$1.bak" "$1"
  exit 1
fi
exit 1`

	cmd := exec.Command("sudo", "sh", "-c", script, "git-user-update", execPath, newBinary)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("update requires sudo and it failed — run 'git-user --update' from your terminal: %w", err)
	}
	return "", nil
}

// scheduleNpmUpdateWindows is only used on Windows (the running executable is
// locked there); npm updates run directly on macOS and Linux.
func scheduleNpmUpdateWindows() error {
	return fmt.Errorf("unsupported on this platform")
}
