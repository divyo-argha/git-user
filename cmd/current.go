package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runCurrent(_ []string) error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	u := store.CurrentUser()
	if u == nil {
		ui.Warn("No active identity set.")
		ui.Info("Run 'git-user switch <n>' to activate one.")
		return nil
	}

	ui.Banner("Active Profile")

	// Identity Section
	verifiedLabel := ""
	if u.SigningKey != "" {
		verifiedLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true).
			Render(" ✔ VERIFIED")
	}

	ui.Header("Identity")
	fmt.Printf("  %-10s: %s%s\n", ui.StyleDim().Render("Name"), u.Name, verifiedLabel)
	fmt.Printf("  %-10s: %s\n", ui.StyleDim().Render("Email"), u.Email)

	// Security Section
	ui.Divider()
	ui.Header("Security")
	signingStatus := "Not Configured"
	if u.SigningKey != "" {
		method := "GPG"
		if u.SigningMethod == "ssh" {
			method = "SSH"
		}
		signingStatus = fmt.Sprintf("%s (%s)", method, u.SigningKey)
	}
	fmt.Printf("  %-10s: %s\n", ui.StyleDim().Render("Signing"), signingStatus)
	
	sshStatus := "Not Configured"
	if u.SSHKey != "" {
		sshStatus = u.SSHKey
	}
	fmt.Printf("  %-10s: %s\n", ui.StyleDim().Render("SSH Key"), sshStatus)

	// Platforms Section
	if len(u.Platforms) > 0 {
		ui.Divider()
		ui.Header("Platforms")
		for p, user := range u.Platforms {
			fmt.Printf("  %-10s: %s\n", ui.StyleDim().Render(strings.Title(p)), user)
		}
	}

	ui.Divider()

	// Cross-check against actual git global config.
	gitName := git.CurrentName()
	gitEmail := git.CurrentEmail()

	if gitName != u.Name || gitEmail != u.Email {
		ui.Warn("Global git config is out of sync with git-user:")
		fmt.Printf("  git config user.name  = %q\n", gitName)
		fmt.Printf("  git config user.email = %q\n", gitEmail)
		ui.Info(fmt.Sprintf("Run 'git-user switch %s' to re-apply.", u.Name))
	}

	// Check for local repository overrides.
	localEmail, _ := git.GetLocalConfig("user.email")
	localName, _ := git.GetLocalConfig("user.name")

	if localEmail != "" || localName != "" {
		isMismatch := false
		if localEmail != "" && localEmail != u.Email {
			isMismatch = true
		}
		if localName != "" && localName != u.Name {
			isMismatch = true
		}

		if isMismatch {
			ui.Warn("Local repository config overrides your active identity:")
			if localName != "" {
				fmt.Printf("  [local] user.name  = %q\n", localName)
			}
			if localEmail != "" {
				fmt.Printf("  [local] user.email = %q\n", localEmail)
			}
			ui.Info("Commits in this directory will use the local config above.")
		}
	}

	return nil
}
