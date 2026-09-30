package core

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	gituserssh "github.com/divyo-argha/git-user/internal/ssh"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"github.com/divyo-argha/git-user/internal/version"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// ── Store Commands ────────────────────────────────────────────────────────────

// RefreshStoreCmd reloads the config from disk.
func RefreshStoreCmd() tea.Cmd {
	return func() tea.Msg {
		store, err := config.Load()
		return StoreRefreshedMsg{Store: store, Err: err}
	}
}

// CheckSyncStatusCmd checks whether the active identity is applied to the git
// config. It never runs git commands for an empty/unset active identity.
func CheckSyncStatusCmd(store *config.Store) tea.Cmd {
	return func() tea.Msg {
		if store == nil || store.Current == "" {
			return SyncStatusMsg{InSync: true}
		}
		u := store.CurrentUser()
		if u == nil {
			return SyncStatusMsg{InSync: true}
		}
		return SyncStatusMsg{InSync: git.IsIdentityInSync(u.Name, u.Email)}
	}
}

// ── Agent Commands ────────────────────────────────────────────────────────────

// CheckAgentCmd checks SSH agent connectivity and loaded key count.
func CheckAgentCmd() tea.Cmd {
	return func() tea.Msg {
		socket := os.Getenv("SSH_AUTH_SOCK")
		if socket == "" {
			if runtime.GOOS == "windows" {
				out, err := exec.Command("ssh-add", "-l").CombinedOutput()
				if err == nil {
					lines := strings.Split(strings.TrimSpace(string(out)), "\n")
					count := 0
					for _, l := range lines {
						if strings.TrimSpace(l) != "" {
							count++
						}
					}
					return AgentStatusMsg{Connected: true, KeyCount: count}
				}
				outStr := strings.ToLower(string(out))
				if strings.Contains(outStr, "no identities") || strings.Contains(outStr, "empty") {
					return AgentStatusMsg{Connected: true, KeyCount: 0}
				}
			}
			return AgentStatusMsg{Connected: false}
		}

		conn, err := net.Dial("unix", socket)
		if err != nil {
			return AgentStatusMsg{Connected: false, Err: err}
		}
		defer conn.Close()

		client := agent.NewClient(conn)
		keys, err := client.List()
		if err != nil {
			return AgentStatusMsg{Connected: true, KeyCount: 0, Err: err}
		}

		return AgentStatusMsg{Connected: true, KeyCount: len(keys)}
	}
}

// ── Toast Commands ────────────────────────────────────────────────────────────

// ToastTimerCmd returns a command that waits for the given duration then sends ToastExpiredMsg.
func ToastTimerCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return ToastExpiredMsg{}
	})
}

// ShowToastCmd creates a toast notification with auto-dismiss.
func ShowToastCmd(text string, style theme.ToastStyleKind, duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		return ToastMsg{Text: text, Style: style, Duration: duration}
	}
}

// ── SSH Key Utility Commands ──────────────────────────────────────────────────

// CheckKeyLoadedCmd checks if a specific key is loaded in the SSH agent.
func CheckKeyLoadedCmd(keyPath string) tea.Cmd {
	return func() tea.Msg {
		loaded := isKeyLoaded(keyPath)
		return KeyLoadedMsg{Path: keyPath, Loaded: loaded}
	}
}

// KeyLoadedMsg reports whether a key is loaded in the agent.
type KeyLoadedMsg struct {
	Path   string
	Loaded bool
}

// isKeyLoaded checks if the given SSH key is loaded in the agent.
func isKeyLoaded(keyPath string) bool {
	return gituserssh.IsSSHKeyLoaded(keyPath)
}

// CheckKeyPassphraseCmd checks if an SSH key is passphrase-protected.
func CheckKeyPassphraseCmd(keyPath string) tea.Cmd {
	return func() tea.Msg {
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return KeyPassphraseMsg{Path: keyPath, Err: err}
		}

		_, err = ssh.ParseRawPrivateKey(data)
		if err == nil {
			return KeyPassphraseMsg{Path: keyPath, Protected: false}
		}

		if _, ok := err.(*ssh.PassphraseMissingError); ok {
			return KeyPassphraseMsg{Path: keyPath, Protected: true}
		}

		return KeyPassphraseMsg{Path: keyPath, Err: err}
	}
}

// KeyPassphraseMsg reports whether a key is passphrase-protected.
type KeyPassphraseMsg struct {
	Path      string
	Protected bool
	Err       error
}

// CheckPlatformConnectionCmd runs ssh -T against a Git host and returns auth status.
func CheckPlatformConnectionCmd(profileName, keyPath, platform, host string, successPatterns []string) tea.Cmd {
	return func() tea.Msg {
		res := gituserssh.CheckPlatformConnection(keyPath, platform, host, successPatterns)
		return PlatformConnectionMsg{
			ProfileName: profileName,
			Platform:    res.Platform,
			Status:      res.Status,
			Username:    res.Username,
		}
	}
}

func extractUsername(output, platform string) string {
	return gituserssh.ExtractPlatformUsername(output, platform)
}

// ── Version Check Commands ───────────────────────────────────────────────────

func fetchGitHubRelease(client *http.Client) string {
	req, err := http.NewRequest("GET", "https://api.github.com/repos/divyo-argha/git-user/releases/latest", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "git-user-tui")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return ""
	}
	return strings.TrimSpace(rel.TagName)
}

func fetchNpmRelease(client *http.Client) string {
	req, err := http.NewRequest("GET", "https://registry.npmjs.org/git-userhub/latest", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "git-user-tui")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return ""
	}
	return strings.TrimSpace(pkg.Version)
}

// CheckVersionCmd checks GitHub releases and the npm registry asynchronously for a newer version.
// It fails silently if the network is unreachable or offline.
func CheckVersionCmd(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		if version.UpdateCheckDisabled() {
			return VersionCheckMsg{CurrentVersion: currentVersion, UpdateAvailable: false}
		}

		// Reuse a recent lookup instead of hitting the network on every launch.
		if cached, ok := version.LoadCachedLatest(); ok {
			return versionCheckResult(currentVersion, cached)
		}

		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		var ghTag, npmTag string
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			ghTag = fetchGitHubRelease(client)
		}()
		go func() {
			defer wg.Done()
			npmTag = fetchNpmRelease(client)
		}()
		wg.Wait()

		bestTag := ghTag
		if npmTag != "" {
			if bestTag == "" || version.IsNewerVersion(npmTag, bestTag) {
				bestTag = npmTag
			}
		}

		if bestTag != "" {
			version.SaveCachedLatest(bestTag)
		}
		return versionCheckResult(currentVersion, bestTag)
	}
}

// versionCheckResult builds the message for a known latest tag ("" = unknown).
func versionCheckResult(currentVersion, latestTag string) VersionCheckMsg {
	if latestTag == "" {
		return VersionCheckMsg{CurrentVersion: currentVersion, UpdateAvailable: false}
	}

	displayTag := latestTag
	if !strings.HasPrefix(strings.ToLower(displayTag), "v") {
		displayTag = "v" + displayTag
	}

	return VersionCheckMsg{
		CurrentVersion:  currentVersion,
		LatestVersion:   displayTag,
		UpdateAvailable: version.IsNewerVersion(latestTag, currentVersion),
	}
}
