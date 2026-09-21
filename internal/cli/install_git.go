package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runInstallGit(args []string) error {
	if git.IsInstalled() {
		ui.Success("Git is already installed and available on this system!")
		out, err := exec.Command("git", "--version").Output()
		if err == nil {
			fmt.Printf("  %s\n", strings.TrimSpace(string(out)))
		}
		return nil
	}

	ui.Banner("GIT INSTALLATION HELPER")
	fmt.Println()

	if runtime.GOOS != "windows" {
		ui.Info("To install Git on your system, please use your distribution's official package manager:")
		fmt.Println("  macOS:         brew install git   (or: xcode-select --install)")
		fmt.Println("  Ubuntu/Debian: sudo apt update && sudo apt install git")
		fmt.Println("  Fedora/RHEL:   sudo dnf install git")
		fmt.Println("  Arch Linux:    sudo pacman -S git")
		return nil
	}

	// Windows: Check if winget is available
	wingetPath, err := exec.LookPath("winget.exe")
	if err != nil {
		wingetPath, err = exec.LookPath("winget")
	}

	if err != nil {
		ui.Warn("Microsoft Windows Package Manager (winget) was not found.")
		ui.Info("Please download and install official Git for Windows from:")
		fmt.Println("  https://git-scm.com/download/win")
		return nil
	}

	fmt.Println("Git was not detected on this system.")
	fmt.Println("You can install official Git for Windows via Microsoft Windows Package Manager (winget).")
	fmt.Println()
	fmt.Println("  Package:  Git.Git (Official Git for Windows)")
	fmt.Println("  Source:   winget (Microsoft official catalog)")
	fmt.Println("  Security: Authenticode signed & SHA-256 verified by Microsoft")
	fmt.Println()

	autoConfirm := false
	for _, arg := range args {
		if arg == "-y" || arg == "--yes" {
			autoConfirm = true
			break
		}
	}

	if !autoConfirm {
		fmt.Print("Proceed with official Git installation? [Y/n]: ")
		reader := bufio.NewReader(os.Stdin)
		ans, _ := reader.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans != "" && ans != "y" && ans != "yes" {
			ui.Info("Installation cancelled.")
			return nil
		}
	}

	ui.Info("Running: winget install --id Git.Git -e --source winget ...")
	cmd := exec.Command(wingetPath, "install", "--id", "Git.Git", "-e", "--source", "winget")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		ui.Errorf("winget installation failed: %v", err)
		ui.Info("You can also manually download Git from https://git-scm.com")
		return err
	}

	ui.Success("Git installation completed! Please restart your terminal so PATH is reloaded.")
	return nil
}
