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

	// Context Awareness (Current Repository)
	if remoteURL, err := git.GetRemoteURL(); err == nil && remoteURL != "" {
		platform, repo := git.DetectPlatformFromURL(remoteURL)
		if platform != "" {
			ui.Divider()
			ui.Header("Project Context")
			icon := "🌐"
			switch strings.ToLower(platform) {
			case "github":
				icon = "🐙"
			case "gitlab":
				icon = "🦊"
			case "bitbucket":
				icon = "💙"
			}
			fmt.Printf("  %-10s: %s %s\n", ui.StyleDim().Render("Platform"), icon, platform)
			fmt.Printf("  %-10s: %s\n", ui.StyleDim().Render("Repository"), repo)

			// Show if this user has a bound account for THIS specific platform
			if username, ok := u.Platforms[strings.ToLower(platform)]; ok {
				fmt.Printf("  %-10s: %s (%s)\n", ui.StyleDim().Render("Linked"), username, ui.StyleSuccess().Render("Verified Auth"))
			} else {
				fmt.Printf("  %-10s: %s\n", ui.StyleDim().Render("Status"), ui.StyleDim().Render("No linked account for this platform"))
			}
		}
	}

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

	// Global Settings Section (Only show if Strict mode is ENFORCED)
	if store.Strict {
		ui.Divider()
		ui.Header("Global Settings")
		fmt.Printf("  %-10s: %s\n", ui.StyleDim().Render("Mode"), "Strict (Enforced)")
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
