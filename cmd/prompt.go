package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
)

var (
	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFFF")). // Cyan
			Bold(true)

	iconStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF00FF")) // Magenta
)

func runPrompt(args []string) error {
	store, err := config.Load()
	if err != nil {
		return nil
	}

	u := store.CurrentUser()
	if u == nil {
		return nil
	}

	icon := "👤"
	if len(args) > 0 && args[0] == "--no-icon" {
		icon = ""
	}

	fmt.Print(iconStyle.Render(icon) + " " + promptStyle.Render(u.Name))
	return nil
}
