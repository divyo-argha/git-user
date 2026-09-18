package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/promptops"
)

// opPromptStatus builds a comprehensive diagnostic status report showing
// the current prompt resolution, active session, repository state, and
// installation status across all supported shells.
func opPromptStatus(store *config.Store) (opResult, error) {
	var b strings.Builder

	b.WriteString("TERMINAL PROMPT INDICATOR STATUS & PREVIEW\n")
	b.WriteString(strings.Repeat("─", 60) + "\n\n")

	// Current Context
	inRepo := git.IsInRepo()
	repoStr := "No"
	if inRepo {
		if root, err := git.RepoRoot(); err == nil {
			repoStr = fmt.Sprintf("Yes (%s)", root)
		} else {
			repoStr = "Yes"
		}
	}

	session := os.Getenv("GIT_USER_SESSION")
	if session == "" {
		session = os.Getenv("GIT_AUTHOR_NAME")
	}
	if session == "" {
		session = "None"
	}

	localOverride := "None"
	if inRepo && git.HasLocalOverride() {
		localOverride = fmt.Sprintf("%s <%s>", git.CurrentName(), git.CurrentEmail())
	}

	globalCurrent := "None"
	if store != nil && store.Current != "" {
		globalCurrent = store.Current
	}

	b.WriteString("CURRENT CONTEXT:\n")
	b.WriteString(fmt.Sprintf("  Inside Git Repository:   %s\n", repoStr))
	b.WriteString(fmt.Sprintf("  Active Terminal Session: %s\n", session))
	b.WriteString(fmt.Sprintf("  Local Repo Override:     %s\n", localOverride))
	b.WriteString(fmt.Sprintf("  Global Active Profile:   %s\n\n", globalCurrent))

	// Live Preview
	b.WriteString("LIVE PROMPT PREVIEWS:\n")
	previewDefault := promptops.ResolvePrompt(store, true, false, true)
	if previewDefault == "" {
		previewDefault = "(empty / hidden)"
	}
	previewPlain := promptops.ResolvePrompt(store, true, true, true)
	previewRaw := promptops.ResolvePrompt(store, false, true, true)

	b.WriteString(fmt.Sprintf("  Standard (with icon):    %s\n", previewDefault))
	b.WriteString(fmt.Sprintf("  Plain (no badges):       %s\n", previewPlain))
	b.WriteString(fmt.Sprintf("  Raw name only:           %s\n\n", previewRaw))

	// Shell Integrations
	b.WriteString("SHELL INTEGRATIONS:\n")
	all := promptops.CheckAllTargets()
	for _, info := range all {
		statusStr := "[ NOT INSTALLED ]"
		if info.Installed {
			statusStr = "[   INSTALLED   ]"
		}
		activeTag := ""
		if info.IsActive {
			activeTag = " (active shell)"
		}
		b.WriteString(fmt.Sprintf("  %-18s %s%s\n", info.Name+":", statusStr, activeTag))
		b.WriteString(fmt.Sprintf("    Config: %s\n", info.ConfigPath))
	}
	b.WriteString("\n")

	// Preferences
	b.WriteString("CONFIGURED PREFERENCES:\n")
	iconPref := " (default Nerd Font)"
	alwaysPref := "False (hidden outside Git repositories)"
	plainPref := "False (includes (session) or (local) badges)"
	if store != nil && store.Prompt != nil {
		if store.Prompt.Icon != "" {
			iconPref = fmt.Sprintf("%q", store.Prompt.Icon)
		}
		if store.Prompt.Always {
			alwaysPref = "True (always visible)"
		}
		if store.Prompt.Plain {
			plainPref = "True (name only without badges)"
		}
	}
	b.WriteString(fmt.Sprintf("  Custom Icon:     %s\n", iconPref))
	b.WriteString(fmt.Sprintf("  Always Show:     %s\n", alwaysPref))
	b.WriteString(fmt.Sprintf("  Plain Format:    %s\n\n", plainPref))

	b.WriteString("QUICK TIPS:\n")
	b.WriteString("  • After installing, run 'source ~/.zshrc' (or restart terminal) to activate.\n")
	b.WriteString("  • Use 'gu switch -s <name>' to switch identities in your current session.\n")
	b.WriteString("  • On standard TTY / dumb consoles, icons automatically fall back to 'git:'.\n")

	return opResult{detail: b.String(), showReport: true}, nil
}

// opInstallPrompt installs the prompt integration for the specified shell target.
func opInstallPrompt(target promptops.Target) (opResult, error) {
	msg, err := promptops.Install(target)
	if err != nil {
		return opResult{}, err
	}
	return opResult{
		detail:     fmt.Sprintf("%s\n\nRestart your shell or reload your rc file to activate.", msg),
		showReport: true,
	}, nil
}

// opUninstallPrompt removes the prompt integration for one target or all targets.
func opUninstallPrompt(target promptops.Target) (opResult, error) {
	var lines []string
	var err error
	if target == "" || target == "all" {
		lines, err = promptops.UninstallAll()
	} else {
		lines, err = promptops.Uninstall(target)
	}
	if err != nil {
		return opResult{}, err
	}
	detail := strings.Join(lines, "\n")
	if detail == "" {
		detail = "Prompt integration removed."
	}
	return opResult{detail: detail, showReport: true}, nil
}

// opSetPromptIcon saves the user's prompt icon preference into config.json.
func opSetPromptIcon(store *config.Store, icon string) (opResult, error) {
	if store == nil {
		return opResult{}, fmt.Errorf("config store not available")
	}
	if store.Prompt == nil {
		store.Prompt = &config.PromptConfig{}
	}
	store.Prompt.Icon = icon
	if err := config.Save(store); err != nil {
		return opResult{}, fmt.Errorf("saving prompt icon: %w", err)
	}
	label := icon
	if label == "" {
		label = "(default)"
	}
	return opResult{detail: fmt.Sprintf("Updated prompt icon to %s", label)}, nil
}

// opTogglePromptAlways toggles the "always show" preference in config.json.
func opTogglePromptAlways(store *config.Store) (opResult, error) {
	if store == nil {
		return opResult{}, fmt.Errorf("config store not available")
	}
	if store.Prompt == nil {
		store.Prompt = &config.PromptConfig{}
	}
	store.Prompt.Always = !store.Prompt.Always
	if err := config.Save(store); err != nil {
		return opResult{}, fmt.Errorf("saving prompt preference: %w", err)
	}
	state := "ON (always displayed in prompt)"
	if !store.Prompt.Always {
		state = "OFF (displayed inside Git repositories only)"
	}
	return opResult{detail: fmt.Sprintf("Prompt indicator always-show is now %s", state)}, nil
}
