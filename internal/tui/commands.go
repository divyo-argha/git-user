package tui

import (
	"net"
	"os"
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
