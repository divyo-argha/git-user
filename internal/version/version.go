package version

import (
	"strconv"
	"strings"
	"sync"
)

var Version = "v5.0.0"

var BuildVersion = ""

// mu guards Version and BuildVersion: SetVersion runs after a self-update
// while the TUI may be reading the version from other goroutines.
var mu sync.RWMutex

func GetVersion() string {
	mu.RLock()
	defer mu.RUnlock()
	if BuildVersion != "" && BuildVersion != "dev" {
		return BuildVersion
	}
	return Version
}

// SetVersion updates the current in-memory version string (e.g. after a self-update).
// The value is normalized to carry a single leading "v".
func SetVersion(v string) {
	v = strings.TrimSpace(v)
	if v == "" {
		return
	}
	v = "v" + strings.TrimLeft(v, "vV")
	mu.Lock()
	defer mu.Unlock()
	Version = v
	BuildVersion = v
}

// semver holds the parts of a version string that matter for ordering.
type semver struct {
	major, minor, patch int
	pre                 string // prerelease identifiers ("rc.1"); empty for a release
}

// parse reads "[vV]MAJOR.MINOR.PATCH[-pre][+build]". Missing minor/patch
// default to 0; anything else non-numeric makes the version invalid.
func parse(v string) (semver, bool) {
	var sv semver
	v = strings.TrimSpace(v)
	v = strings.TrimLeft(v, "vV")
	if i := strings.Index(v, "+"); i != -1 { // build metadata never affects ordering
		v = v[:i]
	}
	if i := strings.Index(v, "-"); i != -1 {
		sv.pre = v[i+1:]
		v = v[:i]
	}
	if v == "" {
		return sv, false
	}
	parts := strings.Split(v, ".")
	if len(parts) > 3 { // only MAJOR.MINOR.PATCH is meaningful; extra parts are ignored
		parts = parts[:3]
	}
	nums := [3]int{}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return sv, false
		}
		nums[i] = n
	}
	sv.major, sv.minor, sv.patch = nums[0], nums[1], nums[2]
	return sv, true
}

// ParseVersion extracts major, minor, and patch numbers from a version string.
// Invalid input yields 0, 0, 0. Prerelease and build suffixes are ignored here;
// use IsNewerVersion for ordering that accounts for them.
func ParseVersion(v string) (int, int, int) {
	sv, ok := parse(v)
	if !ok {
		return 0, 0, 0
	}
	return sv.major, sv.minor, sv.patch
}

// IsValid reports whether v is a parseable version string.
func IsValid(v string) bool {
	_, ok := parse(v)
	return ok
}

// comparePre orders prerelease strings per semver: a release outranks any of
// its prereleases, and dot-separated identifiers compare numerically when both
// are numeric, numeric below alphanumeric, and shorter below longer on a tie.
func comparePre(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == "":
		return 1
	case b == "":
		return -1
	}
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] == bs[i] {
			continue
		}
		an, aerr := strconv.Atoi(as[i])
		bn, berr := strconv.Atoi(bs[i])
		switch {
		case aerr == nil && berr == nil:
			if an < bn {
				return -1
			}
			return 1
		case aerr == nil:
			return -1
		case berr == nil:
			return 1
		case as[i] < bs[i]:
			return -1
		default:
			return 1
		}
	}
	switch {
	case len(as) < len(bs):
		return -1
	case len(as) > len(bs):
		return 1
	}
	return 0
}

// IsNewerVersion reports whether remoteTag is newer than currentVersion according to semver.
// If either side cannot be parsed it returns false, so a malformed tag from a
// remote API never triggers (or suppresses) an update notice by accident.
func IsNewerVersion(remoteTag, currentVersion string) bool {
	r, rok := parse(remoteTag)
	c, cok := parse(currentVersion)
	if !rok || !cok {
		return false
	}
	if r.major != c.major {
		return r.major > c.major
	}
	if r.minor != c.minor {
		return r.minor > c.minor
	}
	if r.patch != c.patch {
		return r.patch > c.patch
	}
	return comparePre(r.pre, c.pre) > 0
}
