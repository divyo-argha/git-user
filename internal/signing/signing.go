package signing

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
)

type Status struct {
	Enabled bool
	Key     string
	Format  string
}

func CurrentStatus(user *config.User) Status {
	if user.SignDisabled || user.SignKey == "" {
		return Status{Enabled: false}
	}
	return Status{Enabled: true, Key: user.SignKey, Format: user.SignFormat}
}

func Disable(store *config.Store, name string) {
	store.ToggleSigning(name, true)
}

var ErrNoKeyBound = errors.New("no SSH key bound to this profile and no --key provided")

// Enable resolves the key/format to use (auto-detecting from the identity's
// bound SSH key and the key string's shape when not given explicitly) and
// records it on store. It does not touch git config — callers that want the
// active profile's live signing config updated do that themselves, since
// that's presentation-coupled (success/warning messages differ by caller).
func Enable(store *config.Store, name, key, format string) (resolvedKey, resolvedFormat string, autoDetected bool, err error) {
	user := store.FindUser(name)
	if user == nil {
		return "", "", false, errors.New("identity not found")
	}

	if key == "" {
		if user.SSHKey == "" {
			return "", "", false, ErrNoKeyBound
		}
		key = user.SSHKey
		autoDetected = true
		if format == "" {
			format = "ssh"
		}
	}

	if format == "" {
		if strings.HasPrefix(key, "ssh-") || strings.Contains(key, "id_") || strings.HasSuffix(key, ".pub") {
			format = "ssh"
		} else {
			format = "gpg"
		}
	}

	if format == "ssh" {
		key = expandPath(key)
	}

	store.SetSigningKey(name, key, format)

	return key, format, autoDetected, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
