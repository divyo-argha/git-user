//go:build windows

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	resolvedGitPath string
	resolveOnce     sync.Once
)

// findWindowsGit checks PATH first, and if not present, scans standard Windows
// installation directories (e.g. Git for Windows, GitHub Desktop, Chocolatey, Scoop).
func findWindowsGit() string {
	resolveOnce.Do(func() {
		// 1. Check if git or git.exe is on PATH
		if p, err := exec.LookPath("git"); err == nil {
			resolvedGitPath = p
			return
		}
		if p, err := exec.LookPath("git.exe"); err == nil {
			resolvedGitPath = p
			return
		}

		// 2. Search standard Windows installation locations
		candidates := []string{
			`C:\Program Files\Git\cmd\git.exe`,
			`C:\Program Files\Git\bin\git.exe`,
			`C:\Program Files (x86)\Git\cmd\git.exe`,
			`C:\Program Files (x86)\Git\bin\git.exe`,
		}

		if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
			candidates = append(candidates,
				filepath.Join(localApp, "Programs", "Git", "cmd", "git.exe"),
				filepath.Join(localApp, "Programs", "Git", "bin", "git.exe"),
			)
			// Check GitHub Desktop bundled git
			matches, _ := filepath.Glob(filepath.Join(localApp, "GitHubDesktop", "app-*", "resources", "app", "git", "cmd", "git.exe"))
			candidates = append(candidates, matches...)
		}

		if progData := os.Getenv("ProgramData"); progData != "" {
			candidates = append(candidates, filepath.Join(progData, "chocolatey", "bin", "git.exe"))
		}

		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(home, "scoop", "shims", "git.exe"))
		}

		for _, c := range candidates {
			if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
				resolvedGitPath = c
				// Prepend to PATH so child processes and subshells inherit it
				gitDir := filepath.Dir(c)
				currPath := os.Getenv("PATH")
				if currPath != "" {
					_ = os.Setenv("PATH", gitDir+string(os.PathListSeparator)+currPath)
				}
				return
			}
		}

		resolvedGitPath = "git"
	})
	return resolvedGitPath
}
