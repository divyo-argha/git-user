package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

var (
	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFFF")). // Cyan
			Bold(true)

	iconStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF00FF")) // Magenta
)

func runPrompt(args []string) error {
	// Only show prompt if inside a git repository
	if !git.IsInGitRepo() {
		return nil
	}

	store, err := config.Load()
	if err != nil {
		return nil
	}

	u := store.CurrentUser()
	if u == nil {
		return nil
	}

	icon := "👤"
	isP10k := false
	noIcon := false

	for _, arg := range args {
		if arg == "--no-icon" {
			noIcon = true
		}
		if arg == "--p10k" {
			isP10k = true
		}
	}

	if noIcon {
		icon = ""
	}

	verifiedTick := ""
	if u.SigningKey != "" {
		if isP10k {
			verifiedTick = " %F{10}✔%f" // Bright Green in P10k
		} else {
			verifiedTick = " ✔"
		}
	}

	output := ""
	if icon != "" {
		output += iconStyle.Render(icon) + " "
	}
	output += promptStyle.Render(u.Name) + verifiedTick

	fmt.Print(output)
	return nil
}
