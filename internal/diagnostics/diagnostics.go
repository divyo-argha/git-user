package diagnostics

import (
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
	"github.com/divyo-argha/git-user/internal/ssh"
	"github.com/divyo-argha/git-user/internal/validate"
)

type Status int

const (
	StatusPass Status = iota
	StatusWarn
	StatusNotice
	StatusInfo
)

type Check struct {
	ID         string
	Category   string
	Name       string
	Status     Status
	Message    string
	Detail     []string
	FixHint    string
	Fixed      bool
	Scored     bool
	Subject    string
	IsProgress bool
}

type Report struct {
	Checks      []Check
	Issues      int
	Fixed       int
	ScoreTotal  int
	ScorePassed int
}

const TokenExpiryWarnDays = 14

func TokenExpiryMessage(expiresAt string) string {
	expiry, err := time.Parse(validate.DateLayout, expiresAt)
	if err != nil {
		return ""
	}
	days := int(time.Until(expiry).Hours() / 24)
	switch {
	case days < 0:
		return fmt.Sprintf("HTTPS token expired %d day(s) ago (%s)", -days, expiresAt)
	case days <= TokenExpiryWarnDays:
		return fmt.Sprintf("HTTPS token expires in %d day(s) (%s)", days, expiresAt)
	default:
		return ""
	}
}

func SigningDisabledMessage(u *config.User) string {
	if u.SignDisabled {
		return "Commit signing is disabled for this identity"
	}
	if u.SignKey == "" {
		return "No commit signing key configured for this identity"
	}
	return ""
}

type Options struct {
	Fix       bool
	VerifySSH func(keyPath string) error
}

func Run(store *config.Store, opts Options) (Report, error) {
	var checks []Check
	add := func(c Check) { checks = append(checks, c) }
	fix := opts.Fix

	activeSSHFailed := false
	activeUserHasToken := false

	add(Check{IsProgress: true, Category: "System", Message: "Checking config file permissions..."})
	configPath := config.ConfigPath()
	if info, err := os.Stat(configPath); err == nil {
		if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable {
			if !pc.Secure {
				if fix {
					if chmodErr := os.Chmod(configPath, 0600); chmodErr != nil {
						add(Check{ID: "config-perms", Category: "System", Name: "Config file permissions", Status: StatusWarn, Scored: true,
							Message: fmt.Sprintf("Could not fix permissions on %s: %v", configPath, chmodErr)})
					} else {
						add(Check{ID: "config-perms", Category: "System", Name: "Config file permissions", Status: StatusPass, Scored: true, Fixed: true,
							Message: fmt.Sprintf("Fixed: %s permissions → 0600", configPath)})
					}
				} else {
					add(Check{ID: "config-perms", Category: "System", Name: "Config file permissions", Status: StatusWarn, Scored: true,
						Message: fmt.Sprintf("Config file has insecure permissions: %o", info.Mode().Perm()),
						FixHint: fmt.Sprintf("chmod 600 %s", configPath)})
				}
			} else {
				add(Check{ID: "config-perms", Category: "System", Name: "Config file permissions", Status: StatusPass, Scored: true,
					Message: "Config file permissions OK (0600)"})
			}
		}
	}

	add(Check{IsProgress: true, Category: "Active Identity", Message: "Checking active identity..."})
	if store == nil {
		loaded, err := config.Load()
		if err != nil {
			add(Check{ID: "config-load", Category: "Active Identity", Name: "Active identity", Status: StatusWarn, Message: "Failed to load config"})
		} else {
			store = loaded
		}
	}

	if store != nil {
		if store.Current == "" {
			add(Check{ID: "active-identity", Category: "Active Identity", Name: "Active identity", Status: StatusWarn,
				Message: "No active identity set", FixHint: "Run 'git-user switch <name>' to activate an identity"})
		} else if user := store.FindUser(store.Current); user == nil {
			add(Check{ID: "active-identity", Category: "Active Identity", Name: "Active identity", Status: StatusWarn,
				Message: fmt.Sprintf("Active identity %q not found in config", store.Current)})
		} else {
			add(Check{ID: "active-identity", Category: "Active Identity", Name: "Active identity", Status: StatusPass,
				Message: fmt.Sprintf("Active identity: %s (%s)", user.Name, user.Email)})

			add(Check{IsProgress: true, Category: "Active Identity", Message: "Checking git config sync..."})
			gitName := git.CurrentName()
			gitEmail := git.CurrentEmail()

			if gitName == user.Name && gitEmail == user.Email {
				add(Check{ID: "git-config-sync", Category: "Active Identity", Name: "Git config sync", Status: StatusPass, Message: "Git config in sync"})
			} else if git.IsInRepo() && git.HasLocalOverride() {
				add(Check{ID: "git-config-sync", Category: "Active Identity", Name: "Git config sync", Status: StatusInfo,
					Message: fmt.Sprintf("Local override active in this repository (resolved identity: %s <%s>) — differs from the global active identity %q by design.", gitName, gitEmail, user.Name)})
			} else if fix {
				if err := git.Apply(user.Name, user.Email); err != nil {
					add(Check{ID: "git-config-sync", Category: "Active Identity", Name: "Git config sync", Status: StatusWarn,
						Message: fmt.Sprintf("Could not resync git config: %v", err)})
				} else if err := git.ApplyIdentitySSHConfig(user.SSHCommand, user.SSHKey, false); err != nil {
					add(Check{ID: "git-config-sync", Category: "Active Identity", Name: "Git config sync", Status: StatusWarn,
						Message: fmt.Sprintf("Could not resync SSH config: %v", err)})
				} else {
					add(Check{ID: "git-config-sync", Category: "Active Identity", Name: "Git config sync", Status: StatusPass, Fixed: true,
						Message: fmt.Sprintf("Fixed: git config re-synced to %q (%s)", user.Name, user.Email)})
				}
			} else if gitName != user.Name {
				add(Check{ID: "git-config-sync", Category: "Active Identity", Name: "Git config sync", Status: StatusWarn,
					Message: fmt.Sprintf("Git name mismatch: expected %q, got %q", user.Name, gitName),
					FixHint: "Run 'git-user switch " + user.Name + "' to resync"})
			} else {
				add(Check{ID: "git-config-sync", Category: "Active Identity", Name: "Git config sync", Status: StatusWarn,
					Message: fmt.Sprintf("Git email mismatch: expected %q, got %q", user.Email, gitEmail),
					FixHint: "Run 'git-user switch " + user.Name + "' to resync"})
			}

			if user.SSHKey != "" {
				add(Check{IsProgress: true, Category: "Active Identity", Message: "Checking SSH key..."})
				info, err := os.Stat(user.SSHKey)
				if os.IsNotExist(err) {
					add(Check{ID: "ssh-key", Category: "Active Identity", Name: "SSH key", Status: StatusWarn,
						Message: fmt.Sprintf("SSH key file not found: %s", user.SSHKey),
						FixHint: "Generate a new key with 'git-user rekey " + user.Name + "'"})
				} else if err != nil {
					add(Check{ID: "ssh-key", Category: "Active Identity", Name: "SSH key", Status: StatusWarn,
						Message: fmt.Sprintf("Error checking SSH key: %v", err)})
				} else {
					if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable {
						if !pc.Secure {
							if fix {
								if chmodErr := os.Chmod(user.SSHKey, 0600); chmodErr != nil {
									add(Check{ID: "ssh-key-perms", Category: "Active Identity", Name: "SSH key permissions", Status: StatusWarn, Scored: true,
										Message: fmt.Sprintf("Could not fix permissions on %s: %v", user.SSHKey, chmodErr)})
								} else {
									add(Check{ID: "ssh-key-perms", Category: "Active Identity", Name: "SSH key permissions", Status: StatusPass, Scored: true, Fixed: true,
										Message: fmt.Sprintf("Fixed: %s permissions → 0600", user.SSHKey)})
								}
							} else {
								add(Check{ID: "ssh-key-perms", Category: "Active Identity", Name: "SSH key permissions", Status: StatusWarn, Scored: true,
									Message: fmt.Sprintf("SSH key has incorrect permissions: %o (should be 0600)", info.Mode().Perm()),
									FixHint: fmt.Sprintf("Run 'chmod 600 %s'", user.SSHKey)})
							}
						} else {
							add(Check{ID: "ssh-key-perms", Category: "Active Identity", Name: "SSH key permissions", Status: StatusPass, Scored: true,
								Message: fmt.Sprintf("SSH key exists with correct permissions: %s", user.SSHKey)})
						}
					} else {
						add(Check{ID: "ssh-key-perms", Category: "Active Identity", Name: "SSH key", Status: StatusPass,
							Message: fmt.Sprintf("SSH key exists: %s", user.SSHKey)})
					}

					if opts.VerifySSH != nil {
						add(Check{IsProgress: true, Category: "Active Identity", Message: "Testing SSH connection to GitHub..."})
						if err := opts.VerifySSH(user.SSHKey); err != nil {
							activeSSHFailed = true
							detail := []string{
								"  This could mean:",
								"    - The public key is not added to your GitHub account",
								"    - The key is not loaded in ssh-agent",
								"    - Network connectivity issues (some networks block SSH's port 22 entirely)",
								fmt.Sprintf("  Fix: Add your public key to GitHub or run 'ssh -i %s -o IdentitiesOnly=yes -T git@github.com' for details", user.SSHKey),
							}
							if !keyring.HasHTTPSToken(user.Name) {
								detail = append(detail, fmt.Sprintf("  If SSH is blocked on this network rather than misconfigured, use an HTTPS token instead: git-user token %s --set", user.Name))
							}
							add(Check{ID: "ssh-connectivity", Category: "Active Identity", Name: "SSH connectivity", Status: StatusWarn,
								Message: "SSH connection failed", Detail: detail})
						} else {
							add(Check{ID: "ssh-connectivity", Category: "Active Identity", Name: "SSH connectivity", Status: StatusPass,
								Message: "SSH connection verified!"})
						}
					}
				}
			} else {
				add(Check{ID: "ssh-key", Category: "Active Identity", Name: "SSH key", Status: StatusWarn,
					Message: "No SSH key configured for this identity",
					FixHint: "Run 'git-user bind-key " + user.Name + " --ssh-key <path>' or 'git-user rekey " + user.Name + "'"})
			}

			if warnMsg := SigningDisabledMessage(user); warnMsg != "" {
				add(Check{ID: "signing", Category: "Active Identity", Name: "Commit signing", Status: StatusWarn, Scored: true,
					Message: warnMsg, FixHint: fmt.Sprintf("Run 'git-user sign %s --on'", user.Name)})
			} else {
				add(Check{ID: "signing", Category: "Active Identity", Name: "Commit signing", Status: StatusPass, Scored: true,
					Message: "Commit signing is configured"})
			}

			activeUserHasToken = keyring.HasHTTPSToken(user.Name)
			if activeUserHasToken {
				add(Check{ID: "https-token", Category: "Active Identity", Name: "HTTPS token", Status: StatusPass,
					Message: "HTTPS token stored (used automatically on HTTPS remotes)"})
				if user.HTTPSTokenExpiresAt != "" {
					if warnMsg := TokenExpiryMessage(user.HTTPSTokenExpiresAt); warnMsg != "" {
						add(Check{ID: "token-expiry", Category: "Active Identity", Name: "Token expiry", Status: StatusWarn, Scored: true,
							Message: warnMsg, FixHint: fmt.Sprintf("Generate a new token on your platform, then run 'git-user token %s --set'", user.Name)})
					} else {
						add(Check{ID: "token-expiry", Category: "Active Identity", Name: "Token expiry", Status: StatusPass, Scored: true,
							Message: fmt.Sprintf("Token expires %s", user.HTTPSTokenExpiresAt)})
					}
				}
			}
		}
	}

	if store != nil && len(store.Users) > 0 {
		add(Check{IsProgress: true, Category: "Profiles & Security Audit", Message: "Auditing profile security & passphrases..."})
		for _, u := range store.Users {
			if u.SSHKey != "" {
				if info, err := os.Stat(u.SSHKey); err == nil {
					if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable && !pc.Secure {
						if fix {
							if chmodErr := os.Chmod(u.SSHKey, 0600); chmodErr != nil {
								add(Check{ID: "profile-ssh-key-perms", Category: "Profiles & Security Audit", Name: "SSH key permissions", Subject: u.Name, Status: StatusWarn,
									Message: fmt.Sprintf("Could not fix permissions on %s: %v", u.SSHKey, chmodErr)})
							} else {
								add(Check{ID: "profile-ssh-key-perms", Category: "Profiles & Security Audit", Name: "SSH key permissions", Subject: u.Name, Status: StatusPass, Fixed: true,
									Message: fmt.Sprintf("Fixed: %s permissions → 0600", u.SSHKey)})
							}
						} else {
							add(Check{ID: "profile-ssh-key-perms", Category: "Profiles & Security Audit", Name: "SSH key permissions", Subject: u.Name, Status: StatusWarn,
								Message: fmt.Sprintf("Profile %q SSH key has insecure permissions: %o", u.Name, info.Mode().Perm()),
								FixHint: fmt.Sprintf("chmod 600 %s", u.SSHKey)})
						}
					}
				}
				if protected, err := ssh.IsPassphraseProtected(u.SSHKey); err == nil {
					if protected {
						add(Check{ID: "profile-passphrase", Category: "Profiles & Security Audit", Name: "Passphrase protection", Subject: u.Name, Status: StatusPass,
							Message: fmt.Sprintf("Profile %q SSH key is passphrase protected", u.Name)})
					} else {
						add(Check{ID: "profile-passphrase", Category: "Profiles & Security Audit", Name: "Passphrase protection", Subject: u.Name, Status: StatusWarn,
							Message: fmt.Sprintf("Profile %q SSH key has no passphrase", u.Name)})
					}
				}
			}
			if u.Name != store.Current {
				if u.HTTPSTokenExpiresAt != "" {
					if warnMsg := TokenExpiryMessage(u.HTTPSTokenExpiresAt); warnMsg != "" {
						add(Check{ID: "profile-token-expiry", Category: "Profiles & Security Audit", Name: "Token expiry", Subject: u.Name, Status: StatusWarn,
							Message: fmt.Sprintf("Profile %q: %s", u.Name, warnMsg)})
					}
				}
				uCopy := u
				if warnMsg := SigningDisabledMessage(&uCopy); warnMsg != "" {
					add(Check{ID: "profile-signing", Category: "Profiles & Security Audit", Name: "Commit signing", Subject: u.Name, Status: StatusWarn,
						Message: fmt.Sprintf("Profile %q: %s", u.Name, warnMsg)})
				}
			}
		}
	}

	add(Check{IsProgress: true, Category: "System", Message: "Checking git installation..."})
	if !git.IsInstalled() {
		fixHint := "Install Git via your package manager"
		if runtime.GOOS == "windows" {
			fixHint = "Run 'git-user install-git' or 'winget install Git.Git'"
		}
		add(Check{
			ID:       "git-installed",
			Category: "System",
			Name:     "Git installation",
			Status:   StatusWarn,
			Message:  "Git is not installed or not on PATH",
			FixHint:  fixHint,
		})
	} else {
		gitVersion, _ := exec.Command("git", "--version").Output()
		add(Check{ID: "git-installed", Category: "System", Name: "Git installation", Status: StatusPass,
			Message: fmt.Sprintf("Git installed: %s", strings.TrimSpace(string(gitVersion)))})
	}

	add(Check{IsProgress: true, Category: "System", Message: "Checking ssh-keygen availability..."})
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		add(Check{ID: "ssh-keygen", Category: "System", Name: "ssh-keygen availability", Status: StatusWarn,
			Message: "ssh-keygen not found on PATH", Detail: []string{"  This is needed for 'git-user register' and 'git-user rekey'"}})
	} else {
		add(Check{ID: "ssh-keygen", Category: "System", Name: "ssh-keygen availability", Status: StatusPass, Message: "ssh-keygen is available"})
	}

	add(Check{IsProgress: true, Category: "System", Message: "Checking for stale SSH key backups..."})
	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	if entries, err := os.ReadDir(sshDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".backup") {
				add(Check{ID: "stale-backup", Category: "System", Name: "Stale SSH key backup", Status: StatusNotice,
					Message: fmt.Sprintf("Stale backup key found: ~/.ssh/%s", e.Name()),
					Detail:  []string{"  Safe to delete once you've confirmed the new key works"}})
			}
		}
	}

	add(Check{IsProgress: true, Category: "System", Message: "Checking shell PATH and binary resolution..."})
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
			var detail []string
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
				detail = append(detail, fmt.Sprintf("%s %s (%s)", prefix, fp, verStr))
			}
			detail = append(detail, "  If commands behave unexpectedly, remove stale versions or adjust your PATH order.")
			add(Check{ID: "path-shadowing", Category: "System", Name: "Binary resolution", Status: StatusNotice,
				Message: fmt.Sprintf("Multiple %s binaries detected in PATH:", binName), Detail: detail})
		} else {
			add(Check{ID: "path-shadowing", Category: "System", Name: "Binary resolution", Status: StatusPass,
				Message: "Binary resolution OK (no shadowing detected)"})
		}
	}

	add(Check{IsProgress: true, Category: "System", Message: "Checking shell integration..."})
	rcFiles := []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".config", "fish", "config.fish"),
	}
	legacyShellFound := false
	for _, rc := range rcFiles {
		if content, err := os.ReadFile(rc); err == nil {
			if strings.Contains(string(content), "eval \"$(git-user init)\"") {
				legacyShellFound = true
				if !fix {
					add(Check{ID: "shell-integration", Category: "System", Name: "Shell integration", Status: StatusWarn,
						Message: fmt.Sprintf("Legacy unshielded shell integration in %s", filepath.Base(rc)),
						FixHint: "Run 'git-user init install' to upgrade to safe invocation"})
				}
			}
		}
	}
	if legacyShellFound && fix {
		if results, err := shellinit.Install(shellinit.Detect(""), ""); err != nil {
			add(Check{ID: "shell-integration", Category: "System", Name: "Shell integration", Status: StatusWarn,
				Message: fmt.Sprintf("Could not upgrade shell integration: %v", err)})
		} else {
			for _, r := range results {
				if r.Status == shellinit.StatusUpgraded {
					add(Check{ID: "shell-integration", Category: "System", Name: "Shell integration", Status: StatusPass, Fixed: true,
						Message: fmt.Sprintf("Fixed: upgraded shell integration in %s", r.File)})
				}
			}
		}
	}

	if git.IsInRepo() {
		add(Check{IsProgress: true, Category: "Repository", Message: "Checking current repository remotes..."})
		if remotes, err := git.ListRemotes(); err == nil && len(remotes) > 0 {
			hasHTTPS := false
			var detail []string
			for _, remote := range remotes {
				if url, err := git.GetRemoteURL(remote); err == nil && strings.HasPrefix(url, "https://") {
					hasHTTPS = true
					detail = append(detail, fmt.Sprintf("  %s: %s", remote, url))
				}
			}
			if hasHTTPS {
				add(Check{ID: "repo-remotes-notice", Category: "Repository", Name: "Repository remotes", Status: StatusNotice,
					Message: "Repository uses HTTPS remotes", Detail: detail})
				if fix {
					if results, err := git.ConvertRemotesToSSH(); err != nil {
						add(Check{ID: "repo-remotes", Category: "Repository", Name: "Repository remotes", Status: StatusWarn,
							Message: fmt.Sprintf("Could not convert remotes: %v", err)})
					} else {
						converted := 0
						var fixDetail []string
						for _, r := range results {
							switch {
							case r.ConvertFailed:
								fixDetail = append(fixDetail, fmt.Sprintf("%s: could not convert %s", r.Remote, r.OldURL))
							case r.UpdateFailed:
								fixDetail = append(fixDetail, fmt.Sprintf("%s: failed to update", r.Remote))
							case r.Converted:
								fixDetail = append(fixDetail, fmt.Sprintf("%s: %s → %s", r.Remote, r.OldURL, r.NewURL))
								converted++
							}
						}
						msg := "All remotes already used SSH"
						if converted > 0 {
							msg = fmt.Sprintf("Converted %d remote(s) to SSH", converted)
						}
						add(Check{ID: "repo-remotes", Category: "Repository", Name: "Repository remotes", Status: StatusPass, Fixed: true,
							Message: msg, Detail: fixDetail})
					}
				} else {
					var fixDetail []string
					fixDetail = append(fixDetail, "  Fix: Run 'git-user fix-remote' to convert to SSH")
					if activeSSHFailed && !activeUserHasToken {
						fixDetail = append(fixDetail, "  Or, since SSH just failed above: git-user token <name> --set")
					}
					add(Check{ID: "repo-remotes", Category: "Repository", Name: "Repository remotes", Status: StatusWarn,
						Message: "Repository uses HTTPS remotes (not converted to SSH)", Detail: fixDetail})
				}
			} else {
				add(Check{ID: "repo-remotes", Category: "Repository", Name: "Repository remotes", Status: StatusPass, Message: "All remotes use SSH"})
			}
		}

		add(Check{IsProgress: true, Category: "Repository", Message: "Checking repository signing policy..."})
		if repoRoot, rootErr := git.RepoRoot(); rootErr == nil {
			if policy, err := config.LoadRepoPolicy(repoRoot); err == nil && (policy.RequireSigning || len(policy.AllowedEmailDomains) > 0) {
				compliant := true

				hookInstalled := false
				if content, err := os.ReadFile(filepath.Join(repoRoot, ".git", "hooks", "pre-commit")); err == nil {
					hookInstalled = strings.HasPrefix(string(content), "#!/bin/sh\n# git-user")
				}
				if !hookInstalled {
					add(Check{ID: "repo-policy", Category: "Repository", Name: "Repository policy", Status: StatusWarn, Scored: true,
						Message: "Repository requires policy enforcement (.git-user-policy) but the enforcing hook isn't installed",
						FixHint: "Run 'git-user hook install'"})
					compliant = false
				}
				if store != nil {
					if activeUser := store.FindUser(store.Current); activeUser != nil {
						if policy.RequireSigning {
							if warnMsg := SigningDisabledMessage(activeUser); warnMsg != "" {
								add(Check{ID: "repo-policy", Category: "Repository", Name: "Repository policy", Status: StatusWarn, Scored: true,
									Message: "Repository requires signed commits (.git-user-policy), but: " + warnMsg,
									FixHint: fmt.Sprintf("Run 'git-user sign %s --on'", activeUser.Name)})
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
								add(Check{ID: "repo-policy", Category: "Repository", Name: "Repository policy", Status: StatusWarn, Scored: true,
									Message: fmt.Sprintf("Repository restricts commits to domains %s, but active identity uses %s", strings.Join(policy.AllowedEmailDomains, ", "), activeUser.Email)})
								compliant = false
							}
						}
					}
				}

				if compliant {
					add(Check{ID: "repo-policy", Category: "Repository", Name: "Repository policy", Status: StatusPass, Scored: true,
						Message: "Repository policy requirements are satisfied"})
				}
			}
		}

		add(Check{IsProgress: true, Category: "Repository", Message: "Checking committed allowed-signers file..."})
		if repoRoot, rootErr := git.RepoRoot(); rootErr == nil {
			if _, statErr := os.Stat(filepath.Join(repoRoot, config.AllowedSignersFileName)); statErr == nil {
				out, _ := exec.Command("git", "config", "--local", "gpg.ssh.allowedSignersFile").Output()
				if strings.TrimSpace(string(out)) == config.AllowedSignersFileName {
					add(Check{ID: "allowed-signers", Category: "Repository", Name: "Allowed-signers wiring", Status: StatusPass, Scored: true,
						Message: fmt.Sprintf("%s is committed and wired into local git config", config.AllowedSignersFileName)})
				} else {
					add(Check{ID: "allowed-signers", Category: "Repository", Name: "Allowed-signers wiring", Status: StatusWarn, Scored: true,
						Message: fmt.Sprintf("%s exists but local git config isn't using it for signature verification", config.AllowedSignersFileName),
						FixHint: "Run 'git-user hook install' to re-wire it"})
				}
			}
		}
	}

	return buildReport(checks), nil
}

func buildReport(checks []Check) Report {
	r := Report{Checks: checks}
	for _, c := range checks {
		if c.IsProgress {
			continue
		}
		if c.Fixed {
			r.Fixed++
		} else if c.Status == StatusWarn {
			r.Issues++
		}
		if c.Scored {
			r.ScoreTotal++
			if c.Status == StatusPass || c.Fixed {
				r.ScorePassed++
			}
		}
	}
	return r
}
