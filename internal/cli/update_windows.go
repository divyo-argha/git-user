//go:build windows

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	detachedProcess       = 0x00000008 // DETACHED_PROCESS
	createNewProcessGroup = 0x00000200 // CREATE_NEW_PROCESS_GROUP
)

// installBinary replaces the installed binary on Windows.
//
// Windows locks an executable file while any process is running from it, so
// the running git-user cannot rename or overwrite itself. Instead a small
// detached batch script is spawned next to the binary: it waits for this
// process to exit and then moves the downloaded binary into place.
func installBinary(execPath, newBinary string) (string, error) {
	tmpl := waitForExitScript() + `
set tries=0
:moveloop
move /Y "{NEW}" "{OLD}" >nul 2>&1
if not errorlevel 1 goto done
ping -n 2 127.0.0.1 >nul
set /a tries+=1
if %tries% lss 10 goto moveloop
exit /b 1
:done
del "%~f0" >nul 2>&1
exit /b 0
`
	scriptPath, err := writeWindowsScript(filepath.Dir(execPath), "git-user-update-*.cmd",
		render(tmpl, newBinary, execPath))
	if err != nil {
		return "", err
	}
	if err := spawnDetached(scriptPath); err != nil {
		return "", fmt.Errorf("starting background updater: %w", err)
	}
	return "Windows locks running executables, so the new git-user is applied in the background.\n" +
		"It will be swapped in right after this command exits — run 'git-user --version' in a few seconds.", nil
}

// scheduleNpmUpdateWindows hands an npm update to a detached background
// process. npm cannot replace the running executable on Windows, so it waits
// for this process to exit and then runs `npm install -g git-userhub@latest`.
func scheduleNpmUpdateWindows() error {
	tmpl := waitForExitScript() + `
call npm install -g git-userhub@latest
del "%~f0" >nul 2>&1
exit /b 0
`
	scriptPath, err := writeWindowsScript(os.TempDir(), "git-user-npm-update-*.cmd",
		render(tmpl, "", ""))
	if err != nil {
		return err
	}
	return spawnDetached(scriptPath)
}

// waitForExitScript returns a batch snippet that blocks until the process
// running the update (this git-user process) has exited.
func waitForExitScript() string {
	return `@echo off
rem Wait for the running git-user process (PID {PID}) to exit.
:waitloop
tasklist /FI "PID eq {PID}" 2>nul | find "{PID}" >nul
if not errorlevel 1 (
  ping -n 2 127.0.0.1 >nul
  goto waitloop
)
`
}

// render substitutes {PID}, {NEW} and {OLD} placeholders.
func render(tmpl, newBinary, execPath string) string {
	return strings.NewReplacer(
		"{PID}", strconv.Itoa(os.Getpid()),
		"{NEW}", newBinary,
		"{OLD}", execPath,
	).Replace(tmpl)
}

// writeWindowsScript writes a batch script (CRLF line endings, which cmd.exe
// expects) and returns its path.
func writeWindowsScript(dir, pattern, content string) (string, error) {
	content = strings.ReplaceAll(content, "\n", "\r\n")
	if !strings.HasSuffix(content, "\r\n") {
		content += "\r\n"
	}
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("creating updater script: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return "", fmt.Errorf("writing updater script: %w", err)
	}
	return f.Name(), nil
}

// spawnDetached starts a batch script fully detached from this process so it
// survives the exit of git-user.
func spawnDetached(scriptPath string) error {
	cmd := exec.Command("cmd", "/c", scriptPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess | createNewProcessGroup}
	return cmd.Start()
}
