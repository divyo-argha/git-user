package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func verifySSHConnection() error {
	return verifySSHConnectionPlatform("github")
}

func verifySSHConnectionPlatform(platform string) error {
	hosts := map[string]string{
		"github":    "git@github.com",
		"gitlab":    "git@gitlab.com",
		"bitbucket": "git@bitbucket.org",
	}
	
	host := hosts[platform]
	if host == "" {
		host = "git@github.com"
	}
	
	cmd := exec.Command("ssh", "-T", host, "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=5")
	output, _ := cmd.CombinedOutput()
	
	if strings.Contains(string(output), "successfully authenticated") || strings.Contains(string(output), "Hi ") {
		return nil
	}
	
	return fmt.Errorf("connection failed")
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
