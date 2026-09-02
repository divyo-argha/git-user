package version

import (
	"fmt"
	"strings"
)

var Version = "v4.11.0"

var BuildVersion = ""

func GetVersion() string {
	if BuildVersion != "" && BuildVersion != "dev" {
		return BuildVersion
	}
	return Version
}

// SetVersion updates the current in-memory version string (e.g. after a self-update).
func SetVersion(v string) {
	v = strings.TrimSpace(v)
	if v == "" {
		return
	}
	Version = v
	BuildVersion = v
}

// ParseVersion extracts major, minor, and patch numbers from a version string.
func ParseVersion(v string) (int, int, int) {
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	parts := strings.Split(v, ".")
	var major, minor, patch int
	if len(parts) > 0 {
		fmt.Sscanf(parts[0], "%d", &major)
	}
	if len(parts) > 1 {
		fmt.Sscanf(parts[1], "%d", &minor)
	}
	if len(parts) > 2 {
		patchStr := parts[2]
		if idx := strings.IndexAny(patchStr, "-+"); idx != -1 {
			patchStr = patchStr[:idx]
		}
		fmt.Sscanf(patchStr, "%d", &patch)
	}
	return major, minor, patch
}

// IsNewerVersion reports whether remoteTag is newer than currentVersion according to semver.
func IsNewerVersion(remoteTag, currentVersion string) bool {
	rMaj, rMin, rPat := ParseVersion(remoteTag)
	cMaj, cMin, cPat := ParseVersion(currentVersion)

	if rMaj != cMaj {
		return rMaj > cMaj
	}
	if rMin != cMin {
		return rMin > cMin
	}
	return rPat > cPat
}
