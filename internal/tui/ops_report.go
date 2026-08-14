package tui

import (
	"fmt"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
	"os"
	"os/exec"
	"path/filepath"
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
		if err := ssh.AddSSHKeyWithPassphrase(user.SSHKey, p); err != nil {
			return opResult{}, fmt.Errorf("could not load key into agent: %w", err)
		}
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
// The key must already be unlocked (in the agent or unprotected) by the
// caller: ssh is never allowed to prompt for a passphrase from within the TUI,
// and the special "unset the agent socket" trick is not used because it would
// force ssh to fall back to the key file (and prompt on the tty for it).
func verifySSHConnectionWithKey(keyPath string) (bool, string) {
	platforms := []struct {
		host    string
		success []string
	}{
		{"git@github.com", []string{"Hi ", "successfully authenticated"}},
		{"git@gitlab.com", []string{"Welcome to GitLab", "successfully authenticated"}},
		{"git@bitbucket.org", []string{"logged in as", "successfully authenticated", "authenticated via ssh key"}},
	}
	for _, p := range platforms {
		args := []string{"-T", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=5"}
		if keyPath != "" {
			args = append(args, "-i", keyPath, "-o", "IdentitiesOnly=yes")
		}
		args = append(args, p.host)
		cmd := exec.Command("ssh", args...)
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

// opDoctor runs a comprehensive health & security check across identities and system configuration.
func opDoctor(store *config.Store) (opResult, error) {
	report := "── SYSTEM & GIT CONFIGURATION ──\n"
	issues := 0

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

	configPath := config.ConfigPath()
	if info, err := os.Stat(configPath); err == nil {
		if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable {
			if !pc.Secure {
				report += fmt.Sprintf("⚠ Config file has insecure permissions: %o (fix: chmod 600 %s)\n", info.Mode().Perm(), configPath)
				issues++
			} else {
				report += "✓ Config file permissions OK (0600)\n"
			}
		}
	}

	report += "\n── ACTIVE IDENTITY ──\n"
	user := store.CurrentUser()
	if store.Current == "" || user == nil {
		report += "⚠ No active identity set — switch to one from the dashboard\n"
		issues++
	} else {
		report += fmt.Sprintf("✓ Active identity: %s (%s)\n", user.Name, user.Email)
		gitName := git.CurrentGlobalName()
		gitEmail := git.CurrentGlobalEmail()
		if gitName != user.Name || gitEmail != user.Email {
			report += fmt.Sprintf("⚠ Git config mismatch: expected %s <%s>, got %s <%s>. Re-switch to resync.\n", user.Name, user.Email, gitName, gitEmail)
			issues++
		} else {
			report += "✓ Git config in sync\n"
		}
	}

	report += "\n── PROFILES & SECURITY AUDIT ──\n"
	if len(store.Users) == 0 {
		report += "ℹ No profiles registered yet\n"
	}
	for _, u := range store.Users {
		report += fmt.Sprintf("Profile: %s (%s)\n", u.Name, u.Email)
		if u.SSHKey == "" {
			report += "  ⚠ No SSH key bound (bind one from profile options)\n"
			issues++
			continue
		}
		info, err := os.Stat(u.SSHKey)
		if err != nil {
			report += fmt.Sprintf("  ⚠ SSH key file not found: %s\n", u.SSHKey)
			issues++
			continue
		}
		if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable {
			if !pc.Secure {
				report += fmt.Sprintf("  ⚠ Insecure key permissions: %o (fix: chmod 600 %s)\n", info.Mode().Perm(), u.SSHKey)
				issues++
			} else {
				report += fmt.Sprintf("  ✓ Key permissions OK (0600): %s\n", filepath.Base(u.SSHKey))
			}
		}

		protected, err := isSSHKeyPassphraseProtected(u.SSHKey)
		if err != nil {
			report += "  ⚠ Could not verify passphrase protection\n"
			issues++
		} else if protected {
			report += "  ✓ Passphrase protected\n"
		} else {
			report += "  ⚠ No passphrase detected (passphrase protection recommended)\n"
			issues++
		}
	}

	home, _ := os.UserHomeDir()
	if entries, err := os.ReadDir(filepath.Join(home, ".ssh")); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".backup") {
				report += fmt.Sprintf("\nℹ Stale backup key found: ~/.ssh/%s (safe to delete once the new key works)\n", e.Name())
			}
		}
	}

	report += "\n"
	if issues == 0 {
		report += "All checks passed! Your git-user setup is 100% healthy and secure.\n"
	} else {
		report += fmt.Sprintf("Found %d health/security issue(s). See details above.\n", issues)
	}

	return opResult{detail: report, showReport: true}, nil
}
