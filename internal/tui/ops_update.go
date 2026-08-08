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
	out = strings.TrimSpace(out)

	if err != nil {
		if out != "" {
			return opResult{}, fmt.Errorf("%s\n%s", err.Error(), out)
		}
		return opResult{}, fmt.Errorf("update failed: %v", err)
	}

	if out == "" {
		out = "Update complete. Restart git-user to use the new version."
	}
	return opResult{detail: out, showReport: true}, nil
}
