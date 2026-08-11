package cli

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/stats"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runStats(args []string) error {
	if !git.IsInRepo() {
		ui.Error("Not in a git repository. Run `git-user stats` within a git repository.")
		return fmt.Errorf("not in repository")
	}

	sortMode := stats.SortByCommits
	targetPath := ""

	for _, arg := range args {
		if arg == "--by=lines" || arg == "--lines" || arg == "-l" {
			sortMode = stats.SortByLines
		} else if arg == "--by=commits" {
			sortMode = stats.SortByCommits
		} else if !strings.HasPrefix(arg, "-") {
			targetPath = arg
		}
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	ui.Banner("REPOSITORY IDENTITY AUDIT")
	if targetPath != "" {
		ui.Info(fmt.Sprintf("Auditing commit history for path: %s", targetPath))
	}
	fmt.Println()

	authorStats, err := stats.AuditRepositoryMode(store, targetPath, sortMode)
	if err != nil {
		ui.Errorf("Failed to audit repository: %v", err)
		return err
	}

	if len(authorStats) == 0 {
		ui.Info("No commits found in this repository.")
		return nil
	}

	if sortMode == stats.SortByLines {
		ui.Header("Commit Authors Summary (Sorted by Code Lines Changed)")
	} else {
		ui.Header("Commit Authors Summary")
	}
	fmt.Println()

	hasUnverified := false

	for _, s := range authorStats {
		statusStr := ""
		if s.UnverifiedCommits > 0 && s.VerifiedCommits > 0 {
			statusStr = fmt.Sprintf("\033[1;33mPartially Verified (\033[1;32m%d Verified\033[1;33m, \033[1;31m%d Unverified\033[1;33m)\033[0m", s.VerifiedCommits, s.UnverifiedCommits)
			hasUnverified = true
		} else if s.UnverifiedCommits > 0 {
			statusStr = "\033[1;31mUnverified (Unsigned)\033[0m"
			hasUnverified = true
		} else {
			statusStr = "\033[1;32mVerified (Cryptographically Signed)\033[0m"
		}

		if sortMode == stats.SortByLines {
			linesStr := fmt.Sprintf("+%d / -%d (Net: %+d)", s.CodeLinesAdded, s.CodeLinesDeleted, s.NetCodeLines)
			fmt.Printf("  %-25s  %-30s  Code Lines: %-22s  Status: %s\n", s.DisplayName, fmt.Sprintf("<%s>", s.Email), linesStr, statusStr)
		} else {
			fmt.Printf("  %-25s  %-30s  Commits: %-5d  Status: %s\n", s.DisplayName, fmt.Sprintf("<%s>", s.Email), s.Commits, statusStr)
		}
	}

	fmt.Println()
	ui.Divider()
	fmt.Println()

	if hasUnverified {
		ui.Warn("Unverified (unsigned) commits were found in the history of this repository.")
	} else {
		ui.Success("All commits in repository history are cryptographically verified!")
	}

	return nil
}
