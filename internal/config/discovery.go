package config

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DiscoveredUser represents a user found during harvesting
type DiscoveredUser struct {
	Name       string
	Email      string
	SSHKey     string
	SigningKey string
	Method     string
}

// Harvest scans the system for existing Git and SSH identities.
func Harvest() ([]DiscoveredUser, error) {
	var results []DiscoveredUser

	// 1. Discover via Global Git Config
	global := discoverGlobalGit()
	if global != nil {
		results = append(results, *global)
	}

	// 2. Discover SSH Keys in ~/.ssh
	sshKeys := discoverSSHKeys()
	
	// If we found a global identity but no SSH key was bound in git config,
	// try to match it with anything in ~/.ssh
	if len(results) > 0 {
		if results[0].SSHKey == "" && len(sshKeys) > 0 {
			results[0].SSHKey = sshKeys[0] // Suggest the first one found
		}
	} else {
		// If no git identity found, but SSH keys exist, create "Unknown" profiles
		for _, key := range sshKeys {
			results = append(results, DiscoveredUser{
				Name:   "discovered-" + filepath.Base(key),
				SSHKey: key,
			})
		}
	}

	return results, nil
}

func discoverGlobalGit() *DiscoveredUser {
	name, _ := getGitConfig("user.name")
	email, _ := getGitConfig("user.email")
	if name == "" && email == "" {
		return nil
	}

	user := &DiscoveredUser{
		Name:  name,
		Email: email,
	}

	signingKey, _ := getGitConfig("user.signingkey")
	if signingKey != "" {
		user.SigningKey = signingKey
		format, _ := getGitConfig("gpg.format")
		if format == "ssh" {
			user.Method = "ssh"
		} else {
			user.Method = "gpg"
		}
	}

	// Check core.sshCommand for identity files
	sshCmd, _ := getGitConfig("core.sshCommand")
	if strings.Contains(sshCmd, "-i ") {
		parts := strings.Split(sshCmd, "-i ")
		if len(parts) > 1 {
			keyPath := strings.Fields(parts[1])[0]
			keyPath = strings.Trim(keyPath, "\"")
			keyPath = strings.Trim(keyPath, "'")
			user.SSHKey = expandHome(keyPath)
		}
	}

	return user
}

func discoverSSHKeys() []string {
	var keys []string
	home, err := os.UserHomeDir()
	if err != nil {
		return keys
	}

	sshDir := filepath.Join(home, ".ssh")
	files, err := os.ReadDir(sshDir)
	if err != nil {
		return keys
	}

	// Common private key patterns
	patterns := []string{"id_rsa", "id_ed25519", "id_ecdsa", "id_dsa"}
	
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		
		// Skip public keys, known_hosts, etc.
		if strings.HasSuffix(name, ".pub") || name == "known_hosts" || name == "authorized_keys" || name == "config" {
			continue
		}

		matched := false
		for _, p := range patterns {
			if strings.HasPrefix(name, p) {
				matched = true
				break
			}
		}

		if matched {
			keys = append(keys, filepath.Join(sshDir, name))
		}
	}

	// Also parse ~/.ssh/config for IdentityFile
	configPath := filepath.Join(sshDir, "config")
	if f, err := os.Open(configPath); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(strings.ToLower(line), "identityfile ") {
				parts := strings.Fields(line)
				if len(parts) > 1 {
					path := expandHome(parts[1])
					// Avoid duplicates
					exists := false
					for _, k := range keys {
						if k == path {
							exists = true
							break
						}
					}
					if !exists {
						keys = append(keys, path)
					}
				}
			}
		}
	}

	return keys
}

func getGitConfig(key string) (string, error) {
	cmd := exec.Command("git", "config", "--global", key)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
