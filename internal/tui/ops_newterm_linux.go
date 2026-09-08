//go:build linux

package tui

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// linuxTerminalSpec describes how to invoke one terminal emulator so it opens
// a new window running a given command.
type linuxTerminalSpec struct {
	bin       string
	buildArgs func(exe, name string) []string
}

// spawnIdentityTerminal tries a list of common Linux terminal emulators, in
// roughly descending order of how likely each is to be installed, and opens
// the first one found on PATH. $TERMINAL — a de-facto convention some window
// managers and dotfiles set to the user's preferred emulator — is tried
// first when set.
func spawnIdentityTerminal(exePath, name string) error {
	specs := []linuxTerminalSpec{
		{"gnome-terminal", func(exe, n string) []string { return []string{"--", exe, "shell", n} }},
		{"konsole", func(exe, n string) []string { return []string{"-e", exe, "shell", n} }},
		{"xfce4-terminal", func(exe, n string) []string { return []string{"-x", exe, "shell", n} }},
		{"terminator", func(exe, n string) []string { return []string{"-x", exe, "shell", n} }},
		{"tilix", func(exe, n string) []string { return []string{"-e", shellJoin(exe, "shell", n)} }},
		{"alacritty", func(exe, n string) []string { return []string{"-e", exe, "shell", n} }},
		{"kitty", func(exe, n string) []string { return []string{exe, "shell", n} }},
		{"wezterm", func(exe, n string) []string { return []string{"start", "--", exe, "shell", n} }},
		{"foot", func(exe, n string) []string { return []string{exe, "shell", n} }},
		{"x-terminal-emulator", func(exe, n string) []string { return []string{"-e", exe, "shell", n} }},
		{"xterm", func(exe, n string) []string { return []string{"-e", exe, "shell", n} }},
	}

	if term := os.Getenv("TERMINAL"); term != "" {
		specs = append([]linuxTerminalSpec{
			{term, func(exe, n string) []string { return []string{"-e", exe, "shell", n} }},
		}, specs...)
	}

	for _, spec := range specs {
		binPath, err := exec.LookPath(spec.bin)
		if err != nil {
			continue
		}
		cmd := exec.Command(binPath, spec.buildArgs(exePath, name)...)
		// Setsid detaches the new terminal from this process's session so it
		// survives (and never receives signals meant for) the TUI process —
		// the whole point is for it to keep running independently.
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no supported terminal emulator found on PATH")
}
