package cli

import (
	"testing"
)

func TestIsNpmInstall(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/usr/local/lib/node_modules/git-userhub-darwin-arm64/bin/git-user", true},
		{"/Users/ops/.nvm/versions/node/v20.0.0/bin/git-user", true},
		{"/usr/local/bin/git-user", false},
		{"/private/user/bin/git-user", false},
	}

	for _, tt := range tests {
		got := isNpmInstall(tt.path)
		if got != tt.want {
			t.Errorf("isNpmInstall(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
