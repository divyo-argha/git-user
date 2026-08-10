package cli

import (
	"fmt"

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

	targetPath := ""
	if len(args) > 0 {
		targetPath = args[0]
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

	authorStats, err := stats.AuditRepository(store, targetPath)
	if err != nil {
		ui.Errorf("Failed to audit repository: %v", err)
		return err
	}

	if len(authorStats) == 0 {
		ui.Info("No commits found in this repository.")
		return nil
	}

	ui.Header("Commit Authors Summary")
	fmt.Println()

	hasUnregistered := false

	for _, s := range authorStats {
		statusStr := ""
		if s.VerifiedUser != nil {
			statusStr = fmt.Sprintf("\033[1;32mVerified (%s)\033[0m", s.VerifiedUser.Name)
		} else {
			statusStr = "\033[1;31mUnregistered (potential identity leak!)\033[0m"
			hasUnregistered = true
		}

		fmt.Printf("  %-25s  %-30s  Commits: %-5d  Status: %s\n", s.DisplayName, fmt.Sprintf("<%s>", s.Email), s.Commits, statusStr)
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
