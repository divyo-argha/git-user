package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runClone(args []string) error {
	var repoURL string
	var destDir string
	var asIdentity string
	var bindFlag bool

	// Parse custom flags manually to avoid strict order issues
	var passArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--as" || arg == "-a" {
			if i+1 < len(args) {
				asIdentity = args[i+1]
				i++
			} else {
				ui.Error("error: --as requires a value")
				return fmt.Errorf("missing identity value")
			}
		} else if arg == "--bind" || arg == "-b" {
			bindFlag = true
		} else {
			passArgs = append(passArgs, arg)
		}
	}

	if len(passArgs) < 1 {
		ui.Error("usage: git-user clone <repo-url> [directory] [--as <identity>] [--bind]")
		return fmt.Errorf("missing repository URL")
	}

	repoURL = passArgs[0]
	if len(passArgs) > 1 {
		destDir = passArgs[1]
	}

	// Resolve the destination directory name if not specified
	if destDir == "" {
		destDir = getRepoDirName(repoURL)
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if len(store.Users) == 0 {
		ui.Error("No registered identities found. Please run `git-user register` first.")
		return fmt.Errorf("no identities registered")
	}

	var targetUser *config.User
	if asIdentity != "" {
		targetUser = store.FindUser(asIdentity)
		if targetUser == nil {
			ui.Errorf("identity %q not found", asIdentity)
			return fmt.Errorf("identity not found")
		}
	} else {
		if ui.IsTTY() {
			var options []string
			for _, u := range store.Users {
				options = append(options, fmt.Sprintf("%s (%s)", u.Name, u.Email))
			}
			options = append(options, "Cancel")

			idx, err := ui.Select("Select identity for this cloned repository:", options)
			if err != nil {
				return err
			}
			if idx == len(options)-1 {
				ui.Info("Clone cancelled")
				return nil
			}
			targetUser = &store.Users[idx]
		} else {
			// Fallback to active global/current identity
			if store.Current != "" {
				targetUser = store.CurrentUser()
			}
			if targetUser == nil {
				targetUser = &store.Users[0] // fallback to first registered
			}
		}
	}

	ui.Info(fmt.Sprintf("Cloning repository using identity: %s (%s)...", targetUser.Name, targetUser.Email))

	// Execute git clone
	cloneArgs := []string{"clone", repoURL}
	if len(passArgs) > 1 {
		cloneArgs = append(cloneArgs, passArgs[1])
	}

	cmd := exec.Command("git", cloneArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		ui.Errorf("git clone failed: %v", err)
		return err
	}

	// Resolve absolute path to cloned repo
	absPath, err := filepath.Abs(destDir)
	if err != nil {
		ui.Errorf("failed to get absolute path of cloned repository: %v", err)
		return err
	}

	// Apply local config to the cloned repository
	if err := configureRepoLocal(absPath, targetUser); err != nil {
		ui.Warn(fmt.Sprintf("Failed to fully configure local identity in repository: %v", err))
	} else {
		ui.Success(fmt.Sprintf("Configured local identity: %s (%s)", targetUser.Name, targetUser.Email))
	}

	// Handle automatic bind-path if specified
	if bindFlag {
		if err := store.BindPathToUser(targetUser.Name, absPath); err != nil {
			ui.Errorf("binding path: %v", err)
		} else {
			if err := config.Save(store); err != nil {
				ui.Errorf("saving config: %v", err)
			} else {
				ui.Success(fmt.Sprintf("Bound directory %q to identity %q", absPath, targetUser.Name))
			}
		}
	}

	return nil
}

// getRepoDirName extracts the repository name from URL (e.g., git@github.com:foo/bar.git -> bar)
func getRepoDirName(repoURL string) string {
	trimmed := strings.TrimSuffix(repoURL, "/")
	trimmed = strings.TrimSuffix(trimmed, ".git")
	parts := strings.Split(trimmed, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		// In case of SSH like git@github.com:foo/bar
		subparts := strings.Split(last, ":")
		return subparts[len(subparts)-1]
	}
	return "repository"
}

// configureRepoLocal runs local git configuration commands inside the target repository directory
func configureRepoLocal(repoPath string, u *config.User) error {
	commands := [][]string{
		{"config", "--local", "user.name", u.Name},
		{"config", "--local", "user.email", u.Email},
	}

	if u.SSHKey != "" {
		sshVal := fmt.Sprintf("ssh -i %q -o IdentitiesOnly=yes", u.SSHKey)
		commands = append(commands, []string{"config", "--local", "core.sshCommand", sshVal})
	}

	if !u.SignDisabled && u.SignKey != "" {
		if u.SignFormat == "ssh" {
			commands = append(commands, []string{"config", "--local", "gpg.format", "ssh"})
		}
		commands = append(commands, []string{"config", "--local", "user.signingkey", u.SignKey})
		commands = append(commands, []string{"config", "--local", "commit.gpgsign", "true"})
	}

	for _, c := range commands {
		cmd := exec.Command("git", c...)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed running git %v: %w", c, err)
		}
	}

	return nil
}
