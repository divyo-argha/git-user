package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
	"golang.org/x/crypto/ssh"
)

func runSecurityCheck(args []string) error {
	ui.Banner("SECURITY CHECK")
	fmt.Println()

	issues := 0

	configPath := config.ConfigPath()
	info, err := os.Stat(configPath)
	if err == nil && runtime.GOOS != "windows" {
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
		if len(store.Users) > 0 {
			fmt.Println()
			ui.Header("IDENTITY SECURITY")
			fmt.Println()
		}

		for _, user := range store.Users {
			ui.Info(fmt.Sprintf("%s (%s)", user.Name, user.Email))

			if user.SSHKey == "" {
				ui.Warn("  No SSH key bound")
				ui.Info(fmt.Sprintf("  Fix: git-user bind %s", user.Name))
				issues++
				continue
			}

			info, err := os.Stat(user.SSHKey)
			if err != nil {
				ui.Warn(fmt.Sprintf("  SSH key not found: %s", user.SSHKey))
				issues++
				continue
			}

			mode := info.Mode().Perm()
			if mode != 0600 {
				ui.Warn(fmt.Sprintf("  Insecure key permissions: %o", mode))
				ui.Info(fmt.Sprintf("  Fix: chmod 600 %s", user.SSHKey))
				issues++
			} else {
				ui.Success(fmt.Sprintf("  Permissions OK: %s", filepath.Base(user.SSHKey)))
			}

			protected, err := isSSHKeyPassphraseProtected(user.SSHKey)
			if err != nil {
				ui.Warn("  Could not verify passphrase protection")
				ui.Info(fmt.Sprintf("  Check manually: ssh-keygen -y -f %s", user.SSHKey))
				issues++
			} else if protected {
				ui.Success("  Passphrase protected")
			} else {
				ui.Warn("  No passphrase detected")
				ui.Info("  Fix: ssh-keygen -p -f " + user.SSHKey)
				issues++
			}

			fmt.Println()
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

func isSSHKeyPassphraseProtected(keyPath string) (bool, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return false, err
	}

	_, err = ssh.ParseRawPrivateKey(data)
	if err == nil {
		return false, nil
	}

	var passphraseErr *ssh.PassphraseMissingError
	if errors.As(err, &passphraseErr) {
		return true, nil
	}

	return false, err
}
