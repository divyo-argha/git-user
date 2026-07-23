package git

import (
	"fmt"
	"os/exec"
	"strings"
)

func Apply(name, email string) error {
	return ApplyScope(name, email, false)
}

func ApplyScope(name, email string, local bool) error {
	if err := setConfig("user.name", name, local); err != nil {
		return err
	}
	if err := setConfig("user.email", email, local); err != nil {
		return err
	}
	return nil
}

// ClearIdentity removes user.name, user.email, and core.sshCommand from global gitconfig.
func ClearIdentity() {
	ClearIdentityScope(false)
}

func ClearIdentityScope(local bool) {
	flag := "--global"
	if local {
		flag = "--local"
	}
	exec.Command("git", "config", flag, "--unset-all", "user.name").Run()
	exec.Command("git", "config", flag, "--unset-all", "user.email").Run()
	exec.Command("git", "config", flag, "--unset-all", "core.sshCommand").Run()
	RemoveSigningConfigScope(local)
}

func CurrentName() string {
	out, _ := getConfigResolved("user.name")
	return out
}

func CurrentGlobalName() string {
	out, _ := getConfig("user.name", false)
	return out
}

func CurrentEmail() string {
	out, _ := getConfigResolved("user.email")
	return out
}

func CurrentGlobalEmail() string {
	out, _ := getConfig("user.email", false)
	return out
}

func CurrentSSHCommand() string {
	out, _ := getConfigResolved("core.sshCommand")
	return out
}

func CurrentGlobalSSHCommand() string {
	out, _ := getConfig("core.sshCommand", false)
	return out
}

func CurrentSigningKey() string {
	out, _ := getConfigResolved("user.signingkey")
	return out
}

func CurrentGlobalSigningKey() string {
	out, _ := getConfig("user.signingkey", false)
	return out
}

func CurrentSignFormat() string {
	out, _ := getConfigResolved("gpg.format")
	return out
}

func CurrentGlobalSignFormat() string {
	out, _ := getConfig("gpg.format", false)
	return out
}

func CurrentCommitGPGSign() string {
	out, _ := getConfigResolved("commit.gpgsign")
	return out
}

func CurrentGlobalCommitGPGSign() string {
	out, _ := getConfig("commit.gpgsign", false)
	return out
}

func ConfigureSSH(keyPath string) error {
	return ConfigureSSHScope(keyPath, false)
}

func ConfigureSSHScope(keyPath string, local bool) error {
	val := fmt.Sprintf("ssh -i %q -o IdentitiesOnly=yes", keyPath)
	return setConfig("core.sshCommand", val, local)
}

func SetSSHCommand(val string) error {
	return SetSSHCommandScope(val, false)
}

func SetSSHCommandScope(val string, local bool) error {
	return setConfig("core.sshCommand", val, local)
}

func RemoveSSHConfig() error {
	return RemoveSSHConfigScope(false)
}

func RemoveSSHConfigScope(local bool) error {
	flag := "--global"
	if local {
		flag = "--local"
	}
	cmd := exec.Command("git", "config", flag, "--unset-all", "core.sshCommand")
	_ = cmd.Run()
	return nil
}

func ConfigureSigning(key, format string) error {
	return ConfigureSigningScope(key, format, false)
}

func ConfigureSigningScope(key, format string, local bool) error {
	flag := "--global"
	if local {
		flag = "--local"
	}
	if format == "ssh" {
		if err := setConfig("gpg.format", "ssh", local); err != nil {
			return err
		}
	} else if format == "gpg" {
		exec.Command("git", "config", flag, "--unset-all", "gpg.format").Run()
	}

	if err := setConfig("user.signingkey", key, local); err != nil {
		return err
	}
	if err := setConfig("commit.gpgsign", "true", local); err != nil {
		return err
	}
	return nil
}

func RemoveSigningConfig() {
	RemoveSigningConfigScope(false)
}

func RemoveSigningConfigScope(local bool) {
	flag := "--global"
	if local {
		flag = "--local"
	}
	exec.Command("git", "config", flag, "--unset-all", "user.signingkey").Run()
	exec.Command("git", "config", flag, "--unset-all", "commit.gpgsign").Run()
	exec.Command("git", "config", flag, "--unset-all", "gpg.format").Run()
}

func IsInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func setConfig(key, value string, local bool) error {
	flag := "--global"
	if local {
		flag = "--local"
	}
	cmd := exec.Command("git", "config", flag, "--replace-all", key, value)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git config %s --replace-all %s: %w\n%s", flag, key, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func getConfig(key string, local bool) (string, error) {
	flag := "--global"
	if local {
		flag = "--local"
	}
	cmd := exec.Command("git", "config", flag, key)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getConfigResolved(key string) (string, error) {
	cmd := exec.Command("git", "config", key)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func HasLocalOverride() bool {
	out, err := getConfig("user.name", true)
	return err == nil && out != ""
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

// HasHTTPSRemotes returns true if the current repo has at least one remote
// whose URL starts with "https://". Returns false when not in a repo, when
// there are no remotes, or when all remotes already use SSH.
func HasHTTPSRemotes() bool {
	remotes, err := ListRemotes()
	if err != nil || len(remotes) == 0 {
		return false
	}
	for _, remote := range remotes {
		url, err := GetRemoteURL(remote)
		if err != nil {
			continue
		}
		if strings.HasPrefix(url, "https://") {
			return true
		}
	}
	return false
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
