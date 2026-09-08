package cli

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/shellinit"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runInit(args []string) error {
	var explicitShell string
	installMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "install", "--install", "-i":
			installMode = true
		case "--shell":
			if i+1 < len(args) {
				explicitShell = args[i+1]
				i++
			}
		case "--fish", "fish":
			explicitShell = "fish"
		case "--powershell", "--pwsh", "powershell", "pwsh":
			explicitShell = "powershell"
		case "--bash", "bash", "--zsh", "zsh", "--sh", "sh":
			explicitShell = "posix"
		case "-h", "--help":
			runSubcommandHelp("init")
			return nil
		default:
			if explicitShell == "" && !strings.HasPrefix(args[i], "-") {
				explicitShell = args[i]
			}
		}
	}

	sh := shellinit.Detect(explicitShell)

	if installMode {
		return runInitInstall(sh, explicitShell)
	}

	fmt.Print(shellinit.Script(sh))
	return nil
}

func runInitInstall(sh shellinit.Shell, explicitShell string) error {
	results, err := shellinit.Install(sh, explicitShell)
	if err != nil {
		return err
	}

	installedCount := 0
	for _, r := range results {
		switch r.Status {
		case shellinit.StatusUpgraded:
			ui.Success(fmt.Sprintf("Upgraded shell integration in %s to safe invocation", r.File))
		case shellinit.StatusAlready:
			ui.Success(fmt.Sprintf("Shell integration is already installed in %s", r.File))
		case shellinit.StatusInstalled:
			ui.Success(fmt.Sprintf("Installed shell integration to %s", r.File))
			installedCount++
		}
	}

	if installedCount > 0 {
		ui.Info("Open a new terminal or reload your shell to start using per-session switching:")
		for _, r := range results {
			if r.Status != shellinit.StatusInstalled {
				continue
			}
			if strings.HasSuffix(r.File, ".zshrc") {
				fmt.Println("  source ~/.zshrc")
			} else if strings.HasSuffix(r.File, ".bashrc") {
				fmt.Println("  source ~/.bashrc")
			}
		}
	}
	return nil
}
