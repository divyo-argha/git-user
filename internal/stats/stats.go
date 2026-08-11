package stats

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

type SortMode string

const (
	SortByCommits SortMode = "commits"
	SortByLines   SortMode = "lines"
)

type AuthorStat struct {
	DisplayName            string
	Email                  string
	Commits                int
	VerifiedCommits        int
	UnregisteredCommits    int
	CodeLinesAdded         int
	CodeLinesDeleted       int
	NetCodeLines           int
	VerifiedLinesAdded     int
	VerifiedLinesDeleted   int
	UnregisteredLinesAdded int
	UnregisteredLinesDel   int
	TotalLines             int
	VerifiedUser           *config.User
	NameVariations         []string
}

// AuditRepository audits commit author identities in the git repository (defaults to sorting by commits).
func AuditRepository(store *config.Store, targetPath string) ([]AuthorStat, error) {
	return AuditRepositoryMode(store, targetPath, SortByCommits)
}

// AuditRepositoryMode audits commit author identities and calculates pure code line changes (excluding comments/blank lines).
func AuditRepositoryMode(store *config.Store, targetPath string, mode SortMode) ([]AuthorStat, error) {
	if !git.IsInRepo() {
		return nil, fmt.Errorf("not in a git repository")
	}

	args := []string{"log", "--all", "--use-mailmap", "-p", "-U0", "--format=COMMIT|%an|%ae"}
	if targetPath != "" {
		args = append(args, "--", targetPath)
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve git log: %w", err)
	}

	type emailGroup struct {
		email                string
		nameCounts           map[string]int
		commits              int
		verifiedCommits      int
		unregisteredCommits  int
		linesAdded           int
		linesDeleted         int
		verifiedLinesAdded   int
		verifiedLinesDeleted int
		unregisteredLinesAdd int
		unregisteredLinesDel int
		matchedUser          *config.User
	}

	groups := make(map[string]*emailGroup)

	var currentGroup *emailGroup
	var isCurrentCommitVerified bool
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "COMMIT|") {
			header := strings.TrimPrefix(line, "COMMIT|")
			parts := strings.SplitN(header, "|", 2)
			name := strings.TrimSpace(parts[0])
			email := ""
			if len(parts) > 1 {
				email = strings.TrimSpace(parts[1])
			}

			normEmail := strings.ToLower(email)
			if normEmail == "" {
				normEmail = "unknown"
			}

			var matched *config.User
			if store != nil {
				matched = store.FindUserByEmail(normEmail)
			}

			groupID := normEmail
			if matched != nil {
				groupID = "user:" + strings.ToLower(matched.Name)
			}

			grp, exists := groups[groupID]
			if !exists {
				primaryEmail := email
				if matched != nil && matched.Email != "" {
					primaryEmail = matched.Email
				}
				grp = &emailGroup{
					email:       primaryEmail,
					nameCounts:  make(map[string]int),
					matchedUser: matched,
				}
				groups[groupID] = grp
			}

			grp.commits++
			if matched != nil {
				grp.verifiedCommits++
				isCurrentCommitVerified = true
			} else {
				grp.unregisteredCommits++
				isCurrentCommitVerified = false
			}

			if name != "" {
				grp.nameCounts[name]++
			}
			currentGroup = grp
			continue
		}

		if currentGroup == nil {
			continue
		}

		// Skip diff header metadata
		if strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") ||
			strings.HasPrefix(line, "@@ ") {
			continue
		}

		if strings.HasPrefix(line, "+") {
			codeText := line[1:]
			if !isCommentOrBlank(codeText) {
				currentGroup.linesAdded++
				if isCurrentCommitVerified {
					currentGroup.verifiedLinesAdded++
				} else {
					currentGroup.unregisteredLinesAdd++
				}
			}
		} else if strings.HasPrefix(line, "-") {
			codeText := line[1:]
			if !isCommentOrBlank(codeText) {
				currentGroup.linesDeleted++
				if isCurrentCommitVerified {
					currentGroup.verifiedLinesDeleted++
				} else {
					currentGroup.unregisteredLinesDel++
				}
			}
		}
	}

	var results []AuthorStat
	for _, grp := range groups {
		var names []string
		var topName string
		maxCount := -1

		for n, cnt := range grp.nameCounts {
			names = append(names, n)
			if cnt > maxCount {
				maxCount = cnt
				topName = n
			}
		}

		sort.Strings(names)

		displayName := topName
		if grp.matchedUser != nil {
			displayName = grp.matchedUser.Name
		}
		if displayName == "" {
			displayName = grp.email
		}

		netLines := grp.linesAdded - grp.linesDeleted
		totalLines := grp.linesAdded + grp.linesDeleted

		results = append(results, AuthorStat{
			DisplayName:            displayName,
			Email:                  grp.email,
			Commits:                grp.commits,
			VerifiedCommits:        grp.verifiedCommits,
			UnregisteredCommits:    grp.unregisteredCommits,
			CodeLinesAdded:         grp.linesAdded,
			CodeLinesDeleted:       grp.linesDeleted,
			NetCodeLines:           netLines,
			VerifiedLinesAdded:     grp.verifiedLinesAdded,
			VerifiedLinesDeleted:   grp.verifiedLinesDeleted,
			UnregisteredLinesAdded: grp.unregisteredLinesAdd,
			UnregisteredLinesDel:   grp.unregisteredLinesDel,
			TotalLines:             totalLines,
			VerifiedUser:           grp.matchedUser,
			NameVariations:         names,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if mode == SortByLines {
			if results[i].NetCodeLines != results[j].NetCodeLines {
				return results[i].NetCodeLines > results[j].NetCodeLines
			}
		}
		return results[i].Commits > results[j].Commits
	})

	return results, nil
}

// isCommentOrBlank checks if a line of code is blank or a comment in common languages (Python, Go, JS, TS, C, Shell, HTML, SQL, etc.).
func isCommentOrBlank(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}

	// Single-line Python / Shell / YAML / Ruby comments
	if strings.HasPrefix(trimmed, "#") {
		return true
	}

	// Python multiline docstrings or strings used as comments
	if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
		return true
	}

	// C-style / Go / Java / JS / TS / Rust / C# comments
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*/") || strings.HasPrefix(trimmed, "*") {
		return true
	}

	// HTML / XML comments
	if strings.HasPrefix(trimmed, "<!--") || strings.HasPrefix(trimmed, "-->") {
		return true
	}

	// SQL / Lua / Haskell comments
	if strings.HasPrefix(trimmed, "--") {
		return true
	}

	return false
}
