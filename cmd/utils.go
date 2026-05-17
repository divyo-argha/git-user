package cmd

import (
	"fmt"
	"os/exec"
	"strings"
)

// verifySSHConnection tests SSH connectivity to GitHub
func verifySSHConnection() error {
	cmd := exec.Command("ssh", "-T", "git@github.com", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=5")
	output, _ := cmd.CombinedOutput()
	
	// GitHub returns exit code 1 even on success with "Hi username!"
	if strings.Contains(string(output), "successfully authenticated") || strings.Contains(string(output), "Hi ") {
		return nil
	}
	
	return fmt.Errorf("connection failed")
}
