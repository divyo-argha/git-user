package cli

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

type authorStats struct {
	Name    string
	Email   string
	Commits int
}

func runStats(args []string) error {
	if !git.IsInRepo() {
		ui.Error("Not in a git repository. Run `git-user stats` within a git repository.")
		return fmt.Errorf("not in repository")
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	ui.Banner("REPOSITORY IDENTITY AUDIT")
	fmt.Println()

	// Get git log with author details
	cmd := exec.Command("git", "log", "--all", "--format=%an <%ae>")
	out, err := cmd.Output()
	if err != nil {
		ui.Errorf("Failed to retrieve git log: %v", err)
		return err
	}

	lines := strings.Split(string(out), "\n")
	counts := make(map[string]int)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			counts[trimmed]++
		}
	}

	if len(counts) == 0 {
		ui.Info("No commits found in this repository.")
		return nil
	}

	var stats []authorStats
	for authorStr, commits := range counts {
		name := authorStr
		email := ""
		if idx := strings.Index(authorStr, "<"); idx != -1 {
			name = strings.TrimSpace(authorStr[:idx])
			email = strings.Trim(authorStr[idx:], "<>")
		}
		stats = append(stats, authorStats{
			Name:    name,
			Email:   email,
			Commits: commits,
		})
	}

	// Sort stats by commit count descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Commits > stats[j].Commits
	})

	ui.Header("Commit Authors Summary")
	fmt.Println()

	hasUnregistered := false

	for _, s := range stats {
		matchedUser := findUserByEmail(store, s.Email)
		statusStr := ""
		if matchedUser != nil {
			statusStr = fmt.Sprintf("\033[1;32mVerified (%s)\033[0m", matchedUser.Name)
		} else {
			statusStr = "\033[1;31mUnregistered (potential identity leak!)\033[0m"
			hasUnregistered = true
		}

		fmt.Printf("  %-25s  %-30s  Commits: %-5d  Status: %s\n", s.Name, fmt.Sprintf("<%s>", s.Email), s.Commits, statusStr)
	}

	fmt.Println()
	ui.Divider()
	fmt.Println()

	if hasUnregistered {
		ui.Warn("Unregistered authors were found in the history of this repository.")
		ui.Info("If these are your commits under a different identity, you can register them using:")
		ui.Info("  git-user register")
	} else {
		ui.Success("All commit authors in history match registered identities!")
	}

	return nil
}

// FindUserByEmail is a helper on config.Store to find user by email.
// Since we don't want to modify config.Store unless necessary, we can implement it locally or check if it exists.
// Wait! Let's check if config.Store already has FindUserByEmail. We saw config.go doesn't have it, but let's check.
func findUserByEmail(store *config.Store, email string) *config.User {
	for i := range store.Users {
		if store.Users[i].Email == email {
			return &store.Users[i]
		}
	}
	return nil
}
