package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ParseConfigGetRegexpLine parses one line of `git config --get-regexp` output into its key/value pair.
func ParseConfigGetRegexpLine(line string) (key, value string, ok bool) {
	line = strings.TrimRight(line, "\r")
	if line == "" {
		return "", "", false
	}
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

// RemoveManagedIncludeIfs removes global includeIf entries pointing to git-user snippet files.
func RemoveManagedIncludeIfs() {
	out, err := exec.Command("git", "config", "--global", "--get-regexp", `includeif\..*\.path`).Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		key, value, ok := ParseConfigGetRegexpLine(line)
		if !ok {
			continue
		}
		if strings.Contains(value, "profile-") && strings.HasSuffix(value, ".gitconfig") {
			_ = exec.Command("git", "config", "--global", "--unset-all", key).Run()
		}
	}
}

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
	unsetConfig("user.name", local)
	unsetConfig("user.email", local)
	unsetConfig("core.sshCommand", local)
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

// IsIdentityInSync reports whether the resolved git config matches the given name and email.
func IsIdentityInSync(name, email string) bool {
	if name == "" && email == "" {
		return false
	}
	return CurrentName() == name && CurrentEmail() == email
}

func CurrentGlobalEmail() string {
	out, _ := getConfig("user.email", false)
	return out
}

// CurrentLocalName returns user.name set at --local scope only.
func CurrentLocalName() string {
	out, _ := getConfig("user.name", true)
	return out
}

// CurrentLocalEmail returns user.email set at --local scope only.
func CurrentLocalEmail() string {
	out, _ := getConfig("user.email", true)
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
	return setConfig("core.sshCommand", IdentitySSHCommand("", keyPath), local)
}

// SSHQuote formats a path for interpolation into an SSH command (e.g. core.sshCommand).
func SSHQuote(s string) string {
	if runtime.GOOS == "windows" {
		clean := filepath.ToSlash(s)
		return `"` + strings.ReplaceAll(clean, `"`, `\"`) + `"`
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// IdentitySSHCommand returns the ssh command configured with the identity's key.
func IdentitySSHCommand(sshCommand, sshKey string) string {
	if sshCommand != "" {
		return sshCommand
	}
	if sshKey != "" {
		return fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", SSHQuote(sshKey))
	}
	return ""
}

func gitBinary() string {
	return findWindowsGit()
}

// BinaryPath returns the resolved path to the git executable.
func BinaryPath() string {
	return gitBinary()
}

func gitCmd(args ...string) *exec.Cmd {
	return exec.Command(gitBinary(), args...)
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
	unsetConfig("core.sshCommand", local)
	return nil
}

func ApplyIdentitySSHConfig(sshCommand, sshKey string, local bool) error {
	if sshCommand != "" {
		return SetSSHCommandScope(sshCommand, local)
	}
	if sshKey != "" {
		return ConfigureSSHScope(sshKey, local)
	}
	return RemoveSSHConfigScope(local)
}

func ConfigureSigning(key, format string) error {
	return ConfigureSigningScope(key, format, false)
}

func ConfigureSigningScope(key, format string, local bool) error {
	if format == "ssh" {
		if err := setConfig("gpg.format", "ssh", local); err != nil {
			return err
		}
	} else if format == "gpg" {
		unsetConfig("gpg.format", local)
	}

	if err := setConfig("user.signingkey", key, local); err != nil {
		return err
	}
	if err := setConfig("commit.gpgsign", "true", local); err != nil {
		return err
	}
	return nil
}

// ConfigureAskpass points core.askpass at cmd (the git-user binary plus
// arguments identifying which identity to answer for — see
// ConfigureAskpass configures core.askpass for HTTPS token helper invocation.
func ConfigureAskpass(cmd string) error {
	return ConfigureAskpassScope(cmd, false)
}

func ConfigureAskpassScope(cmd string, local bool) error {
	return setConfig("core.askpass", cmd, local)
}

func RemoveAskpassConfig() {
	RemoveAskpassConfigScope(false)
}

func RemoveAskpassConfigScope(local bool) {
	unsetConfig("core.askpass", local)
}

func CurrentAskpass() string {
	out, _ := getConfigResolved("core.askpass")
	return out
}

func RemoveSigningConfig() {
	RemoveSigningConfigScope(false)
}

func RemoveSigningConfigScope(local bool) {
	unsetConfig("user.signingkey", local)
	unsetConfig("commit.gpgsign", local)
	unsetConfig("gpg.format", local)
}

func IsInstalled() bool {
	bin := gitBinary()
	if bin != "git" {
		return true
	}
	_, err := exec.LookPath("git")
	return err == nil
}

func setConfig(key, value string, local bool) error {
	if IsInstalled() {
		flag := "--global"
		if local {
			flag = "--local"
		}
		cmd := gitCmd("config", flag, "--replace-all", key, value)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git config %s --replace-all %s: %w\n%s", flag, key, err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	cfgPath := GlobalConfigPath()
	if local {
		cfgPath = RepoConfigPath()
	}
	if cfgPath == "" {
		return fmt.Errorf("git is not installed and could not locate config path")
	}
	return setDirectConfig(cfgPath, key, value)
}

func unsetConfig(key string, local bool) {
	if IsInstalled() {
		flag := "--global"
		if local {
			flag = "--local"
		}
		_ = gitCmd("config", flag, "--unset-all", key).Run()
		return
	}
	cfgPath := GlobalConfigPath()
	if local {
		cfgPath = RepoConfigPath()
	}
	if cfgPath != "" {
		_ = unsetDirectConfig(cfgPath, key)
	}
}

func getConfig(key string, local bool) (string, error) {
	if IsInstalled() {
		flag := "--global"
		if local {
			flag = "--local"
		}
		cmd := gitCmd("config", flag, key)
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}
	cfgPath := GlobalConfigPath()
	if local {
		cfgPath = RepoConfigPath()
	}
	return getDirectConfig(cfgPath, key)
}

func getConfigResolved(key string) (string, error) {
	if IsInstalled() {
		cmd := gitCmd("config", key)
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}
	if loc := RepoConfigPath(); loc != "" {
		if val, err := getDirectConfig(loc, key); err == nil && val != "" {
			return val, nil
		}
	}
	return getDirectConfig(GlobalConfigPath(), key)
}

func HasLocalOverride() bool {
	name, _ := getConfig("user.name", true)
	email, _ := getConfig("user.email", true)
	key, _ := getConfig("user.signingkey", true)
	return name != "" || email != "" || key != ""
}

func IsInRepo() bool {
	if IsInstalled() {
		cmd := gitCmd("rev-parse", "--git-dir")
		return cmd.Run() == nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return false
	}
	for {
		if fi, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return fi.IsDir() || fi.Mode().IsRegular()
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return false
}

func GetRemoteURL(remote string) (string, error) {
	cmd := gitCmd("remote", "get-url", "--", remote)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetPushRemoteURL returns the resolved push URL for a remote.
func GetPushRemoteURL(remote string) (string, error) {
	cmd := gitCmd("remote", "get-url", "--push", "--", remote)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ConfigurePushInsteadOf configures Git to route pushes to https://<host>/ over SSH (git@<host>:).
func ConfigurePushInsteadOf(host string, local bool) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	sshBase := fmt.Sprintf("git@%s:", host)
	httpsBase := fmt.Sprintf("https://%s/", host)
	return setConfig(fmt.Sprintf("url.%s.pushInsteadOf", sshBase), httpsBase, local)
}

// RemovePushInsteadOf removes a pushInsteadOf rewrite for a given host.
func RemovePushInsteadOf(host string, local bool) {
	sshBase := fmt.Sprintf("git@%s:", host)
	unsetConfig(fmt.Sprintf("url.%s.pushInsteadOf", sshBase), local)
}

// DefaultPushInsteadOfHosts returns the standard Git hosting platforms.
func DefaultPushInsteadOfHosts() []string {
	return []string{"github.com", "gitlab.com", "bitbucket.org"}
}

// ConfigureInsteadOf routes all operations for https://<host>/ over SSH (git@<host>:).
func ConfigureInsteadOf(host string, local bool) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	sshBase := fmt.Sprintf("git@%s:", host)
	httpsBase := fmt.Sprintf("https://%s/", host)
	return setConfig(fmt.Sprintf("url.%s.insteadOf", sshBase), httpsBase, local)
}

// RemoveInsteadOf removes an insteadOf rewrite for a given host.
func RemoveInsteadOf(host string, local bool) {
	sshBase := fmt.Sprintf("git@%s:", host)
	unsetConfig(fmt.Sprintf("url.%s.insteadOf", sshBase), local)
}

// HTTPSRemoteHost extracts the host from an https:// remote URL (e.g.
// "https://github.com/owner/repo.git" -> "github.com"), stripping any
// embedded credentials. Returns "" if url isn't an https:// URL.
func HTTPSRemoteHost(httpsURL string) string {
	if !strings.HasPrefix(httpsURL, "https://") {
		return ""
	}
	rest := strings.TrimPrefix(httpsURL, "https://")
	host := rest
	if idx := strings.Index(rest, "/"); idx >= 0 {
		host = rest[:idx]
	}
	if atIdx := strings.LastIndex(host, "@"); atIdx >= 0 {
		host = host[atIdx+1:]
	}
	return host
}

// HasHTTPSPushRemotes reports whether any remote in the current repository
// resolves its push URL to HTTPS (meaning pushes are not protected by SSH).
func HasHTTPSPushRemotes() bool {
	if !IsInRepo() {
		return false
	}
	remotes, err := ListRemotes()
	if err != nil || len(remotes) == 0 {
		return false
	}
	for _, remote := range remotes {
		pushURL, err := GetPushRemoteURL(remote)
		if err == nil && strings.HasPrefix(pushURL, "https://") {
			return true
		}
		if err != nil {
			fetchURL, fErr := GetRemoteURL(remote)
			if fErr == nil && strings.HasPrefix(fetchURL, "https://") {
				return true
			}
		}
	}
	return false
}

func SetRemoteURL(remote, url string) error {
	cmd := gitCmd("remote", "set-url", "--", remote, url)
	return cmd.Run()
}

type RemoteConversionResult struct {
	Remote        string
	OldURL        string
	NewURL        string
	Converted     bool
	ConvertFailed bool
	UpdateFailed  bool
}

func ConvertRemotesToSSH() ([]RemoteConversionResult, error) {
	if !IsInstalled() {
		return nil, fmt.Errorf("git is not installed")
	}
	if !IsInRepo() {
		return nil, fmt.Errorf("not in a git repository")
	}
	remotes, err := ListRemotes()
	if err != nil || len(remotes) == 0 {
		return nil, fmt.Errorf("no remotes found")
	}

	var results []RemoteConversionResult
	for _, remote := range remotes {
		url, err := GetRemoteURL(remote)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(url, "https://") {
			continue
		}

		sshURL, ok := ConvertHTTPSToSSH(url)
		if !ok {
			results = append(results, RemoteConversionResult{Remote: remote, OldURL: url, ConvertFailed: true})
			continue
		}

		if err := SetRemoteURL(remote, sshURL); err != nil {
			results = append(results, RemoteConversionResult{Remote: remote, OldURL: url, NewURL: sshURL, UpdateFailed: true})
			continue
		}

		results = append(results, RemoteConversionResult{Remote: remote, OldURL: url, NewURL: sshURL, Converted: true})
	}
	return results, nil
}

func ListRemotes() ([]string, error) {
	cmd := gitCmd("remote")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var remotes []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			remotes = append(remotes, line)
		}
	}
	return remotes, nil
}

func HasHTTPSRemotes() bool {
	if !IsInRepo() {
		return false
	}
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

	// Strip embedded credentials (user:token@host → host)
	if atIdx := strings.LastIndex(host, "@"); atIdx >= 0 {
		host = host[atIdx+1:]
	}

	return fmt.Sprintf("git@%s:%s.git", host, path), true
}

// RepoConfigPath resolves the path to the current repository's .git/config file,
// correctly navigating parent directories and following gitdir pointers in worktrees/submodules.
func RepoConfigPath() string {
	root, err := RepoRoot()
	if err != nil || root == "" {
		return ""
	}
	gitEntry := filepath.Join(root, ".git")
	fi, err := os.Stat(gitEntry)
	if err != nil {
		return ""
	}
	if fi.IsDir() {
		return filepath.Join(gitEntry, "config")
	}
	// In git worktrees and submodules, .git is a regular file containing "gitdir: <path>"
	data, err := os.ReadFile(gitEntry)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	if strings.HasPrefix(line, "gitdir:") {
		target := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
		if !filepath.IsAbs(target) {
			target = filepath.Join(root, target)
		}
		cfg := filepath.Join(target, "config")
		if _, err := os.Stat(cfg); err == nil {
			return cfg
		}
		commondir := filepath.Join(target, "commondir")
		if cData, err := os.ReadFile(commondir); err == nil {
			cTarget := strings.TrimSpace(string(cData))
			if !filepath.IsAbs(cTarget) {
				cTarget = filepath.Join(target, cTarget)
			}
			cCfg := filepath.Join(cTarget, "config")
			if _, err := os.Stat(cCfg); err == nil {
				return cCfg
			}
		}
		return cfg
	}
	return ""
}

// CurrentBranch returns the name of the currently checked out branch.
func CurrentBranch() string {
	if IsInstalled() {
		out, err := gitCmd("rev-parse", "--abbrev-ref", "HEAD").Output()
		if err == nil {
			branch := strings.TrimSpace(string(out))
			if branch != "" && branch != "HEAD" {
				return branch
			}
		}
	}
	root, err := RepoRoot()
	if err != nil || root == "" {
		return ""
	}
	gitEntry := filepath.Join(root, ".git")
	headPath := filepath.Join(gitEntry, "HEAD")
	if fi, err := os.Stat(gitEntry); err == nil && !fi.IsDir() {
		if data, err := os.ReadFile(gitEntry); err == nil {
			line := strings.TrimSpace(string(data))
			if strings.HasPrefix(line, "gitdir:") {
				target := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
				if !filepath.IsAbs(target) {
					target = filepath.Join(root, target)
				}
				headPath = filepath.Join(target, "HEAD")
			}
		}
	}
	if data, err := os.ReadFile(headPath); err == nil {
		line := strings.TrimSpace(string(data))
		if strings.HasPrefix(line, "ref: refs/heads/") {
			return strings.TrimPrefix(line, "ref: refs/heads/")
		}
	}
	return ""
}

func RepoRoot() (string, error) {
	if IsInstalled() {
		out, err := gitCmd("rev-parse", "--show-toplevel").Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		gitEntry := filepath.Join(dir, ".git")
		if fi, err := os.Stat(gitEntry); err == nil {
			if fi.IsDir() || fi.Mode().IsRegular() {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("not in a git repository")
}

// CurrentRepoName returns the directory name of the current git repository root.
func CurrentRepoName() string {
	root, err := RepoRoot()
	if err != nil || root == "" {
		return ""
	}
	return filepath.Base(filepath.Clean(root))
}
