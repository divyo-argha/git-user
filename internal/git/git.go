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

// IsInstalled checks that git is available on PATH.
func IsInstalled() bool {
	_, err := exec.LookPath("git")
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
