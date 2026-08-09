package cli

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runImportOriginal(args []string) error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	// Check if already imported
	for _, u := range store.Users {
		if u.Source == "original" {
			ui.Warn(fmt.Sprintf("Original identity already imported as %q", u.Name))
			ui.Info("To rename: git-user edit " + u.Name + " <email>")
			return nil
		}
	}

	// Read current gitconfig (before any git-user switch has happened)
	// If original snapshot exists, use that — it's the pre-git-user state
	var name, email, sshCommand string
	if store.Original != nil {
		name = store.Original.Name
		email = store.Original.Email
		sshCommand = store.Original.SSHCommand
	} else {
		name = git.CurrentName()
		email = git.CurrentEmail()
		sshCommand = git.CurrentSSHCommand()
	}

	if name == "" && email == "" {
		ui.Error("No user.name or user.email found — nothing to import")
		return fmt.Errorf("no original identity")
	}

	// Determine name for the identity — the user always chooses it. The current
	// git user.name (or "original") is only offered as a suggestion.
	importName := name
	if len(args) > 0 {
		importName = args[0]
	} else {
		defaultName := name
		if defaultName == "" {
			defaultName = "original"
		}
		chosen, promptErr := ui.Prompt(fmt.Sprintf("Name for this identity [%s]:", defaultName))
		if promptErr != nil {
			return promptErr
		}
		chosen = strings.TrimSpace(chosen)
		if chosen == "" {
			chosen = defaultName
		}
		importName = chosen
	}

	if store.FindUser(importName) != nil {
		ui.Errorf("Identity %q already exists — nothing was imported.", importName)
		ui.Info("Resolve this by either:")
		ui.Info("  • importing under a different name: git-user switch --original <unique-name>")
		ui.Info("  • renaming the conflicting profile first: git-user rename " + importName + " <new-name>, then: git-user switch --original " + importName)
		return fmt.Errorf("identity exists")
	}

	if email == "" {
		promptEmail, promptErr := ui.Prompt("Email for this identity:")
		if promptErr != nil || promptEmail == "" {
			ui.Error("Email is required")
			return fmt.Errorf("missing email")
		}
		email = promptEmail
	}

	// Extract SSH key path from core.sshCommand if present, but keep the full
	// command so switching to this identity restores the exact original setup.
	sshKey := extractSSHKeyFromCommand(sshCommand)

	store.Users = append(store.Users, config.User{
		Name:       importName,
		Email:      email,
		SSHKey:     sshKey,
		SSHCommand: sshCommand,
		Source:     "original",
	})

	// The imported identity stays active: the gitconfig already holds it, so we
	// keep it as-is and simply record it as the current profile.
	store.Current = importName

	// Snapshot the original for --original restore if not already done
	store.SnapshotOriginal(name, email, sshCommand, git.CurrentSigningKey(), git.CurrentSignFormat(), git.CurrentCommitGPGSign())
	store.ImportPrompted = true

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	fmt.Println()
	ui.Success(fmt.Sprintf("Imported original identity as %q and set it active", importName))
	ui.Info(fmt.Sprintf("  Email: %s", email))
	if sshKey != "" {
		ui.Info(fmt.Sprintf("  SSH Key: %s", sshKey))
	}
	if sshCommand != "" {
		ui.Info(fmt.Sprintf("  SSH Command: %s", sshCommand))
	}
	fmt.Println()
	ui.Info(fmt.Sprintf("It stays your active identity. Switch away with: git-user switch %s", importName))
	ui.Info("Or restore raw original: git-user switch --original")

	return nil
}

// extractSSHKeyFromCommand parses "ssh -i /path/to/key ..." and returns the key path.
func extractSSHKeyFromCommand(cmd string) string {
	parts := strings.Fields(cmd)
	for i, p := range parts {
		if p == "-i" && i+1 < len(parts) {
			return strings.Trim(parts[i+1], `"'`)
		}
	}
	return ""
}
