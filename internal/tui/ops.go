package tui

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/gitenv"
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

// unlockIdentitySSHKeyForShell verifies and loads a protected, not-yet-loaded
// SSH key into ssh-agent before an in-TUI isolated shell (openIdentityShellCmd)
// starts using it — without this, `git-user shell` inside a TUI-suspended
// terminal would look ready immediately but the first push/pull would stall
// on ssh's own passphrase prompt. Mirrors the passphrase gate in opSwitch
// (ops_identity.go), minus the parts of a full switch (config apply, signing,
// previous-identity logout) that don't apply here. An empty warning with a
// nil error means the key is unlocked and loaded (or didn't need to be);
// ErrNeedsPassphrase means the caller must collect one interactively.
func unlockIdentitySSHKeyForShell(user *config.User, passphrase string) (warning string, err error) {
	if user.SSHKey == "" {
		return "", nil
	}
	protected, perr := isSSHKeyPassphraseProtected(user.SSHKey)
	if perr != nil || !protected || ssh.IsSSHKeyLoaded(user.SSHKey) {
		return "", nil
	}

	mode := user.GetPassphraseMode()
	p := passphrase
	if p == "" && mode == "persistent" {
		if secret, kerr := keyring.GetKeychainPassphrase(user.Name); kerr == nil && secret != "" {
			if ssh.VerifyPassphrase(user.SSHKey, secret) {
				p = secret
			} else {
				_ = keyring.DeleteKeychainPassphrase(user.Name)
			}
		}
	}
	if p == "" {
		return "", ErrNeedsPassphrase
	}
	if !ssh.VerifyPassphrase(user.SSHKey, p) {
		return "", fmt.Errorf("incorrect passphrase")
	}
	if passphrase != "" && mode == "persistent" {
		_ = keyring.SetKeychainPassphrase(user.Name, passphrase)
	}

	if agentErr := ssh.EnsureSSHAgent(); agentErr != nil {
		return fmt.Sprintf("Key for %q was NOT loaded into any ssh-agent (no agent reachable) — the next push/pull may hang or fail asking for a passphrase.", user.Name), nil
	}
	if err := ssh.AddSSHKeyWithPassphrase(user.SSHKey, p); err != nil {
		return fmt.Sprintf("Could not load key into agent: %v", err), nil
	}
	return "", nil
}

// applyHTTPSCredentialConfig wires core.askpass to this identity's stored
// HTTPS token (if any), or removes it if not — the TUI-side twin of the same
// helper in internal/cli/switch.go, called at the same point opSwitch
// applies (or clears) signing config for the new identity. Returns a warning
// string (empty on success) for the caller to fold into its own warnings.
func applyHTTPSCredentialConfig(user *config.User, local bool) string {
	if !keyring.HasHTTPSToken(user.Name) {
		git.RemoveAskpassConfigScope(local)
		return ""
	}
	cmd, err := gitenv.AskpassCommand(user.Name)
	if err != nil {
		return fmt.Sprintf("Could not resolve git-user's own path to wire up the HTTPS token: %v", err)
	}
	if err := git.ConfigureAskpassScope(cmd, local); err != nil {
		return fmt.Sprintf("Could not apply core.askpass: %v", err)
	}
	return ""
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

