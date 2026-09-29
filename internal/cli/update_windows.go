//go:build windows

package cli

import (
	"fmt"
	"io"
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
// On Windows, an executing binary is locked against modification and deletion,
// but the Windows filesystem permits renaming an active executable. We rename
// the currently-running executable to a backup path (.old), move the new binary
// into place, and clean up or defer the deletion of the old binary.
//
// This avoids spawning external background batch scripts or console utilities,
// preventing rogue console tabs and window flashes in Windows Terminal.
func installBinary(execPath, newBinary string) (string, error) {
	// Clean up any stale .old files from previous runs
	cleanStaleBackups(execPath)

	oldPath := execPath + ".old"
	_ = os.Remove(oldPath)

	if err := os.Rename(execPath, oldPath); err != nil {
		// If renaming to .old failed (e.g. an existing .old is locked by another process),
		// try with a unique PID suffix.
		oldPath = fmt.Sprintf("%s.old.%d", execPath, os.Getpid())
		_ = os.Remove(oldPath)
		if err := os.Rename(execPath, oldPath); err != nil {
			return "", fmt.Errorf("backing up current binary: %w", err)
		}
	}

	if err := moveOrCopy(newBinary, execPath); err != nil {
		// Rollback: restore backup
		_ = os.Rename(oldPath, execPath)
		return "", fmt.Errorf("installing new binary: %w", err)
	}

	// Attempt to delete the old binary. On Windows this might fail with an
	// access error while the current process is still open; that is harmless
	// and will be cleaned up on the next run.
	_ = os.Remove(oldPath)

	return "", nil
}

// cleanStaleBackups removes any leftover .old backup binaries in the binary's directory.
func cleanStaleBackups(execPath string) {
	dir := filepath.Dir(execPath)
	base := filepath.Base(execPath)
	matches, err := filepath.Glob(filepath.Join(dir, base+".old*"))
	if err == nil {
		for _, m := range matches {
			_ = os.Remove(m)
		}
	}
}

// moveOrCopy moves src to dst, falling back to copy+delete if across volumes.
func moveOrCopy(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		_ = os.Remove(dst)
		return err
	}

	_ = os.Remove(src)
	return nil
}

// scheduleNpmUpdateWindows hands an npm update to a background process.
// version must already be validated (isSafeNpmVersion) by the caller.
//
// To prevent locking conflicts with npm, we first rename the running binary
// to .old. To avoid Windows Terminal popping up cmd tabs (which occurs when
// console utilities like find/tasklist are run in detached mode without a console),
// we execute the npm update via wscript (a Windows GUI subsystem host that allocates
// no console window or tabs), falling back to a hidden PowerShell launcher.
func scheduleNpmUpdateWindows(version string) error {
	if version == "" || (version != "latest" && !isSafeNpmVersion(version)) {
		return fmt.Errorf("refusing to schedule npm update: unsafe version %q", version)
	}

	// Rename the current binary to .old immediately so npm can overwrite the target
	// binary without encountering an OS file lock.
	if execPath, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
			execPath = resolved
		}
		cleanStaleBackups(execPath)
		_ = os.Rename(execPath, execPath+".old")
	}

	targetPID := getTargetPID()

	// 1. Try wscript (built-in Windows GUI subsystem host, completely windowless and tabless)
	wscriptExe := findWscriptExe()
	if wscriptExe != "" {
		vbsTmpl := `Option Explicit
Dim wmi, procs, sh, fso, targetPID, i
targetPID = {PID}
If targetPID > 0 Then
  On Error Resume Next
  Set wmi = GetObject("winmgmts:")
  If Not wmi Is Nothing Then
    For i = 1 To 60
      Set procs = wmi.ExecQuery("Select ProcessId From Win32_Process Where ProcessId = " & targetPID)
      If procs Is Nothing Then Exit For
      If procs.Count = 0 Then Exit For
      WScript.Sleep 1000
    Next
  End If
  On Error Goto 0
End If
Set sh = CreateObject("WScript.Shell")
sh.Run "cmd.exe /c npm install -g git-userhub@{VERSION}", 0, True
Set fso = CreateObject("Scripting.FileSystemObject")
On Error Resume Next
fso.DeleteFile WScript.ScriptFullName
`
		content := strings.NewReplacer(
			"{PID}", strconv.Itoa(targetPID),
			"{VERSION}", version,
		).Replace(vbsTmpl)

		if vbsPath, err := writeWindowsScript(os.TempDir(), "git-user-npm-update-*.vbs", content); err == nil {
			cmd := exec.Command(wscriptExe, "//B", "//Nologo", vbsPath)
			cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
			if err := cmd.Start(); err == nil {
				return nil
			}
			_ = os.Remove(vbsPath)
		}
	}

	// 2. Fallback to PowerShell if wscript is unavailable or restricted
	psCmd := fmt.Sprintf(
		`$p=%d; if ($p -gt 0) { try { Wait-Process -Id $p -Timeout 60 -ErrorAction SilentlyContinue } catch {} }; Start-Process -WindowStyle Hidden -FilePath "cmd.exe" -ArgumentList "/c npm install -g git-userhub@%s"`,
		targetPID,
		version,
	)
	fallbackCmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psCmd)
	fallbackCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow | createNewProcessGroup}
	if err := fallbackCmd.Start(); err == nil {
		return nil
	}

	// 3. Last-resort fallback: direct detached cmd.exe running npm
	directCmd := exec.Command("cmd.exe", "/c", "npm install -g git-userhub@"+version)
	directCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow | createNewProcessGroup}
	return directCmd.Start()
}

// findWscriptExe locates wscript.exe on the system.
func findWscriptExe() string {
	if p, err := exec.LookPath("wscript.exe"); err == nil {
		return p
	}
	sysRoot := os.Getenv("SystemRoot")
	if sysRoot == "" {
		sysRoot = `C:\Windows`
	}
	candidate := filepath.Join(sysRoot, "System32", "wscript.exe")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

// getTargetPID returns the PID of the process to wait for before performing
// the npm update. If launched from the TUI, it returns the TUI's parent PID
// so the update only runs after the TUI has exited.
func getTargetPID() int {
	if parentPID := os.Getenv("GIT_USER_PARENT_PID"); parentPID != "" {
		if p, err := strconv.Atoi(parentPID); err == nil && p > 0 {
			return p
		}
	}
	return os.Getpid()
}

// writeWindowsScript writes a script file (with CRLF endings) and returns its path.
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
