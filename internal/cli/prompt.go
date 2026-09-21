package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/promptops"
	"github.com/divyo-argha/git-user/internal/ui"
)

func parseTarget(name string) (promptops.Target, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "zsh":
		return promptops.TargetZsh, nil
	case "bash":
		return promptops.TargetBash, nil
	case "fish":
		return promptops.TargetFish, nil
	case "starship":
		return promptops.TargetStarship, nil
	case "powershell", "pwsh":
		return promptops.TargetPowerShell, nil
	case "nushell", "nu":
		return promptops.TargetNushell, nil
	default:
		return "", fmt.Errorf("unknown target shell %q (supported: zsh, bash, fish, starship, powershell, nushell)", name)
	}
}

func runPrompt(args []string) error {
	withIcon := false
	always := false
	plain := false

	for i, arg := range args {
		switch arg {
		case "install":
			targetArg := ""
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				targetArg = args[i+1]
			}
			return runPromptInstall(targetArg)
		case "uninstall":
			targetArg := ""
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				targetArg = args[i+1]
			}
			return runPromptUninstall(targetArg)
		case "--icon", "-i":
			withIcon = true
		case "--always", "-a":
			always = true
		case "--plain", "-p":
			plain = true
		}
	}

	store, _ := config.Load()
	out := promptops.ResolvePrompt(store, withIcon, plain, always)
	if out != "" {
		fmt.Print(out)
	}
	return nil
}

func runPromptInstall(targetArg string) error {
	if targetArg != "" {
		target, err := parseTarget(targetArg)
		if err != nil {
			return err
		}
		msg, err := promptops.Install(target)
		if err != nil {
			return err
		}
		ui.Success(msg)
		printPostInstallHint(target)
		return nil
	}

	ui.Banner("PROMPT INTEGRATION INSTALLER")
	fmt.Println()

	var options []string
	home, _ := os.UserHomeDir()
	starshipPath := filepath.Join(home, ".config", "starship.toml")

	hasStarship := false
	if _, err := os.Stat(starshipPath); err == nil {
		hasStarship = true
	}

	shellEnv := strings.ToLower(os.Getenv("SHELL"))
	isGitBash := strings.Contains(shellEnv, "bash") ||
		strings.Contains(shellEnv, "sh") ||
		os.Getenv("BASH") != "" ||
		os.Getenv("MSYSTEM") != ""

	if hasStarship {
		options = append(options, "Starship Prompt (recommended - detected)")
	}

	if isGitBash && runtime.GOOS == "windows" {
		options = append(options, "Bash / Git Bash (recommended - active shell)")
		options = append(options, "PowerShell")
		options = append(options, "Nushell")
	} else if runtime.GOOS == "windows" {
		options = append(options, "PowerShell (recommended - Windows)")
		options = append(options, "Bash / Git Bash")
		options = append(options, "Nushell")
	} else if strings.Contains(shellEnv, "zsh") {
		options = append(options, "Zsh (recommended - active shell)")
		options = append(options, "Bash")
		options = append(options, "Fish")
		options = append(options, "PowerShell")
		options = append(options, "Nushell")
	} else if strings.Contains(shellEnv, "fish") {
		options = append(options, "Fish (recommended - active shell)")
		options = append(options, "Zsh")
		options = append(options, "Bash")
		options = append(options, "PowerShell")
		options = append(options, "Nushell")
	} else if strings.Contains(shellEnv, "nu") {
		options = append(options, "Nushell (recommended - active shell)")
		options = append(options, "Zsh")
		options = append(options, "Bash")
		options = append(options, "Fish")
		options = append(options, "PowerShell")
	} else {
		options = append(options, "Bash (recommended - active shell)")
		options = append(options, "Zsh")
		options = append(options, "Fish")
		options = append(options, "PowerShell")
		options = append(options, "Nushell")
	}

	if !hasStarship {
		options = append(options, "Starship Prompt")
	}

	options = append(options, "Cancel")

	idx, err := ui.Select("Choose where to install the git-user prompt integration:", options)
	if err != nil {
		return err
	}

	choice := options[idx]
	if choice == "Cancel" {
		ui.Info("Cancelled")
		return nil
	}

	var target promptops.Target
	if strings.Contains(choice, "Starship") {
		target = promptops.TargetStarship
	} else if strings.Contains(choice, "Zsh") {
		target = promptops.TargetZsh
	} else if strings.Contains(choice, "Bash") {
		target = promptops.TargetBash
	} else if strings.Contains(choice, "Fish") {
		target = promptops.TargetFish
	} else if strings.Contains(choice, "PowerShell") {
		target = promptops.TargetPowerShell
	} else if strings.Contains(choice, "Nushell") {
		target = promptops.TargetNushell
	} else {
		return nil
	}

	msg, err := promptops.Install(target)
	if err != nil {
		return err
	}
	ui.Success(msg)
	printPostInstallHint(target)
	return nil
}

func printPostInstallHint(target promptops.Target) {
	switch target {
	case promptops.TargetZsh:
		ui.Info("Run 'source ~/.zshrc' to apply the changes to your current terminal session.")
	case promptops.TargetBash:
		ui.Info("Run 'source ~/.bashrc' to apply the changes to your current terminal session.")
	case promptops.TargetFish:
		ui.Info("Open a new fish session or run 'source ~/.config/fish/conf.d/git_user_prompt.fish' to apply.")
	case promptops.TargetPowerShell:
		ui.Info("Restart your PowerShell session or run '. $PROFILE' to apply.")
	case promptops.TargetNushell:
		ui.Info("Restart Nushell to see your active profile in the right prompt.")
	case promptops.TargetStarship:
		ui.Info("Starship automatically reloads configurations. Try navigating to a git repository now.")
	}
}

func runPromptUninstall(targetArg string) error {
	ui.Banner("PROMPT INTEGRATION UNINSTALLER")
	fmt.Println()

	if targetArg != "" {
		if strings.ToLower(targetArg) == "all" {
			lines, err := promptops.UninstallAll()
			if err != nil {
				return err
			}
			for _, l := range lines {
				ui.Info(l)
			}
			return nil
		}
		target, err := parseTarget(targetArg)
		if err != nil {
			return err
		}
		lines, err := promptops.Uninstall(target)
		if err != nil {
			return err
		}
		for _, l := range lines {
			ui.Info(l)
		}
		return nil
	}

	var options []string
	for _, t := range promptops.AllTargets() {
		info := promptops.CheckTarget(t)
		if info.Installed {
			options = append(options, fmt.Sprintf("%s (installed)", promptops.TargetName(t)))
		} else {
			options = append(options, promptops.TargetName(t))
		}
	}
	options = append(options, "All shells", "Cancel")

	idx, err := ui.Select("Choose which prompt integration to uninstall:", options)
	if err != nil {
		return err
	}
	choice := options[idx]
	if choice == "Cancel" {
		ui.Info("Cancelled")
		return nil
	}
	if choice == "All shells" {
		lines, err := promptops.UninstallAll()
		if err != nil {
			return err
		}
		for _, l := range lines {
			ui.Info(l)
		}
		return nil
	}

	for _, t := range promptops.AllTargets() {
		if strings.HasPrefix(choice, promptops.TargetName(t)) {
			lines, err := promptops.Uninstall(t)
			if err != nil {
				return err
			}
			for _, l := range lines {
				ui.Info(l)
			}
			return nil
		}
	}
	return nil
}
