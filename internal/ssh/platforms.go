package ssh

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
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
	{Name: "Bitbucket", Host: "git@bitbucket.org", Patterns: []string{"logged in as", "successfully authenticated", "authenticated via ssh key", "connect to bitbucket"}},
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
			rest := output[idx+3:]
			end := strings.IndexAny(rest, "!.,\n\r")
			if end != -1 {
				return "@" + strings.TrimSpace(rest[:end])
			}
		}
	case "GitLab":
		// "Welcome to GitLab, @username!" or "Welcome to GitLab, username!"
		idx := strings.Index(output, "Welcome to GitLab")
		if idx != -1 {
			rest := output[idx+17:]
			rest = strings.TrimLeft(rest, ",: ")
			end := strings.IndexAny(rest, "!.\n\r")
			if end != -1 {
				user := strings.TrimSpace(rest[:end])
				if user != "" {
					if !strings.HasPrefix(user, "@") {
						user = "@" + user
					}
					return user
				}
			}
		}
	case "Bitbucket":
		// "logged in as username."
		idx := strings.Index(strings.ToLower(output), "logged in as ")
		if idx != -1 {
			rest := output[idx+13:]
			end := strings.IndexAny(rest, ".!\n\r")
			if end != -1 {
				user := strings.TrimSpace(rest[:end])
				if user != "" {
					if !strings.HasPrefix(user, "@") {
						user = "@" + user
					}
					return user
				}
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
	// BatchMode=yes prevents hanging on interactive passphrase/password prompts.
	args := []string{"-T", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new", "-o", "ConnectTimeout=5", "-o", "ConnectionAttempts=1"}
	if keyPath != "" {
		args = append(args, "-i", keyPath, "-o", "IdentitiesOnly=yes")
	}
	args = append(args, host)

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh", args...)
	output, err := cmd.CombinedOutput()
	out := string(output)

	if ctx.Err() != nil || (err != nil && (strings.Contains(out, "Connection timed out") ||
		strings.Contains(out, "Connection refused") ||
		strings.Contains(out, "Could not resolve hostname") ||
		strings.Contains(out, "Network is unreachable") ||
		strings.Contains(out, "No route to host") ||
		strings.Contains(out, "Operation timed out") ||
		strings.Contains(out, "Connection reset") ||
		strings.Contains(out, "Connection closed") ||
		strings.Contains(out, "Name or service not known") ||
		strings.Contains(out, "Temporary failure in name resolution") ||
		strings.Contains(out, "Host is down") ||
		strings.Contains(out, "Resource temporarily unavailable"))) {
		return PlatformResult{
			Platform: platform,
			Host:     host,
			Status:   "network_error",
		}
	}

	lowerOut := strings.ToLower(out)
	for _, marker := range successPatterns {
		if strings.Contains(lowerOut, strings.ToLower(marker)) {
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

// CheckAllPlatforms tests SSH authentication against all supported Git platforms concurrently.
func CheckAllPlatforms(keyPath string) []PlatformResult {
	results := make([]PlatformResult, len(DefaultPlatforms))
	var wg sync.WaitGroup
	for i, p := range DefaultPlatforms {
		wg.Add(1)
		go func(idx int, target PlatformTarget) {
			defer wg.Done()
			results[idx] = CheckPlatformConnection(keyPath, target.Name, target.Host, target.Patterns)
		}(i, p)
	}
	wg.Wait()
	return results
}
