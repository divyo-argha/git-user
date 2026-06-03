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
		ui.Info("Run 'git-user switch <name>' to activate one.")
		return nil
	}

	ui.Banner("Active Identity")
	fmt.Printf("  Name:  %s", u.Name)
	if u.Source == "original" {
		fmt.Printf(" [imported from original]\n")
	} else {
		fmt.Println()
	}
	fmt.Printf("  Email: %s\n", u.Email)
	
	if u.SSHKey != "" {
		fmt.Printf("  SSH:   %s\n", u.SSHKey)
	}

	gitName := git.CurrentName()
	gitEmail := git.CurrentEmail()

	if gitName != u.Name || gitEmail != u.Email {
		ui.Divider()
		ui.Warn("Git config is out of sync")
		ui.Info(fmt.Sprintf("Run 'git-user switch %s' to re-apply", u.Name))
	}

	return nil
}
