package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/stats"
	"github.com/divyo-argha/git-user/internal/ui"
)

const defaultVerifyDepth = 50

func resolveDefaultRange() string {
	n := fmt.Sprintf("HEAD~%d", defaultVerifyDepth)
	if err := exec.Command("git", "rev-parse", "--verify", "--quiet", n).Run(); err != nil {
		return ""
	}
	return n + "..HEAD"
}

func runVerify(args []string) error {
	if !git.IsInRepo() {
		ui.Error("Not in a git repository. Run `git-user verify` within a git repository.")
		return fmt.Errorf("not in repository")
	}

	revRange := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--range", "-r":
			if i+1 < len(args) {
				revRange = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") && revRange == "" {
				revRange = args[i]
			}
		}
	}
	if revRange == "" {
		revRange = resolveDefaultRange()
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	displayRange := revRange
	if displayRange == "" {
		displayRange = "full history"
	}

	jsonOutput := ui.IsJSONOutput(args)
	if !jsonOutput {
		ui.Banner("COMMIT SIGNATURE VERIFICATION")
		ui.Info(fmt.Sprintf("Verifying commit signatures for range: %s", displayRange))
		fmt.Println()
	}

	authorStats, err := stats.VerifyRange(store, revRange)
	if err != nil {
		if !jsonOutput {
			ui.Errorf("Failed to verify commit range: %v", err)
		}
		return err
	}

	totalNotSigned := 0
	for _, s := range authorStats {
		_, notSigned := formatSignatureStatus(s)
		totalNotSigned += notSigned
	}

	if jsonOutput {
		_ = json.NewEncoder(os.Stdout).Encode(struct {
			Range         string             `json:"range"`
			Authors       []stats.AuthorStat `json:"authors"`
			TotalUnsigned int                `json:"total_unsigned"`
		}{
			Range:         displayRange,
			Authors:       authorStats,
			TotalUnsigned: totalNotSigned,
		})
		if totalNotSigned > 0 {
			return fmt.Errorf("unsigned or invalid commits found in range")
		}
		return nil
	}

	if len(authorStats) == 0 {
		ui.Info("No commits found in this range.")
		return nil
	}

	for _, s := range authorStats {
		sigStr, _ := formatSignatureStatus(s)
		fmt.Printf("  %-25s  %-30s  Commits: %-5d  Signature: %s\n", s.DisplayName, fmt.Sprintf("<%s>", s.Email), s.Commits, sigStr)
	}

	fmt.Println()
	ui.Divider()
	fmt.Println()

	if totalNotSigned > 0 {
		ui.Warn(fmt.Sprintf("%d commit(s) in range %s are unsigned or carry an invalid/revoked/unverifiable signature.", totalNotSigned, displayRange))
		return fmt.Errorf("unsigned or invalid commits found in range")
	}

	ui.Success(fmt.Sprintf("All commits in range %s carry a valid, currently-trusted cryptographic signature.", displayRange))
	return nil
}
