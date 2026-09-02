package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runDoctor(args []string) error {
	ui.Banner("GIT-USER DIAGNOSTICS & SECURITY")
	fmt.Println()

	issues := 0

	ui.Info("Checking config file permissions...")
	configPath := config.ConfigPath()
	info, err := os.Stat(configPath)
	if err == nil {
		if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable {
			if !pc.Secure {
				ui.Warn(fmt.Sprintf("Config file has insecure permissions: %o", info.Mode().Perm()))
				ui.Info(fmt.Sprintf("  Fix: chmod 600 %s", configPath))
				issues++
			} else {
				ui.Success("Config file permissions OK (0600)")
			}
		}
	}

	ui.Info("Checking active identity...")
	store, err := config.Load()
	if err != nil {
		ui.Error("Failed to load config")
		issues++
	} else if store.Current == "" {
		ui.Warn("No active identity set")
		ui.Info("  Fix: Run 'git-user switch <name>' to activate an identity")
		issues++
	} else {
		user := store.FindUser(store.Current)
		if user == nil {
			ui.Error(fmt.Sprintf("Active identity %q not found in config", store.Current))
			issues++
		} else {
			ui.Success(fmt.Sprintf("Active identity: %s (%s)", user.Name, user.Email))

			ui.Info("Checking git config sync...")
			// Resolved (not --global-only), so this reflects what git will
			// actually use right now — including a repo-local override.
			gitName := git.CurrentName()
			gitEmail := git.CurrentEmail()

			if gitName == user.Name && gitEmail == user.Email {
				ui.Success("Git config in sync")
			} else if git.IsInRepo() && git.HasLocalOverride() {
				// A `switch --local` (or an equivalent manual local config)
				// deliberately makes the resolved identity differ from the
				// global active one in just this repo — that's the feature
				// working as intended, not drift. Comparing against
				// `--global` instead would have missed genuine drift in
				// repos that happen to have no local override, so this
				// checks resolved config but only warns when there's no
				// local override to explain the difference.
				ui.Info(fmt.Sprintf("Local override active in this repository (resolved identity: %s <%s>) — differs from the global active identity %q by design.", gitName, gitEmail, user.Name))
			} else if gitName != user.Name {
				ui.Warn(fmt.Sprintf("Git name mismatch: expected %q, got %q", user.Name, gitName))
				ui.Info("  Fix: Run 'git-user switch " + user.Name + "' to resync")
				issues++
			} else {
				ui.Warn(fmt.Sprintf("Git email mismatch: expected %q, got %q", user.Email, gitEmail))
				ui.Info("  Fix: Run 'git-user switch " + user.Name + "' to resync")
				issues++
			}

			if user.SSHKey != "" {
				ui.Info("Checking SSH key...")
				info, err := os.Stat(user.SSHKey)
				if os.IsNotExist(err) {
					ui.Error(fmt.Sprintf("SSH key file not found: %s", user.SSHKey))
					ui.Info("  Fix: Generate a new key with 'git-user rekey " + user.Name + "'")
					issues++
				} else if err != nil {
					ui.Error(fmt.Sprintf("Error checking SSH key: %v", err))
					issues++
				} else {
					if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable {
						if !pc.Secure {
							ui.Warn(fmt.Sprintf("SSH key has incorrect permissions: %o (should be 0600)", info.Mode().Perm()))
							ui.Info(fmt.Sprintf("  Fix: Run 'chmod 600 %s'", user.SSHKey))
							issues++
						} else {
							ui.Success(fmt.Sprintf("SSH key exists with correct permissions: %s", user.SSHKey))
						}
					} else {
						ui.Success(fmt.Sprintf("SSH key exists: %s", user.SSHKey))
					}

					ui.Info("Testing SSH connection to GitHub...")
					if err := verifySSHConnectionWithKey(user.SSHKey); err != nil {
						ui.Warn("SSH connection failed")
						ui.Info("  This could mean:")
						ui.Info("    - The public key is not added to your GitHub account")
						ui.Info("    - The key is not loaded in ssh-agent")
						ui.Info("    - Network connectivity issues")
						ui.Info(fmt.Sprintf("  Fix: Add your public key to GitHub or run 'ssh -i %s -o IdentitiesOnly=yes -T git@github.com' for details", user.SSHKey))
						issues++
					} else {
						ui.Success("SSH connection verified!")
					}
				}
			} else {
				ui.Warn("No SSH key configured for this identity")
				ui.Info("  Fix: Run 'git-user bind-key " + user.Name + " --ssh-key <path>' or 'git-user rekey " + user.Name + "'")
				issues++
			}
		}
	}

	if store != nil && len(store.Users) > 0 {
		ui.Info("Auditing profile security & passphrases...")
		for _, u := range store.Users {
			if u.SSHKey != "" {
				info, err := os.Stat(u.SSHKey)
				if err == nil {
					if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable && !pc.Secure {
						ui.Warn(fmt.Sprintf("Profile %q SSH key has insecure permissions: %o", u.Name, info.Mode().Perm()))
						ui.Info(fmt.Sprintf("  Fix: chmod 600 %s", u.SSHKey))
						issues++
					}
				}
				protected, err := isSSHKeyPassphraseProtected(u.SSHKey)
				if err == nil {
					if protected {
						ui.Success(fmt.Sprintf("Profile %q SSH key is passphrase protected", u.Name))
					} else {
						ui.Warn(fmt.Sprintf("Profile %q SSH key has no passphrase", u.Name))
						issues++
					}
				}
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
		ui.Info("  This is needed for 'git-user register' and 'git-user rekey'")
		issues++
	} else {
		ui.Success("ssh-keygen is available")
	}

	ui.Info("Checking for stale SSH key backups...")
	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	if entries, err := os.ReadDir(sshDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".backup") {
				ui.Warn(fmt.Sprintf("Stale backup key found: ~/.ssh/%s", e.Name()))
				ui.Info("  Safe to delete once you've confirmed the new key works")
			}
		}
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
		ui.Success("All checks passed! Your git-user setup is 100% healthy and secure.")
	} else {
		ui.Warn(fmt.Sprintf("Found %d issue(s). See suggestions above.", issues))
	}

	return nil
}
