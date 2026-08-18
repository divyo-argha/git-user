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

	if len(gitParams) > 0 {
		var quoted []string
		for _, p := range gitParams {
			quoted = append(quoted, fmt.Sprintf("'%s'", p))
		}
		vars["GIT_CONFIG_PARAMETERS"] = strings.Join(quoted, " ")
	}

	return vars
}
