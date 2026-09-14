package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runPolicy(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user policy <init|show>")
		return fmt.Errorf("missing subcommand")
	}

	switch args[0] {
	case "init":
		return runPolicyInit(args[1:])
	case "show":
		return runPolicyShow()
	default:
		ui.Errorf("unknown policy subcommand: %s", args[0])
		return fmt.Errorf("unknown subcommand")
	}
}

func runPolicyInit(args []string) error {
	repoRoot, err := git.RepoRoot()
	if err != nil {
		ui.Error("Not in a git repository")
		return fmt.Errorf("not in repo")
	}
	policyPath := filepath.Join(repoRoot, config.RepoPolicyFileName)

	var domainsFlag string
	haveDomainsFlag := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--domains" && i+1 < len(args) {
			domainsFlag = args[i+1]
			haveDomainsFlag = true
			i++
		}
	}

	if content, statErr := os.ReadFile(policyPath); statErr == nil {
		ui.Warn(fmt.Sprintf("%s already exists:", config.RepoPolicyFileName))
		fmt.Println()
		fmt.Println(string(content))
		if !ui.Confirm("Overwrite it?", false) {
			ui.Info("Cancelled")
			return nil
		}
	}

	ui.Banner("REPOSITORY POLICY SETUP")
	fmt.Println()

	requireSigning := ui.Confirm("Require commits to be signed in this repository?", true)

	domainsInput := domainsFlag
	if !haveDomainsFlag {
		var promptErr error
		domainsInput, promptErr = ui.Prompt("Allowed email domains (comma-separated, blank to skip):")
		if promptErr != nil && !errors.Is(promptErr, ui.ErrNotInteractive) {
			return promptErr
		}
	}
	var domains []string
	for _, d := range strings.Split(domainsInput, ",") {
		d = strings.TrimSpace(strings.ToLower(d))
		if d != "" {
			domains = append(domains, d)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# git-user repository policy\n# See: git-user hook --help\n\n")
	fmt.Fprintf(&b, "require_signing=%t\n", requireSigning)
	if len(domains) > 0 {
		fmt.Fprintf(&b, "allowed_email_domains=%s\n", strings.Join(domains, ","))
	}

	if err := os.WriteFile(policyPath, []byte(b.String()), 0644); err != nil {
		ui.Errorf("writing %s: %v", config.RepoPolicyFileName, err)
		return err
	}

	ui.Success(fmt.Sprintf("Wrote %s", config.RepoPolicyFileName))

	if content, err := os.ReadFile(filepath.Join(repoRoot, ".git", "hooks", "pre-commit")); err != nil || !strings.HasPrefix(string(content), "#!/bin/sh\n# git-user") {
		ui.Info("Run 'git-user hook install' to enforce this policy locally.")
	}

	return nil
}

func runPolicyShow() error {
	repoRoot, err := git.RepoRoot()
	if err != nil {
		ui.Error("Not in a git repository")
		return fmt.Errorf("not in repo")
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, config.RepoPolicyFileName))
	if err != nil {
		if os.IsNotExist(err) {
			ui.Info(fmt.Sprintf("No %s file in this repository.", config.RepoPolicyFileName))
			ui.Info("Run 'git-user policy init' to create one.")
			return nil
		}
		ui.Errorf("reading %s: %v", config.RepoPolicyFileName, err)
		return err
	}

	fmt.Println(string(content))
	return nil
}
