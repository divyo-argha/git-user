package ssh

import (
	"strings"
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
			name:     "GitLab greeting with period",
			platform: "GitLab",
			output:   "Welcome to GitLab, @john_doe.",
			expected: "@john_doe",
		},
		{
			name:     "Bitbucket standard greeting",
			platform: "Bitbucket",
			output:   "logged in as bitbucketuser.\n\nYou can use git or hg to connect to Bitbucket. Shell access is disabled.",
			expected: "@bitbucketuser",
		},
		{
			name:     "Bitbucket modern ssh key greeting (no username)",
			platform: "Bitbucket",
			output:   "authenticated via ssh key.\n\nYou can use git to connect to Bitbucket. Shell access is disabled",
			expected: "",
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

func TestDefaultPlatformsPatterns(t *testing.T) {
	cases := []struct {
		platform string
		output   string
		matched  bool
	}{
		{
			platform: "GitHub",
			output:   "Hi octocat! You've successfully authenticated, but GitHub does not provide shell access.",
			matched:  true,
		},
		{
			platform: "GitLab",
			output:   "Welcome to GitLab, @john_doe!",
			matched:  true,
		},
		{
			platform: "GitLab",
			output:   "welcome to gitlab, @john_doe!",
			matched:  true,
		},
		{
			platform: "Bitbucket",
			output:   "logged in as bitbucketuser.\n\nYou can use git or hg to connect to Bitbucket. Shell access is disabled.",
			matched:  true,
		},
		{
			platform: "Bitbucket",
			output:   "authenticated via ssh key.\n\nYou can use git to connect to Bitbucket. Shell access is disabled",
			matched:  true,
		},
		{
			platform: "Bitbucket",
			output:   "Authenticated via SSH key.\n\nYou can use git to connect to Bitbucket. Shell access is disabled",
			matched:  true,
		},
		{
			platform: "GitHub",
			output:   "git@github.com: Permission denied (publickey).",
			matched:  false,
		},
		{
			platform: "GitLab",
			output:   "git@gitlab.com: Permission denied (publickey).",
			matched:  false,
		},
		{
			platform: "Bitbucket",
			output:   "git@bitbucket.org: Permission denied (publickey).",
			matched:  false,
		},
	}

	platformsByName := make(map[string]PlatformTarget)
	for _, p := range DefaultPlatforms {
		platformsByName[p.Name] = p
	}

	for _, tc := range cases {
		p, ok := platformsByName[tc.platform]
		if !ok {
			t.Fatalf("platform %q not found in DefaultPlatforms", tc.platform)
		}

		lowerOut := strings.ToLower(tc.output)
		match := false
		for _, marker := range p.Patterns {
			if strings.Contains(lowerOut, strings.ToLower(marker)) {
				match = true
				break
			}
		}

		if match != tc.matched {
			t.Errorf("[%s] output matched = %v, want %v. Output: %q", tc.platform, match, tc.matched, tc.output)
		}
	}
}
