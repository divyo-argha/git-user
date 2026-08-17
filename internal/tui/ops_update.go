package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// opUpdate attempts to self-update the git-user binary in-TUI.
// It looks for the git-user binary on PATH and re-runs it with "update".
// The output is captured and returned as a Report so nothing leaks to the
// terminal while the alt-screen is active.
func opUpdate() (opResult, error) {
	selfPath, err := exec.LookPath("git-user")
	if err != nil {
		// Try the running executable as fallback.
		selfPath, err = os.Executable()
		if err != nil {
			return opResult{}, fmt.Errorf("could not locate git-user binary: %v", err)
		}
	}

	out, err := runCaptured("", selfPath, "update")
	cleanOut := stripAnsi(strings.TrimSpace(out))

	if err != nil {
		if strings.Contains(strings.ToLower(cleanOut), "sudo") || strings.Contains(strings.ToLower(err.Error()), "sudo") {
			msg := "Updating git-user requires administrator (sudo) privileges.\n\n" +
				"Please exit git-user and run from your terminal:\n" +
				"  sudo git-user --update\n\n" +
				"Or reinstall using the installer:\n" +
				"  curl -fsSL https://raw.githubusercontent.com/divyo-argha/git-user/main/install.sh | bash"
			return opResult{detail: msg, showReport: true}, nil
		}
		if cleanOut != "" {
			return opResult{}, fmt.Errorf("%s\n%s", err.Error(), cleanOut)
		}
		return opResult{}, fmt.Errorf("update failed: %v", err)
	}

	if cleanOut == "" {
		cleanOut = "Update complete. Restart git-user to use the new version."
	}
	return opResult{detail: cleanOut, showReport: true}, nil
}
