package ssh

import (
	"testing"
)

func TestExtractPlatformUsername(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		output   string
		expected string
	}{
		{
			name:     "GitHub standard greeting",
			platform: "GitHub",
			output:   "Hi octocat! You've successfully authenticated, but GitHub does not provide shell access.",
			expected: "@octocat",
		},
		{
			name:     "GitHub unrecognized output",
			platform: "GitHub",
			output:   "Permission denied (publickey).",
			expected: "",
		},
		{
			name:     "GitLab standard greeting with @",
			platform: "GitLab",
			output:   "Welcome to GitLab, @john_doe!",
			expected: "@john_doe",
		},
		{
			name:     "GitLab greeting without @",
			platform: "GitLab",
			output:   "Welcome to GitLab, jane_doe!",
			expected: "@jane_doe",
		},
		{
			name:     "Bitbucket standard greeting",
			platform: "Bitbucket",
			output:   "logged in as bitbucketuser.\n\nYou can use git or hg to connect to Bitbucket. Shell access is disabled.",
			expected: "@bitbucketuser",
		},
		{
			name:     "Unknown platform",
			platform: "SourceForge",
			output:   "Hi user!",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPlatformUsername(tt.output, tt.platform)
			if got != tt.expected {
				t.Errorf("ExtractPlatformUsername() = %q, want %q", got, tt.expected)
			}
		})
	}
}
