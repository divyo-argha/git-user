//go:build windows

package ssh

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/divyo-argha/git-user/internal/git"
)

var ensureSSHOnPathOnce sync.Once

// EnsureSSHBinariesOnPath makes a bare exec.Command("ssh-keygen", ...) (or
// "ssh-add"/"ssh") resolvable even when Git for Windows was found by
// internal/git's own PATH-independent discovery but its usr\bin directory —
// where the OpenSSH client tools it bundles actually live — was never added
// to PATH itself. Without this, git-user can successfully locate Git yet
// still fail every SSH operation with a bare "executable file not found".
// Mirrors internal/git's own findWindowsGit(), which does the same for
// git.exe.
func EnsureSSHBinariesOnPath() {
	ensureSSHOnPathOnce.Do(func() {
		if _, err := exec.LookPath("ssh-keygen.exe"); err == nil {
			return
		}
		if _, err := exec.LookPath("ssh-keygen"); err == nil {
			return
		}

		var candidates []string

		// 1. Native Windows OpenSSH client (installed by default in Windows 10/11)
		sysRoot := os.Getenv("SystemRoot")
		if sysRoot == "" {
			sysRoot = `C:\Windows`
		}
		candidates = append(candidates, filepath.Join(sysRoot, "System32", "OpenSSH"))

		// 2. Git for Windows layout relative to discovered git binary (<root>\usr\bin)
		gitPath := git.BinaryPath()
		if gitPath != "" && gitPath != "git" && gitPath != "git.exe" {
			root := filepath.Dir(filepath.Dir(gitPath))
			candidates = append(candidates, filepath.Join(root, "usr", "bin"))
		}

		// 3. Standard Git for Windows installation locations
		candidates = append(candidates,
			`C:\Program Files\Git\usr\bin`,
			`C:\Program Files (x86)\Git\usr\bin`,
		)
		if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
			candidates = append(candidates, filepath.Join(localApp, "Programs", "Git", "usr", "bin"))
		}

		for _, candidate := range candidates {
			if fi, err := os.Stat(filepath.Join(candidate, "ssh-keygen.exe")); err == nil && !fi.IsDir() {
				currPath := os.Getenv("PATH")
				if currPath != "" {
					_ = os.Setenv("PATH", candidate+string(os.PathListSeparator)+currPath)
				}
				return
			}
		}
	})
}
