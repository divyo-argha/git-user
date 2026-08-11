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

	hasUnsignedCommits := false
	hasUnregisteredAuthors := false

	for _, s := range authorStats {
		notSigned := s.UnsignedCommits + s.RevokedSignatureCommits + s.BadSignatureCommits + s.UnverifiableCommits

		var sigStr string
		switch {
		case s.SignedCommits > 0 && notSigned == 0:
			sigStr = fmt.Sprintf("\033[1;32mSigned (%d/%d cryptographically signed)\033[0m", s.SignedCommits, s.Commits)
		case s.SignedCommits > 0:
			sigStr = fmt.Sprintf("\033[1;33mPartially Signed (\033[1;32m%d Signed\033[1;33m, \033[1;31m%d Not Signed\033[1;33m)\033[0m", s.SignedCommits, notSigned)
			hasUnsignedCommits = true
		default:
			sigStr = fmt.Sprintf("\033[1;31mNot Signed (0/%d cryptographically signed)\033[0m", s.Commits)
			hasUnsignedCommits = true
		}
		if s.RevokedSignatureCommits > 0 {
			sigStr += fmt.Sprintf(" \033[1;31m[%d signed by a revoked key]\033[0m", s.RevokedSignatureCommits)
		}
		if s.BadSignatureCommits > 0 {
			sigStr += fmt.Sprintf(" \033[1;31m[%d invalid signature]\033[0m", s.BadSignatureCommits)
		}
		if s.UnverifiableCommits > 0 {
			sigStr += fmt.Sprintf(" \033[1;33m[%d unverifiable locally]\033[0m", s.UnverifiableCommits)
		}

		identityStr := "\033[1;31mUnregistered identity\033[0m"
		if s.IsRegisteredIdentity() {
			identityStr = fmt.Sprintf("\033[1;32mRegistered (%s)\033[0m", s.VerifiedUser.Name)
		} else {
			hasUnregisteredAuthors = true
		}

		if sortMode == stats.SortByLines {
			linesStr := fmt.Sprintf("+%d / -%d (Net: %+d)", s.CodeLinesAdded, s.CodeLinesDeleted, s.NetCodeLines)
			fmt.Printf("  %-25s  %-30s  Code Lines: %-22s  Signature: %-s  Identity: %s\n", s.DisplayName, fmt.Sprintf("<%s>", s.Email), linesStr, sigStr, identityStr)
		} else {
			fmt.Printf("  %-25s  %-30s  Commits: %-5d  Signature: %-s  Identity: %s\n", s.DisplayName, fmt.Sprintf("<%s>", s.Email), s.Commits, sigStr, identityStr)
		}
	}

	fmt.Println()
	ui.Divider()
	fmt.Println()

	if hasUnsignedCommits {
		ui.Warn("Some commits are not cryptographically signed (or carry a signature that is invalid, revoked, or unverifiable locally — git could not check it without the signer's public key / allowedSignersFile).")
	} else {
		ui.Success("All commits in repository history carry a valid, currently-trusted cryptographic signature (per git's own signature check).")
	}

	if hasUnregisteredAuthors {
		ui.Warn("Some commit authors do not match any identity registered in git-user. This is independent of signing status.")
	}

	return nil
}
