package tui

import (
	"fmt"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
	"os"
	"strings"
)

// ── Reports ───────────────────────────────────────────────────────────────────

// opPubkey returns the active identity's public key text.
func opPubkey(store *config.Store, name string) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	if name != store.Current {
		return opResult{}, fmt.Errorf("to view %s's public key, switch to it first", name)
	}
	if user.SSHKey == "" {
		return opResult{}, fmt.Errorf("identity %q has no SSH key bound", name)
	}
	pubKeyPath := user.SSHKey + ".pub"
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return opResult{}, fmt.Errorf("public key not found at %s", pubKeyPath)
	}
	report := fmt.Sprintf("PUBLIC KEY — %s (%s)\n\n", user.Name, user.Email)
	report += strings.TrimSpace(string(pubKeyBytes)) + "\n"
	if fp, err := runCaptured("", "ssh-keygen", "-lf", pubKeyPath); err == nil {
		report += "\nFingerprint: " + strings.TrimSpace(fp) + "\n"
	}
	report += "\nAdd this key to your Git platform(s):\n"
	report += "  GitHub:    Settings → SSH and GPG keys → New SSH key\n"
	report += "  GitLab:    Preferences → SSH Keys → Add new key\n"
	report += "  Bitbucket: Personal settings → SSH keys → Add key\n"
	report += "\nThe same public key can be added to multiple platforms.\n"
	report += "The private key stays on your machine and is never shared.\n"
	return opResult{detail: report, showReport: true}, nil
}

// opCheckSSH tests the SSH connection for an identity. The passphrase is
// optional: it is used to unlock a passphrase-protected key that is not in
// the agent. Without one, the keychain is consulted (persistent mode) and the
// TUI asks for it in-app otherwise — ssh itself never prompts on the terminal.
func opCheckSSH(store *config.Store, name, passphrase string) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	if user.SSHKey == "" {
		return opResult{}, fmt.Errorf("no SSH key bound to identity %q", name)
	}
	protected, perr := isSSHKeyPassphraseProtected(user.SSHKey)
	if perr == nil && protected && !ssh.IsSSHKeyLoaded(user.SSHKey) {
		p := passphrase
		if p == "" && user.GetPassphraseMode() == "persistent" {
			if secret, kerr := keyring.GetKeychainPassphrase(user.Name); kerr == nil && secret != "" {
				if ssh.VerifyPassphrase(user.SSHKey, secret) {
					p = secret
				} else {
					_ = keyring.DeleteKeychainPassphrase(user.Name)
				}
			}
		}
		if p == "" {
			return opResult{}, ErrNeedsPassphrase
		}
		if !ssh.VerifyPassphrase(user.SSHKey, p) {
			return opResult{}, fmt.Errorf("incorrect passphrase")
		}
		opts := ssh.AgentLoadOptions{LifetimeSecs: uint32(user.GetAgentTTL().Seconds()), ConfirmBeforeUse: user.AgentConfirmBeforeUse}
		if err := ssh.AddSSHKeyWithOptions(user.SSHKey, p, opts); err != nil {
			return opResult{}, fmt.Errorf("could not load key into agent: %w", err)
		}
	}
	results := ssh.CheckAllPlatforms(user.SSHKey)
	report := fmt.Sprintf("Checking SSH connection for %s (%s)\nSSH Key: %s\n\nPlatform Status:\n", user.Name, user.Email, user.SSHKey)
	connectedCount := 0
	var connectedNames []string
	for _, res := range results {
		switch res.Status {
		case "connected":
			connectedCount++
			if res.Username != "" {
				connectedNames = append(connectedNames, fmt.Sprintf("%s (%s)", res.Platform, res.Username))
				report += fmt.Sprintf("  • %-10s : Connected ✓ (%s)\n", res.Platform, res.Username)
			} else {
				connectedNames = append(connectedNames, res.Platform)
				report += fmt.Sprintf("  • %-10s : Connected ✓\n", res.Platform)
			}
		case "network_error":
			report += fmt.Sprintf("  • %-10s : Network error (could not connect)\n", res.Platform)
		case "not_added":
			report += fmt.Sprintf("  • %-10s : Not connected (key not added)\n", res.Platform)
		default:
			report += fmt.Sprintf("  • %-10s : Not connected\n", res.Platform)
		}
	}

	report += "\n"
	if connectedCount == 0 {
		report += "Result: Nothing connected — this SSH key is not added to GitHub, GitLab, or Bitbucket.\n"
		report += "Publish it using 'Publish SSH key to platform' or copy it via 'Show public key'.\n"
	} else {
		report += fmt.Sprintf("Result: Connected to %s\n", strings.Join(connectedNames, ", "))
	}

	if !ssh.IsSSHKeyLoaded(user.SSHKey) {
		report += "\nNote: Key is not loaded in the SSH agent. Unlock it by switching to this identity.\n"
	}
	return opResult{detail: report, showReport: true}, nil
}

// opRefresh re-syncs git-user's stored state onto the live environment,
// actually fixing drift instead of only reporting it (unlike doctor, which is
// read-only). Mirrors `git-user refresh` (internal/cli/refresh.go).
func opRefresh(store *config.Store) (opResult, error) {
	report := ""
	fixed := 0

	configPath := config.ConfigPath()
	if info, statErr := os.Stat(configPath); statErr == nil {
		if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable && !pc.Secure {
			if err := os.Chmod(configPath, 0600); err != nil {
				report += fmt.Sprintf("⚠ Could not fix permissions on %s: %v\n", configPath, err)
			} else {
				report += fmt.Sprintf("Fixed permissions: %s → 0600\n", configPath)
				fixed++
			}
		}
	}
	for _, u := range store.Users {
		if u.SSHKey == "" {
			continue
		}
		if info, statErr := os.Stat(u.SSHKey); statErr == nil {
			if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable && !pc.Secure {
				if err := os.Chmod(u.SSHKey, 0600); err != nil {
					report += fmt.Sprintf("⚠ Could not fix permissions on %s: %v\n", u.SSHKey, err)
				} else {
					report += fmt.Sprintf("Fixed permissions: %s → 0600\n", u.SSHKey)
					fixed++
				}
			}
		}
	}

	if err := config.Save(store); err != nil {
		report += fmt.Sprintf("⚠ Could not resync directory-binding config: %v\n", err)
	} else {
		report += "Directory-binding config is in sync.\n"
	}

	if store.Current == "" {
		report += "No active identity set — nothing to re-apply to git config.\n"
	} else if user := store.FindUser(store.Current); user == nil {
		report += fmt.Sprintf("Active identity %q no longer exists in config — clearing it.\n", store.Current)
		store.Current = ""
		_ = config.Save(store)
		fixed++
	} else {
		before := refreshGitConfigFingerprint()
		var applyErrs []string

		if err := git.Apply(user.Name, user.Email); err != nil {
			applyErrs = append(applyErrs, fmt.Sprintf("re-apply name/email: %v", err))
		}
		var sshErr error
		if user.SSHCommand != "" {
			sshErr = git.SetSSHCommand(user.SSHCommand)
		} else if user.SSHKey != "" {
			sshErr = git.ConfigureSSH(user.SSHKey)
		} else {
			sshErr = git.RemoveSSHConfig()
		}
		if sshErr != nil {
			applyErrs = append(applyErrs, fmt.Sprintf("re-apply SSH config: %v", sshErr))
		}
		if !user.SignDisabled && user.SignKey != "" {
			if err := git.ConfigureSigning(user.SignKey, user.SignFormat); err != nil {
				applyErrs = append(applyErrs, fmt.Sprintf("re-apply signing config: %v", err))
			}
		} else {
			git.RemoveSigningConfig()
		}

		for _, e := range applyErrs {
			report += "⚠ Could not " + e + "\n"
		}
		if before != refreshGitConfigFingerprint() {
			report += fmt.Sprintf("Fixed: git config had drifted from identity %q — corrected.\n", user.Name)
			fixed++
		} else if len(applyErrs) > 0 {
			report += fmt.Sprintf("Git config unchanged for identity %q — the fix attempt above failed, this was NOT verified as healthy.\n", user.Name)
			fixed++
		} else {
			report += fmt.Sprintf("Git config already matched identity %q — nothing to fix.\n", user.Name)
		}
	}

	if fixed == 0 {
		report = "Nothing to fix — git-user's config is already healthy.\n\n" + report
	} else {
		report = fmt.Sprintf("Fixed %d issue(s).\n\n", fixed) + report
	}
	return opResult{detail: report, showReport: true}, nil
}

// refreshGitConfigFingerprint captures the slice of global git config that
// git-user manages, so opRefresh can tell whether re-applying an identity
// actually changed anything or the config was already in sync.
func refreshGitConfigFingerprint() string {
	return strings.Join([]string{
		git.CurrentName(),
		git.CurrentEmail(),
		git.CurrentSSHCommand(),
		git.CurrentSigningKey(),
		git.CurrentSignFormat(),
		git.CurrentCommitGPGSign(),
	}, "\x00")
}

// opLog returns the identity-switch audit log recorded by
// config.AppendSwitchLog, most recent last. Mirrors `git-user log`
// (internal/cli/log.go), capped at the most recent 50 entries.
func opLog() (opResult, error) {
	entries, err := config.ReadSwitchLog()
	if err != nil {
		return opResult{}, err
	}
	if len(entries) == 0 {
		return opResult{detail: "No identity switches recorded yet.\n", showReport: true}, nil
	}

	const limit = 50
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	var report strings.Builder
	for _, line := range entries {
		parts := strings.SplitN(line, "\t", 3)
		var ts, name, repo string
		if len(parts) > 0 {
			ts = parts[0]
		}
		if len(parts) > 1 {
			name = parts[1]
		}
		if len(parts) > 2 {
			repo = parts[2]
		}
		fmt.Fprintf(&report, "%s  %-20s  %s\n", ts, name, repo)
	}
	return opResult{detail: report.String(), showReport: true}, nil
}

// opFixRemote converts HTTPS remotes to SSH.
func opFixRemote() (opResult, error) {
	if !git.IsInstalled() {
		return opResult{}, fmt.Errorf("git is not installed")
	}
	if !git.IsInRepo() {
		return opResult{}, fmt.Errorf("not in a git repository")
	}
	remotes, err := git.ListRemotes()
	if err != nil || len(remotes) == 0 {
		return opResult{}, fmt.Errorf("no remotes found")
	}
	converted := 0
	report := ""
	for _, remote := range remotes {
		url, err := git.GetRemoteURL(remote)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(url, "https://") {
			continue
		}
		sshURL, ok := git.ConvertHTTPSToSSH(url)
		if !ok {
			report += fmt.Sprintf("%s: could not convert %s\n", remote, url)
			continue
		}
		if err := git.SetRemoteURL(remote, sshURL); err != nil {
			report += fmt.Sprintf("%s: failed to update\n", remote)
			continue
		}
		report += fmt.Sprintf("%s: %s → %s\n", remote, url, sshURL)
		converted++
	}
	if converted == 0 {
		report = "All remotes already use SSH.\n"
	} else {
		report = fmt.Sprintf("Converted %d remote(s) to SSH.\n", converted) + report
	}
	return opResult{detail: report, showReport: true}, nil
}
