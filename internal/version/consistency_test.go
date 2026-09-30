package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The version lives in version.go and npm/package.json, both bumped by hand.
// This fails the build the moment they drift, instead of after a release.
func TestVersionMatchesNpmPackage(t *testing.T) {
	path := filepath.Join("..", "..", "npm", "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("npm/package.json not available: %v", err)
	}
	var pkg struct {
		Version              string            `json:"version"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}

	want := strings.TrimPrefix(Version, "v")
	if pkg.Version != want {
		t.Errorf("npm/package.json version %q does not match version.go %q — bump both together", pkg.Version, Version)
	}
	for dep, v := range pkg.OptionalDependencies {
		if strings.HasPrefix(dep, "git-userhub-") && strings.TrimLeft(v, "^~") != want {
			t.Errorf("optionalDependencies[%s] = %q, want %q", dep, v, want)
		}
	}
}
