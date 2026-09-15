package cli

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/hookops"
	"github.com/divyo-argha/git-user/internal/policyops"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runHook(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user hook <install|uninstall|check>")
		fmt.Println()
		ui.Info("Git hooks help prevent commits with the wrong identity")
		fmt.Println()
		fmt.Println("  install   - Install pre-commit/pre-push/post-merge hooks in current repo")
		fmt.Println("  uninstall - Remove git-user's hooks from current repo")
		fmt.Println("  check     - Verify current identity (used by hook)")
		return fmt.Errorf("missing subcommand")
	}

	subcommand := args[0]

	switch subcommand {
	case "install":
		return installHook()
	case "uninstall":
		return uninstallHook()
	case "check":
		return checkIdentity(args[1:])
	default:
		ui.Errorf("unknown hook subcommand: %s", subcommand)
		return fmt.Errorf("unknown subcommand")
	}
}

func cliConfirmOverwriteHook(spec hookops.HookSpec, existing string) bool {
	ui.Warn(fmt.Sprintf("%s hook already exists", spec.Name))
	ui.Info("Current hook content:")
	fmt.Println(existing)
	fmt.Println()
	return ui.Confirm(fmt.Sprintf("Overwrite %s?", spec.Name), false)
}

func installHook() error {
	if !git.IsInRepo() {
		ui.Error("Not in a git repository")
		return fmt.Errorf("not in repo")
	}

	results, err := hookops.Install(cliConfirmOverwriteHook)
	if err != nil {
		ui.Errorf("%v", err)
		return err
	}

	installed := 0
	for _, r := range results {
		switch r.Outcome {
		case hookops.OutcomeAlreadyInstalled:
			ui.Info(fmt.Sprintf("%s hook already installed by git-user — skipping", r.Spec.Name))
		case hookops.OutcomeSkippedExisting:
			ui.Info(fmt.Sprintf("Skipped %s", r.Spec.Name))
		case hookops.OutcomeInstalled, hookops.OutcomeOverwritten:
			ui.Success(fmt.Sprintf("%s hook installed — verifies your identity %s", r.Spec.Name, r.Spec.Comment))
			installed++
		}
	}

	if installed > 0 {
		ui.Info("To remove: git-user hook uninstall")
	}

	if repoRoot, err := git.RepoRoot(); err == nil {
		if _, statErr := os.Stat(repoRoot + "/" + config.AllowedSignersFileName); statErr == nil {
			suggestAllowedSignersConfig(repoRoot)
		}
	}

	return nil
}

func uninstallHook() error {
	if !git.IsInRepo() {
		ui.Error("Not in a git repository")
		return fmt.Errorf("not in repo")
	}

	results, err := hookops.Uninstall()
	if err != nil {
		ui.Errorf("%v", err)
		return err
	}

	removed := 0
	for _, r := range results {
		switch r.Outcome {
		case hookops.OutcomeRemoved:
			ui.Success(fmt.Sprintf("%s hook removed", r.Spec.Name))
			removed++
		case hookops.OutcomeForeign:
			ui.Warn(fmt.Sprintf("%s hook exists but wasn't created by git-user", r.Spec.Name))
			ui.Info("Remove manually if needed: " + r.Existing)
		}
	}

	if removed == 0 {
		ui.Info("No git-user hooks found")
	}

	return nil
}

func checkIdentity(hookArgs []string) error {
	hookName := ""
	if len(hookArgs) > 0 {
		hookName = hookArgs[0]
	}

	store, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "✖ Failed to load git-user config: %v\n", err)
		return err
	}

	if hookName == "post-merge" {
		for _, s := range hookops.WarnOnUnsignedMerge(store) {
			notSigned := s.UnsignedCommits + s.RevokedSignatureCommits + s.BadSignatureCommits + s.UnverifiableCommits
			fmt.Fprintf(os.Stderr, "⚠ Just merged in %d unsigned/untrusted commit(s) from %s <%s>\n", notSigned, s.DisplayName, s.Email)
		}
		return nil
	}

	if store.Current == "" {
		fmt.Fprintf(os.Stderr, "✖ No active git-user identity\n")
		fmt.Fprintf(os.Stderr, "  Run: git-user switch <name>\n")
		return fmt.Errorf("no active identity")
	}

	user := store.FindUser(store.Current)
	if user == nil {
		fmt.Fprintf(os.Stderr, "✖ Active identity %q not found\n", store.Current)
		return fmt.Errorf("identity not found")
	}

	err = hookops.CheckIdentity(store)
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *hookops.IdentityMismatchError:
		fmt.Fprintf(os.Stderr, "✖ Identity mismatch!\n")
		fmt.Fprintf(os.Stderr, "  Expected: %s (%s)\n", e.ExpectedName, e.Expected)
		fmt.Fprintf(os.Stderr, "  Git config: %s\n", e.Got)
		fmt.Fprintf(os.Stderr, "  Run: git-user switch %s\n", e.ExpectedName)
		return fmt.Errorf("identity mismatch")
	case *hookops.PolicyViolation:
		fmt.Fprintf(os.Stderr, "✖ %s\n", e.Message)
		fmt.Fprintf(os.Stderr, "  %s\n", e.FixHint)
		return err
	default:
		fmt.Fprintf(os.Stderr, "✖ %v\n", err)
		return err
	}
}

func suggestAllowedSignersConfig(repoRoot string) {
	changed, err := policyops.WireAllowedSignersConfig(repoRoot)
	if err != nil {
		ui.Warn(fmt.Sprintf("Could not set gpg.ssh.allowedSignersFile: %v", err))
		return
	}
	if changed {
		ui.Info(fmt.Sprintf("Set local git config to use %s for signature verification", config.AllowedSignersFileName))
	}
}
