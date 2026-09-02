package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/keyring"
)

// opUninstall removes git-user's footprint from the machine. This is the TUI
// counterpart of `git-user uninstall` (internal/cli/uninstall.go), scoped
// down to what's safe to do without the CLI's follow-up interactive prompts:
// it always keeps SSH keys and the binary itself, restores the original git
// identity/config, removes git-user's own directory-binding config and
// keychain-stored passphrases, and deletes git-user's config directory. Key,
// binary, and shell-prompt-integration removal are left to the CLI, which the
// report below points to.
func opUninstall(store *config.Store) (opResult, error) {
	dataDir := filepath.Dir(config.ConfigPath())
	var report strings.Builder

	if store.Original != nil {
		if err := uninstallRestoreOriginal(store.Original); err != nil {
			fmt.Fprintf(&report, "⚠ Could not fully restore original git config: %v\n", err)
		} else {
			report.WriteString("Restored original git identity/config.\n")
		}
	} else {
		git.RemoveSSHConfig()
		git.RemoveSigningConfig()
		report.WriteString("No pre-git-user snapshot was recorded — left user.name/user.email untouched, removed git-user's sshCommand/signing config.\n")
	}

	uninstallRemoveManagedIncludeIfs()
	report.WriteString("Removed directory-binding (includeIf) git config.\n")

	for _, u := range store.Users {
		_ = keyring.DeleteKeychainPassphrase(u.Name)
	}
	report.WriteString("Removed stored keychain passphrases.\n")

	var keptKeys []string
	seen := make(map[string]bool)
	for _, u := range store.Users {
		if u.SSHKey == "" || seen[u.SSHKey] {
			continue
		}
		seen[u.SSHKey] = true
		if _, err := os.Stat(u.SSHKey); err == nil {
			keptKeys = append(keptKeys, u.SSHKey)
		}
	}
	if len(keptKeys) > 0 {
		report.WriteString("Kept your SSH keys (run 'git-user uninstall --delete-keys' from a terminal to remove them too):\n")
		for _, path := range keptKeys {
			fmt.Fprintf(&report, "  %s\n", path)
		}
	}

	// GIT_USER_CONFIG is an unvalidated override (test sandboxing today, but
	// nothing stops it being set elsewhere): if it ever pointed straight at a
	// file with no dedicated parent directory, dataDir would resolve to $HOME
	// or worse, and RemoveAll would wipe far more than git-user's own state.
	// Refuse rather than risk that.
	if !isGitUserDataDir(dataDir) {
		fmt.Fprintf(&report, "⚠ Refused to remove %s — it doesn't look like git-user's own data directory.\n", dataDir)
	} else if err := os.RemoveAll(dataDir); err != nil {
		fmt.Fprintf(&report, "⚠ Could not remove %s: %v\n", dataDir, err)
	} else {
		fmt.Fprintf(&report, "Removed %s\n", dataDir)
	}

	report.WriteString("\nNot removed automatically — clean these up yourself if applicable:\n")
	report.WriteString("  • Shell prompt integration (zsh/bash/starship/fish) — run 'git-user uninstall' from a terminal for full cleanup.\n")
	report.WriteString("  • Pre-commit hooks from 'git-user hook install' in individual repos.\n")
	report.WriteString("  • The git-user binary itself — run 'git-user uninstall --remove-binary' from a terminal, or delete it manually.\n")

	return opResult{detail: report.String(), showReport: true}, nil
}

// isGitUserDataDir reports whether dir is safe to recursively delete as
// git-user's own data directory: it must be named ".git-users" (the fixed
// name git-user itself creates it under) and must not be the user's home
// directory or filesystem root, however it was reached.
func isGitUserDataDir(dir string) bool {
	if dir == "" || dir == "." || dir == string(filepath.Separator) {
		return false
	}
	if filepath.Base(dir) != ".git-users" {
		return false
	}
	if home, err := os.UserHomeDir(); err == nil && dir == home {
		return false
	}
	return true
}

// uninstallRestoreOriginal applies a pre-git-user gitconfig snapshot back onto
// the live global git config. Mirrors restoreOriginalGitConfig in
// internal/cli/switch.go (unexported there, so duplicated here rather than
// shared across packages).
func uninstallRestoreOriginal(o *config.OriginalConfig) error {
	if err := git.Apply(o.Name, o.Email); err != nil {
		return err
	}
	var errs []error
	if o.SSHCommand != "" {
		if err := git.SetSSHCommand(o.SSHCommand); err != nil {
			errs = append(errs, fmt.Errorf("restoring core.sshCommand: %w", err))
		}
	} else {
		git.RemoveSSHConfig()
	}
	if o.SignKey != "" || o.CommitGPGSign != "" {
		format := "gpg"
		if o.SignFormat == "ssh" {
			format = "ssh"
		}
		if err := git.ConfigureSigning(o.SignKey, format); err != nil {
			errs = append(errs, fmt.Errorf("restoring signing config: %w", err))
		}
	} else {
		git.RemoveSigningConfig()
	}
	return errors.Join(errs...)
}

// uninstallRemoveManagedIncludeIfs strips only the global includeIf entries
// that point at git-user's own profile-*.gitconfig snippet files, leaving any
// unrelated includeIf rules the user configured by hand alone. Mirrors
// removeManagedIncludeIfs in internal/cli/uninstall.go.
func uninstallRemoveManagedIncludeIfs() {
	out, err := exec.Command("git", "config", "--global", "--get-regexp", `includeif\..*\.path`).Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		if strings.Contains(value, "profile-") && strings.HasSuffix(value, ".gitconfig") {
			_ = exec.Command("git", "config", "--global", "--unset-all", key).Run()
		}
	}
}
