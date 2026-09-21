package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/diagnostics"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runDoctor(args []string) error {
	fix := false
	for _, a := range args {
		if a == "--fix" || a == "-f" {
			fix = true
		}
	}

	jsonOutput := ui.IsJSONOutput(args)

	var store *config.Store
	if loaded, err := config.Load(); err == nil {
		store = loaded
	}

	report, _ := diagnostics.Run(store, diagnostics.Options{Fix: fix, VerifySSH: verifySSHConnectionWithKey, Interactive: ui.IsTTY()})

	if jsonOutput {
		var warnings []string
		for _, c := range report.Checks {
			if !c.IsProgress && c.Status == diagnostics.StatusWarn && !c.Fixed {
				warnings = append(warnings, c.Message)
			}
		}
		securityScore := ""
		if report.ScoreTotal > 0 {
			securityScore = fmt.Sprintf("%d/%d", report.ScorePassed, report.ScoreTotal)
		}
		_ = json.NewEncoder(os.Stdout).Encode(struct {
			Issues        int      `json:"issues"`
			Fixed         int      `json:"fixed"`
			SecurityScore string   `json:"security_score,omitempty"`
			Warnings      []string `json:"warnings"`
		}{
			Issues:        report.Issues,
			Fixed:         report.Fixed,
			SecurityScore: securityScore,
			Warnings:      warnings,
		})
		if report.Issues > 0 {
			return fmt.Errorf("%d issue(s) found", report.Issues)
		}
		return nil
	}

	ui.Banner("GIT-USER DIAGNOSTICS & SECURITY")
	fmt.Println()
	if fix {
		ui.Info("Running with --fix: auto-correctable issues below are fixed in place, not just reported.")
		fmt.Println()
	}

	for _, c := range report.Checks {
		if c.IsProgress {
			ui.Info(c.Message)
			continue
		}
		switch c.Status {
		case diagnostics.StatusPass:
			ui.Success(c.Message)
		case diagnostics.StatusInfo:
			ui.Info(c.Message)
		case diagnostics.StatusNotice, diagnostics.StatusWarn:
			ui.Warn(c.Message)
		}
		for _, d := range c.Detail {
			ui.Info(d)
		}
		if c.FixHint != "" {
			ui.Info("  Fix: " + c.FixHint)
		}
	}

	fmt.Println()
	ui.Divider()
	if report.ScoreTotal > 0 {
		ui.Info(fmt.Sprintf("Security Score: %d/%d", report.ScorePassed, report.ScoreTotal))
	}
	if report.Issues == 0 && report.Fixed == 0 {
		ui.Success("All checks passed! Your git-user setup is 100% healthy and secure.")
	} else if fix {
		if report.Fixed > 0 {
			ui.Success(fmt.Sprintf("Fixed %d issue(s).", report.Fixed))
		}
		if report.Issues > 0 {
			ui.Warn(fmt.Sprintf("%d issue(s) need manual attention (see warnings above).", report.Issues))
		}
	} else {
		ui.Warn(fmt.Sprintf("Found %d issue(s). See suggestions above.", report.Issues))
		ui.Info("Run 'git-user doctor --fix' to automatically correct what can be fixed.")
	}

	return nil
}
