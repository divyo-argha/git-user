package tui

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/stats"
)

// ── Stats ─────────────────────────────────────────────────────────────────────

// opStats audits commit authors in the current repository against registered
// identities and reports potential identity leaks.
func opStats(store *config.Store) (opResult, error) {
	if !git.IsInRepo() {
		return opResult{}, fmt.Errorf("not in a git repository — run Stats inside a repository")
	}

	authorStats, err := stats.AuditRepository(store, "")
	if err != nil {
		return opResult{}, fmt.Errorf("failed to retrieve git log: %v", err)
	}

	if len(authorStats) == 0 {
		return opResult{detail: "No commits found in this repository.", showReport: true}, nil
	}

	report := "REPOSITORY IDENTITY AUDIT\n\n"
	report += "Commit Authors Summary\n\n"

	hasUnregistered := false
	for _, s := range authorStats {
		statusStr := "⚠ Unregistered (potential identity leak!)"
		if s.VerifiedUser != nil {
			statusStr = fmt.Sprintf("✓ Verified (%s)", s.VerifiedUser.Name)
		} else {
			hasUnregistered = true
		}
		report += fmt.Sprintf("  %-25s  %-30s  Commits: %-5d  %s\n", s.DisplayName, fmt.Sprintf("<%s>", s.Email), s.Commits, statusStr)
	}

	report += "\n"
	if hasUnregistered {
		report += "Unregistered authors were found in the history of this repository.\n"
		report += "If these are your commits, register the identity and re-switch.\n"
	} else {
		report += "All commit authors in history match registered identities!\n"
	}
	return opResult{detail: report, showReport: true}, nil
}
