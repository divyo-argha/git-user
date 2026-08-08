package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

// ── Stats ─────────────────────────────────────────────────────────────────────

type authorStats struct {
	Name    string
	Email   string
	Commits int
}

// opStats audits commit authors in the current repository against registered
// identities and reports potential identity leaks.
func opStats(store *config.Store) (opResult, error) {
	if !git.IsInRepo() {
		return opResult{}, fmt.Errorf("not in a git repository — run Stats inside a repository")
	}

	out, err := runCaptured("", "git", "log", "--all", "--format=%an <%ae>")
	if err != nil {
		return opResult{}, fmt.Errorf("failed to retrieve git log: %v", err)
	}

	counts := make(map[string]int)
	for _, line := range strings.Split(string(out), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			counts[trimmed]++
		}
	}

	if len(counts) == 0 {
		return opResult{detail: "No commits found in this repository.", showReport: true}, nil
	}

	var stats []authorStats
	for authorStr, commits := range counts {
		name := authorStr
		email := ""
		if idx := strings.Index(authorStr, "<"); idx != -1 {
			name = strings.TrimSpace(authorStr[:idx])
			email = strings.Trim(authorStr[idx:], "<>")
		}
		stats = append(stats, authorStats{Name: name, Email: email, Commits: commits})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Commits > stats[j].Commits
	})

	report := "REPOSITORY IDENTITY AUDIT\n\n"
	report += "Commit Authors Summary\n\n"

	hasUnregistered := false
	for _, s := range stats {
		statusStr := "⚠ Unregistered (potential identity leak!)"
		if findUserByEmail(store, s.Email) != nil {
			statusStr = "✓ Verified"
			if u := findUserByEmail(store, s.Email); u != nil {
				statusStr = fmt.Sprintf("✓ Verified (%s)", u.Name)
			}
		} else {
			hasUnregistered = true
		}
		report += fmt.Sprintf("  %-25s  %-30s  Commits: %-5d  %s\n", s.Name, fmt.Sprintf("<%s>", s.Email), s.Commits, statusStr)
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

func findUserByEmail(store *config.Store, email string) *config.User {
	for i := range store.Users {
		if store.Users[i].Email == email {
			return &store.Users[i]
		}
	}
	return nil
}
