package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/shellinit"
	"github.com/divyo-argha/git-user/internal/ui"
	"github.com/divyo-argha/git-user/internal/validate"
)

func runDoctor(args []string) error {
	fix := false
	for _, a := range args {
		if a == "--fix" || a == "-f" {
			fix = true
		}
	}

	jsonOutput := ui.IsJSONOutput(args)
	var warnings []string
	record := func(msg string) {
		if jsonOutput {
			warnings = append(warnings, msg)
			return
		}
		ui.Warn(msg)
	}

	realStdout := os.Stdout
	if jsonOutput {
		if devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0); err == nil {
			os.Stdout = devNull
			defer devNull.Close()
		}
	}

	ui.Banner("GIT-USER DIAGNOSTICS & SECURITY")
	fmt.Println()
	if fix {
		ui.Info("Running with --fix: auto-correctable issues below are fixed in place, not just reported.")
		fmt.Println()
	}

	issues := 0
	fixed := 0
	scoreTotal := 0
	scorePassed := 0
	// Tracked across the two checks below (SSH connectivity for the active
	// identity, HTTPS remotes in the current repo) so the HTTPS-remotes
	// suggestion can offer a token as an alternative to fix-remote when SSH
	// demonstrably isn't working for this identity right now — instead of
	// just repeating "convert to SSH" as if that were always the fix.
	activeSSHFailed := false
	activeUserHasToken := false

	ui.Info("Checking config file permissions...")
	configPath := config.ConfigPath()
	info, err := os.Stat(configPath)
	if err == nil {
		if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable {
			scoreTotal++
			if !pc.Secure {
				if fix {
					if chmodErr := os.Chmod(configPath, 0600); chmodErr != nil {
						record(fmt.Sprintf("Could not fix permissions on %s: %v", configPath, chmodErr))
						issues++
					} else {
						ui.Success(fmt.Sprintf("Fixed: %s permissions → 0600", configPath))
						fixed++
						scorePassed++
					}
				} else {
					record(fmt.Sprintf("Config file has insecure permissions: %o", info.Mode().Perm()))
					ui.Info(fmt.Sprintf("  Fix: chmod 600 %s", configPath))
					issues++
				}
			} else {
				ui.Success("Config file permissions OK (0600)")
				scorePassed++
			}
		}
	}

	ui.Info("Checking active identity...")
	store, err := config.Load()
	if err != nil {
		record("Failed to load config")
		issues++
	} else if store.Current == "" {
		record("No active identity set")
		ui.Info("  Fix: Run 'git-user switch <name>' to activate an identity")
		issues++
	} else {
		user := store.FindUser(store.Current)
		if user == nil {
			record(fmt.Sprintf("Active identity %q not found in config", store.Current))
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
			} else if fix {
				if err := git.Apply(user.Name, user.Email); err != nil {
					record(fmt.Sprintf("Could not resync git config: %v", err))
					issues++
				} else if err := applyUserSSHConfig(user, false); err != nil {
					record(fmt.Sprintf("Could not resync SSH config: %v", err))
					issues++
				} else {
					ui.Success(fmt.Sprintf("Fixed: git config re-synced to %q (%s)", user.Name, user.Email))
					fixed++
				}
			} else if gitName != user.Name {
				record(fmt.Sprintf("Git name mismatch: expected %q, got %q", user.Name, gitName))
				ui.Info("  Fix: Run 'git-user switch " + user.Name + "' to resync")
				issues++
			} else {
				record(fmt.Sprintf("Git email mismatch: expected %q, got %q", user.Email, gitEmail))
				ui.Info("  Fix: Run 'git-user switch " + user.Name + "' to resync")
				issues++
			}

			if user.SSHKey != "" {
				ui.Info("Checking SSH key...")
				info, err := os.Stat(user.SSHKey)
				if os.IsNotExist(err) {
					record(fmt.Sprintf("SSH key file not found: %s", user.SSHKey))
					ui.Info("  Fix: Generate a new key with 'git-user rekey " + user.Name + "'")
					issues++
				} else if err != nil {
					record(fmt.Sprintf("Error checking SSH key: %v", err))
					issues++
				} else {
					if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable {
						scoreTotal++
						if !pc.Secure {
							if fix {
								if chmodErr := os.Chmod(user.SSHKey, 0600); chmodErr != nil {
									record(fmt.Sprintf("Could not fix permissions on %s: %v", user.SSHKey, chmodErr))
									issues++
								} else {
									ui.Success(fmt.Sprintf("Fixed: %s permissions → 0600", user.SSHKey))
									fixed++
									scorePassed++
								}
							} else {
								record(fmt.Sprintf("SSH key has incorrect permissions: %o (should be 0600)", info.Mode().Perm()))
								ui.Info(fmt.Sprintf("  Fix: Run 'chmod 600 %s'", user.SSHKey))
								issues++
							}
						} else {
							ui.Success(fmt.Sprintf("SSH key exists with correct permissions: %s", user.SSHKey))
							scorePassed++
						}
					} else {
						ui.Success(fmt.Sprintf("SSH key exists: %s", user.SSHKey))
					}

					ui.Info("Testing SSH connection to GitHub...")
					if err := verifySSHConnectionWithKey(user.SSHKey); err != nil {
						activeSSHFailed = true
						record("SSH connection failed")
						ui.Info("  This could mean:")
						ui.Info("    - The public key is not added to your GitHub account")
						ui.Info("    - The key is not loaded in ssh-agent")
						ui.Info("    - Network connectivity issues (some networks block SSH's port 22 entirely)")
						ui.Info(fmt.Sprintf("  Fix: Add your public key to GitHub or run 'ssh -i %s -o IdentitiesOnly=yes -T git@github.com' for details", user.SSHKey))
						if !keyring.HasHTTPSToken(user.Name) {
							ui.Info(fmt.Sprintf("  If SSH is blocked on this network rather than misconfigured, use an HTTPS token instead: git-user token %s --set", user.Name))
						}
						issues++
					} else {
						ui.Success("SSH connection verified!")
					}
				}
			} else {
				record("No SSH key configured for this identity")
				ui.Info("  Fix: Run 'git-user bind-key " + user.Name + " --ssh-key <path>' or 'git-user rekey " + user.Name + "'")
				issues++
			}

			scoreTotal++
			if warnMsg := signingDisabledWarning(user); warnMsg != "" {
				record(warnMsg)
				ui.Info(fmt.Sprintf("  Fix: Run 'git-user sign %s --on'", user.Name))
				issues++
			} else {
				ui.Success("Commit signing is configured")
				scorePassed++
			}

			activeUserHasToken = keyring.HasHTTPSToken(user.Name)
			if activeUserHasToken {
				ui.Success("HTTPS token stored (used automatically on HTTPS remotes)")
				if user.HTTPSTokenExpiresAt != "" {
					scoreTotal++
					if warnMsg := tokenExpiryWarning(user.HTTPSTokenExpiresAt); warnMsg != "" {
						record(warnMsg)
						ui.Info(fmt.Sprintf("  Fix: Generate a new token on your platform, then run 'git-user token %s --set'", user.Name))
						issues++
					} else {
						ui.Success(fmt.Sprintf("Token expires %s", user.HTTPSTokenExpiresAt))
						scorePassed++
					}
				}
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
						if fix {
							if chmodErr := os.Chmod(u.SSHKey, 0600); chmodErr != nil {
								record(fmt.Sprintf("Could not fix permissions on %s: %v", u.SSHKey, chmodErr))
								issues++
							} else {
								ui.Success(fmt.Sprintf("Fixed: %s permissions → 0600", u.SSHKey))
								fixed++
							}
						} else {
							record(fmt.Sprintf("Profile %q SSH key has insecure permissions: %o", u.Name, info.Mode().Perm()))
							ui.Info(fmt.Sprintf("  Fix: chmod 600 %s", u.SSHKey))
							issues++
						}
					}
				}
				protected, err := isSSHKeyPassphraseProtected(u.SSHKey)
				if err == nil {
					if protected {
						ui.Success(fmt.Sprintf("Profile %q SSH key is passphrase protected", u.Name))
					} else {
						record(fmt.Sprintf("Profile %q SSH key has no passphrase", u.Name))
						issues++
					}
				}
			}
			// The active identity's own token/expiry and signing status were
			// already checked above (with more context — it's the one doctor
			// just tested SSH connectivity for); this only covers the rest.
			if u.Name != store.Current {
				if u.HTTPSTokenExpiresAt != "" {
					if warnMsg := tokenExpiryWarning(u.HTTPSTokenExpiresAt); warnMsg != "" {
						record(fmt.Sprintf("Profile %q: %s", u.Name, warnMsg))
						issues++
					}
				}
				uCopy := u
				if warnMsg := signingDisabledWarning(&uCopy); warnMsg != "" {
					record(fmt.Sprintf("Profile %q: %s", u.Name, warnMsg))
					issues++
				}
			}
		}
	}

	ui.Info("Checking git installation...")
	if !git.IsInstalled() {
		record("Git is not installed or not on PATH")
		issues++
	} else {
		gitVersion, _ := exec.Command("git", "--version").Output()
		ui.Success(fmt.Sprintf("Git installed: %s", strings.TrimSpace(string(gitVersion))))
	}

	ui.Info("Checking ssh-keygen availability...")
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		record("ssh-keygen not found on PATH")
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

	ui.Info("Checking shell PATH and binary resolution...")
	pathEnv := os.Getenv("PATH")
	if pathEnv != "" {
		var foundPaths []string
		binName := "git-user"
		if runtime.GOOS == "windows" {
			binName = "git-user.exe"
		}
		for _, dir := range filepath.SplitList(pathEnv) {
			p := filepath.Join(dir, binName)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				duplicate := false
				for _, prev := range foundPaths {
					if prev == p {
						duplicate = true
						break
					}
				}
				if !duplicate {
					foundPaths = append(foundPaths, p)
				}
			}
		}
		if len(foundPaths) > 1 {
			ui.Warn(fmt.Sprintf("Multiple %s binaries detected in PATH:", binName))
			for idx, fp := range foundPaths {
				prefix := "  •"
				if idx == 0 {
					prefix = "  ▶ (active)"
				}
				verOut, _ := exec.Command(fp, "--version").Output()
				verStr := strings.TrimSpace(string(verOut))
				if verStr == "" {
					verStr = "unknown version"
				}
				ui.Info(fmt.Sprintf("%s %s (%s)", prefix, fp, verStr))
			}
			ui.Info("  If commands behave unexpectedly, remove stale versions or adjust your PATH order.")
		} else {
			ui.Success("Binary resolution OK (no shadowing detected)")
		}
	}

	ui.Info("Checking shell integration...")
	rcFiles := []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".config", "fish", "config.fish"),
	}
	legacyShellFound := false
	for _, rc := range rcFiles {
		if content, err := os.ReadFile(rc); err == nil {
			str := string(content)
			if strings.Contains(str, "eval \"$(git-user init)\"") {
				legacyShellFound = true
				if !fix {
					record(fmt.Sprintf("Legacy unshielded shell integration in %s", filepath.Base(rc)))
					ui.Info("  Fix: Run 'git-user init install' to upgrade to safe invocation")
					issues++
				}
			}
		}
	}
	if legacyShellFound && fix {
		if results, err := shellinit.Install(shellinit.Detect(""), ""); err != nil {
			record(fmt.Sprintf("Could not upgrade shell integration: %v", err))
			issues++
		} else {
			for _, r := range results {
				if r.Status == shellinit.StatusUpgraded {
					ui.Success(fmt.Sprintf("Fixed: upgraded shell integration in %s", r.File))
					fixed++
				}
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
				if fix {
					if err := runFixRemote(nil); err != nil {
						record(fmt.Sprintf("Could not convert remotes: %v", err))
						issues++
					} else {
						fixed++
					}
				} else {
					ui.Info("  Fix: Run 'git-user fix-remote' to convert to SSH")
					// fix-remote isn't really "the fix" when SSH has just
					// demonstrably failed for this identity — offer the
					// alternative this doctor run already has evidence for,
					// instead of only ever pointing at SSH.
					if activeSSHFailed && !activeUserHasToken {
						ui.Info("  Or, since SSH just failed above: git-user token <name> --set")
					}
					record("Repository uses HTTPS remotes (not converted to SSH)")
					issues++
				}
			} else {
				ui.Success("All remotes use SSH")
			}
		}

		ui.Info("Checking repository signing policy...")
		if repoRoot, rootErr := git.RepoRoot(); rootErr == nil {
			policy, err := config.LoadRepoPolicy(repoRoot)
			if err == nil && (policy.RequireSigning || len(policy.AllowedEmailDomains) > 0) {
				scoreTotal++
				compliant := true

				hookInstalled := false
				if content, err := os.ReadFile(filepath.Join(repoRoot, ".git", "hooks", "pre-commit")); err == nil {
					hookInstalled = strings.HasPrefix(string(content), "#!/bin/sh\n# git-user")
				}
				if !hookInstalled {
					record("Repository requires policy enforcement (.git-user-policy) but the enforcing hook isn't installed")
					ui.Info("  Fix: Run 'git-user hook install'")
					issues++
					compliant = false
				}
				if store != nil {
					if activeUser := store.FindUser(store.Current); activeUser != nil {
						if policy.RequireSigning {
							if warnMsg := signingDisabledWarning(activeUser); warnMsg != "" {
								record("Repository requires signed commits (.git-user-policy), but: " + warnMsg)
								ui.Info(fmt.Sprintf("  Fix: Run 'git-user sign %s --on'", activeUser.Name))
								issues++
								compliant = false
							}
						}
						if len(policy.AllowedEmailDomains) > 0 {
							_, domain, _ := strings.Cut(strings.ToLower(activeUser.Email), "@")
							allowed := false
							for _, d := range policy.AllowedEmailDomains {
								if domain == d {
									allowed = true
									break
								}
							}
							if !allowed {
								record(fmt.Sprintf("Repository restricts commits to domains %s, but active identity uses %s", strings.Join(policy.AllowedEmailDomains, ", "), activeUser.Email))
								issues++
								compliant = false
							}
						}
					}
				}

				if compliant {
					ui.Success("Repository policy requirements are satisfied")
					scorePassed++
				}
			}
		}
	}

	os.Stdout = realStdout

	if jsonOutput {
		securityScore := ""
		if scoreTotal > 0 {
			securityScore = fmt.Sprintf("%d/%d", scorePassed, scoreTotal)
		}
		_ = json.NewEncoder(os.Stdout).Encode(struct {
			Issues        int      `json:"issues"`
			Fixed         int      `json:"fixed"`
			SecurityScore string   `json:"security_score,omitempty"`
			Warnings      []string `json:"warnings"`
		}{
			Issues:        issues,
			Fixed:         fixed,
			SecurityScore: securityScore,
			Warnings:      warnings,
		})
		if issues > 0 {
			return fmt.Errorf("%d issue(s) found", issues)
		}
		return nil
	}

	fmt.Println()
	ui.Divider()
	if scoreTotal > 0 {
		ui.Info(fmt.Sprintf("Security Score: %d/%d", scorePassed, scoreTotal))
	}
	if issues == 0 && fixed == 0 {
		ui.Success("All checks passed! Your git-user setup is 100% healthy and secure.")
	} else if fix {
		if fixed > 0 {
			ui.Success(fmt.Sprintf("Fixed %d issue(s).", fixed))
		}
		if issues > 0 {
			ui.Warn(fmt.Sprintf("%d issue(s) need manual attention (see warnings above).", issues))
		}
	} else {
		ui.Warn(fmt.Sprintf("Found %d issue(s). See suggestions above.", issues))
		ui.Info("Run 'git-user doctor --fix' to automatically correct what can be fixed.")
	}

	return nil
}

// tokenExpiryWarnDays is how far ahead of an HTTPS token's recorded expiry
// doctor starts warning — long enough to generate and swap in a replacement
// before it actually lapses mid-push.
const tokenExpiryWarnDays = 14

// tokenExpiryWarning returns a warning message if expiresAt (a
// validate.DateLayout date) is already past, or within tokenExpiryWarnDays,
// and "" if it's further out (or unparseable — doctor has no fix for a
// corrupted date, and refusing to run over it would be worse than skipping
// it silently).
func tokenExpiryWarning(expiresAt string) string {
	expiry, err := time.Parse(validate.DateLayout, expiresAt)
	if err != nil {
		return ""
	}
	days := int(time.Until(expiry).Hours() / 24)
	switch {
	case days < 0:
		return fmt.Sprintf("HTTPS token expired %d day(s) ago (%s)", -days, expiresAt)
	case days <= tokenExpiryWarnDays:
		return fmt.Sprintf("HTTPS token expires in %d day(s) (%s)", days, expiresAt)
	default:
		return ""
	}
}

func signingDisabledWarning(u *config.User) string {
	if u.SignDisabled {
		return "Commit signing is disabled for this identity"
	}
	if u.SignKey == "" {
		return "No commit signing key configured for this identity"
	}
	return ""
}
