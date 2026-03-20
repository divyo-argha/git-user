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

	ui.Header("Active Identity")
	ui.Divider()
	fmt.Printf("  Name  : %s\n", u.Name)
	fmt.Printf("  Email : %s\n", u.Email)
	if u.SSHKey != "" {
		fmt.Printf("  Key   : %s\n", u.SSHKey)
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

	return nil
}
