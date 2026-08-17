package screens

import (
	"fmt"
	"os/exec"
	"strings"
)

// clipboardWrite writes text to the system clipboard using the first available
// clipboard tool: pbcopy (macOS), wl-copy (Wayland), xclip/xsel (Linux/X11),
// or clip.exe / PowerShell (Windows).
func clipboardWrite(text string) error {
	tools := [][]string{
		{"pbcopy"},
		{"wl-copy"},
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"clip.exe"},
		{"clip"},
		{"powershell.exe", "-NoProfile", "-Command", "Set-Clipboard"},
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
