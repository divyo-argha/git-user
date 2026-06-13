package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/divyo-argha/git-user/internal/config"
)

func runPrompt(_ []string) error {
	// Check if we are inside a git repository
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		// Not in a git repo or git is not installed; exit silently
		os.Exit(0)
	}

	// Load git-user config
	store, err := config.Load()
	if err != nil {
		// Error loading config, exit silently to avoid breaking the prompt
		os.Exit(0)
	}

	// Output the active identity name if there is one
	if store.Current != "" {
		fmt.Print(store.Current)
	}

	return nil
}
