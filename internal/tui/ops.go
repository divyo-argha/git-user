package tui

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
)

// opResult carries the outcome of an in-TUI operation. Detail is rendered on a
// Report screen when ShowReport is true; otherwise it is shown as a toast.
type opResult struct {
	detail     string
	showReport bool
}

// Sentinel errors used to signal that the UI must prompt for more input.
var (
	ErrNeedsPassphrase = errors.New("passphrase required")
	ErrNeedsCredential = errors.New("platform credential required")
)

// ── Shared helpers ────────────────────────────────────────────────────────────

// runCaptured runs a command with its output captured so nothing leaks to the
// terminal while the TUI is on the alternate screen. Git is forced into
// non-interactive mode so it never blocks on a credential prompt.
func runCaptured(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func isValidEmail(email string) bool {
	return config.ValidEmail(email)
}

func isSSHKeyPassphraseProtected(keyPath string) (bool, error) {
	return ssh.IsPassphraseProtected(keyPath)
}

// needsPassphraseForSwitch reports whether switching to the identity requires
// interactive passphrase entry (protected, not loaded, not in keychain).
func needsPassphraseForSwitch(store *config.Store, name string) bool {
	user := store.FindUser(name)
	if user == nil || user.SSHKey == "" {
		return false
	}
	protected, err := isSSHKeyPassphraseProtected(user.SSHKey)
	if err != nil || !protected || ssh.IsSSHKeyLoaded(user.SSHKey) {
		return false
	}
	if user.GetPassphraseMode() == "persistent" {
		if secret, kerr := keyring.GetKeychainPassphrase(user.Name); kerr == nil && secret != "" {
			return false
		}
	}
	return true
}

func stripAnsi(s string) string {
	var result strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

