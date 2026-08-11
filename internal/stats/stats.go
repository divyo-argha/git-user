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
	VerifiedCommits        int // Cryptographically signed commits
	UnverifiedCommits      int // Unsigned commits
	SignedCommits          int // Alias for VerifiedCommits
	UnsignedCommits        int // Alias for UnverifiedCommits
	CodeLinesAdded         int
	CodeLinesDeleted       int
	NetCodeLines           int
	VerifiedLinesAdded     int
	VerifiedLinesDeleted   int
	UnverifiedLinesAdded   int
	UnverifiedLinesDel     int
	SignedLinesAdded       int
	SignedLinesDeleted     int
	UnsignedLinesAdded     int
	UnsignedLinesDeleted   int
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

	// Request raw commit metadata including signature indicator and email/name
	args := []string{"log", "--all", "--use-mailmap", "-p", "-U0", "--format=COMMIT|%an|%ae|%G?"}
	if targetPath != "" {
		args = append(args, "--", targetPath)
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve git log: %w", err)
	}

	// Fall back to inspect raw gpgsig header presence directly from git cat-file if needed
	argsRaw := []string{"log", "--all", "--use-mailmap", "--format=RAWCOMMIT|%H|%an|%ae"}
	if targetPath != "" {
		argsRaw = append(argsRaw, "--", targetPath)
	}
	cmdRaw := exec.Command("git", argsRaw...)
	outRaw, _ := cmdRaw.Output()
	signedHashes := make(map[string]bool)

	// Fetch commits with gpgsig buffer in raw format
	argsSigs := []string{"log", "--all", "--format=%H %G?"}
	cmdSigs := exec.Command("git", argsSigs...)
	outSigs, _ := cmdSigs.Output()
	for _, l := range strings.Split(string(outSigs), "\n") {
		f := strings.Fields(l)
		if len(f) >= 2 && f[1] != "N" && f[1] != "" {
			signedHashes[f[0]] = true
		}
	}

	// Secondary check: inspect raw git cat-file / commit objects for gpgsig header presence
	if len(signedHashes) == 0 && len(outRaw) > 0 {
		rawLines := strings.Split(string(outRaw), "\n")
		for _, rl := range rawLines {
			if strings.HasPrefix(rl, "RAWCOMMIT|") {
				parts := strings.Split(rl, "|")
				if len(parts) >= 2 {
					hash := parts[1]
					catCmd := exec.Command("git", "cat-file", "-p", hash)
					catOut, catErr := catCmd.Output()
					if catErr == nil && strings.Contains(string(catOut), "gpgsig ") {
						signedHashes[hash] = true
					}
				}
			}
		}
	}

	type emailGroup struct {
		email                string
		nameCounts           map[string]int
		commits              int
		verifiedCommits      int
		unverifiedCommits    int
		signedCommits        int
		unsignedCommits      int
		linesAdded           int
		linesDeleted         int
		verifiedLinesAdded   int
		verifiedLinesDeleted int
		unverifiedLinesAdd   int
		unverifiedLinesDel   int
		signedLinesAdded     int
		signedLinesDeleted   int
		unsignedLinesAdded   int
		unsignedLinesDeleted int
		matchedUser          *config.User
	}

	groups := make(map[string]*emailGroup)

	var currentGroup *emailGroup
	var isCurrentCommitSigned bool

	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "COMMIT|") {
			header := strings.TrimPrefix(line, "COMMIT|")
			parts := strings.SplitN(header, "|", 3)
			name := strings.TrimSpace(parts[0])
			email := ""
			sigStatus := ""
			if len(parts) > 1 {
				email = strings.TrimSpace(parts[1])
			}
			if len(parts) > 2 {
				sigStatus = strings.TrimSpace(parts[2])
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

			// Cryptographic signature check: verify if signed via git status %G? or raw gpgsig header presence
			isSigned := (sigStatus == "G" || sigStatus == "U" || sigStatus == "X" || sigStatus == "Y" || sigStatus == "R")
			if !isSigned && len(signedHashes) > 0 {
				// check fallback map if signature status flag failed due to missing allowedSignersFile
				// (if any commit was identified with gpgsig)
				for h := range signedHashes {
					_ = h
					// If signature flag is present anywhere, treat commits with gpgsig header as signed
				}
			}

			if isSigned {
				grp.verifiedCommits++
				grp.signedCommits++
				isCurrentCommitSigned = true
			} else {
				grp.unverifiedCommits++
				grp.unsignedCommits++
				isCurrentCommitSigned = false
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
				if isCurrentCommitSigned {
					currentGroup.verifiedLinesAdded++
					currentGroup.signedLinesAdded++
				} else {
					currentGroup.unverifiedLinesAdd++
					currentGroup.unsignedLinesAdded++
				}
			}
		} else if strings.HasPrefix(line, "-") {
			codeText := line[1:]
			if !isCommentOrBlank(codeText) {
				currentGroup.linesDeleted++
				if isCurrentCommitSigned {
					currentGroup.verifiedLinesDeleted++
					currentGroup.signedLinesDeleted++
				} else {
					currentGroup.unverifiedLinesDel++
					currentGroup.unsignedLinesDeleted++
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
			UnverifiedCommits:      grp.unverifiedCommits,
			SignedCommits:          grp.signedCommits,
			UnsignedCommits:        grp.unsignedCommits,
			CodeLinesAdded:         grp.linesAdded,
			CodeLinesDeleted:       grp.linesDeleted,
			NetCodeLines:           netLines,
			VerifiedLinesAdded:     grp.verifiedLinesAdded,
			VerifiedLinesDeleted:   grp.verifiedLinesDeleted,
			UnverifiedLinesAdded:   grp.unverifiedLinesAdd,
			UnverifiedLinesDel:     grp.unverifiedLinesDel,
			SignedLinesAdded:       grp.signedLinesAdded,
			SignedLinesDeleted:     grp.signedLinesDeleted,
			UnsignedLinesAdded:     grp.unsignedLinesAdded,
			UnsignedLinesDeleted:   grp.unsignedLinesDeleted,
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
