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

		gitPath := git.BinaryPath()
		if gitPath == "" || gitPath == "git" || gitPath == "git.exe" {
			return
		}

		// Git for Windows layout: <root>\cmd\git.exe or <root>\bin\git.exe,
		// with its bundled OpenSSH client at <root>\usr\bin.
		root := filepath.Dir(filepath.Dir(gitPath))
		candidate := filepath.Join(root, "usr", "bin")
		if fi, err := os.Stat(filepath.Join(candidate, "ssh-keygen.exe")); err != nil || fi.IsDir() {
			return
		}

		currPath := os.Getenv("PATH")
		if currPath != "" {
			_ = os.Setenv("PATH", candidate+string(os.PathListSeparator)+currPath)
		}
	})
}
