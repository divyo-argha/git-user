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
	createNewProcessGroup = 0x00000200 // CREATE_NEW_PROCESS_GROUP
	createNoWindow        = 0x08000000 // CREATE_NO_WINDOW
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
del "%~f0" >nul 2>&1
exit /b 1
:done
del "%~f0" >nul 2>&1
exit /b 0
`
	scriptPath, err := writeWindowsScript(filepath.Dir(execPath), "git-user-update-*.cmd",
		render(tmpl, newBinary, execPath, ""))
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
// for this process to exit and then runs `npm install -g git-userhub@<version>`.
// version must already be validated (isSafeNpmVersion) by the caller — it is
// spliced directly into a batch script, so an unvalidated value here would be
// a command-injection hole.
func scheduleNpmUpdateWindows(version string) error {
	if version == "" || (version != "latest" && !isSafeNpmVersion(version)) {
		return fmt.Errorf("refusing to schedule npm update: unsafe version %q", version)
	}
	tmpl := waitForExitScript() + `
call npm install -g git-userhub@{VERSION}
del "%~f0" >nul 2>&1
exit /b 0
`
	scriptPath, err := writeWindowsScript(os.TempDir(), "git-user-npm-update-*.cmd",
		render(tmpl, "", "", version))
	if err != nil {
		return err
	}
	return spawnDetached(scriptPath)
}

// waitForExitScript returns a batch snippet that blocks until the process
// running the update (this git-user process) has exited, giving up after
// ~2 minutes (60 tries * ~2s) rather than looping forever. Without this
// bound, a stale or reused {PID} (e.g. the TUI process it was meant to wait
// for was killed rather than exiting normally, or the PID got recycled by an
// unrelated long-lived process) would leave this script polling forever in
// the background.
func waitForExitScript() string {
	return `@echo off
rem Wait for the running git-user process (PID {PID}) to exit.
set wait_tries=0
:waitloop
tasklist /FI "PID eq {PID}" 2>nul | find "{PID}" >nul
if not errorlevel 1 (
  set /a wait_tries+=1
  if %wait_tries% geq 60 goto waitdone
  ping -n 2 127.0.0.1 >nul
  goto waitloop
)
:waitdone
`
}

// getTargetPID returns the PID of the process to wait for before performing
// the binary swap or npm update. If launched from the TUI, it returns the
// TUI's parent PID so the update only runs after the TUI has exited.
func getTargetPID() int {
	if parentPID := os.Getenv("GIT_USER_PARENT_PID"); parentPID != "" {
		if p, err := strconv.Atoi(parentPID); err == nil && p > 0 {
			return p
		}
	}
	return os.Getpid()
}

// render substitutes {PID}, {NEW}, {OLD} and {VERSION} placeholders.
func render(tmpl, newBinary, execPath, npmVersion string) string {
	return strings.NewReplacer(
		"{PID}", strconv.Itoa(getTargetPID()),
		"{NEW}", newBinary,
		"{OLD}", execPath,
		"{VERSION}", npmVersion,
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
//
// CREATE_NO_WINDOW (not DETACHED_PROCESS) is what keeps this invisible: the
// script's wait loop repeatedly shells out to tasklist/find/ping, and a
// DETACHED_PROCESS parent has no console for those console-mode children to
// inherit, so each one would allocate and flash its own new console window
// every iteration until the watched PID exits — potentially forever if the
// user leaves that process running. CREATE_NO_WINDOW instead gives the
// script a single hidden console that its children silently share.
func spawnDetached(scriptPath string) error {
	cmd := exec.Command("cmd", "/c", scriptPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow | createNewProcessGroup}
	return cmd.Start()
}
