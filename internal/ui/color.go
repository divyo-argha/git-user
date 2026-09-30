package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// NoColorRequested reports whether colour output should be disabled, following
// https://no-color.org (any non-empty NO_COLOR) plus TERM=dumb.
func NoColorRequested() bool {
	return os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
}

// StripNoColorFlag removes a --no-color flag from args and reports whether it
// was present. Arguments after a literal "--" are left alone so commands like
// `git-user exec -- tool --no-color` still reach the wrapped tool.
func StripNoColorFlag(args []string) ([]string, bool) {
	out := make([]string, 0, len(args))
	found := false
	for i, a := range args {
		if a == "--" {
			out = append(out, args[i:]...)
			break
		}
		if a == "--no-color" {
			found = true
			continue
		}
		out = append(out, a)
	}
	return out, found
}

// DisableColor switches all styled output (CLI and TUI) to plain text. It also
// exports NO_COLOR so child processes spawned by git-user honour it.
func DisableColor() {
	_ = os.Setenv("NO_COLOR", "1")
	lipgloss.SetColorProfile(termenv.Ascii)
}

// ApplyColorPreference honours --no-color and the NO_COLOR/TERM=dumb
// environment, returning args with the flag removed. Call it once at startup,
// before any output is rendered.
func ApplyColorPreference(args []string) []string {
	args, flag := StripNoColorFlag(args)
	if flag || NoColorRequested() {
		DisableColor()
	}
	return args
}
