package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// NoColorRequested reports whether color output should be disabled (via NO_COLOR or TERM=dumb).
func NoColorRequested() bool {
	return os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
}

// StripNoColorFlag removes --no-color from args and reports whether it was present.
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

// DisableColor switches all styled output to plain text and sets NO_COLOR=1.
func DisableColor() {
	_ = os.Setenv("NO_COLOR", "1")
	lipgloss.SetColorProfile(termenv.Ascii)
}

// ApplyColorPreference applies --no-color or NO_COLOR settings, returning updated args.
func ApplyColorPreference(args []string) []string {
	args, flag := StripNoColorFlag(args)
	if flag || NoColorRequested() {
		DisableColor()
	}
	return args
}
