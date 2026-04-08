package cmd

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runSignIn(args []string) error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	u := store.CurrentUser()
	if u == nil {
		ui.Error("No active identity. Switch to an identity first with 'git user switch <name>'")
		return fmt.Errorf("no active identity")
	}

	// Parse flags
	remember := false
	for _, arg := range args {
		if arg == "--remember" {
			remember = true
		}
	}

	// Set remember mode
	if err := store.SetRemember(u.Name, remember); err != nil {
		ui.Errorf("setting remember mode: %v", err)
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	mode := "forget"
	if remember {
		mode = "remember"
	}

	ui.Success(fmt.Sprintf("Signed in as %q in %s mode", u.Name, mode))
	if remember {
		ui.Info("Credentials will persist across profile switches")
	} else {
		ui.Info("Credentials will be cleared when switching profiles")
	}

	return nil
}
