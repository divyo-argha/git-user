package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runDoctor(args []string) error {
	ui.Banner("Git-User Diagnostics")
	fmt.Println()

	issues := 0

	ui.Info("Checking active identity...")
	store, err := config.Load()
	if err != nil {
		ui.Error("Failed to load config")
		issues++
	} else if store.Current == "" {
		ui.Warn("No active identity set")
		ui.Info("  Fix: Run 'git user switch <name>' to activate an identity")
		issues++
	} else {
		user := store.FindUser(store.Current)
		if user == nil {
			ui.Error(fmt.Sprintf("Active identity %q not found in config", store.Current))
			issues++
		} else {
			ui.Success(fmt.Sprintf("Active identity: %s (%s)", user.Name, user.Email))

			ui.Info("Checking git config sync...")
			gitName, _ := exec.Command("git", "config", "--global", "user.name").Output()
			gitEmail, _ := exec.Command("git", "config", "--global", "user.email").Output()

			if strings.TrimSpace(string(gitName)) != user.Name {
				ui.Warn(fmt.Sprintf("Git name mismatch: expected %q, got %q", user.Name, strings.TrimSpace(string(gitName))))
				ui.Info("  Fix: Run 'git user switch " + user.Name + "' to resync")
				issues++
			} else if strings.TrimSpace(string(gitEmail)) != user.Email {
				ui.Warn(fmt.Sprintf("Git email mismatch: expected %q, got %q", user.Email, strings.TrimSpace(string(gitEmail))))
				ui.Info("  Fix: Run 'git user switch " + user.Name + "' to resync")
				issues++
			} else {
				ui.Success("Git config in sync")
			}

			if user.SSHKey != "" {
				ui.Info("Checking SSH key...")
				info, err := os.Stat(user.SSHKey)
				if os.IsNotExist(err) {
					ui.Error(fmt.Sprintf("SSH key file not found: %s", user.SSHKey))
					ui.Info("  Fix: Generate a new key with 'git user rekey " + user.Name + "'")
					issues++
				} else if err != nil {
					ui.Error(fmt.Sprintf("Error checking SSH key: %v", err))
					issues++
				} else {
					mode := info.Mode().Perm()
					if mode != 0600 {
						ui.Warn(fmt.Sprintf("SSH key has incorrect permissions: %o (should be 0600)", mode))
						ui.Info(fmt.Sprintf("  Fix: Run 'chmod 600 %s'", user.SSHKey))
						issues++
					} else {
						ui.Success(fmt.Sprintf("SSH key exists with correct permissions: %s", user.SSHKey))
					}

					ui.Info("Testing SSH connection to GitHub...")
					if err := verifySSHConnection(); err != nil {
						ui.Warn("SSH connection failed")
						ui.Info("  This could mean:")
						ui.Info("    - The public key is not added to your GitHub account")
						ui.Info("    - The key is not loaded in ssh-agent")
						ui.Info("    - Network connectivity issues")
						ui.Info("  Fix: Add your public key to GitHub or run 'ssh -T git@github.com' for details")
						issues++
					} else {
						ui.Success("SSH connection verified!")
					}
				}
			} else {
				ui.Warn("No SSH key configured for this identity")
				ui.Info("  Fix: Run 'git user bind " + user.Name + " --ssh-key <path>' or 'git user rekey " + user.Name + "'")
				issues++
			}
		}
	}

	ui.Info("Checking git installation...")
	if !git.IsInstalled() {
		ui.Error("Git is not installed or not on PATH")
		issues++
	} else {
		gitVersion, _ := exec.Command("git", "--version").Output()
		ui.Success(fmt.Sprintf("Git installed: %s", strings.TrimSpace(string(gitVersion))))
	}

	ui.Info("Checking ssh-keygen availability...")
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		ui.Warn("ssh-keygen not found on PATH")
		ui.Info("  This is needed for 'git user register' and 'git user rekey'")
		issues++
	} else {
		ui.Success("ssh-keygen is available")
	}

	if git.IsInRepo() {
		ui.Info("Checking current repository remotes...")
		remotes, err := git.ListRemotes()
		if err == nil && len(remotes) > 0 {
			hasHTTPS := false
			for _, remote := range remotes {
				url, err := git.GetRemoteURL(remote)
				if err == nil && strings.HasPrefix(url, "https://") {
					if !hasHTTPS {
						ui.Warn("Repository uses HTTPS remotes")
						hasHTTPS = true
					}
					ui.Info(fmt.Sprintf("  %s: %s", remote, url))
				}
			}
			if hasHTTPS {
				ui.Info("  Fix: Run 'git-user fix-remote' to convert to SSH")
				issues++
			} else {
				ui.Success("All remotes use SSH")
			}
		}
	}

	fmt.Println()
	ui.Divider()
	if issues == 0 {
		ui.Success("All checks passed! Your git-user setup is healthy.")
	} else {
		ui.Warn(fmt.Sprintf("Found %d issue(s). See suggestions above.", issues))
	}

	return nil
}
