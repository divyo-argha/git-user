package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ssh"
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
	if fp, err := exec.Command("ssh-keygen", "-lf", pubKeyPath).Output(); err == nil {
		report += "\nFingerprint: " + strings.TrimSpace(string(fp)) + "\n"
	}
	report += "\nAdd this key to your Git platform(s):\n"
	report += "  GitHub:    Settings → SSH and GPG keys → New SSH key\n"
	report += "  GitLab:    Preferences → SSH Keys → Add new key\n"
	report += "  Bitbucket: Personal settings → SSH keys → Add key\n"
	report += "\nThe same public key can be added to multiple platforms.\n"
	report += "The private key stays on your machine and is never shared.\n"
	return opResult{detail: report, showReport: true}, nil
}

// opCheckSSH tests the SSH connection for an identity.
func opCheckSSH(store *config.Store, name string) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	if user.SSHKey == "" {
		return opResult{}, fmt.Errorf("no SSH key bound to identity %q", name)
	}
	report := fmt.Sprintf("Checking SSH connection for %s (%s)\n\n", user.Name, user.Email)
	ok, detail := verifySSHConnectionWithKey(user.SSHKey)
	if ok {
		report += "SSH connection verified successfully!\n"
	} else {
		report += "SSH verification failed.\n"
		report += detail
	}
	if !ssh.IsSSHKeyLoaded(user.SSHKey) {
		report += "\nKey is not loaded in the SSH agent. Unlock it by switching to this identity.\n"
	}
	return opResult{detail: report, showReport: true}, nil
}

// verifySSHConnectionWithKey tests SSH auth across the major platforms.
func verifySSHConnectionWithKey(keyPath string) (bool, string) {
	platforms := []struct {
		host    string
		success []string
	}{
		{"git@github.com", []string{"Hi ", "successfully authenticated"}},
		{"git@gitlab.com", []string{"Welcome to GitLab", "successfully authenticated"}},
		{"git@bitbucket.org", []string{"logged in as", "successfully authenticated"}},
	}
	for _, p := range platforms {
		args := []string{"-T", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=5"}
		if keyPath != "" {
			args = append(args, "-i", keyPath, "-o", "IdentitiesOnly=yes")
		}
		args = append(args, p.host)
		cmd := exec.Command("ssh", args...)
		if keyPath != "" {
			cmd.Env = append(os.Environ(), "SSH_AUTH_SOCK=")
		}
		output, _ := cmd.CombinedOutput()
		out := string(output)
		for _, marker := range p.success {
			if strings.Contains(out, marker) {
				return true, ""
			}
		}
	}
	return false, "  • The public key may not be added to GitHub/GitLab yet\n  • The key may not be loaded in ssh-agent\n  • Network connectivity issues\n"
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

// opSecurity audits identity and config file security.
func opSecurity(store *config.Store) (opResult, error) {
	report := ""
	issues := 0
	configPath := config.ConfigPath()
	if info, err := os.Stat(configPath); err == nil {
		mode := info.Mode().Perm()
		if mode != 0600 {
			report += fmt.Sprintf("⚠ Config file has insecure permissions: %o (fix: chmod 600 %s)\n", mode, configPath)
			issues++
		} else {
			report += "✓ Config file permissions OK (0600)\n"
		}
	}
	for _, user := range store.Users {
		report += fmt.Sprintf("\n%s (%s)\n", user.Name, user.Email)
		if user.SSHKey == "" {
			report += "  ⚠ No SSH key bound (bind one from the profile view)\n"
			issues++
			continue
		}
		info, err := os.Stat(user.SSHKey)
		if err != nil {
			report += fmt.Sprintf("  ⚠ SSH key not found: %s\n", user.SSHKey)
			issues++
			continue
		}
		mode := info.Mode().Perm()
		if mode != 0600 {
			report += fmt.Sprintf("  ⚠ Insecure key permissions: %o (fix: chmod 600 %s)\n", mode, user.SSHKey)
			issues++
		} else {
			report += fmt.Sprintf("  ✓ Permissions OK: %s\n", filepath.Base(user.SSHKey))
		}
		protected, err := isSSHKeyPassphraseProtected(user.SSHKey)
		if err != nil {
			report += "  ⚠ Could not verify passphrase protection\n"
			issues++
		} else if protected {
			report += "  ✓ Passphrase protected\n"
		} else {
			report += "  ⚠ No passphrase detected — use the Passphrase menu to add one\n"
			issues++
		}
	}
	report += "\n"
	if issues == 0 {
		report += "No security issues found\n"
	} else {
		report += fmt.Sprintf("Found %d security issue(s)\n", issues)
	}
	return opResult{detail: report, showReport: true}, nil
}

// opDoctor runs a health check.
func opDoctor(store *config.Store) (opResult, error) {
	report := ""
	issues := 0
	user := store.CurrentUser()
	if store.Current == "" || user == nil {
		report += "⚠ No active identity set — switch to one from the dashboard\n"
		issues++
	} else {
		report += fmt.Sprintf("✓ Active identity: %s (%s)\n", user.Name, user.Email)
	}
	if user != nil && store.Current != "" {
		gitName := git.CurrentGlobalName()
		gitEmail := git.CurrentGlobalEmail()
		if gitName != user.Name || gitEmail != user.Email {
			report += fmt.Sprintf("⚠ Git config mismatch: expected %s <%s>, got %s <%s>. Re-switch to resync.\n", user.Name, user.Email, gitName, gitEmail)
			issues++
		} else {
			report += "✓ Git config in sync\n"
		}
		if user.SSHKey != "" {
			if info, err := os.Stat(user.SSHKey); err != nil {
				report += fmt.Sprintf("⚠ SSH key file not found: %s\n", user.SSHKey)
				issues++
			} else {
				report += fmt.Sprintf("✓ SSH key exists: %s\n", user.SSHKey)
				_ = info
			}
		} else {
			report += "⚠ No SSH key configured for this identity\n"
			issues++
		}
	}
	if git.IsInstalled() {
		report += "✓ Git is installed\n"
	} else {
		report += "⚠ Git is not installed or not on PATH\n"
		issues++
	}
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		report += "⚠ ssh-keygen not found on PATH\n"
		issues++
	} else {
		report += "✓ ssh-keygen is available\n"
	}
	home, _ := os.UserHomeDir()
	if entries, err := os.ReadDir(filepath.Join(home, ".ssh")); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".backup") {
				report += fmt.Sprintf("ℹ Stale backup key found: ~/.ssh/%s (safe to delete once the new key works)\n", e.Name())
			}
		}
	}
	report += "\n"
	if issues == 0 {
		report += "All checks passed! Your git-user setup is healthy.\n"
	} else {
		report += fmt.Sprintf("Found %d issue(s). See suggestions above.\n", issues)
	}
	return opResult{detail: report, showReport: true}, nil
}

