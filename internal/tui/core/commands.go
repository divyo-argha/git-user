package core

import (
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/theme"
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

// ── Agent Commands ────────────────────────────────────────────────────────────

// CheckAgentCmd checks SSH agent connectivity and loaded key count.
func CheckAgentCmd() tea.Cmd {
	return func() tea.Msg {
		socket := os.Getenv("SSH_AUTH_SOCK")
		if socket == "" {
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
	pubKeyPath := keyPath + ".pub"
	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return false
	}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		return false
	}

	targetFP := ssh.FingerprintSHA256(pubKey)

	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return false
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return false
	}
	defer conn.Close()

	client := agent.NewClient(conn)
	keys, err := client.List()
	if err != nil {
		return false
	}

	for _, key := range keys {
		if ssh.FingerprintSHA256(key) == targetFP {
			return true
		}
	}
	return false
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
func CheckPlatformConnectionCmd(keyPath, platform, host string, successPatterns []string) tea.Cmd {
	return func() tea.Msg {
		args := []string{"-T", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=4", "-o", "ConnectionAttempts=1"}
		if keyPath != "" {
			args = append(args, "-i", keyPath, "-o", "IdentitiesOnly=yes")
		}
		args = append(args, host)

		cmd := exec.Command("ssh", args...)
		if keyPath != "" {
			cmd.Env = append(os.Environ(), "SSH_AUTH_SOCK=")
		}

		output, err := cmd.CombinedOutput()
		out := string(output)

		// Check for network errors or unreachability
		// Common ssh exit code for timeout/network failure is 255.
		// Also parse standard connection failed strings.
		if err != nil && (strings.Contains(out, "Connection timed out") || strings.Contains(out, "Connection refused") || strings.Contains(out, "Could not resolve hostname")) {
			return PlatformConnectionMsg{
				Platform: platform,
				Status:   "network_error",
			}
		}

		for _, marker := range successPatterns {
			if strings.Contains(out, marker) {
				// Try to extract username
				username := extractUsername(out, platform)
				return PlatformConnectionMsg{
					Platform: platform,
					Status:   "connected",
					Username: username,
				}
			}
		}

		return PlatformConnectionMsg{
			Platform: platform,
			Status:   "not_added",
		}
	}
}

func extractUsername(output, platform string) string {
	switch platform {
	case "GitHub":
		// "Hi username! You've successfully authenticated..."
		idx := strings.Index(output, "Hi ")
		if idx != -1 {
			end := strings.Index(output[idx+3:], "!")
			if end != -1 {
				return "@" + output[idx+3:idx+3+end]
			}
		}
	case "GitLab":
		// "Welcome to GitLab, @username!"
		idx := strings.Index(output, "Welcome to GitLab, ")
		if idx != -1 {
			end := strings.Index(output[idx+19:], "!")
			if end != -1 {
				return output[idx+19 : idx+19+end]
			}
		}
	case "Bitbucket":
		// "logged in as username."
		idx := strings.Index(output, "logged in as ")
		if idx != -1 {
			end := strings.Index(output[idx+13:], ".")
			if end != -1 {
				return "@" + output[idx+13:idx+13+end]
			}
		}
	}
	return ""
}
