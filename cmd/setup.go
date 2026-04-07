package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/ui"
)

const (
	markerStart = "# git-user:prompt:start"
	markerEnd   = "# git-user:prompt:end"
)

func runSetupPrompt(_ []string) error {
	shell := DetectShell()
	if shell == "" {
		ui.Error("Could not detect your shell. Please specify manually (zsh or bash).")
		return fmt.Errorf("unknown shell")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		ui.Errorf("could not find home directory: %v", err)
		return err
	}

	configFile := ""
	if shell == "zsh" || shell == "zsh-p10k" {
		configFile = filepath.Join(home, ".zshrc")
	} else if shell == "bash" {
		configFile = filepath.Join(home, ".bashrc")
	}

	if configFile == "" {
		ui.Errorf("Unsupported shell: %s", shell)
		return fmt.Errorf("unsupported shell")
	}

	// Prepare injection block
	exe, err := os.Executable()
	if err != nil {
		exe = "git-user"
	}

	// Deep P10k integration
	if shell == "zsh-p10k" {
		p10kFile := filepath.Join(home, ".p10k.zsh")
		if err := setupP10kDeep(p10kFile, exe); err == nil {
			ui.Success("Deep Powerlevel10k integration applied to ~/.p10k.zsh")
		} else {
			ui.Warn(fmt.Sprintf("Could not apply deep P10k integration: %v. Falling back to simple hook.", err))
		}
	}

	// Standard shell config injection (e.g., .zshrc)
	content, _ := os.ReadFile(configFile)
	injection := fmt.Sprintf("%s\neval \"$(%s init %s)\"\n%s\n", markerStart, exe, shell, markerEnd)

	// Determine where to insert if P10k
	finalContent := ""
	if shell == "zsh-p10k" {
		lines := strings.Split(string(content), "\n")
		insertIdx := -1
		for i, line := range lines {
			if strings.Contains(line, "p10k.zsh") {
				insertIdx = i
				break
			}
		}
		if insertIdx != -1 {
			newLines := append(lines[:insertIdx], append([]string{injection}, lines[insertIdx:]...)...)
			finalContent = strings.Join(newLines, "\n")
		}
	}

	if finalContent == "" {
		finalContent = string(content)
		if len(finalContent) > 0 && !strings.HasSuffix(finalContent, "\n") {
			finalContent += "\n"
		}
		finalContent += "\n" + injection
	}

	os.WriteFile(configFile, []byte(finalContent), 0644)

	ui.Success(fmt.Sprintf("Successfully added prompt integration to %s", configFile))
	ui.Info("Please restart your terminal or run:")
	fmt.Printf("  source %s\n\n", configFile)
	return nil
}

func setupP10kDeep(path string, exe string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Backup
	os.WriteFile(path+".bak", content, 0644)

	strContent := string(content)
	if strings.Contains(strContent, "prompt_git_user") {
		return nil // Already deep integrated
	}

	// 1. Inject into array: typeset -g POWERLEVEL9K_RIGHT_PROMPT_ELEMENTS=(
	target := "typeset -g POWERLEVEL9K_RIGHT_PROMPT_ELEMENTS=("
	idx := strings.Index(strContent, target)
	if idx != -1 {
		insertPos := idx + len(target)
		strContent = strContent[:insertPos] + "\n    git_user                # active git-user identity" + strContent[insertPos:]
	}

	// 2. Append styling and function definition
	// COLOR 15 (Bright White) name with COLOR 14 (Bright Cyan) icon
	funcDef := fmt.Sprintf("\n%s\n# git-user:styling\ntypeset -g POWERLEVEL9K_GIT_USER_FOREGROUND=15\ntypeset -g POWERLEVEL9K_GIT_USER_VISUAL_IDENTIFIER_COLOR=14\ntypeset -g POWERLEVEL9K_GIT_USER_BOLD=true\n\nprompt_git_user() {\n  local name=$(%s prompt --no-icon)\n  [[ -n $name ]] && p10k segment -f 15 -i '👤' -t \"$name\"\n}\n%s\n", markerStart, exe, markerEnd)
	strContent += funcDef

	return os.WriteFile(path, []byte(strContent), 0644)
}

func runReload(args []string) error {
	ui.Info("Reloading git-user configuration...")
	runRemovePrompt(args)
	return runSetupPrompt(args)
}

func runRemovePrompt(_ []string) error {
	home, _ := os.UserHomeDir()
	files := []string{filepath.Join(home, ".zshrc"), filepath.Join(home, ".bashrc"), filepath.Join(home, ".p10k.zsh")}

	found := false
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}

		if err := removeMarkersFromFile(file); err == nil {
			found = true
			ui.Success(fmt.Sprintf("Removed prompt integration from %s", file))
		}
		
		// Specialized P10k array cleanup
		if strings.HasSuffix(file, ".p10k.zsh") {
			content, _ := os.ReadFile(file)
			newContent := strings.ReplaceAll(string(content), "\n    git_user                # active git-user identity", "")
			os.WriteFile(file, []byte(newContent), 0644)
		}
	}

	if !found {
		ui.Info("No git-user prompt integration found in your shell config files.")
	} else {
		ui.Info("Please restart your terminal to complete removal.")
	}

	return nil
}

func removeMarkersFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var newLines []string
	scanner := bufio.NewScanner(f)
	inBlock := false
	foundAny := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == markerStart {
			inBlock = true
			foundAny = true
			continue
		}
		if strings.TrimSpace(line) == markerEnd {
			inBlock = false
			continue
		}
		if !inBlock {
			newLines = append(newLines, line)
		}
	}

	if !foundAny {
		return fmt.Errorf("markers not found")
	}

	// Write back
	output := strings.Join(newLines, "\n")
	if len(newLines) > 0 && newLines[len(newLines)-1] != "" {
		output += "\n"
	}

	return os.WriteFile(path, []byte(output), 0644)
}
