package cmd

import (
	"fmt"

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

	ui.Header("Active Profile")
	ui.Divider()
	fmt.Printf("  Name  : %s\n", u.Name)
	fmt.Printf("  Email : %s\n", u.Email)
	if u.SSHKey != "" {
		fmt.Printf("  SSH Key  : %s\n", u.SSHKey)
	}
	if u.SigningKey != "" {
		fmt.Printf("  Signing  : %s (%s)\n", u.SigningKey, u.SigningMethod)
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
