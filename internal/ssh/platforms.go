package ssh

import (
	"os/exec"
	"strings"
)

// PlatformTarget represents a Git platform SSH endpoint and expected greeting patterns.
type PlatformTarget struct {
	Name     string
	Host     string
	Patterns []string
}

// DefaultPlatforms returns the supported Git hosting platforms.
var DefaultPlatforms = []PlatformTarget{
	{Name: "GitHub", Host: "git@github.com", Patterns: []string{"Hi ", "successfully authenticated"}},
	{Name: "GitLab", Host: "git@gitlab.com", Patterns: []string{"Welcome to GitLab", "successfully authenticated"}},
	{Name: "Bitbucket", Host: "git@bitbucket.org", Patterns: []string{"logged in as", "successfully authenticated", "authenticated via ssh key"}},
}

// PlatformResult represents the outcome of probing a platform SSH connection.
type PlatformResult struct {
	Platform string `json:"platform"`
	Host     string `json:"host"`
	Status   string `json:"status"` // "connected", "not_added", "network_error"
	Username string `json:"username,omitempty"`
}

// ExtractPlatformUsername parses SSH output for the authenticated username.
func ExtractPlatformUsername(output, platform string) string {
	switch platform {
	case "GitHub":
		// "Hi username! You've successfully authenticated..."
		idx := strings.Index(output, "Hi ")
		if idx != -1 {
			end := strings.Index(output[idx+3:], "!")
			if end != -1 {
				return "@" + output[idx+3:idx+3+end]
			}
		}
	case "GitLab":
		// "Welcome to GitLab, @username!"
		idx := strings.Index(output, "Welcome to GitLab, ")
		if idx != -1 {
			end := strings.Index(output[idx+19:], "!")
			if end != -1 {
				user := strings.TrimSpace(output[idx+19 : idx+19+end])
				if !strings.HasPrefix(user, "@") {
					user = "@" + user
				}
				return user
			}
		}
	case "Bitbucket":
		// "logged in as username."
		idx := strings.Index(output, "logged in as ")
		if idx != -1 {
			end := strings.Index(output[idx+13:], ".")
			if end != -1 {
				return "@" + output[idx+13:idx+13+end]
			}
		}
	}
	return ""
}

// CheckPlatformConnection tests SSH connection to a single platform.
func CheckPlatformConnection(keyPath, platform, host string, successPatterns []string) PlatformResult {
	// accept-new (not "no"): pin an unknown host's key on first contact like
	// "no" does, but unlike "no" it still rejects a host whose key changed
	// after being trusted — "no" silently accepts that on every single
	// connection, which is exactly the MITM scenario host-key checking exists
	// to catch.
	args := []string{"-T", "-o", "StrictHostKeyChecking=accept-new", "-o", "ConnectTimeout=5", "-o", "ConnectionAttempts=1"}
	if keyPath != "" {
		args = append(args, "-i", keyPath, "-o", "IdentitiesOnly=yes")
	}
	args = append(args, host)

	cmd := exec.Command("ssh", args...)
	output, err := cmd.CombinedOutput()
	out := string(output)

	if err != nil && (strings.Contains(out, "Connection timed out") ||
		strings.Contains(out, "Connection refused") ||
		strings.Contains(out, "Could not resolve hostname") ||
		strings.Contains(out, "Network is unreachable") ||
		strings.Contains(out, "No route to host")) {
		return PlatformResult{
			Platform: platform,
			Host:     host,
			Status:   "network_error",
		}
	}

	for _, marker := range successPatterns {
		if strings.Contains(out, marker) {
			return PlatformResult{
				Platform: platform,
				Host:     host,
				Status:   "connected",
				Username: ExtractPlatformUsername(out, platform),
			}
		}
	}

	return PlatformResult{
		Platform: platform,
		Host:     host,
		Status:   "not_added",
	}
}

// CheckAllPlatforms tests SSH authentication against all supported Git platforms.
func CheckAllPlatforms(keyPath string) []PlatformResult {
	results := make([]PlatformResult, len(DefaultPlatforms))
	for i, p := range DefaultPlatforms {
		results[i] = CheckPlatformConnection(keyPath, p.Name, p.Host, p.Patterns)
	}
	return results
}
