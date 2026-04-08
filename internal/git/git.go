package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Apply sets the global git user.name and user.email.
func Apply(name, email string) error {
	if err := setConfig("user.name", name); err != nil {
		return err
	}
	if err := setConfig("user.email", email); err != nil {
		return err
	}
	return nil
}

// ApplySigning sets the global git signing configuration.
func ApplySigning(key, method string) error {
	if err := setConfig("user.signingkey", key); err != nil {
		return err
	}
	if method == "ssh" {
		if err := setConfig("gpg.format", "ssh"); err != nil {
			return err
		}
	} else {
		// Default or explicit gpg
		if err := setConfig("gpg.format", "openpgp"); err != nil {
			return err
		}
	}
	if err := setConfig("commit.gpgsign", "true"); err != nil {
		return err
	}
	return nil
}

// RemoveSigningConfig unsets the global git signing configuration.
func RemoveSigningConfig() error {
	_ = unsetConfig("user.signingkey")
	_ = unsetConfig("gpg.format")
	_ = setConfig("commit.gpgsign", "false")
	return nil
}

// CurrentName returns the global git user.name (empty string if unset).
func CurrentName() string {
	out, _ := getConfig("user.name")
	return out
}

// CurrentEmail returns the global git user.email (empty string if unset).
func CurrentEmail() string {
	out, _ := getConfig("user.email")
	return out
}

// ConfigureSSH sets the global core.sshCommand to use a specific key.
func ConfigureSSH(keyPath string) error {
	// Quote the path to handle spaces and prevent injection.
	val := fmt.Sprintf("ssh -i %q -o IdentitiesOnly=yes", keyPath)
	return setConfig("core.sshCommand", val)
}

// RemoveSSHConfig unsets the global core.sshCommand.
func RemoveSSHConfig() error {
	cmd := exec.Command("git", "config", "--global", "--unset", "core.sshCommand")
	// ignore error if it was already unset
	_ = cmd.Run()
	return nil
}

// IsInstalled checks that git is available on PATH.
func IsInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// IsInGitRepo checks if the current working directory is inside a git repository.
func IsInGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

func setConfig(key, value string) error {
	cmd := exec.Command("git", "config", "--global", key, value)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git config --global %s: %w\n%s", key, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func getConfig(key string) (string, error) {
	cmd := exec.Command("git", "config", "--global", key)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetLocalConfig returns the local (repository-level) config value.
func GetLocalConfig(key string) (string, error) {
	cmd := exec.Command("git", "config", "--local", key)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetRemoteURL returns the URL of the 'origin' remote if it exists.
func GetRemoteURL() (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// DetectPlatformFromURL parses a git remote URL and returns the platform name and repository identifier.
func DetectPlatformFromURL(url string) (platform, repo string) {
	if url == "" {
		return "", ""
	}

	// Examples:
	// https://github.com/divyo/git-user.git
	// git@github.com:divyo/git-user.git

	url = strings.TrimSuffix(url, ".git")

	if strings.Contains(url, "github.com") {
		platform = "GitHub"
		parts := strings.Split(url, "github.com")
		if len(parts) > 1 {
			repo = strings.Trim(parts[1], ":/")
		}
	} else if strings.Contains(url, "gitlab.com") {
		platform = "GitLab"
		parts := strings.Split(url, "gitlab.com")
		if len(parts) > 1 {
			repo = strings.Trim(parts[1], ":/")
		}
	} else if strings.Contains(url, "bitbucket.org") {
		platform = "Bitbucket"
		parts := strings.Split(url, "bitbucket.org")
		if len(parts) > 1 {
			repo = strings.Trim(parts[1], ":/")
		}
	}

	return platform, repo
}

func unsetConfig(key string) error {
	cmd := exec.Command("git", "config", "--global", "--unset", key)
	return cmd.Run()
}
