package logo

import (
	"strings"
)

var SmallPixelLines = []string{}

// NewSmallPixelLines is the handcrafted high-contrast vector block logo representing gu_logo.png.
var NewSmallPixelLines = []string{
	"  \x1b[38;2;249;115;22m▄▀▀ █ ▀█▀\x1b[0m       \x1b[38;2;226;232;240m█ █ ▀▀▀ █▀▀ █▀▄\x1b[0m",
	"  \x1b[38;2;249;115;22m█ ▄ █  █ \x1b[0m  \x1b[38;2;148;163;184m▄▄▄\x1b[0m  \x1b[38;2;226;232;240m█ █ ▀▀▄ █▀  █▀▀\x1b[0m",
	"  \x1b[38;2;249;115;22m▀▀▀ ▀  ▀ \x1b[0m       \x1b[38;2;226;232;240m▀▀▀ ▀▀▀ ▀▀▀ ▀ ▀\x1b[0m",
}

// IsInlineGraphicsSupported checks if the terminal supports native inline PNG image protocols.
func IsInlineGraphicsSupported() bool {
	return false
}

// GetTrimmedLogo returns NewSmallPixelLines without leading/trailing empty lines.
func GetTrimmedLogo() []string {
	var out []string
	for _, line := range NewSmallPixelLines {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
