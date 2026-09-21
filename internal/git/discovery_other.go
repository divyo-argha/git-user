//go:build !windows

package git

import (
	"os/exec"
)

func findWindowsGit() string {
	if p, err := exec.LookPath("git"); err == nil {
		return p
	}
	return "git"
}
