package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NoUpdateCheckEnv disables the background "is there a newer release?" network
// check when set to anything other than "", "0" or "false". Explicit
// `git-user update` is unaffected.
const NoUpdateCheckEnv = "GIT_USER_NO_UPDATE_CHECK"

// UpdateCheckTTL is how long a successful remote lookup is reused before the
// network is hit again.
const UpdateCheckTTL = 24 * time.Hour

// UpdateCheckDisabled reports whether the user opted out of background update checks.
func UpdateCheckDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(NoUpdateCheckEnv))) {
	case "", "0", "false", "no", "off":
		return false
	}
	return true
}

type updateCache struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

func updateCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "git-user", "latest-version.json"), nil
}

// LoadCachedLatest returns the last remote version seen, if it was recorded
// within UpdateCheckTTL and is a valid version. Any read or parse problem is a
// cache miss.
func LoadCachedLatest() (string, bool) {
	path, err := updateCachePath()
	if err != nil {
		return "", false
	}
	return loadCachedLatest(path, time.Now())
}

func loadCachedLatest(path string, now time.Time) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var c updateCache
	if json.Unmarshal(data, &c) != nil || !IsValid(c.Latest) {
		return "", false
	}
	// A timestamp in the future means a clock change or a corrupt file: refetch.
	if age := now.Sub(c.CheckedAt); age < 0 || age >= UpdateCheckTTL {
		return "", false
	}
	return c.Latest, true
}

// SaveCachedLatest records a successful remote lookup. Failures are ignored:
// the cache is an optimization only.
func SaveCachedLatest(latest string) {
	if !IsValid(latest) {
		return
	}
	if path, err := updateCachePath(); err == nil {
		saveCachedLatest(path, latest, time.Now())
	}
}

func saveCachedLatest(path, latest string, now time.Time) {
	data, err := json.Marshal(updateCache{CheckedAt: now, Latest: latest})
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, data, 0o600) != nil {
		return
	}
	if os.Rename(tmp, path) != nil {
		_ = os.Remove(tmp)
	}
}
