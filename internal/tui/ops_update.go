package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/divyo-argha/git-user/internal/version"
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

	// Detect new version from updated binary or captured output, and update in-memory version.
	newVer := detectInstalledVersion(selfPath)
	if newVer == "" {
		newVer = extractUpdatedVersion(cleanOut)
	}
	if newVer != "" {
		version.SetVersion(newVer)
	}

	if cleanOut == "" {
		cleanOut = "Update complete. Restart git-user to use the new version."
	}
	return opResult{detail: cleanOut, showReport: true}, nil
}

// detectInstalledVersion attempts to query the newly installed git-user binary for its version.
func detectInstalledVersion(selfPath string) string {
	candidates := []string{selfPath}
	if p, err := exec.LookPath("git-user"); err == nil && p != selfPath {
		candidates = append(candidates, p)
	}
	if p, err := os.Executable(); err == nil && p != selfPath {
		candidates = append(candidates, p)
	}

	for _, cand := range candidates {
		if cand == "" {
			continue
		}
		if out, err := exec.Command(cand, "--version").Output(); err == nil {
			s := strings.TrimSpace(string(out))
			s = strings.TrimPrefix(s, "git-user")
			s = strings.TrimSpace(s)
			if isValidVersionStr(s) {
				return s
			}
		}
	}
	return ""
}

// extractUpdatedVersion parses the new version string from update output.
func extractUpdatedVersion(output string) string {
	// Look for transition arrows: "v4.8.1 ──▶ v4.8.2" or "v4.8.1 → v4.8.2" or "v4.8.1 -> v4.8.2"
	for _, sep := range []string{"──▶", "→", "->", "-->"} {
		if idx := strings.Index(output, sep); idx != -1 {
			rem := strings.TrimSpace(output[idx+len(sep):])
			fields := strings.Fields(rem)
			if len(fields) > 0 {
				ver := strings.Trim(fields[0], "(),:; \t\r\n")
				if isValidVersionStr(ver) {
					return ver
				}
			}
		}
	}

	// Look for "Download verified: git-user v4.8.2" or "Download verified: v4.8.2"
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Download verified:") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				ver := strings.Trim(fields[len(fields)-1], "(),:; \t\r\n")
				if isValidVersionStr(ver) {
					return ver
				}
			}
		}
		// Look for "already up to date!\n\n   v4.8.1" or "(latest release)"
		if strings.Contains(line, "(latest release)") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				ver := strings.Trim(fields[0], "(),:; \t\r\n")
				if isValidVersionStr(ver) {
					return ver
				}
			}
		}
	}

	return ""
}

func isValidVersionStr(s string) bool {
	if s == "" {
		return false
	}
	s = strings.TrimPrefix(strings.TrimPrefix(s, "v"), "V")
	if len(s) == 0 {
		return false
	}
	// Must start with a digit
	return s[0] >= '0' && s[0] <= '9'
}
