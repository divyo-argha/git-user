package tui

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/shellinit"
)

// opInstallShellIntegration writes the shell-integration snippet for sh to
// the user's rc file(s) — the one actual filesystem write this whole feature
// performs, and only on explicit confirmation from a Confirm dialog. It
// reports exactly what happened to each targeted file: freshly installed, a
// legacy snippet upgraded in place, or already present.
func opInstallShellIntegration(sh shellinit.Shell, explicitShell string) (opResult, error) {
	results, err := shellinit.Install(sh, explicitShell)
	if err != nil {
		return opResult{}, err
	}

	var lines []string
	for _, r := range results {
		switch r.Status {
		case shellinit.StatusInstalled:
			lines = append(lines, fmt.Sprintf("Installed to %s", r.File))
		case shellinit.StatusUpgraded:
			lines = append(lines, fmt.Sprintf("Upgraded existing integration in %s", r.File))
		case shellinit.StatusAlready:
			lines = append(lines, fmt.Sprintf("Already installed in %s", r.File))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, "Nothing to do.")
	}
	lines = append(lines, "", "Open a new terminal (or `source` the file) to start using it.")

	return opResult{detail: strings.Join(lines, "\n"), showReport: true}, nil
}
