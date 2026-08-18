package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

// runRefresh re-syncs git-user's stored state onto the live environment,
// actually fixing drift instead of only reporting it (unlike `doctor`, which
// is read-only). This is what to run when the config git-user *thinks* is
// active no longer matches what's really in ~/.gitconfig — e.g. after a
// manual edit, another tool touching global git config, or a crashed/partial
// switch.
func runRefresh(args []string) error {
	ui.Banner("GIT-USER REFRESH")
	fmt.Println()

	fixed := 0

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	ui.Info("Fixing file permissions...")
	configPath := config.ConfigPath()
	if info, statErr := os.Stat(configPath); statErr == nil {
		if pc := config.CheckFilePermissions(info.Mode()); pc.Applicable && !pc.Secure {
			if err := os.Chmod(configPath, 0600); err != nil {
				ui.Warn(fmt.Sprintf("Could not fix permissions on %s: %v", configPath, err))
			} else {
				ui.Success(fmt.Sprintf("Fixed permissions: %s → 0600", configPath))
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
					ui.Warn(fmt.Sprintf("Could not fix permissions on %s: %v", u.SSHKey, err))
				} else {
					ui.Success(fmt.Sprintf("Fixed permissions: %s → 0600", u.SSHKey))
					fixed++
				}
			}
		}
	}

	ui.Info("Re-syncing directory-binding (includeIf) config...")
	// config.Save prunes stale profile-*.gitconfig snippet files and any
	// global includeIf entries left over from a renamed/removed identity or a
	// bind-path that was cleared out of band — this is the same self-heal
	// that already runs after every normal mutation, just re-run standalone.
	if err := config.Save(store); err != nil {
		ui.Warn(fmt.Sprintf("Could not resync directory-binding config: %v", err))
	} else {
		ui.Success("Directory-binding config is in sync")
	}

	if store.Current == "" {
		ui.Info("No active identity set — nothing to re-apply to git config.")
	} else {
		user := store.FindUser(store.Current)
		if user == nil {
			ui.Warn(fmt.Sprintf("Active identity %q no longer exists in config — clearing it.", store.Current))
			store.Current = ""
			if err := config.Save(store); err != nil {
				ui.Warn(fmt.Sprintf("Could not save config: %v", err))
			}
			fixed++
		} else {
			ui.Info(fmt.Sprintf("Re-applying identity %q to git config...", user.Name))
			before := gitConfigFingerprint()

			if err := git.Apply(user.Name, user.Email); err != nil {
				ui.Warn(fmt.Sprintf("Could not re-apply name/email: %v", err))
			}
			if err := applyUserSSHConfig(user, false); err != nil {
				ui.Warn(fmt.Sprintf("Could not re-apply SSH config: %v", err))
			}
			if !user.SignDisabled && user.SignKey != "" {
				if err := git.ConfigureSigning(user.SignKey, user.SignFormat); err != nil {
					ui.Warn(fmt.Sprintf("Could not re-apply signing config: %v", err))
				}
			} else {
				git.RemoveSigningConfig()
			}

			if before != gitConfigFingerprint() {
				ui.Success(fmt.Sprintf("Fixed: git config had drifted from identity %q — corrected.", user.Name))
				fixed++
			} else {
				ui.Success(fmt.Sprintf("Git config already matched identity %q — nothing to fix.", user.Name))
			}
		}
	}

	fmt.Println()
	ui.Divider()
	if fixed == 0 {
		ui.Success("Nothing to fix — git-user's config is already healthy.")
	} else {
		ui.Success(fmt.Sprintf("Fixed %d issue(s).", fixed))
	}
	ui.Info("Run 'git-user doctor' for a full read-only diagnostic, including live SSH connectivity checks.")

	return nil
}

// gitConfigFingerprint captures the slice of global git config that
// git-user manages, so runRefresh can tell whether re-applying an identity
// actually changed anything or the config was already in sync.
func gitConfigFingerprint() string {
	return strings.Join([]string{
		git.CurrentName(),
		git.CurrentEmail(),
		git.CurrentSSHCommand(),
		git.CurrentSigningKey(),
		git.CurrentSignFormat(),
		git.CurrentCommitGPGSign(),
	}, "\x00")
}
