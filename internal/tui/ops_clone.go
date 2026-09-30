package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

// ── Clone ─────────────────────────────────────────────────────────────────────

// opClone clones a repository and configures the local identity for it.
func opClone(store *config.Store, repoURL, destDir, identity string, bind bool) (opResult, error) {
	if !git.IsInstalled() {
		return opResult{}, fmt.Errorf("git is not installed or not on PATH")
	}
	if repoURL == "" {
		return opResult{}, fmt.Errorf("repository URL is required")
	}
	user := store.FindUser(identity)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", identity)
	}
	if destDir == "" {
		destDir = repoDirName(repoURL)
	}

	args := []string{"clone", repoURL}
	if destDir != "" {
		args = append(args, destDir)
	}
	// The clone itself must authenticate as the chosen identity: there is no
	// repo config yet, so pin the identity's key through GIT_SSH_COMMAND
	// (otherwise a private repo is fetched with whatever key the agent offers).
	var env []string
	if sshCmd := git.IdentitySSHCommand(user.SSHCommand, user.SSHKey); sshCmd != "" {
		env = append(env, "GIT_SSH_COMMAND="+sshCmd)
	}
	out, err := runCapturedEnv("", env, "git", args...)
	if err != nil {
		return opResult{}, fmt.Errorf("git clone failed: %v\n%s", err, strings.TrimSpace(out))
	}

	absPath, err := filepath.Abs(destDir)
	if err != nil {
		return opResult{}, fmt.Errorf("resolving clone path: %v", err)
	}

	report := fmt.Sprintf("Cloned repository using identity %q (%s)\n", user.Name, user.Email)
	if err := configureRepoLocal(absPath, user); err != nil {
		report += fmt.Sprintf("⚠ Could not configure local identity in repository: %v\n", err)
		return opResult{detail: report, showReport: true}, nil
	}
	report += fmt.Sprintf("Configured local identity in %s\n", absPath)

	if bind {
		if err := store.BindPathToUser(user.Name, absPath); err != nil {
			report += fmt.Sprintf("⚠ Could not bind directory: %v\n", err)
		} else {
			if err := config.Save(store); err != nil {
				report += fmt.Sprintf("⚠ Saving config failed: %v\n", err)
			} else {
				report += fmt.Sprintf("Bound directory %q to identity %q\n", absPath, user.Name)
			}
		}
	}

	return opResult{detail: report, showReport: true}, nil
}

// repoDirName extracts the repository name from a clone URL.
func repoDirName(repoURL string) string {
	trimmed := strings.TrimSuffix(repoURL, "/")
	trimmed = strings.TrimSuffix(trimmed, ".git")
	parts := strings.Split(trimmed, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if sub := strings.Split(last, ":"); len(sub) > 1 {
			return sub[len(sub)-1]
		}
		return last
	}
	return "repository"
}

// configureRepoLocal applies local git configuration for the identity,
// mirroring what opSwitch/opBind apply globally: identity, SSH command
// (custom override or derived key path), commit signing, and any custom
// config keys — previously this only set name/email/key/signing and silently
// dropped a custom SSHCommand override and CustomConfig entries.
func configureRepoLocal(repoPath string, u *config.User) error {
	commands := [][]string{
		{"config", "--local", "user.name", u.Name},
		{"config", "--local", "user.email", u.Email},
	}

	if sshVal := git.IdentitySSHCommand(u.SSHCommand, u.SSHKey); sshVal != "" {
		commands = append(commands, []string{"config", "--local", "core.sshCommand", sshVal})
	}

	if !u.SignDisabled && u.SignKey != "" {
		if u.SignFormat == "ssh" {
			commands = append(commands, []string{"config", "--local", "gpg.format", "ssh"})
		}
		commands = append(commands, []string{"config", "--local", "user.signingkey", u.SignKey})
		commands = append(commands, []string{"config", "--local", "commit.gpgsign", "true"})
	}

	for k, v := range u.CustomConfig {
		commands = append(commands, []string{"config", "--local", k, v})
	}

	for _, c := range commands {
		if out, err := runCaptured(repoPath, "git", c...); err != nil {
			return fmt.Errorf("failed running git %v: %w\n%s", c, err, strings.TrimSpace(out))
		}
	}
	return nil
}
