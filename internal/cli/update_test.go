package cli

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input  string
		m, n, p int
	}{
		{"v4.3.1", 4, 3, 1},
		{"V3.3.4", 3, 3, 4},
		{"1.0.0", 1, 0, 0},
		{"v2.5.0-beta1", 2, 5, 0},
		{"v4.10.12+build123", 4, 10, 12},
	}

	for _, tt := range tests {
		m, n, p := parseVersion(tt.input)
		if m != tt.m || n != tt.n || p != tt.p {
			t.Errorf("parseVersion(%q) = (%d, %d, %d), expected (%d, %d, %d)", tt.input, m, n, p, tt.m, tt.n, tt.p)
		}
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		remote  string
		current string
		want    bool
	}{
		{"v3.3.4", "v4.3.1", false},
		{"v4.3.1", "v4.3.1", false},
		{"v4.3.2", "v4.3.1", true},
		{"v5.0.0", "v4.3.1", true},
		{"v4.4.0", "v4.3.1", true},
		{"v3.9.9", "v4.0.0", false},
	}

	for _, tt := range tests {
		got := isNewerVersion(tt.remote, tt.current)
		if got != tt.want {
			t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tt.remote, tt.current, got, tt.want)
		}
	}
}

func TestIsNpmInstall(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/usr/local/lib/node_modules/git-userhub-darwin-arm64/bin/git-user", true},
		{"/Users/bob/.nvm/versions/node/v20.0.0/bin/git-user", true},
		{"/usr/local/bin/git-user", false},
		{"/home/user/bin/git-user", false},
	}

	for _, tt := range tests {
		got := isNpmInstall(tt.path)
		if got != tt.want {
			t.Errorf("isNpmInstall(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
