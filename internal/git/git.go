package git

import (
	"fmt"
	"os/exec"
	"strings"
)

func Apply(name, email string) error {
	if err := setConfig("user.name", name); err != nil {
		return err
	}
	if err := setConfig("user.email", email); err != nil {
		return err
	}
	return nil
}

// ClearIdentity removes user.name, user.email, and core.sshCommand from global gitconfig.
func ClearIdentity() {
	exec.Command("git", "config", "--global", "--unset-all", "user.name").Run()
	exec.Command("git", "config", "--global", "--unset-all", "user.email").Run()
	exec.Command("git", "config", "--global", "--unset-all", "core.sshCommand").Run()
	RemoveSigningConfig()
}

func CurrentName() string {
	out, _ := getConfig("user.name")
	return out
}

func CurrentEmail() string {
	out, _ := getConfig("user.email")
	return out
}

func CurrentSSHCommand() string {
	out, _ := getConfig("core.sshCommand")
	return out
}

func CurrentSigningKey() string {
	out, _ := getConfig("user.signingkey")
	return out
}

func CurrentSignFormat() string {
	out, _ := getConfig("gpg.format")
	return out
}

func CurrentCommitGPGSign() string {
	out, _ := getConfig("commit.gpgsign")
	return out
}

func ConfigureSSH(keyPath string) error {
	val := fmt.Sprintf("ssh -i %q -o IdentitiesOnly=yes", keyPath)
	return setConfig("core.sshCommand", val)
}

func SetSSHCommand(val string) error {
	return setConfig("core.sshCommand", val)
}

func RemoveSSHConfig() error {
	cmd := exec.Command("git", "config", "--global", "--unset-all", "core.sshCommand")
	_ = cmd.Run()
	return nil
}

func ConfigureSigning(key, format string) error {
	if format == "ssh" {
		if err := setConfig("gpg.format", "ssh"); err != nil {
			return err
		}
	} else if format == "gpg" {
		// explicitly unset gpg.format so it falls back to default gpg
		exec.Command("git", "config", "--global", "--unset-all", "gpg.format").Run()
	}

	if err := setConfig("user.signingkey", key); err != nil {
		return err
	}
	if err := setConfig("commit.gpgsign", "true"); err != nil {
		return err
	}
	return nil
}

func RemoveSigningConfig() {
	exec.Command("git", "config", "--global", "--unset-all", "user.signingkey").Run()
	exec.Command("git", "config", "--global", "--unset-all", "commit.gpgsign").Run()
	exec.Command("git", "config", "--global", "--unset-all", "gpg.format").Run()
}

func IsInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func setConfig(key, value string) error {
	cmd := exec.Command("git", "config", "--global", "--replace-all", key, value)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git config --global --replace-all %s: %w\n%s", key, err, strings.TrimSpace(string(out)))
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

func IsInRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

func GetRemoteURL(remote string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", remote)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func SetRemoteURL(remote, url string) error {
	cmd := exec.Command("git", "remote", "set-url", remote, url)
	return cmd.Run()
}

func ListRemotes() ([]string, error) {
	cmd := exec.Command("git", "remote")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var remotes []string
	for _, line := range lines {
		if line != "" {
			remotes = append(remotes, line)
		}
	}
	return remotes, nil
}

func ConvertHTTPSToSSH(httpsURL string) (string, bool) {
	if !strings.HasPrefix(httpsURL, "https://") {
		return httpsURL, false
	}
	
	httpsURL = strings.TrimPrefix(httpsURL, "https://")
	httpsURL = strings.TrimSuffix(httpsURL, ".git")
	
	parts := strings.SplitN(httpsURL, "/", 2)
	if len(parts) != 2 {
		return "", false
	}
	
	host := parts[0]
	path := parts[1]
	
	return fmt.Sprintf("git@%s:%s.git", host, path), true
}
