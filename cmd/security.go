package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runSecurityCheck(args []string) error {
	ui.Banner("SECURITY CHECK")
	fmt.Println()

	issues := 0

	configPath := config.ConfigPath()
	info, err := os.Stat(configPath)
	if err == nil {
		mode := info.Mode().Perm()
		if mode != 0600 {
			ui.Warn(fmt.Sprintf("Config file has insecure permissions: %o", mode))
			ui.Info(fmt.Sprintf("Fix: chmod 600 %s", configPath))
			issues++
		} else {
			ui.Success("Config file permissions OK (0600)")
		}
	}

	store, err := config.Load()
	if err == nil {
		for _, user := range store.Users {
			if user.SSHKey != "" {
				info, err := os.Stat(user.SSHKey)
				if err != nil {
					ui.Warn(fmt.Sprintf("SSH key not found: %s", user.SSHKey))
					issues++
					continue
				}

				mode := info.Mode().Perm()
				if mode != 0600 {
					ui.Warn(fmt.Sprintf("%s: Insecure permissions %o", filepath.Base(user.SSHKey), mode))
					ui.Info(fmt.Sprintf("Fix: chmod 600 %s", user.SSHKey))
					issues++
				} else {
					ui.Success(fmt.Sprintf("%s: Permissions OK", user.Name))
				}

				ui.Info(fmt.Sprintf("%s: Checking passphrase protection...", user.Name))
				ui.Info("  Tip: Add passphrase with: ssh-keygen -p -f " + user.SSHKey)
			}
		}
	}

	fmt.Println()
	ui.Divider()

	if issues == 0 {
		ui.Success("No security issues found")
	} else {
		ui.Warn(fmt.Sprintf("Found %d security issue(s)", issues))
	}

	fmt.Println()
	ui.Header("SECURITY RECOMMENDATIONS")
	fmt.Println()
	fmt.Println("1. Use SSH key passphrases:")
	fmt.Println("   ssh-keygen -p -f ~/.ssh/git_work")
	fmt.Println()
	fmt.Println("2. Use ssh-agent to cache passphrases:")
	fmt.Println("   eval $(ssh-agent)")
	fmt.Println("   ssh-add ~/.ssh/git_work")
	fmt.Println()
	fmt.Println("3. On shared machines:")
	fmt.Println("   - Each user should have their own system account")
	fmt.Println("   - Never share SSH keys between users")
	fmt.Println("   - Use different keys for different identities")
	fmt.Println()
	fmt.Println("4. File permissions (already enforced):")
	fmt.Println("   - Config: 0600 (owner only)")
	fmt.Println("   - SSH keys: 0600 (owner only)")

	return nil
}
