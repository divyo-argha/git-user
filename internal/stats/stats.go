package stats

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

type AuthorStat struct {
	DisplayName    string
	Email          string
	Commits        int
	VerifiedUser   *config.User
	NameVariations []string
}

// AuditRepository audits commit author identities in the git repository.
// Optional targetPath specifies a subdirectory or file path to constrain git log.
func AuditRepository(store *config.Store, targetPath string) ([]AuthorStat, error) {
	if !git.IsInRepo() {
		return nil, fmt.Errorf("not in a git repository")
	}

	args := []string{"log", "--all", "--use-mailmap", "--format=%an|%ae"}
	if targetPath != "" {
		args = append(args, "--", targetPath)
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve git log: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	type emailGroup struct {
		email       string
		nameCounts  map[string]int
		commits     int
		matchedUser *config.User
	}

	groups := make(map[string]*emailGroup)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		parts := strings.SplitN(trimmed, "|", 2)
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
		if name != "" {
			grp.nameCounts[name]++
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

		results = append(results, AuthorStat{
			DisplayName:    displayName,
			Email:          grp.email,
			Commits:        grp.commits,
			VerifiedUser:   grp.matchedUser,
			NameVariations: names,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Commits > results[j].Commits
	})

	return results, nil
}
