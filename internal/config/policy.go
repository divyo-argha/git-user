package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const RepoPolicyFileName = ".git-user-policy"

type RepoPolicy struct {
	RequireSigning      bool
	AllowedEmailDomains []string
}

func LoadRepoPolicy(repoRoot string) (RepoPolicy, error) {
	var policy RepoPolicy

	f, err := os.Open(filepath.Join(repoRoot, RepoPolicyFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return policy, nil
		}
		return policy, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.ToLower(strings.TrimSpace(value))
		truthy := value == "true" || value == "1" || value == "yes"

		switch key {
		case "require_signing":
			policy.RequireSigning = truthy
		case "allowed_email_domains":
			var domains []string
			for _, d := range strings.Split(value, ",") {
				d = strings.TrimSpace(d)
				if d != "" {
					domains = append(domains, d)
				}
			}
			policy.AllowedEmailDomains = domains
		}
	}
	if err := scanner.Err(); err != nil {
		return policy, err
	}

	return policy, nil
}
