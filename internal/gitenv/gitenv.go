// Package gitenv builds the Git environment-variable overrides for a single
// identity. It has no dependency on internal/cli or internal/tui so both can
// import it without creating a cycle.
package gitenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/keyring"
)

// Vars returns the map of Git environment variables for a given user identity.
func Vars(u *config.User) map[string]string {
	vars := map[string]string{
		"GIT_AUTHOR_NAME":     u.Name,
		"GIT_AUTHOR_EMAIL":    u.Email,
		"GIT_COMMITTER_NAME":  u.Name,
		"GIT_COMMITTER_EMAIL": u.Email,
		"GIT_USER_SESSION":    u.Name,
	}

	// SSH command configuration
	if u.SSHCommand != "" {
		vars["GIT_SSH_COMMAND"] = u.SSHCommand
	} else if u.SSHKey != "" {
		keyPath := u.SSHKey
		if strings.HasPrefix(keyPath, "~/") || strings.HasPrefix(keyPath, "~\\") {
			if home, err := os.UserHomeDir(); err == nil {
				keyPath = filepath.Join(home, keyPath[2:])
			}
		}
		vars["GIT_SSH_COMMAND"] = fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", keyPath)
	}

	// Git config parameters (ensures git config user.name, git config user.email, and signing are overridden)
	var gitParams []string
	gitParams = append(gitParams, fmt.Sprintf("user.name=%s", u.Name))
	gitParams = append(gitParams, fmt.Sprintf("user.email=%s", u.Email))
	if u.SignKey != "" && !u.SignDisabled {
		gitParams = append(gitParams, fmt.Sprintf("user.signingkey=%s", u.SignKey))
		gitParams = append(gitParams, "commit.gpgsign=true")
		if u.SignFormat != "" {
			gitParams = append(gitParams, fmt.Sprintf("gpg.format=%s", u.SignFormat))
		}
	}
	for k, v := range u.CustomConfig {
		gitParams = append(gitParams, fmt.Sprintf("%s=%s", k, v))
	}

	// HTTPS credential (for hosts/proxies where SSH isn't available): route
	// credential prompts back through this same git-user binary rather than
	// ever placing the token in git config — see AskpassCommand and
	// internal/cli/askpass_helper.go.
	if keyring.HasHTTPSToken(u.Name) {
		if cmd, err := AskpassCommand(u.Name); err == nil {
			gitParams = append(gitParams, fmt.Sprintf("core.askpass=%s", cmd))
		}
	}

	if len(gitParams) > 0 {
		var quoted []string
		for _, p := range gitParams {
			quoted = append(quoted, sqQuote(p))
		}
		vars["GIT_CONFIG_PARAMETERS"] = strings.Join(quoted, " ")
	}

	return vars
}

// AskpassCommand builds the core.askpass value that answers HTTPS credential
// prompts for identityName by shelling back into this same git-user binary
// (see internal/cli/askpass_helper.go). The token itself is never written
// into git config — the helper looks it up from the OS keyring at prompt
// time, by identity name.
func AskpassCommand(identityName string) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s __askpass %s", shellQuote(exePath), shellQuote(identityName)), nil
}

// shellQuote wraps s in single quotes for safe interpolation into a POSIX
// shell command string — core.askpass, like core.sshCommand, is executed via
// the shell, so this (not sqQuote below, which is for a different consumer:
// git's own GIT_CONFIG_PARAMETERS parser) is what protects against a
// pathological exe path or identity name breaking out of the command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// sqQuote wraps s in single quotes using the same escaping git's own
// sq_dequote (config.c) expects when parsing GIT_CONFIG_PARAMETERS: an
// embedded single quote is closed, escaped with a backslash, and reopened.
// Without this, a value containing a literal quote — e.g. an email local
// part like o'brien@example.com, which validate.Email explicitly allows —
// corrupts the quoting and desyncs every parameter after it.
func sqQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
