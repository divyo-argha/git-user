package cli

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

// onboardingPromptCommands is the allowlist of interactive entry points that may
// show the first-run import prompt. Read-only/system commands (list, current,
// doctor, prompt, completion, ...) never ask, so automation and shell
// integrations are never interrupted. The interactive TUI launcher (no args,
// "tui", "-i", "--interactive") is intentionally excluded: the TUI asks the
// import question itself inside the UI so the user never leaves the TUI.
var onboardingPromptCommands = map[string]bool{
	"register": true,
	"reg":      true,
	"switch":   true,
	"sw":       true,
}

// shouldPromptFirstRunImport reports whether a command may trigger the
// first-run import onboarding. The prompt itself is additionally gated on an
// interactive terminal and an unanswered store, so this only narrows the set of
// commands that are allowed to ask.
func shouldPromptFirstRunImport(sub string) bool {
	return onboardingPromptCommands[sub]
}

// maybePromptFirstRunImport offers the user a chance to import their existing
// git identity into git-user instead of importing it silently. It never
// modifies anything without an explicit choice, and it only runs once per
// machine (the decision is recorded in the store).
func maybePromptFirstRunImport() error {
	if !ui.IsTTY() {
		return nil
	}

	store, err := config.Load()
	if err != nil {
		return nil
	}
	if store.ImportPrompted || len(store.Users) > 0 {
		return nil
	}

	name := git.CurrentName()
	email := git.CurrentEmail()

	// Record that the prompt was shown even when there is nothing to import,
	// so we never nag again on future runs.
	if name == "" && email == "" {
		store.ImportPrompted = true
		_ = config.Save(store)
		return nil
	}

	ui.Banner("FIRST RUN SETUP")
	fmt.Println()
	ui.Info("git-user found an existing Git identity on this machine:")
	fmt.Printf("    name:  %s\n", name)
	fmt.Printf("    email: %s\n", email)
	fmt.Println()

	// Mark the prompt as shown as soon as we actually ask, so it only appears
	// once regardless of the user's choice.
	store.ImportPrompted = true

	if !ui.Confirm("Import this identity into git-user? Nothing will be changed if you choose No.", false) {
		fmt.Println()
		ui.Info("No problem — nothing was imported.")
		ui.Info("You can import it later with: git-user switch --original")
		ui.Info("Or from the TUI:  System Utilities → Import / Export → Import original gitconfig")
		return config.Save(store)
	}

	sshCommand := git.CurrentSSHCommand()

	defaultName := name
	if defaultName == "" {
		defaultName = "original"
	}

	fmt.Println()
	ui.Info("What should this identity be known as? You'll use this name to switch to it later.")
	importName, promptErr := ui.Prompt(fmt.Sprintf("Identity name [%s]:", defaultName))
	if promptErr != nil {
		return promptErr
	}
	importName = strings.TrimSpace(importName)
	if importName == "" {
		importName = defaultName
	}

	if store.FindUser(importName) != nil {
		ui.Errorf("Identity %q already exists — nothing was imported.", importName)
		ui.Info("Resolve this by either:")
		ui.Info("  • importing under a different name: git-user switch --original <unique-name>")
		ui.Info("  • renaming the conflicting profile first: git-user rename " + importName + " <new-name>, then: git-user switch --original " + importName)
		return config.Save(store)
	}

	store.SnapshotOriginal(name, email, sshCommand, git.CurrentSigningKey(), git.CurrentSignFormat(), git.CurrentCommitGPGSign())

	store.Users = append(store.Users, config.User{
		Name:       importName,
		Email:      email,
		SSHKey:     extractSSHKeyFromCommand(sshCommand),
		SSHCommand: sshCommand,
		Source:     "original",
	})

	// The imported identity stays active: the gitconfig already holds it, so we
	// keep it as-is and simply record it as the current profile.
	store.Current = importName

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	fmt.Println()
	ui.Success(fmt.Sprintf("Imported your existing identity as %q and set it active", importName))
	ui.Info(fmt.Sprintf("  name:  %s", name))
	ui.Info(fmt.Sprintf("  email: %s", email))
	if sshCommand != "" {
		ui.Info(fmt.Sprintf("  SSH: preserving core.sshCommand: %s", sshCommand))
	}
	fmt.Println()
	ui.Info(fmt.Sprintf("It stays your active identity. Switch away with: git-user switch %s", importName))
	return nil
}
