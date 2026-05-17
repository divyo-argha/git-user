package cmd

import (
	"fmt"
	"os/exec"
	"strings"
)

// verifySSHConnection tests SSH connectivity to GitHub
func verifySSHConnection() error {
	return verifySSHConnectionPlatform("github")
}

// verifySSHConnectionPlatform tests SSH connectivity to a specific platform
func verifySSHConnectionPlatform(platform string) error {
	hosts := map[string]string{
		"github":    "git@github.com",
		"gitlab":    "git@gitlab.com",
		"bitbucket": "git@bitbucket.org",
	}
	
	host := hosts[platform]
	if host == "" {
		host = "git@github.com" // default
	}
	
	cmd := exec.Command("ssh", "-T", host, "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=5")
	output, _ := cmd.CombinedOutput()
	
	// GitHub/GitLab/Bitbucket return exit code 1 even on success with "Hi username!"
	if strings.Contains(string(output), "successfully authenticated") || strings.Contains(string(output), "Hi ") {
		return nil
	}
	
	return fmt.Errorf("connection failed")
}
