package screens

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
)

// clipboardWriteFn is the seam copyToClipboardCmd calls through — swapped in
// tests to assert on exactly what text was copied without touching a real
// clipboard.
var clipboardWriteFn = ClipboardWrite

// ClipboardWrite writes text to the system clipboard.
// It uses native clipboard APIs via github.com/atotto/clipboard (direct Win32 API on Windows,
// pbcopy on macOS, wl-copy/xclip/xsel on Linux), with fallbacks to CLI tools.
func ClipboardWrite(text string) error {
	if err := clipboard.WriteAll(text); err == nil {
		return nil
	}

	tools := [][]string{
		{"pbcopy"},
		{"wl-copy"},
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"clip.exe"},
		{"clip"},
		{"powershell.exe", "-NoProfile", "-Command", "$input | Set-Clipboard"},
	}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool[0]); err != nil {
			continue
		}
		cmd := exec.Command(tool[0], tool[1:]...) //nolint:gosec
		cmd.Stdin = strings.NewReader(text)
		if _, err := cmd.CombinedOutput(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no clipboard tool found (tried pbcopy, wl-copy, xclip, xsel, clip)")
}
