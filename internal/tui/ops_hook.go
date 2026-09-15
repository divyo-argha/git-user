package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/hookops"
	"github.com/divyo-argha/git-user/internal/policyops"
)

// ── Hooks ─────────────────────────────────────────────────────────────────────

// opHook installs, uninstalls, or checks git-user's hooks (pre-commit,
// pre-push, post-merge), backed by internal/hookops — the same package
// internal/cli/hook.go uses, so the TUI and CLI can never again drift on
// what "installed" means (this closes the gap where the TUI's own bespoke
// version only ever managed a bare pre-commit hook with no policy
// enforcement or allowed-signers wiring).
func opHook(action string) (opResult, error) {
	switch action {
	case "install":
		return opHookInstall()
	case "uninstall":
		return opHookUninstall()
	case "check":
		return opHookCheck()
	default:
		return opResult{}, fmt.Errorf("unknown hook action %q", action)
	}
}

func opHookInstall() (opResult, error) {
	if !git.IsInRepo() {
		return opResult{}, fmt.Errorf("not in a git repository")
	}

	// v1 TUI policy: never silently overwrite a foreign (non-git-user) hook —
	// skip it and report the conflict, since there's no interactive
	// ui.Confirm-style prompt mid-flow the way the CLI has.
	results, err := hookops.Install(nil)
	if err != nil {
		return opResult{}, err
	}

	var report strings.Builder
	for _, r := range results {
		switch r.Outcome {
		case hookops.OutcomeInstalled, hookops.OutcomeOverwritten:
			fmt.Fprintf(&report, "%s hook installed — verifies your identity %s\n", r.Spec.Name, r.Spec.Comment)
		case hookops.OutcomeAlreadyInstalled:
			fmt.Fprintf(&report, "%s hook already installed by git-user — skipping\n", r.Spec.Name)
		case hookops.OutcomeSkippedExisting:
			fmt.Fprintf(&report, "%s hook exists and wasn't created by git-user — skipped. Remove it manually if you want git-user to manage it.\n", r.Spec.Name)
		}
	}

	if repoRoot, err := git.RepoRoot(); err == nil {
		if _, statErr := os.Stat(filepath.Join(repoRoot, config.AllowedSignersFileName)); statErr == nil {
			if changed, _ := policyops.WireAllowedSignersConfig(repoRoot); changed {
				report.WriteString("Set local git config to use .allowed-signers for signature verification\n")
			}
		}
	}

	return opResult{detail: report.String()}, nil
}

func opHookUninstall() (opResult, error) {
	if !git.IsInRepo() {
		return opResult{}, fmt.Errorf("not in a git repository")
	}

	results, err := hookops.Uninstall()
	if err != nil {
		return opResult{}, err
	}

	var report strings.Builder
	removed := 0
	for _, r := range results {
		switch r.Outcome {
		case hookops.OutcomeRemoved:
			fmt.Fprintf(&report, "%s hook removed\n", r.Spec.Name)
			removed++
		case hookops.OutcomeForeign:
			fmt.Fprintf(&report, "%s hook exists but wasn't created by git-user — remove manually if needed: %s\n", r.Spec.Name, r.Existing)
		}
	}

	if report.Len() == 0 {
		return opResult{detail: "No git-user hooks found"}, nil
	}
	return opResult{detail: report.String()}, nil
}

func opHookCheck() (opResult, error) {
	store, err := config.Load()
	if err != nil {
		return opResult{}, fmt.Errorf("failed to load git-user config: %v", err)
	}
	if store.Current == "" {
		return opResult{}, fmt.Errorf("no active git-user identity")
	}
	if store.FindUser(store.Current) == nil {
		return opResult{}, fmt.Errorf("active identity %q not found", store.Current)
	}

	if err := hookops.CheckIdentity(store); err != nil {
		switch e := err.(type) {
		case *hookops.IdentityMismatchError:
			return opResult{}, fmt.Errorf("identity mismatch — expected %s <%s>, git config has %s", e.ExpectedName, e.Expected, e.Got)
		case *hookops.PolicyViolation:
			return opResult{}, fmt.Errorf("%s (%s)", e.Message, e.FixHint)
		default:
			return opResult{}, err
		}
	}
	return opResult{detail: "Identity verified — commit is using the correct identity"}, nil
}
